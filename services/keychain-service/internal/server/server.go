package server

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/helixdevelopment/keychain-service/internal/handler"
	"github.com/helixdevelopment/keychain-service/internal/repository"
)

// Claims represents the identity claims extracted from a validated JWT.
// Mirrors gateway-service's Claims (services/gateway-service/internal/
// server/server.go) and billing-service's Claims (services/billing-service/
// internal/server/server.go) — the gateway forwards the original signed
// Authorization bearer token to upstream services untouched (proxyTo clones
// the client's request headers verbatim and never strips Authorization), so
// keychain-service independently validates the SAME token with the SAME
// public key to obtain the caller's identity (T19). Prior to this fix,
// keychain-service's /api/v1/keychain routes — which store encrypted
// private keys / passphrases — had NO authentication middleware at all:
// any caller could create, list, read, update, or delete keychain items
// completely unauthenticated.
type Claims struct {
	UserID string `json:"userId"`
	OrgID  string `json:"orgId,omitempty"`
	jwt.RegisteredClaims
}

// Server wraps the Gin engine and HTTP server.
type Server struct {
	router       *gin.Engine
	srv          *http.Server
	repo         *repository.Repository
	jwtPublicKey ed25519.PublicKey
}

// New creates a new Server with routes wired.
func New(repo *repository.Repository) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	s := &Server{
		router: r,
		repo:   repo,
	}

	// Load JWT public key from environment — same JWT_PUBLIC_KEY secret
	// gateway-service and billing-service are provisioned with, so a token
	// minted by auth-service and validated at the edge validates identically
	// here.
	if pubKeyB64 := os.Getenv("JWT_PUBLIC_KEY"); pubKeyB64 != "" {
		pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyB64)
		if err != nil {
			log.Printf("warning: failed to decode JWT_PUBLIC_KEY: %v", err)
		} else if len(pubKeyBytes) != ed25519.PublicKeySize {
			log.Printf("warning: invalid JWT_PUBLIC_KEY size: expected %d, got %d", ed25519.PublicKeySize, len(pubKeyBytes))
		} else {
			s.jwtPublicKey = ed25519.PublicKey(pubKeyBytes)
		}
	}

	r.Use(gin.Recovery())
	r.Use(requestIDMiddleware())
	r.Use(loggingMiddleware())

	h := handler.New(repo)

	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/health", h.HealthCheck)
	r.GET("/ready", h.ReadinessCheck)

	// Keychain routes — store encrypted private keys / passphrases.
	// T19: this group previously had NO authentication middleware at all;
	// every route below is now gated by the canonical Ed25519 JWT chain
	// (services/billing-service/internal/server/server.go /
	// services/gateway-service/internal/server/server.go), matching the
	// gateway's forwarded Authorization bearer token.
	api := r.Group("/api/v1")
	api.Use(s.authMiddleware())
	{
		api.POST("/keychain", h.CreateItem)
		api.GET("/keychain", h.ListItems)
		api.GET("/keychain/:id", h.GetItem)
		api.PUT("/keychain/:id", h.UpdateItem)
		api.DELETE("/keychain/:id", h.DeleteItem)
	}

	return s
}

// Router exposes the underlying engine for testing.
func (s *Server) Router() http.Handler {
	return s.router
}

// authMiddleware validates the caller's bearer JWT on every /api/v1/keychain
// route (T19). This is the canonical Ed25519 JWT_PUBLIC_KEY chain shared
// with gateway-service and billing-service — the gateway forwards the
// caller's original signed Authorization bearer token through to upstream
// services untouched, so keychain-service must validate that SAME token.
// Requests with no token, a malformed token, a token that fails
// signature/expiry validation, or an unconfigured JWT_PUBLIC_KEY are
// rejected outright (fail-closed) rather than served unauthenticated —
// closing the previous gap where this route group had no auth check
// whatsoever. On success, the validated userId/orgId claims are set into
// the gin context under the SAME "userID" key the pre-existing CreateItem
// handler already reads (internal/handler/handler.go), so that handler's
// caller-identity wiring — previously always empty because nothing ever
// populated it — now genuinely receives the authenticated caller's ID.
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		token := parts[1]
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "empty token"})
			return
		}

		if s.jwtPublicKey == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "JWT validation not configured"})
			return
		}

		claims, err := s.validateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("orgID", claims.OrgID)
		c.Next()
	}
}

func (s *Server) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtPublicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

// Run starts the HTTP server with graceful shutdown.
func (s *Server) Run(addr string) error {
	s.srv = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		log.Printf("keychain-service starting on %s", addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		log.Printf("keychain-service received signal %v, shutting down gracefully", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.srv.Shutdown(ctx)
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Set("requestID", reqID)
		c.Writer.Header().Set("X-Request-ID", reqID)
		c.Next()
	}
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		reqID, _ := c.Get("requestID")
		log.Printf("[%s] %s %s %d %s %s", reqID, method, path, status, latency, clientIP)
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Printf("[ERROR] %s %s: %v", method, path, e.Err)
			}
		}
	}
}
