package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/user-service/internal/repository"
	"github.com/helixdevelopment/user-service/internal/server"
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

	// Initialize database connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
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
