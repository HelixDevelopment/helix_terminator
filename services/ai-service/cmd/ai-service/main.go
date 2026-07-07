package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/ai-service/internal/llmclient"
	"github.com/helixdevelopment/ai-service/internal/repository"
	"github.com/helixdevelopment/ai-service/internal/server"
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
