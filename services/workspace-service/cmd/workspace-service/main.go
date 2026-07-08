package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/workspace-service/internal/repository"
	"github.com/helixdevelopment/workspace-service/internal/server"
	"github.com/helixdevelopment/workspace-service/migrations"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger := log.New(os.Stdout, "[workspace-service] ", log.LstdFlags)

	// Initialize database connection.
	//
	// workspace-service has TWO independent DB-init sites: this one and
	// internal/server/server.go's New() (called just below), which opens
	// its own separate pgxpool.Pool and is the pool that actually backs
	// the running service's repository - the pool built here is not
	// passed anywhere and is otherwise unused by this binary. Both sites
	// apply pending schema migrations (migrations.Run) BEFORE opening
	// their own pool so neither ever queries the schema pre-migration;
	// Run is idempotent (a second invocation against an already-migrated
	// database is a no-op, migrate.ErrNoChange), so calling it from both
	// sites is safe regardless of call order. Both pools are opened via
	// migrations.ConnectionURL(dbURL) so they consistently resolve
	// unqualified table names against the migrated
	// search_path=workspace_service schema, not the shared database's
	// default "public" schema (schema-per-service, GAP-01).
	var repo *repository.Repository
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		if version, merr := migrations.Run(dbURL, logger); merr != nil {
			logger.Printf("warning: failed to apply database migrations: %v", merr)
		} else {
			logger.Printf("database migrations applied - schema version %d", version)

			poolURL, perr := migrations.ConnectionURL(dbURL)
			if perr != nil {
				logger.Printf("warning: failed to build schema-scoped connection URL: %v", perr)
			} else {
				pool, err := pgxpool.New(context.Background(), poolURL)
				if err != nil {
					logger.Printf("warning: failed to connect to database: %v", err)
				} else {
					repo = repository.New(pool)
				}
			}
		}
	}

	if repo == nil {
		logger.Println("warning: no database connection, using in-memory mode")
	}

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
