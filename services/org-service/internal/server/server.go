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

	"github.com/helixdevelopment/org-service/internal/handler"
	"github.com/helixdevelopment/org-service/internal/repository"
	"github.com/helixdevelopment/org-service/migrations"
)

// Claims represents the identity claims extracted from a validated JWT.
// Mirrors gateway-service's and billing-service's Claims — the gateway
// forwards the caller's original signed Authorization bearer token to upstream
// services untouched, so org-service independently validates the SAME token
// with the SAME public key (JWT_PUBLIC_KEY) to obtain the caller's identity
// (T19). This closes the previous fail-OPEN hole where a request with no
// Authorization header was silently injected a default
// "00000000-0000-0000-0000-000000000000" userID/orgID and served
// unauthenticated.
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

// Server wraps the Gin engine with org service functionality.
type Server struct {
	router       *gin.Engine
	logger       Logger
	handler      *handler.Handler
	repo         *repository.Repository
	jwtPublicKey ed25519.PublicKey
}

// New creates a new Server with dependencies.
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
			// steady-state pool's unqualified queries resolve
			// against the schema migrations.Run just migrated, not
			// the shared database's default "public" schema
			// (schema-per-service, GAP-01).
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

	// Load JWT public key from environment — the same JWT_PUBLIC_KEY secret
	// gateway-service and billing-service are provisioned with, so a token
	// minted by auth-service and validated at the edge validates identically
	// here. A missing/malformed key leaves s.jwtPublicKey nil, and the
	// authMiddleware then fails CLOSED (rejects every request) rather than
	// serving unauthenticated traffic.
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

	// API routes (no auth required for this service scaffold)
	api := r.Group("/api/v1")
	api.Use(s.authMiddleware())
	{
		// Organization routes
		api.POST("/orgs", h.CreateOrg)
		api.GET("/orgs", h.ListOrgs)
		api.GET("/orgs/:id", h.GetOrg)
		api.GET("/orgs/by-slug/:slug", h.GetOrgBySlug)
		api.PUT("/orgs/:id", h.UpdateOrg)
		api.DELETE("/orgs/:id", h.DeleteOrg)

		// Team routes under org
		api.POST("/orgs/:id/teams", h.CreateTeam)
		api.GET("/orgs/:id/teams", h.ListTeams)

		// Team routes standalone
		api.GET("/teams/:id", h.GetTeam)
		api.PUT("/teams/:id", h.UpdateTeam)
		api.DELETE("/teams/:id", h.DeleteTeam)

		// Membership routes
		api.POST("/orgs/:id/members", h.AddMember)
		api.GET("/orgs/:id/members", h.ListMembers)
		api.PUT("/orgs/:id/members/:user_id", h.UpdateMember)
		api.DELETE("/orgs/:id/members/:user_id", h.RemoveMember)
	}

	return s, nil
}

// Router exposes the underlying engine for testing.
func (s *Server) Router() http.Handler {
	return s.router
}

// Run starts the HTTP server.
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// --- Middleware ---

func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.Recovery()
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

		s.logger.Printf("[ORG] %v | %3d | %13v | %15s | %-7s %s",
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

// authMiddleware validates the caller's bearer JWT on every /api/v1 org/team/
// membership route (T19). It fails CLOSED: a request with no Authorization
// header, a malformed header, an empty/invalid/expired/wrongly-signed token, or
// an unconfigured JWT_PUBLIC_KEY is rejected 401 outright — never served with a
// default userID/orgID injected. On success the validated claims are set into
// the gin context under the same "userID"/"orgID" keys the handlers read.
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

// validateToken parses and verifies tokenString the same way gateway-service
// and billing-service do: it MUST be signed with Ed25519 (EdDSA) — any other
// alg is rejected outright — and MUST verify against s.jwtPublicKey.
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
