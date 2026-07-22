# AI Service

HelixTerminator microservice — synchronous AI-completion request API backed by a
local HelixLLM-compatible LLM provider (llama.cpp / Ollama / HelixLLM gateway,
OpenAI-compatible `/v1/chat/completions`). See
[`docs/guides/AI_SERVICE.md`](../../docs/guides/AI_SERVICE.md) for the full
architecture, honest-failure-semantics table, and live-integration-test
instructions.

## Features
- AI request creation: `CreateRequest` calls the configured local LLM
  **synchronously** and persists only a terminal `completed`/`failed` status —
  no fabricated `"pending"` row is ever written
- AI request history: list + fetch a single request by ID
- Local-only LLM tier — the cloud tier (OpenAI/Anthropic API keys) is
  deliberately not wired (operator-blocked)

## Module Path

`github.com/helixdevelopment/ai-service`

## Database

PostgreSQL, schema `ai_service` (schema-per-service) within the shared
`helixterminator` database — see `migrations/migrate.go`. No Redis or Kafka
dependency.

## Upstream Dependencies

`digital.vasic.llmprovider` (owned-org submodule, `submodules/llmprovider`) —
the generic OpenAI-compatible LLM client adapter `internal/llmclient` wires to
a local HelixLLM-compatible backend.

## API Endpoints

- `POST` `/api/v1/ai/requests` — Create an AI request. Body:
  `{"prompt": "...", "model": "...", "context"?, "maxTokens"?, "temperature"?}`.
  Calls the LLM provider synchronously; responds `201` with a terminal
  `status` (`completed` or `failed`), or `504` on a clean provider timeout.
- `GET` `/api/v1/ai/requests` — List AI requests for the current user
  (`?limit=&offset=`, default `limit=20`, max `100`).
- `GET` `/api/v1/ai/requests/:id` — Fetch a single AI request by ID.

## Health Checks

- `GET /healthz` / `GET /health` — Liveness check (200 = healthy)
- `GET /healthz/ready` / `GET /ready` — Readiness check (200 = ready, 503 = database unavailable)

## Running

```bash
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/helixterminator?sslmode=disable
export PORT=8088
export AI_LOCAL_PROVIDER_BASE_URL=http://127.0.0.1:18434/v1/chat/completions
go run ./cmd/ai-service
```

## Testing

```bash
go test -v -race -cover ./...
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | No | `postgres://postgres:postgres@localhost:5432/helixterminator?sslmode=disable` | PostgreSQL connection string (schema `ai_service`) |
| `PORT` | No | `8088` | HTTP listen port |
| `AI_LOCAL_PROVIDER_BASE_URL` | No | `http://127.0.0.1:18434/v1/chat/completions` | **Full** chat-completions URL of the local HelixLLM-compatible backend — used as-is, a bare host silently 404s |
| `AI_LOCAL_PROVIDER_MODEL` | No | `qwen2.5-1.5b-instruct` | Default model ID used when a request does not specify one |
| `AI_LLM_TIMEOUT` | No | `90s` | Per-completion budget the synchronous LLM call is bounded to; exceeding it returns a clean `504 Gateway Timeout` |
| `AI_HTTP_WRITE_TIMEOUT` | No | `150s` | HTTP server `WriteTimeout`; **must** exceed `AI_LLM_TIMEOUT` by ≥10s — the process refuses to start otherwise (see `internal/server.ValidateTimeoutInvariant`) |

---

*HelixTerminator AI Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
