package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/helixdevelopment/auth-service/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger := log.New(os.Stdout, "[auth] ", log.LstdFlags|log.Lshortfile)

	// JWT Ed25519 signing key: loaded from the JWT_PRIVATE_KEY env var
	// (mounted Kubernetes Secret in production - see
	// infrastructure/kubernetes/base/services/auth-service/deployment.yaml
	// and docs/guides/JWT_KEY_PROVISIONING.md), with a loudly-logged
	// ephemeral fallback for dev/test only. See
	// internal/server.loadJWTManager for the full fail-closed/fallback
	// contract. KMS-backed signing is real future hardening, tracked as
	// an explicit operator decision (§11.4.101/§11.4.112) - not
	// implemented here.
	srv, err := server.New(logger)
	if err != nil {
		logger.Fatalf("failed to create server: %v", err)
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Printf("auth-service starting on port %s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatalf("server forced to shutdown: %v", err)
	}

	logger.Println("server exited gracefully")
}
