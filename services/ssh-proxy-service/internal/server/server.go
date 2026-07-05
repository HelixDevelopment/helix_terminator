package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/ssh-proxy-service/internal/handler"
	"github.com/helixdevelopment/ssh-proxy-service/internal/repository"
	"github.com/helixdevelopment/ssh-proxy-service/internal/wshandler"
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

// Server wraps the Gin engine with SSH proxy service functionality.
type Server struct {
	router     *gin.Engine
	logger     Logger
	handler    *handler.Handler
	sessionMgr *wshandler.SessionManager
}

// New creates a new Server with dependencies.
func New(logger Logger) (*Server, error) {
	if logger == nil {
		logger = &defaultLogger{}
	}

	var repo repository.Repository
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		pool, err := pgxpool.New(context.Background(), dbURL)
		if err != nil {
			logger.Printf("warning: failed to connect to database: %v", err)
		} else {
			repo = repository.NewPostgresRepository(pool)
		}
	}

	if repo == nil {
		logger.Println("warning: no database connection, using in-memory mode")
		repo = &repository.InMemoryRepository{}
	}

	sessionMgr := wshandler.NewSessionManager()
	h := handler.New(repo, sessionMgr)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	s := &Server{
		router:     r,
		logger:     logger,
		handler:    h,
		sessionMgr: sessionMgr,
	}

	// Global middleware
	r.Use(s.recoveryMiddleware())
	r.Use(s.requestIDMiddleware())
	r.Use(s.loggingMiddleware())
	r.Use(s.corsMiddleware())

	// Health endpoints
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	// API routes
	api := r.Group("/api/v1/ssh")
	{
		api.GET("/sessions", h.ListSSHSessions)
		api.GET("/sessions/:id", h.GetSSHSession)
		api.DELETE("/sessions/:id", h.TerminateSSHSession)
	}

	// WebSocket endpoint (must avoid body-buffering middleware)
	r.GET("/ws/ssh", h.HandleWebSocket)

	return s, nil
}

// Router exposes the underlying engine for testing.
func (s *Server) Router() http.Handler {
	return s.router
}

// SessionManager exposes the session manager for graceful shutdown.
func (s *Server) SessionManager() *wshandler.SessionManager {
	return s.sessionMgr
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

		s.logger.Printf("[SSH-PROXY] %v | %3d | %13v | %15s | %-7s %s",
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
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-API-Key")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
