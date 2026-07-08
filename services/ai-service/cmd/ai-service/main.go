package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/helixdevelopment/ai-service/internal/llmclient"
	"github.com/helixdevelopment/ai-service/internal/repository"
	"github.com/helixdevelopment/ai-service/internal/server"
	"github.com/helixdevelopment/ai-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultLocalLLMBaseURL / defaultLocalLLMModel are the fallback values applied when
// AI_LOCAL_PROVIDER_BASE_URL / AI_LOCAL_PROVIDER_MODEL are unset (§11.4.28B config
// injection — overridable via env, never hardcoded past this single seam). The port
// matches the project's local HelixLLM llama.cpp smoke-test convention.
const (
	defaultLocalLLMBaseURL = "http://127.0.0.1:18434/v1/chat/completions"
	defaultLocalLLMModel   = "qwen2.5-1.5b-instruct"
	// localLLMAPIKeyPlaceholder is a non-secret filler: llama-server enforces no
	// auth, but the llmprovider adapter's ValidateConfig contract requires a
	// non-empty apiKey — see internal/llmclient.NewGenericClient.
	localLLMAPIKeyPlaceholder = "local-no-auth-required"
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

	// Apply pending schema migrations before opening the steady-state pool.
	// ai-service already fails fast on DB connectivity trouble (see
	// pgxpool.New below), so a migration failure (including a dirty schema
	// state) is fatal here too - never serve against an unmigrated schema.
	// This is purely additive schema-per-service DDL; it does not touch
	// internal/llmclient's LLM-provider dispatch.
	version, merr := migrations.Run(databaseURL, log.Default())
	if merr != nil {
		log.Fatalf("failed to apply database migrations: %v", merr)
	}
	log.Printf("database migrations applied - schema version %d", version)

	// Use the same schema-scoped connection URL the migrator applied
	// (search_path=migrations.Schema) so the steady-state pool's
	// unqualified "ai_requests" queries resolve against the schema
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

	repo := repository.New(pool)

	llmBaseURL := os.Getenv("AI_LOCAL_PROVIDER_BASE_URL")
	if llmBaseURL == "" {
		llmBaseURL = defaultLocalLLMBaseURL
	}
	llmModel := os.Getenv("AI_LOCAL_PROVIDER_MODEL")
	if llmModel == "" {
		llmModel = defaultLocalLLMModel
	}
	// Local HelixLLM tier only — the cloud tier (OpenAI/Anthropic API keys) is
	// OPERATOR-BLOCKED and deliberately not wired (see docs/CONTINUATION.md).
	llmClient := llmclient.NewGenericClient("helixllm-local", localLLMAPIKeyPlaceholder, llmBaseURL, llmModel)
	log.Printf("ai-service: local LLM provider base_url=%s model=%s", llmBaseURL, llmModel)

	srv := server.New(repo, llmClient)

	log.Printf("ai-service starting on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
