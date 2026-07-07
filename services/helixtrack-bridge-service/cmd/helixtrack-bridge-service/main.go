package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/coreclient"
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

	// Real HelixTrack Core JWT client (§11.4.28(B): injected via env, never
	// hardcoded). A nil core (base URL unset) is intentional fail-closed —
	// CreateBridge refuses to fabricate "active" without it.
	var core handler.Authenticator
	if coreBaseURL := os.Getenv("HELIXTRACK_CORE_BASE_URL"); coreBaseURL != "" {
		core = coreclient.New(coreBaseURL, os.Getenv("HELIXTRACK_CORE_USERNAME"), os.Getenv("HELIXTRACK_CORE_PASSWORD"))
	}

	h := handler.New(repo, core)
	srv := server.New(h)

	return srv.Run()
}
