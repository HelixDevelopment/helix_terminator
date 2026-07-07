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

	"github.com/helixdevelopment/pki-service/internal/handler"
	"github.com/helixdevelopment/pki-service/internal/repository"
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

// Server wraps the Gin engine with PKI service functionality.
type Server struct {
	router  *gin.Engine
	logger  Logger
	handler *handler.Handler
	repo    repository.Repository
}

// New creates a new PKI Server with dependencies.
func New(logger Logger) (*Server, error) {
	if logger == nil {
		logger = &defaultLogger{}
	}

	// Initialize database connection
	var repo repository.Repository
	dbURL := os.Getenv("DATABASE_URL")
	encKey := os.Getenv("PKI_ENCRYPTION_KEY")
	if encKey == "" {
		return nil, fmt.Errorf("PKI_ENCRYPTION_KEY environment variable is required")
	}

	if dbURL != "" {
		pool, err := pgxpool.New(context.Background(), dbURL)
		if err != nil {
			logger.Printf("warning: failed to connect to database: %v", err)
		} else {
			repo = repository.NewPostgresRepository(pool)
		}
	}

	if repo == nil {
		logger.Println("warning: no database connection, using nil repository")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	h := handler.New(repo, encKey)

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

	// Health endpoints
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	// PKI routes
	r.POST("/api/v1/pki/ca", h.CreateCA)
	r.GET("/api/v1/pki/ca", h.ListCAs)
	r.GET("/api/v1/pki/ca/:id", h.GetCA)
	r.DELETE("/api/v1/pki/ca/:id", h.DeleteCA)
	r.POST("/api/v1/pki/ca/:id/certs", h.CreateCertificate)
	r.GET("/api/v1/pki/certs", h.ListCerts)
	r.GET("/api/v1/pki/certs/:id", h.GetCert)
	r.POST("/api/v1/pki/certs/:id/revoke", h.RevokeCert)

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

		s.logger.Printf("[PKI] %v | %3d | %13v | %15s | %-7s %s",
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
