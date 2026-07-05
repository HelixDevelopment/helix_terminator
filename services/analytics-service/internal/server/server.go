package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/analytics-service/internal/handler"
)

// Server wraps the Gin engine.
type Server struct {
	router *gin.Engine
}

// New creates a new Server with routes wired.
func New() *Server {
	// TODO: configure middleware (logging, recovery, auth, tracing)
	r := gin.New()
	h := handler.New()

	// Health endpoints
	r.GET("/health", h.HealthCheck)
	r.GET("/ready", h.ReadinessCheck)

	// TODO: wire service-specific routes

	return &Server{router: r}
}

// Run starts the HTTP server.
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// Router exposes the underlying engine for testing.
func (s *Server) Router() http.Handler {
	return s.router
}
