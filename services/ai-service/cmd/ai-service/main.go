package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/helixdevelopment/ai-service/internal/handler"
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

	// Cloud-tier defaults (§11.4.28B config injection — overridable via
	// AI_CLOUD_BASE_URL / AI_CLOUD_MODEL, never hardcoded past this seam). The
	// generic OpenAI-compatible adapter (llmclient.NewGenericClient) targets a
	// FULL chat-completions path; the Anthropic default is its OpenAI-compatible
	// endpoint, so the SAME adapter serves both cloud providers.
	defaultOpenAIBaseURL    = "https://api.openai.com/v1/chat/completions"
	defaultOpenAIModel      = "gpt-4o-mini"
	defaultAnthropicBaseURL = "https://api.anthropic.com/v1/chat/completions"
	defaultAnthropicModel   = "claude-3-5-haiku-latest"
)

func main() {
	// §11.4.108 startup env-invariant: AI_HTTP_WRITE_TIMEOUT MUST exceed
	// AI_LLM_TIMEOUT by server.MinWriteTimeoutMargin — see
	// internal/server.ValidateTimeoutInvariant's doc comment for the T8-x
	// truncation defect a misconfigured deploy would otherwise silently
	// reintroduce (a slow-but-SUCCESSFUL completion has its HTTP response
	// truncated underneath CreateRequest's synchronous LLM call). Checked FIRST,
	// before any DB/LLM wiring, so a bad deploy config fails fast instead of
	// serving traffic under a latent truncation risk.
	if err := server.ValidateTimeoutInvariant(server.ResolveHTTPWriteTimeout(), handler.ResolveLLMTimeout()); err != nil {
		log.Fatalf("ai-service: startup env-invariant violated: %v", err)
	}

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

	// Optional monthly cloud-LLM spend ceiling. HONEST BOUNDARY (Constitution
	// §11.4 anti-bluff): the value is parsed, validated, and logged for operator
	// visibility, but it is NOT enforced here — no request path meters spend
	// against it yet. A future change wires real cost metering; until then the
	// ceiling is surfaced, never claimed active.
	if raw := os.Getenv("LLM_MONTHLY_COST_CEILING_USD"); raw != "" {
		ceiling, cerr := strconv.ParseFloat(raw, 64)
		if cerr != nil || ceiling < 0 {
			log.Fatalf("ai-service: LLM_MONTHLY_COST_CEILING_USD must be a non-negative number, got %q", raw)
		}
		log.Printf("ai-service: LLM monthly cost ceiling configured at $%.2f (SURFACED, NOT ENFORCED)", ceiling)
	}

	// LLM provider-tier selection. A cloud tier arms ONLY when its API key is
	// present (env-sourced, never hardcoded — §11.4.10); with no cloud key the
	// service falls back to the local HelixLLM tier (the honest internal
	// default). Presence of a key SELECTS the tier — no API call is made here;
	// the first real completion runs through the handler at request time. The
	// same generic OpenAI-compatible adapter serves all three tiers.
	var llmClient *llmclient.GenericClient
	switch {
	case os.Getenv("OPENAI_API_KEY") != "":
		baseURL := os.Getenv("AI_CLOUD_BASE_URL")
		if baseURL == "" {
			baseURL = defaultOpenAIBaseURL
		}
		model := os.Getenv("AI_CLOUD_MODEL")
		if model == "" {
			model = defaultOpenAIModel
		}
		llmClient = llmclient.NewGenericClient("openai-cloud", os.Getenv("OPENAI_API_KEY"), baseURL, model)
		log.Printf("ai-service: cloud LLM provider=openai base_url=%s model=%s", baseURL, model)
	case os.Getenv("ANTHROPIC_API_KEY") != "":
		baseURL := os.Getenv("AI_CLOUD_BASE_URL")
		if baseURL == "" {
			baseURL = defaultAnthropicBaseURL
		}
		model := os.Getenv("AI_CLOUD_MODEL")
		if model == "" {
			model = defaultAnthropicModel
		}
		llmClient = llmclient.NewGenericClient("anthropic-cloud", os.Getenv("ANTHROPIC_API_KEY"), baseURL, model)
		log.Printf("ai-service: cloud LLM provider=anthropic base_url=%s model=%s", baseURL, model)
	default:
		llmBaseURL := os.Getenv("AI_LOCAL_PROVIDER_BASE_URL")
		if llmBaseURL == "" {
			llmBaseURL = defaultLocalLLMBaseURL
		}
		llmModel := os.Getenv("AI_LOCAL_PROVIDER_MODEL")
		if llmModel == "" {
			llmModel = defaultLocalLLMModel
		}
		llmClient = llmclient.NewGenericClient("helixllm-local", localLLMAPIKeyPlaceholder, llmBaseURL, llmModel)
		log.Printf("ai-service: local LLM provider base_url=%s model=%s", llmBaseURL, llmModel)
	}

	srv := server.New(repo, llmClient)

	log.Printf("ai-service starting on port %s", port)
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
