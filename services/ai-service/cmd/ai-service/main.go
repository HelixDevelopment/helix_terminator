package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/ai-service/internal/repository"
	"github.com/helixdevelopment/ai-service/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/helixterminator?sslmode=disable"
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

	repo := repository.New(pool)
	srv := server.New(repo)

	log.Printf("ai-service starting on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
