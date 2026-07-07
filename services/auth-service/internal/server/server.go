package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/auth-service/internal/crypto"
	"github.com/helixdevelopment/auth-service/internal/handler"
	"github.com/helixdevelopment/auth-service/internal/repository"
	"github.com/helixdevelopment/auth-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

// Server wraps the Gin engine with auth service functionality
type Server struct {
	router     *gin.Engine
	logger     Logger
	handler    *handler.Handler
	jwtManager *crypto.JWTManager
}

// New creates a new Auth Server with dependencies
func New(logger Logger) (*Server, error) {
	if logger == nil {
		logger = &defaultLogger{}
	}

	// Initialize JWT manager with Ed25519
	jwtManager, err := crypto.NewJWTManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT manager: %w", err)
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
			// steady-state pool's unqualified "users" queries resolve
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

	// If no DB, create a mock repository for testing
	if repo == nil {
		logger.Println("warning: no database connection, using in-memory mode")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Create handler
	h := handler.New(repo, jwtManager)

	s := &Server{
		router:     r,
		logger:     logger,
		handler:    h,
		jwtManager: jwtManager,
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

	// Auth routes (no auth required). /mfa/verify deliberately stays
	// here even though its handler needs a specific userID: Login()
	// withholds real tokens for an MFA-enabled user until MFA
	// verification succeeds, so the caller completing login has no
	// bearer token to present yet. VerifyMFA resolves the user from the
	// request's challengeId (bound to a user at /login time) rather
	// than from an authenticated-request context value - see
	// internal/handler/mfa_challenge.go.
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/mfa/verify", h.VerifyMFA)
	r.POST("/refresh", h.RefreshToken)
	r.POST("/validate", h.ValidateToken)

	// Authenticated routes - require a valid, non-revoked bearer access
	// token. /logout and /mfa/setup live here (not in the "no auth
	// required" block above) because their handlers resolve the acting
	// user from the authenticated userID the middleware sets on the
	// context: a logout call has no session to revoke, and an MFA-setup
	// call has no account to enable MFA for, without an authenticated
	// caller. Registering either outside this group means "userID" is
	// never set on the context, so the handler's userID lookup silently
	// resolves to nothing and every call fails - the bug both routes
	// had before this fix.
	auth := r.Group("/")
	auth.Use(s.jwtValidationMiddleware())
	{
		auth.POST("/logout", h.Logout)
		auth.POST("/mfa/setup", h.SetupMFA)
		// TODO: add authenticated routes (profile, sessions, etc.)
		auth.GET("/me", func(c *gin.Context) {
			userID, _ := c.Get("userID")
			c.JSON(http.StatusOK, gin.H{"userId": userID})
		})
	}

	return s, nil
}

// Router exposes the underlying engine for testing
func (s *Server) Router() http.Handler {
	return s.router
}

// JWTManager exposes the JWT manager
func (s *Server) JWTManager() *crypto.JWTManager {
	return s.jwtManager
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

		s.logger.Printf("[AUTH] %v | %3d | %13v | %15s | %-7s %s",
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

func (s *Server) jwtValidationMiddleware() gin.HandlerFunc {
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

		claims, err := s.jwtManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// JWT signature validation alone is stateless: a token revoked
		// by a prior /logout still verifies cryptographically until it
		// naturally expires. Reject it here too so a replayed,
		// logged-out access token is genuinely denied, not just an
		// unenforced session row in the database.
		if !s.handler.IsAccessTokenActive(c.Request.Context(), token) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("orgID", claims.OrgID)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Next()
	}
}
