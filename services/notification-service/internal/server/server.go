package server

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/notification-service/internal/handler"
	"github.com/helixdevelopment/notification-service/internal/repository"
	"github.com/helixdevelopment/notification-service/migrations"
)

// Claims represents the identity claims extracted from a validated JWT.
// Mirrors gateway-service's Claims (services/gateway-service/internal/
// server/server.go) and billing-service's Claims (services/billing-service/
// internal/server/server.go) — the gateway forwards the caller's original
// signed Authorization bearer token to every proxied upstream untouched
// (proxyTo clones the client's request headers verbatim via
// c.Request.Header.Clone() and never strips or replaces Authorization, nor
// does it ever inject a service API key), and gateway-service DOES proxy
// real end-user notification traffic to notification-service the same way
// (services/gateway-service/internal/server/server.go: `api.POST(
// "/notifications", s.proxyTo("notification-service", ...))`). So
// notification-service independently validates the SAME token with the
// SAME public key to obtain the caller's identity (T11), exactly as
// billing-service does for T12, rather than demanding a header (X-API-Key)
// that no real caller in this request path ever sends.
type Claims struct {
	UserID string `json:"userId"`
	OrgID  string `json:"orgId,omitempty"`
	jwt.RegisteredClaims
}

// Logger interface for logging
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type defaultLogger struct{}

func (d *defaultLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

func (d *defaultLogger) Println(v ...interface{}) {
	fmt.Println(v...)
}

// Server wraps the Gin engine with notification service functionality
type Server struct {
	router       *gin.Engine
	logger       Logger
	handler      *handler.Handler
	jwtPublicKey ed25519.PublicKey
}

// New creates a new Notification Server with dependencies
func New(logger Logger) (*Server, error) {
	if logger == nil {
		logger = &defaultLogger{}
	}

	// Initialize database connection
	var repo *repository.Repository
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		// Apply pending schema migrations before opening the steady-state
		// pool. A migration failure (including a dirty schema state) MUST
		// NOT be served against, so on failure we deliberately skip pool
		// creation and fall through to in-memory mode below, matching this
		// service's existing degrade-gracefully-on-DB-trouble behaviour.
		if version, merr := migrations.Run(dbURL, logger); merr != nil {
			logger.Printf("warning: failed to apply database migrations: %v", merr)
		} else {
			logger.Printf("database migrations applied - schema version %d", version)

			// Use the same schema-scoped connection URL the migrator
			// applied (search_path=migrations.Schema) so the
			// steady-state pool's unqualified "notifications" /
			// "notification_preferences" queries resolve against the
			// schema migrations.Run just migrated, not the shared
			// database's default "public" schema (schema-per-service,
			// GAP-01).
			poolURL, perr := migrations.ConnectionURL(dbURL)
			if perr != nil {
				logger.Printf("warning: failed to build schema-scoped connection URL: %v", perr)
			} else {
				pool, err := pgxpool.New(context.Background(), poolURL)
				if err != nil {
					logger.Printf("warning: failed to connect to database: %v", err)
				} else {
					repo = repository.New(pool)
				}
			}
		}
	}

	if repo == nil {
		logger.Println("warning: no database connection, using in-memory mode")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Create handler
	h := handler.New(repo)

	s := &Server{
		router:  r,
		logger:  logger,
		handler: h,
	}

	// Load JWT public key from environment — the SAME JWT_PUBLIC_KEY secret
	// gateway-service and billing-service are provisioned with (services/
	// gateway-service/internal/server/server.go, services/billing-service/
	// internal/server/server.go), so a token minted by auth-service and
	// validated at the edge validates identically here.
	if pubKeyB64 := os.Getenv("JWT_PUBLIC_KEY"); pubKeyB64 != "" {
		pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyB64)
		if err != nil {
			logger.Printf("warning: failed to decode JWT_PUBLIC_KEY: %v", err)
		} else if len(pubKeyBytes) != ed25519.PublicKeySize {
			logger.Printf("warning: invalid JWT_PUBLIC_KEY size: expected %d, got %d", ed25519.PublicKeySize, len(pubKeyBytes))
		} else {
			s.jwtPublicKey = ed25519.PublicKey(pubKeyBytes)
		}
	}

	// Global middleware
	r.Use(s.recoveryMiddleware())
	r.Use(s.requestIDMiddleware())
	r.Use(s.loggingMiddleware())
	r.Use(s.corsMiddleware())

	// Health endpoints (no auth required)
	r.GET("/healthz/live", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/healthz", h.HealthCheck)

	// Notification routes — require a valid service-to-service API key.
	// This closes the open-relay gap the real email/webhook delivery sinks
	// introduced: without authentication, CreateNotification could be
	// abused by anyone as an unauthenticated spam / SSRF-amplification
	// relay (Constitution §11.4.133 security-hardening finding). The whole
	// group is protected, not just CreateNotification, mirroring
	// vault-service's convention and closing the related gap where any
	// caller could otherwise list/read/delete another user's notifications
	// by simply supplying their user_id.
	api := r.Group("/api/v1/notifications")
	api.Use(s.authMiddleware())
	{
		api.POST("", h.CreateNotification)
		api.GET("", h.ListNotifications)
		api.GET("/unread-count", h.CountUnread)
		api.GET("/:id", h.GetNotification)
		api.POST("/:id/read", h.MarkRead)
		api.POST("/read-all", h.MarkAllRead)
		api.DELETE("/:id", h.DeleteNotification)
		api.GET("/preferences", h.GetPreference)
		api.PUT("/preferences", h.UpdatePreference)
	}

	return s, nil
}

// Router exposes the underlying engine for testing
func (s *Server) Router() http.Handler {
	return s.router
}

// --- Middleware ---

func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.Recovery()
}

// authMiddleware validates the caller's bearer JWT and extracts its
// userID/orgID claims into the gin context (T11), aligning
// notification-service with the SAME canonical Ed25519 JWT_PUBLIC_KEY chain
// gateway-service and billing-service use (T12) instead of the previous
// X-API-Key scheme.
//
// FORENSIC FINDING (T11, confirmed by reading gateway-service's proxyTo,
// services/gateway-service/internal/server/server.go:1097-1152): the
// gateway clones the caller's original request headers verbatim
// (`proxyReq.Header = c.Request.Header.Clone()`) and forwards the caller's
// signed "Authorization: Bearer <Ed25519-JWT>" untouched to
// notification-service; it never sets an X-API-Key header. The prior
// X-API-Key-based authMiddleware therefore rejected EVERY real end-user
// notification request routed through the canonical gateway path with 401,
// regardless of how valid the caller's auth-service-issued JWT was — an
// unauthenticated-in-practice open-relay/soft-lockout defect at the same
// time (real callers could never reach it; only a caller holding the
// separate, never-issued NOTIFICATION_SERVICE_API_KEY secret could). This
// middleware closes that gap by validating the SAME forwarded token the
// gateway already validated, the SAME way billing-service does, so it
// fails CLOSED on a missing/invalid/wrongly-signed token (never silently
// allowing unauthenticated access to a channel that performs real outbound
// SMTP/HTTP delivery) while genuinely accepting the real, canonical caller
// identity.
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: invalid authorization header format"})
			return
		}

		token := parts[1]
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: empty token"})
			return
		}

		if s.jwtPublicKey == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: JWT validation not configured"})
			return
		}

		claims, err := s.validateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: invalid token"})
			return
		}

		if claims.UserID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: token missing user identity"})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("orgID", claims.OrgID)
		c.Next()
	}
}

// validateToken parses and verifies tokenString exactly the way
// gateway-service/billing-service do: it MUST be signed with Ed25519
// (EdDSA) — any other alg is rejected outright, never silently accepted —
// and MUST verify against s.jwtPublicKey.
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

func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		s.logger.Printf("[NOTIFY] %v | %3d | %13v | %15s | %-7s %s",
			start.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	allowedOrigins := parseCORSAllowedOrigins()
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if isAllowedOrigin(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-API-Key")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func parseCORSAllowedOrigins() []string {
	env := os.Getenv("CORS_ALLOWED_ORIGINS")
	if env == "" {
		return []string{"http://localhost:3000"}
	}
	var origins []string
	for _, o := range strings.Split(env, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

func isAllowedOrigin(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}
