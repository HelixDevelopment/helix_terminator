package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/helixdevelopment/user-service/internal/repository"
	"github.com/helixdevelopment/user-service/internal/server"
	"github.com/helixdevelopment/user-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/helixterminator?sslmode=disable"
	}

	// Apply pending schema migrations before opening the steady-state pool.
	// user-service already fails fast on DB connectivity trouble (see the
	// Ping check below), so a migration failure (including a dirty schema
	// state) is fatal here too - never serve against an unmigrated schema.
	if version, err := migrations.Run(databaseURL, log.Default()); err != nil {
		log.Fatalf("failed to apply database migrations: %v", err)
	} else {
		log.Printf("database migrations applied - schema version %d", version)
	}

	// Use the same schema-scoped connection URL the migrator applied
	// (search_path=migrations.Schema) so the steady-state pool's
	// unqualified "users" queries resolve against the schema
	// migrations.Run just migrated, not the shared database's default
	// "public" schema (schema-per-service, GAP-01).
	poolURL, err := migrations.ConnectionURL(databaseURL)
	if err != nil {
		log.Fatalf("failed to build schema-scoped connection URL: %v", err)
	}

	// Initialize database connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, poolURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("database connection established")

	repo := repository.New(pool)
	srv := server.New(repo)

	log.Printf("user-service starting on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
