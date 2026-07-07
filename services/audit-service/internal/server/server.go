package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/audit-service/internal/handler"
	"github.com/helixdevelopment/audit-service/internal/repository"
	"github.com/helixdevelopment/audit-service/migrations"
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

// Server wraps the Gin engine with audit service functionality
type Server struct {
	router  *gin.Engine
	logger  Logger
	handler *handler.Handler
	repo    *repository.Repository
}

// New creates a new Audit Server with dependencies
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
			// steady-state pool's unqualified "audit_logs" queries
			// resolve against the schema migrations.Run just migrated,
			// not the shared database's default "public" schema
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
		repo = repository.New(nil)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Create handler
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
	r.GET("/healthz/live", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/healthz", h.HealthCheck)

	// Audit routes
	r.POST("/api/v1/audit/logs", h.CreateAuditLog)
	r.GET("/api/v1/audit/logs", h.ListAuditLogs)
	r.GET("/api/v1/audit/logs/:id", h.GetAuditLog)
	r.GET("/api/v1/audit/stats/actions", h.CountByAction)
	r.GET("/api/v1/audit/stats/resources", h.CountByResourceType)

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

		s.logger.Printf("[AUDIT] %v | %3d | %13v | %15s | %-7s %s",
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
