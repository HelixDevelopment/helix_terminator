package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/helixdevelopment/container-bridge-service/internal/containerrt"
	"github.com/helixdevelopment/container-bridge-service/internal/handler"
	"github.com/helixdevelopment/container-bridge-service/internal/repository"
	"github.com/helixdevelopment/container-bridge-service/internal/server"
	"github.com/helixdevelopment/container-bridge-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
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
		dbURL = "postgres://postgres:postgres@localhost:5432/containerbridge?sslmode=disable"
	}

	// Apply pending schema migrations before opening the steady-state pool.
	// container-bridge-service already fails fast on DB connectivity
	// trouble (see pgxpool.New below), so a migration failure (including a
	// dirty schema state) is fatal here too - never serve against an
	// unmigrated schema.
	version, merr := migrations.Run(dbURL, log.Default())
	if merr != nil {
		return fmt.Errorf("failed to apply database migrations: %w", merr)
	}
	log.Printf("database migrations applied - schema version %d", version)

	// Use the same schema-scoped connection URL the migrator applied
	// (search_path=migrations.Schema) so the steady-state pool's
	// unqualified "container_bridges" queries resolve against the schema
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

	// Detect the local container runtime (Podman-first per §11.4.161, override
	// via CONTAINER_RUNTIME_PRIORITY). A detection failure is NOT fatal to the
	// service — it degrades every container-lifecycle route to an honest 503
	// rather than fabricating container state (see internal/handler).
	rtCtx, rtCancel := context.WithTimeout(context.Background(), 10*time.Second)
	backend, backendErr := containerrt.Detect(rtCtx, os.Getenv("CONTAINER_RUNTIME_PRIORITY"))
	rtCancel()
	if backendErr != nil {
		fmt.Fprintf(os.Stderr, "warning: no container runtime available: %v\n", backendErr)
		backend = nil
	}

	h := handler.New(repo, backend)
	srv := server.New(h)

	return srv.Run()
}
