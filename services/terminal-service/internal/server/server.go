package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/terminal-service/internal/handler"
	"github.com/helixdevelopment/terminal-service/internal/recorder"
	"github.com/helixdevelopment/terminal-service/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
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

// Server wraps the Gin engine with terminal service functionality.
type Server struct {
	router  *gin.Engine
	logger  Logger
	handler *handler.Handler
}

// New creates a new Server with dependencies.
func New(logger Logger) (*Server, error) {
	if logger == nil {
		logger = &defaultLogger{}
	}

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
		logger.Println("warning: no database connection, some features will be unavailable")
	}

	outputDir := os.Getenv("RECORDING_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "/tmp/terminal-recordings"
	}
	rec := recorder.NewRecorder(outputDir, repo)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	h := handler.New(repo, rec)

	s := &Server{
		router:  r,
		logger:  logger,
		handler: h,
	}

	// Global middleware
	r.Use(s.recoveryMiddleware())
	r.Use(s.requestIDMiddleware())
	r.Use(s.loggingMiddleware())
	r.Use(s.corsMiddleware())

	// Health endpoints
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/healthz/live", h.HealthCheck)

	// Terminal session routes
	r.POST("/api/v1/terminal/sessions", h.CreateTerminalSession)
	r.GET("/api/v1/terminal/sessions", h.ListTerminalSessions)
	r.GET("/api/v1/terminal/sessions/:id", h.GetTerminalSession)
	r.PUT("/api/v1/terminal/sessions/:id", h.UpdateTerminalSession)
	r.POST("/api/v1/terminal/sessions/:id/close", h.CloseTerminalSession)

	// Output routes
	r.POST("/api/v1/terminal/sessions/:id/output", h.WriteTerminalOutput)
	r.GET("/api/v1/terminal/sessions/:id/output", h.GetTerminalOutput)

	// Playback and recording routes
	r.GET("/api/v1/terminal/sessions/:id/playback", h.GetPlayback)
	r.POST("/api/v1/terminal/sessions/:id/recording", h.StartRecording)
	r.GET("/api/v1/terminal/sessions/:id/recording", h.GetRecording)

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

		s.logger.Printf("[TERM] %v | %3d | %13v | %15s | %-7s %s",
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
