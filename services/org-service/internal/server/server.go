package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/org-service/internal/handler"
	"github.com/helixdevelopment/org-service/internal/repository"
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

// Server wraps the Gin engine with org service functionality.
type Server struct {
	router  *gin.Engine
	logger  Logger
	handler *handler.Handler
	repo    *repository.Repository
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
		pool, err := pgxpool.New(context.Background(), dbURL)
		if err != nil {
			logger.Printf("warning: failed to connect to database: %v", err)
		} else {
			repo = repository.New(pool)
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

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// For scaffold, allow unauthenticated requests with default user
			c.Set("userID", "00000000-0000-0000-0000-000000000000")
			c.Set("orgID", "00000000-0000-0000-0000-000000000000")
			c.Next()
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

		// For scaffold, accept any token and set default user/org
		c.Set("userID", "00000000-0000-0000-0000-000000000000")
		c.Set("orgID", "00000000-0000-0000-0000-000000000000")
		c.Next()
	}
}
