package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/helixdevelopment/keychain-service/internal/repository"
	"github.com/helixdevelopment/keychain-service/internal/server"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
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
