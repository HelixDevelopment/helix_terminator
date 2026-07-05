package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/keychain-service/internal/handler"
	"github.com/helixdevelopment/keychain-service/internal/repository"
)

// Server wraps the Gin engine and HTTP server.
type Server struct {
	router *gin.Engine
	srv    *http.Server
	repo   *repository.Repository
}

// New creates a new Server with routes wired.
func New(repo *repository.Repository) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(requestIDMiddleware())
	r.Use(loggingMiddleware())

	h := handler.New(repo)

	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/health", h.HealthCheck)
	r.GET("/ready", h.ReadinessCheck)

	api := r.Group("/api/v1")
	{
		api.POST("/keychain", h.CreateItem)
		api.GET("/keychain", h.ListItems)
		api.GET("/keychain/:id", h.GetItem)
		api.PUT("/keychain/:id", h.UpdateItem)
		api.DELETE("/keychain/:id", h.DeleteItem)
	}

	return &Server{
		router: r,
		repo:   repo,
	}
}

// Run starts the HTTP server with graceful shutdown.
func (s *Server) Run(addr string) error {
	s.srv = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		log.Printf("keychain-service starting on %s", addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		log.Printf("keychain-service received signal %v, shutting down gracefully", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.srv.Shutdown(ctx)
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Set("requestID", reqID)
		c.Writer.Header().Set("X-Request-ID", reqID)
		c.Next()
	}
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		reqID, _ := c.Get("requestID")
		log.Printf("[%s] %s %s %d %s %s", reqID, method, path, status, latency, clientIP)
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Printf("[ERROR] %s %s: %v", method, path, e.Err)
			}
		}
	}
}
