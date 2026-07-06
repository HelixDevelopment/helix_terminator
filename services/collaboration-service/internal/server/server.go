package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/collaboration-service/internal/handler"
)

// Server wraps the HTTP server
type Server struct {
	httpServer *http.Server
	handler    *handler.Handler
}

// New creates a new Server
func New(h *handler.Handler) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	s := &Server{handler: h}

	router.GET("/healthz", h.HealthCheck)
	router.GET("/healthz/ready", h.ReadinessCheck)

	api := router.Group("/api/v1")
	{
		api.POST("/sessions", h.CreateSession)
		api.GET("/sessions", h.ListSessions)
		api.GET("/sessions/:id", h.GetSession)
		api.POST("/sessions/:id/join", h.JoinSession)
		api.POST("/sessions/:id/leave", h.LeaveSession)
		api.POST("/sessions/:id/end", h.EndSession)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	s.httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start begins listening
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Run starts the server and listens for shutdown signals
func (s *Server) Run() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("Collaboration Service listening on %s\n", s.httpServer.Addr)
		errCh <- s.Start()
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		fmt.Printf("Received signal %v, shutting down...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.Shutdown(ctx)
	}
}
