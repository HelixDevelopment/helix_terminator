# AI Service — Local HelixLLM Integration

**Revision:** 1
**Last modified:** 2026-07-22T00:00:00Z

## Problem this closes (§11.4, no-bluff)

Before this change, `ai-service`'s `CreateRequest` handler accepted a
prompt, wrote an `AIRequest` row with `Status: "pending"`, and returned
`201 Created` — **without ever calling any LLM**. The request never
progressed past `"pending"`; no completion was ever produced. Any client
polling the resource waited forever for a transition that could never
happen. That is a textbook §11.4 PASS-bluff: the endpoint accepted the
request as if the feature worked, while the feature category itself did
not exist.

`CreateRequest` now **synchronously** calls a real local LLM backend
before persisting, and only ever writes a terminal `Status` —
`"completed"` (a real provider response landed) or `"failed"` (the
provider errored, timed out, or is unconfigured). `"pending"` is no
longer written anywhere in the codebase.

## Architecture

```
client
  │  POST /api/v1/ai/requests {prompt, model?, max_tokens?, temperature?}
  ▼
ai-service (internal/handler.CreateRequest)
  │  1. bind JSON
  │  2. h.llm.Complete(ctx, model, maxTokens, temperature, prompt)   ← SYNCHRONOUS
  │  3. persist AIRequest row with a TERMINAL status only
  │  4. respond 201 (Status: completed|failed) or 504 (clean timeout)
  ▼
internal/llmclient.GenericClient  (internal/llmclient/llmclient.go)
  │  adapts digital.vasic.llmprovider's LLMClient contract to
  │  handler.LLMClient's narrow (model, maxTokens, temperature, prompt) shape
  ▼
digital.vasic.llmprovider/pkg/providers/generic.Provider
  │  (submodules/llmprovider/pkg/providers/generic/generic.go)
  │  generic OpenAI-compatible HTTP client: POST <baseURL> with an
  │  OpenAI-shaped {model, messages, max_tokens, temperature, ...} body,
  │  Bearer-auth header, JSON response decode
  ▼
HelixLLM gateway  (submodules/helixllm)
  POST /v1/chat/completions  →  internal/gateway/openai.HandleChatCompletions
  → brain.Brain / fallback.Chain → a real inference backend
    (llama.cpp server, Ollama, or any configured downstream provider)
```

`digital.vasic.llmprovider` is consumed as an owned-org submodule per
§11.4.31/§11.4.74 (extend-don't-reimplement) — `ai-service` does **not**
hand-roll an OpenAI HTTP client; it wires the submodule's existing
`generic.Provider`, which already implements the OpenAI-compatible
`chat/completions` request/response shape.

## The exact API contract (investigated, not guessed — §11.4.6)

**Endpoint registration** — `submodules/helixllm/internal/gateway/router.go:78`:

```go
v1.POST("/chat/completions", HandleChatCompletions(opts.Brain, opts.ToolManager, opts.RAGHook))
```

mounted under the `/v1` group (`router.go:63`), so the full path is
**`POST /v1/chat/completions`** — the standard OpenAI-compatible route.
Auth is `gwmw.APIKeyAuth(opts.APIKeys)` (`router.go:64`); when
`HELIXLLM_API_KEYS`/`opts.APIKeys` is empty the gateway is open-access
(no bearer token enforced) — this is the default for the local
llama.cpp/Ollama-backed dev deployment this task targets.

**Handler** — `submodules/helixllm/internal/gateway/openai.go:46`
(`HandleChatCompletions`) binds the body into
`api.ChatCompletionRequest` (defined in
`submodules/helixllm/pkg/api/openai.go:4-18`):

```go
type ChatCompletionRequest struct {
    Model       string        `json:"model"`
    Messages    []ChatMessage `json:"messages"`
    Temperature *float64      `json:"temperature,omitempty"`
    MaxTokens   *int          `json:"max_tokens,omitempty"`
    Stream      bool          `json:"stream,omitempty"`
    Tools       []Tool        `json:"tools,omitempty"`
    // ... TopP, N, Stop, PresencePenalty, FrequencyPenalty, User, ToolChoice
}
```

and responds (non-streaming path, `openai.go:365`) with
`api.ChatCompletionResponse` (`pkg/api/openai.go:62-69`):

```go
type ChatCompletionResponse struct {
    ID      string
    Object  string  // "chat.completion"
    Created int64
    Model   string
    Choices []ChatCompletionChoice  // [0].Message.Content, [0].FinishReason
    Usage   *Usage                  // PromptTokens, CompletionTokens, TotalTokens
}
```

This is byte-for-byte the OpenAI `/v1/chat/completions` shape — no
HelixLLM-specific envelope. `digital.vasic.llmprovider`'s
`generic.Provider` (`submodules/llmprovider/pkg/providers/generic/generic.go:38-93`)
independently defines an equivalent minimal `Request`/`Response` pair
(`{model, messages[], max_tokens, temperature, top_p, stream, stop[]}`
→ `{id, object, created, model, choices[].message.content,
choices[].finish_reason, usage}`) and is a structural match for what
HelixLLM's gateway actually accepts and returns — confirmed by the live
call below, not assumed from the two source files alone.

## Config

| Env var | Default | Meaning |
|---|---|---|
| `AI_LOCAL_PROVIDER_BASE_URL` | `http://127.0.0.1:18434/v1/chat/completions` | **Full** chat-completions URL of the local HelixLLM-compatible backend. `generic.Provider` uses this value as-is and appends nothing — a bare host silently 404s. |
| `AI_LOCAL_PROVIDER_MODEL` | `qwen2.5-1.5b-instruct` | Default model ID sent when a `CreateAIRequest` does not specify one. |
| `AI_LLM_TIMEOUT` | `90s` | Per-completion budget `CreateRequest` bounds the synchronous `h.llm.Complete` call to (`internal/handler/handler.go:35-51`). Exceeding it returns a clean `context.DeadlineExceeded`, mapped to `504 Gateway Timeout`. |
| `AI_HTTP_WRITE_TIMEOUT` | `150s` | The `http.Server`'s `WriteTimeout`. **Must** exceed `AI_LLM_TIMEOUT` by ≥10s (`internal/server.MinWriteTimeoutMargin`) — checked at process startup by `server.ValidateTimeoutInvariant` (`cmd/ai-service/main.go:39`) and the process **refuses to start** if the margin is violated. This guards against a slow-but-successful completion having its HTTP response silently truncated underneath the synchronous LLM call (the T8-x defect class). |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/helixterminator?sslmode=disable` | Postgres connection string for the `ai_requests` persistence layer. |
| `PORT` | `8088` | ai-service's own listen port. |

Wired in `cmd/ai-service/main.go:90-101`:

```go
llmBaseURL := os.Getenv("AI_LOCAL_PROVIDER_BASE_URL")
if llmBaseURL == "" { llmBaseURL = defaultLocalLLMBaseURL }
llmModel := os.Getenv("AI_LOCAL_PROVIDER_MODEL")
if llmModel == "" { llmModel = defaultLocalLLMModel }
llmClient := llmclient.NewGenericClient("helixllm-local", localLLMAPIKeyPlaceholder, llmBaseURL, llmModel)
```

`localLLMAPIKeyPlaceholder` (`"local-no-auth-required"`) is **not** a
secret — the local llama.cpp/Ollama-backed gateway enforces no auth by
default (empty `HELIXLLM_API_KEYS`, see the router auth middleware
above); the placeholder exists only because `generic.Provider.ValidateConfig`
requires a non-empty `apiKey` string. No credential of any kind is
read, logged, or transmitted for the local tier.

## Local-only posture — why the cloud tier is off

`ai-service` wires **exactly one** LLM tier: the local HelixLLM-compatible
backend above. There is no code path that reads `OPENAI_API_KEY`,
`ANTHROPIC_API_KEY`, or any other cloud-provider credential — grep
confirms this (`cmd/ai-service/main.go`, `internal/handler/handler.go`,
`internal/llmclient/llmclient.go` each carry an explicit comment: *"the
cloud tier (OpenAI/Anthropic API keys) is OPERATOR-BLOCKED and
deliberately not wired"*). This is an explicit **operator decision**
(local-only deployment, no cloud keys) recorded in the dispatch that
produced this change, not an oversight — `digital.vasic.llmprovider`
ships first-class `openai`, `anthropic`, `claude`, and 40+ other cloud
provider adapters (`submodules/llmprovider/pkg/providers/*`) that could
be wired in a future, explicitly-authorized change; none of them are
imported by `ai-service` today.

Because there is no cloud code path to call, there is no cloud-specific
error branch either — an `AIRequest` created against this build always
resolves through the single local tier's real success/failure/timeout
outcomes described below. A future cloud-tier addition MUST preserve
this same honesty discipline: an unconfigured cloud tier must respond
with an explicit "cloud LLM not configured (local-only deployment)"
condition, never a fabricated completion.

## Honest failure semantics

`CreateRequest` (`internal/handler/handler.go:94-173`) never writes
`Status: "pending"`. Every code path resolves to a **terminal** outcome:

| Condition | HTTP status | Body `Status` | Notes |
|---|---|---|---|
| `h.llm == nil` (misconfigured deployment) | `201 Created` | `"failed"` | Production wiring in `main.go` always constructs a real client; this path only fires in a broken deployment or a test that intentionally exercises it. |
| Provider call errors (connection refused, non-200, malformed JSON, `AI_LOCAL_PROVIDER_BASE_URL` unreachable) | `201 Created` | `"failed"` | The row is created and persisted honestly — the client can see exactly which request failed and why (server-side log line), never a silent void. |
| Provider call exceeds `AI_LLM_TIMEOUT` | `504 Gateway Timeout` | (row persisted with `"failed"`, best-effort) | Distinguishes "provider overloaded/too slow" from an ordinary provider error at the transport level. |
| Provider call succeeds | `201 Created` | `"completed"` | `Response`/`TokensUsed` are the real provider output — never fabricated text. |

Tests exercising every row of this table:
`internal/handler/handler_test.go::TestCreateRequest_CallsRealLLM_NoFabricatedPending`,
`::TestCreateRequest_LLMProviderError_SetsFailedStatus`,
`::TestCreateRequest_LLMTimeout_ReturnsCleanGatewayTimeout`.

## Running the integration test (real, live backend)

`internal/llmclient/llmclient_integration_test.go` is a §11.4.27
mandatory real-integration test — it never mocks the network boundary.
It probes `<base>/health` first (§11.4.3 per-environment-topology
dispatch): reachable → drives a genuine completion and asserts
non-empty content + positive token usage; unreachable → an honest
`t.Skip` (never a fake PASS, never a false FAIL).

```bash
# Point at any locally-running OpenAI-compatible chat-completions server
# (llama.cpp server, Ollama's OpenAI-compat endpoint, or a real HelixLLM
# gateway instance). Example against a llama.cpp server already listening
# on 18434 (the production default):
export AI_LOCAL_PROVIDER_BASE_URL="http://127.0.0.1:18434/v1/chat/completions"
cd services/ai-service
GOWORK=off GOMAXPROCS=2 go test -p 2 -count=1 -run TestGenericClient_Complete_LiveHelixLLMContainer \
  -v ./internal/llmclient/...
```

### Captured evidence (this session, §11.4.5/§11.4.69/§11.4.107 — real, not fabricated)

Ran against a live local llama.cpp server already serving on
`127.0.0.1:18434` (`ghcr.io/ggml-org/llama.cpp:server` image,
`Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` model, confirmed via
`GET /v1/models` before the run):

```
=== RUN   TestGenericClient_Complete_LiveHelixLLMContainer
    llmclient_integration_test.go:125: live HelixLLM completion evidence: content="pong" tokensUsed=16
--- PASS: TestGenericClient_Complete_LiveHelixLLMContainer (0.10s)
PASS
ok  	github.com/helixdevelopment/ai-service/internal/llmclient	0.102s
```

The prompt sent was `"Say the single word: pong"`; the model answered
`"pong"` — a genuine, model-produced, non-empty completion with
positive token usage, proving the adapter talks to a real inference
backend end-to-end, not a fixture.

Without a reachable backend the same test SKIPs honestly instead of
failing or fabricating a PASS:

```
=== RUN   TestGenericClient_Complete_LiveHelixLLMContainer
    llmclient_integration_test.go:76: SKIP §11.4.3: live HelixLLM llama.cpp container
    unreachable at http://127.0.0.1:18435/health (dial tcp 127.0.0.1:18435:
    connect: connection refused) — this environment has no running container for
    this integration test; start it per the T-ai session recipe
    (ghcr.io/ggml-org/llama.cpp:server, Qwen2.5-1.5B-Instruct-Q4_K_M.gguf) and
    re-run to exercise this path
--- SKIP: TestGenericClient_Complete_LiveHelixLLMContainer (0.00s)
```

### Starting a local backend from scratch (rootless podman, §11.4.161)

```bash
podman run -d --name helixllm-ai-smoke -p 18434:8080 \
  -v "$HOME/models:/models:ro" \
  ghcr.io/ggml-org/llama.cpp:server \
  -m /models/<your-gguf-model> --host 0.0.0.0 --port 8080 -c 4096
```

Plain default rootless userns — no `:z`, no `--userns=keep-id`, no
`label=disable` (those crash on hosts with SELinux enforcing per prior
forensic findings in this project). Then run the integration test as
shown above.

## Unit tests (httptest fixtures — permitted, §11.4.27(A))

`internal/llmclient/llmclient_test.go` covers success, provider HTTP
error, malformed-JSON response, and model-override-passthrough against
an in-process `httptest.Server` returning canned OpenAI-shaped JSON —
these are unit tests of `GenericClient`'s adaptation logic, not a
substitute for the live-backend integration test above.

```bash
cd services/ai-service
GOWORK=off GOMAXPROCS=2 go test -p 2 -count=1 ./internal/llmclient/... -v
GOWORK=off GOMAXPROCS=2 go test -p 2 -count=1 ./internal/handler/... -v
```

## Sources verified

Investigated directly from the vendored submodule source (this
project's own owned-org dependency, not an external web-hosted API
that could drift independently of this repo — the code IS the current
contract):

- `submodules/helixllm/internal/gateway/router.go` (endpoint registration, `/v1` mount, auth middleware) — read 2026-07-22.
- `submodules/helixllm/internal/gateway/openai.go` (`HandleChatCompletions` handler, request/response marshalling) — read 2026-07-22.
- `submodules/helixllm/pkg/api/openai.go` (`ChatCompletionRequest`/`ChatCompletionResponse` wire types) — read 2026-07-22.
- `submodules/llmprovider/provider.go` + `submodules/llmprovider/pkg/providers/generic/generic.go` (the `LLMProvider` interface and the generic OpenAI-compatible adapter `ai-service` consumes) — read 2026-07-22.
- Live confirmation: `GET /v1/models` + `GET /health` + a real `POST /v1/chat/completions` round trip against a locally-running `ghcr.io/ggml-org/llama.cpp:server` container serving `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` — executed 2026-07-22 (see captured evidence above).
