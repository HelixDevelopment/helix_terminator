package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/ai-service/internal/handler"
	"github.com/helixdevelopment/ai-service/internal/repository"
)

// DefaultHTTPWriteTimeout is applied to the ai-service http.Server's WriteTimeout
// when AI_HTTP_WRITE_TIMEOUT is unset. It MUST stay comfortably above
// handler.DefaultLLMTimeout (see the paired invariant test
// TestHTTPWriteTimeoutExceedsLLMBudget) — CreateRequest calls the configured LLM
// provider SYNCHRONOUSLY (§11.4.108), so a WriteTimeout shorter than (or too close
// to) the LLM completion budget truncates the HTTP response on a slow-but-successful
// completion even though the DB row was written correctly (T8-x independent-review
// finding: the pre-fix 15s WriteTimeout vs. the LLM provider's up-to-120s budget).
const DefaultHTTPWriteTimeout = 150 * time.Second

// httpWriteTimeoutEnvVar overrides DefaultHTTPWriteTimeout — see
// ResolveHTTPWriteTimeout.
const httpWriteTimeoutEnvVar = "AI_HTTP_WRITE_TIMEOUT"

// ResolveHTTPWriteTimeout reads AI_HTTP_WRITE_TIMEOUT (a Go duration string, e.g.
// "150s") and returns it when present and valid (> 0); otherwise returns
// DefaultHTTPWriteTimeout.
func ResolveHTTPWriteTimeout() time.Duration {
	if v := os.Getenv(httpWriteTimeoutEnvVar); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return DefaultHTTPWriteTimeout
}

// MinWriteTimeoutMargin is the minimum gap ValidateTimeoutInvariant enforces between
// the effective HTTP WriteTimeout and the effective LLM completion budget, whatever
// their configured values (env-overridden or default). See DefaultHTTPWriteTimeout's
// doc comment for the T8-x truncation defect this margin guards against.
const MinWriteTimeoutMargin = 10 * time.Second

// ValidateTimeoutInvariant asserts writeTimeout exceeds llmTimeout by at least
// MinWriteTimeoutMargin — the SAME invariant TestHTTPWriteTimeoutExceedsLLMBudget
// checks at test time, but callable at process-startup time (see
// cmd/ai-service/main.go) so a misconfigured deploy that raises AI_LLM_TIMEOUT
// without raising AI_HTTP_WRITE_TIMEOUT to match is caught BEFORE the server starts
// accepting traffic, not merely by a test that runs in CI and can be skipped or
// missed on a raw `go build` + deploy. A margin below MinWriteTimeoutMargin (or a
// non-positive margin) silently reintroduces the T8-x truncation defect: a
// slow-but-successful completion gets its HTTP response truncated by WriteTimeout
// underneath the synchronous CreateRequest call, even though the DB row was written
// correctly.
func ValidateTimeoutInvariant(writeTimeout, llmTimeout time.Duration) error {
	margin := writeTimeout - llmTimeout
	if margin < MinWriteTimeoutMargin {
		return fmt.Errorf(
			"AI_HTTP_WRITE_TIMEOUT (%s) must exceed AI_LLM_TIMEOUT (%s) by at least %s "+
				"(current margin %s) — otherwise a slow-but-SUCCESSFUL LLM completion "+
				"truncates the HTTP response underneath CreateRequest's synchronous call "+
				"(see DefaultHTTPWriteTimeout doc comment, T8-x finding); "+
				"raise AI_HTTP_WRITE_TIMEOUT or lower AI_LLM_TIMEOUT",
			writeTimeout, llmTimeout, MinWriteTimeoutMargin, margin)
	}
	return nil
}

// Server wraps the Gin engine and HTTP server.
type Server struct {
	router *gin.Engine
	srv    *http.Server
	repo   *repository.Repository
}

// New creates a new Server with routes wired. llm is the real-completion client
// (production: *llmclient.GenericClient pointed at the local HelixLLM llama.cpp
// server) that CreateRequest calls synchronously — see internal/handler.LLMClient.
func New(repo *repository.Repository, llm handler.LLMClient) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(requestIDMiddleware())
	r.Use(loggingMiddleware())

	h := handler.New(repo, llm)

	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/health", h.HealthCheck)
	r.GET("/ready", h.ReadinessCheck)

	api := r.Group("/api/v1")
	{
		api.POST("/ai/requests", h.CreateRequest)
		api.GET("/ai/requests", h.ListRequests)
		api.GET("/ai/requests/:id", h.GetRequest)
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
		WriteTimeout: ResolveHTTPWriteTimeout(),
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		log.Printf("ai-service starting on %s", addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		log.Printf("ai-service received signal %v, shutting down gracefully", sig)
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
