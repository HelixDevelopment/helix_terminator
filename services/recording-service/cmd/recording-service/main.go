package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/recording-service/internal/handler"
	"github.com/helixdevelopment/recording-service/internal/repository"
	"github.com/helixdevelopment/recording-service/internal/server"
	"github.com/helixdevelopment/recording-service/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/recording?sslmode=disable"
	}

	// Apply pending schema migrations before opening the steady-state pool.
	// recording-service already fails fast on DB connectivity trouble (see
	// pgxpool.New below), so a migration failure (including a dirty schema
	// state) is fatal here too - never serve against an unmigrated schema.
	version, merr := migrations.Run(dbURL, log.Default())
	if merr != nil {
		return fmt.Errorf("failed to apply database migrations: %w", merr)
	}
	log.Printf("database migrations applied - schema version %d", version)

	// Use the same schema-scoped connection URL the migrator applied
	// (search_path=migrations.Schema) so the steady-state pool's
	// unqualified "recordings" queries resolve against the schema
	// migrations.Run just migrated, not the shared database's default
	// "public" schema (schema-per-service, GAP-01).
	poolURL, perr := migrations.ConnectionURL(dbURL)
	if perr != nil {
		return fmt.Errorf("failed to build schema-scoped connection URL: %w", perr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, poolURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	repo := repository.New(pool)
	h := handler.New(repo)
	srv := server.New(h)

	return srv.Run()
}
