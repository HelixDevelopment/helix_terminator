package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/vault-service/internal/handler"
	"github.com/helixdevelopment/vault-service/internal/repository"
	"github.com/helixdevelopment/vault-service/migrations"
)

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
	router  *gin.Engine
	logger  Logger
	handler *handler.Handler
	repo    *repository.Repository
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

// authMiddleware enforces service-to-service API key authentication on every
// /api/v1/vault/* route. The expected key is provisioned via the
// VAULT_SERVICE_API_KEY environment variable. A request with a missing,
// empty, or mismatched X-API-Key header is rejected with 401 Unauthorized.
// If VAULT_SERVICE_API_KEY is not configured at all, the service fails
// closed (every vault request is rejected) rather than silently allowing
// unauthenticated access to a secrets store.
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := os.Getenv("VAULT_SERVICE_API_KEY")
		got := c.GetHeader("X-API-Key")
		if expected == "" || got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid X-API-Key"})
			return
		}
		c.Next()
	}
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
