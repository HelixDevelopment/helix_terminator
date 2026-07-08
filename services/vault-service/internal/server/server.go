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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/vault-service/internal/handler"
	"github.com/helixdevelopment/vault-service/internal/repository"
	"github.com/helixdevelopment/vault-service/migrations"
)

// Claims represents the identity claims extracted from a validated JWT.
// Mirrors gateway-service's Claims (services/gateway-service/internal/
// server/server.go) and billing-service's Claims (services/billing-service/
// internal/server/server.go) — the gateway forwards the original signed
// Authorization bearer token to upstream services untouched (proxyTo clones
// the client's request headers verbatim and never strips Authorization), so
// vault-service independently validates the SAME token with the SAME public
// key rather than demanding a separate service-to-service X-API-Key the
// gateway never sends (T19; same defect class as T11's notification-service
// finding).
type Claims struct {
	UserID string `json:"userId"`
	OrgID  string `json:"orgId,omitempty"`
	jwt.RegisteredClaims
}

// Logger interface for logging.
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

// Server wraps the Gin engine with vault service functionality.
type Server struct {
	router       *gin.Engine
	logger       Logger
	handler      *handler.Handler
	repo         *repository.Repository
	jwtPublicKey ed25519.PublicKey
}

// New creates a new Vault Server with dependencies.
func New(logger Logger) (*Server, error) {
	if logger == nil {
		logger = &defaultLogger{}
	}

	var repo *repository.Repository
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		// Apply pending schema migrations before opening the steady-state
		// pool. A migration failure (including a dirty schema state) MUST
		// NOT be served against, so on failure we deliberately skip pool
		// creation and fall through to in-memory mode below, matching this
		// service's existing degrade-gracefully-on-DB-trouble behaviour.
		// This does NOT touch secret material - Run only creates/alters
		// schema objects (extension/tables/indexes/triggers); the vault's
		// own encryption-at-rest of encrypted_value stays entirely in
		// this service's repository/crypto layer, unchanged.
		if version, merr := migrations.Run(dbURL, logger); merr != nil {
			logger.Printf("warning: failed to apply database migrations: %v", merr)
		} else {
			logger.Printf("database migrations applied - schema version %d", version)

			// Use the same schema-scoped connection URL the migrator
			// applied (search_path=migrations.Schema) so the
			// steady-state pool's unqualified "secrets" /
			// "secret_versions" queries resolve against the schema
			// migrations.Run just migrated, not the shared database's
			// default "public" schema (schema-per-service, GAP-01).
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

	h := handler.New(repo)

	s := &Server{
		router:  r,
		logger:  logger,
		handler: h,
		repo:    repo,
	}

	// Load JWT public key from environment — same JWT_PUBLIC_KEY secret
	// gateway-service and billing-service are provisioned with, so a token
	// minted by auth-service and validated at the edge validates identically
	// here.
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
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	// Vault routes — all require a valid service-to-service API key (§security).
	v1 := r.Group("/api/v1/vault")
	v1.Use(s.authMiddleware())
	{
		// Collection-level routes additionally require a valid caller
		// identity (X-User-ID) before the handler runs: ListSecrets and
		// CreateSecret have no target secret ID for a pre-handler ownership
		// lookup (unlike the secret-ID-scoped routes below), so the
		// handlers themselves derive/enforce the authoritative tenant scope
		// from that identity (see handler.CallerUserID) — this middleware
		// guarantees a valid identity reaches them at all, short-circuiting
		// unauthenticated collection requests at the router layer just like
		// tenantIsolationMiddleware does for the ID-scoped routes (T7).
		v1.POST("/secrets", s.requireCallerIdentityMiddleware(), h.CreateSecret)
		v1.GET("/secrets", s.requireCallerIdentityMiddleware(), h.ListSecrets)
		// Secret-ID-scoped routes additionally require tenant isolation: the
		// caller-asserted X-User-ID MUST match the target secret's owner, so
		// one tenant can never read/modify/rotate another tenant's secret.
		v1.GET("/secrets/:id", s.tenantIsolationMiddleware(), h.GetSecret)
		v1.PUT("/secrets/:id", s.tenantIsolationMiddleware(), h.UpdateSecret)
		v1.DELETE("/secrets/:id", s.tenantIsolationMiddleware(), h.DeleteSecret)
		v1.GET("/secrets/:id/versions", s.tenantIsolationMiddleware(), h.GetSecretVersions)
		v1.POST("/secrets/:id/rotate", s.tenantIsolationMiddleware(), h.RotateSecret)
	}

	return s, nil
}

// Router exposes the underlying engine for testing.
func (s *Server) Router() http.Handler {
	return s.router
}

// --- Middleware ---

func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.Recovery()
}

// authMiddleware validates the caller's bearer JWT on every /api/v1/vault/*
// route (T19). This is the canonical Ed25519 JWT_PUBLIC_KEY chain shared
// with gateway-service and billing-service — the gateway forwards the
// caller's original signed Authorization bearer token through to upstream
// services untouched, so vault-service must validate that SAME token rather
// than demand a service-to-service X-API-Key the gateway never sends
// (previously this middleware rejected every real gateway-routed request).
// Requests with no token, a malformed token, a token that fails
// signature/expiry validation, or an unconfigured JWT_PUBLIC_KEY are
// rejected outright (fail-closed) rather than served unauthenticated.
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

// requireCallerIdentityMiddleware enforces that a caller of a tenant-scoped
// collection route (ListSecrets, CreateSecret) presents a valid X-User-ID
// header before the handler runs. It shares its identity-parsing logic with
// tenantIsolationMiddleware via handler.CallerUserID (§11.4.124
// reuse-don't-duplicate) rather than re-implementing header validation.
func (s *Server) requireCallerIdentityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := handler.CallerUserID(c); !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid X-User-ID"})
			return
		}
		c.Next()
	}
}

// tenantIsolationMiddleware enforces object-level access control on
// secret-ID-scoped routes: the caller MUST present a valid X-User-ID header,
// and that identity MUST match the target secret's owning user_id. A
// mismatch returns 404 (not 403) so the endpoint never confirms or denies
// the existence of another tenant's secret to an unauthorized caller.
func (s *Server) tenantIsolationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			// Malformed ID: let the handler produce its own 400.
			c.Next()
			return
		}

		callerID, ok := handler.CallerUserID(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid X-User-ID"})
			return
		}

		if s.repo == nil {
			// No database wired (e.g. degraded mode) — defer to the handler.
			c.Next()
			return
		}

		secret, err := s.repo.GetSecretByID(c.Request.Context(), id)
		if err != nil {
			// Secret does not exist (or already soft-deleted) — let the
			// handler return its own not-found response.
			c.Next()
			return
		}
		if secret.UserID != callerID {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "secret not found"})
			return
		}
		c.Next()
	}
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

		s.logger.Printf("[VAULT] %v | %3d | %13v | %15s | %-7s %s",
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
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
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
		return nil
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
