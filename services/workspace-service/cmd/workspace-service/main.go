package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/helixdevelopment/workspace-service/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger := log.New(os.Stdout, "[workspace-service] ", log.LstdFlags)

	// Database connection + schema migrations are owned exclusively by
	// internal/server/server.go's New() (called just below): it applies
	// migrations.Run + opens the one pgxpool.Pool that actually backs the
	// running service's repository via migrations.ConnectionURL(dbURL).
	// A second, independent DB-init site used to live here (its own
	// pgxpool.Pool + a redundant migrations.Run call), but its pool was
	// never passed to server.New (which takes only a Logger) and was
	// therefore provably dead code - removed per §11.4.124 investigation
	// (git history: introduced in commit eb4701d's scaffold, which already
	// called server.New(&logAdapter{logger}) with no pool/repo parameter;
	// migration wiring in commit 88c0661 added migrations.Run to this dead
	// site too, but never made it live). server.New() below is the sole
	// live migration + DB-connection path.

	// Create server
	srv, err := server.New(&logAdapter{logger})
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Start server in a goroutine
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: srv.Router(),
	}

	go func() {
		log.Printf("workspace-service starting on port %s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down workspace-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	log.Println("workspace-service stopped")
}

type logAdapter struct {
	logger *log.Logger
}

func (l *logAdapter) Printf(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}

func (l *logAdapter) Println(v ...interface{}) {
	l.logger.Println(v...)
}
