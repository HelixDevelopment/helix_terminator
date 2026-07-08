package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/helixdevelopment/billing-service/internal/repository"
	"github.com/helixdevelopment/billing-service/internal/server"
	"github.com/helixdevelopment/billing-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8087"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/helixterminator?sslmode=disable"
	}

	// Apply pending schema migrations before opening the steady-state pool.
	// billing-service already fails fast on DB connectivity trouble (see
	// pgxpool.New below), so a migration failure (including a dirty schema
	// state) is fatal here too - never serve against an unmigrated schema.
	// This is purely additive schema setup; the auth-middleware +
	// tenant-scoping (T12/T14) handler/model logic is untouched.
	version, merr := migrations.Run(databaseURL, log.Default())
	if merr != nil {
		log.Fatalf("failed to apply database migrations: %v", merr)
	}
	log.Printf("database migrations applied - schema version %d", version)

	// Use the same schema-scoped connection URL the migrator applied
	// (search_path=migrations.Schema) so the steady-state pool's
	// unqualified "billing_plans"/"subscriptions"/"invoices"/
	// "usage_records" queries resolve against the schema migrations.Run
	// just migrated, not the shared database's default "public" schema
	// (schema-per-service, GAP-01).
	poolURL, perr := migrations.ConnectionURL(databaseURL)
	if perr != nil {
		log.Fatalf("failed to build schema-scoped connection URL: %v", perr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, poolURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("database connection established")

	repo := repository.New(pool)
	srv := server.New(repo)

	log.Printf("billing-service starting on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
