package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/helixdevelopment/keychain-service/internal/repository"
	"github.com/helixdevelopment/keychain-service/internal/server"
	"github.com/helixdevelopment/keychain-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/helixterminator?sslmode=disable"
	}

	// Encryption-at-rest key for private_key + passphrase (§11.4.10 / T10).
	// Fail-closed: the service MUST NOT start without it — there is no
	// silent plaintext fallback.
	encKey := os.Getenv("KEYCHAIN_ENCRYPTION_KEY")
	if encKey == "" {
		log.Fatalf("KEYCHAIN_ENCRYPTION_KEY environment variable is required")
	}

	// Apply pending schema migrations before opening the steady-state pool.
	// keychain-service already fails fast on DB connectivity trouble (see
	// pgxpool.New below), so a migration failure (including a dirty schema
	// state) is fatal here too - never serve against an unmigrated schema.
	// Scope note: this touches SCHEMA INIT ONLY - it never reads, writes,
	// or reasons about the private_key/passphrase ciphertext handled by
	// internal/repository + internal/crypto (T10 encryption-at-rest).
	version, merr := migrations.Run(databaseURL, log.Default())
	if merr != nil {
		log.Fatalf("failed to apply database migrations: %v", merr)
	}
	log.Printf("database migrations applied - schema version %d", version)

	// Use the same schema-scoped connection URL the migrator applied
	// (search_path=migrations.Schema) so the steady-state pool's
	// unqualified "keychain_items" queries resolve against the schema
	// migrations.Run just migrated, not the shared database's default
	// "public" schema (schema-per-service, GAP-01).
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

	repo, err := repository.New(pool, encKey)
	if err != nil {
		log.Fatalf("failed to initialize repository: %v", err)
	}
	srv := server.New(repo)

	log.Printf("keychain-service starting on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
