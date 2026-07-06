package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/helixdevelopment/health-service/internal/checker"
	"github.com/helixdevelopment/health-service/internal/handler"
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

// Server wraps the Gin engine.
type Server struct {
	router  *gin.Engine
	logger  Logger
	handler *handler.Handler
}

// New creates a new Server with routes wired.
func New(logger Logger, endpoints map[string]string, timeout time.Duration) *Server {
	if logger == nil {
		logger = &defaultLogger{}
	}

	chk := checker.New(endpoints, timeout)
	h := handler.New(chk)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	s := &Server{
		router:  r,
		logger:  logger,
		handler: h,
	}

	// Global middleware
	r.Use(s.recoveryMiddleware())
	r.Use(s.loggingMiddleware())

	// Health endpoints (no auth required)
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	// API v1 routes
	v1 := r.Group("/api/v1/health")
	{
		v1.GET("/system", h.GetSystemHealth)
		v1.GET("/services/:name", h.GetServiceHealth)
		v1.POST("/check", h.RunHealthCheck)
	}

	return s
}

// Run starts the HTTP server.
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// Router exposes the underlying engine for testing.
func (s *Server) Router() http.Handler {
	return s.router
}

// --- Middleware ---

func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.Recovery()
}

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		s.logger.Printf("[HEALTH] %3d | %13v | %-7s %s",
			statusCode,
			latency,
			method,
			path,
		)
	}
}
