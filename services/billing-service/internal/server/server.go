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
	"github.com/helixdevelopment/billing-service/internal/billing"
	"github.com/helixdevelopment/billing-service/internal/handler"
	"github.com/helixdevelopment/billing-service/internal/repository"
)

// Claims represents the identity claims extracted from a validated JWT.
// Mirrors gateway-service's Claims (services/gateway-service/internal/
// server/server.go) — the gateway forwards the original signed
// Authorization bearer token to upstream services untouched (proxyTo
// clones the client's request headers verbatim and never strips
// Authorization), so billing-service independently validates the SAME
// token with the SAME public key to obtain the caller's tenant identity
// (T12) rather than ever trusting a client-supplied query parameter for
// tenant scoping.
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
	// gateway-service is provisioned with (services/gateway-service/
	// internal/server/server.go), so a token minted by auth-service and
	// validated at the edge validates identically here.
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

	// Constitution §11.4 anti-bluff honest feature-flag: read
	// STRIPE_SECRET_KEY (+ STRIPE_WEBHOOK_SECRET) from the process
	// environment exactly once at startup, mirroring the JWT_PUBLIC_KEY
	// env-read pattern above. When absent, provider is nil and every
	// subscription-lifecycle-mutating handler honestly responds 501
	// "payments provider not configured" — NEVER a fabricated success.
	// When present, every subsequent request this process serves makes
	// REAL calls against the real Stripe API. See
	// internal/billing.NewProviderFromEnv + docs/guides/BILLING.md.
	provider, perr := billing.NewProviderFromEnv()
	if perr != nil {
		log.Printf("warning: failed to construct payments provider from environment: %v", perr)
	} else if provider != nil {
		log.Printf("billing-service: payments provider %q configured — subscription lifecycle calls are REAL", provider.Name())
	} else {
		log.Printf("billing-service: no payments provider configured (STRIPE_SECRET_KEY unset) — subscription-lifecycle-mutating endpoints will respond 501 Not Implemented")
	}

	h := handler.New(repo, handler.WithProvider(provider))

	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/health", h.HealthCheck)
	r.GET("/ready", h.ReadinessCheck)

	// Stripe (or any configured processor) authenticates a webhook
	// delivery via its own payload-signature scheme (Stripe-Signature
	// header + shared secret, verified inside h.StripeWebhook via
	// billing.PaymentProvider.VerifyWebhook) — NEVER a bearer JWT, so
	// this route is deliberately mounted OUTSIDE the api.Use(s.authMiddleware())
	// group below.
	r.POST("/api/v1/webhooks/stripe", h.StripeWebhook)

	api := r.Group("/api/v1")
	api.Use(s.authMiddleware())
	{
		api.POST("/subscriptions", h.CreateSubscription)
		api.GET("/subscriptions", h.ListSubscriptions)
		api.GET("/subscriptions/:id", h.GetSubscription)
		api.PUT("/subscriptions/:id", h.UpdateSubscription)
		api.POST("/subscriptions/:id/cancel", h.CancelSubscription)
		api.GET("/invoices", h.ListInvoices)
		api.GET("/invoices/:id", h.GetInvoice)
	}

	return s
}

// authMiddleware validates the caller's bearer JWT and extracts its
// userID/orgID claims into the gin context (T12). It NEVER derives tenant
// identity from client-supplied query parameters or path segments — the
// only source of truth for "which tenant is this?" is a cryptographically
// validated token claim. Requests with no token, a malformed token, a
// token that fails signature/expiry validation, or a token missing an
// orgId claim are rejected outright (fail-closed) rather than served
// unscoped or mis-scoped data.
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "empty token"})
			c.Abort()
			return
		}

		if s.jwtPublicKey == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "JWT validation not configured"})
			c.Abort()
			return
		}

		claims, err := s.validateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		if claims.OrgID == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "token missing org identity"})
			c.Abort()
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

// Router exposes the underlying engine for testing.
func (s *Server) Router() http.Handler {
	return s.router
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
		log.Printf("billing-service starting on %s", addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		log.Printf("billing-service received signal %v, shutting down gracefully", sig)
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
