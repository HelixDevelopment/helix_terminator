package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/handler"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/repository"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/server"
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
		dbURL = "postgres://postgres:postgres@localhost:5432/helixtrackbridge?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	repo := repository.New(pool)
	h := handler.New(repo)
	srv := server.New(h)

	return srv.Run()
}
