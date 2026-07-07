package llmclient_test

// §11.4.27 mandatory real-integration test: unlike llmclient_test.go's httptest
// fixtures, this test MUST hit the LIVE HelixLLM llama.cpp OpenAI-compatible server
// (submodules/helixllm's ghcr.io/ggml-org/llama.cpp:server prebuilt image running
// CPU-only with the Qwen2.5-1.5B-Instruct-Q4_K_M.gguf model — see the T-ai session's
// container recipe). It is gated per §11.4.3 (per-environment-topology dispatch): a
// health-endpoint probe at test start decides PASS/SKIP — never a fake PASS when the
// container is unreachable, never a false FAIL when it is genuinely absent from the
// current environment.
//
// Real evidence captured this session (container helixllm-ai-smoke, host port 18435
// — 18434 was already bound by an unrelated pre-existing process on the shared host,
// so this dispatch's assigned port could not be used without disturbing a resource
// this task does not own; §11.4.174 shared-host process-ownership applies):
//
//   $ curl -s http://127.0.0.1:18435/v1/chat/completions -H 'Content-Type: application/json' \
//       -d '{"messages":[{"role":"user","content":"Say the single word: pong"}],"stream":false,"max_tokens":16}'
//   {"choices":[{"finish_reason":"stop","index":0,"message":{"role":"assistant","content":"Pong."}}],
//    ...,"usage":{"completion_tokens":4,"prompt_tokens":35,"total_tokens":39,...}}
//
// The override that dispatched this worktree assigned AI_LOCAL_PROVIDER_BASE_URL a
// fixed port; this test instead reads it from the SAME env var the production binary
// uses (falling back to the locally-verified port) so it stays correct if the
// container is redeployed on a different port.

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/helixdevelopment/ai-service/internal/llmclient"
)

// integrationBaseURL resolves the chat-completions URL under test the same way
// cmd/ai-service/main.go resolves it in production (env override, else the
// locally-verified default for this session's container).
func integrationBaseURL() string {
	if v := os.Getenv("AI_LOCAL_PROVIDER_BASE_URL"); v != "" {
		return v
	}
	return "http://127.0.0.1:18435/v1/chat/completions"
}

// healthURLFromChatCompletionsURL derives the llama.cpp /health endpoint from the
// configured chat-completions URL for the reachability probe.
func healthURLFromChatCompletionsURL(chatURL string) string {
	i := strings.Index(chatURL, "/v1/chat/completions")
	if i < 0 {
		return chatURL
	}
	return chatURL[:i] + "/health"
}

func TestGenericClient_Complete_LiveHelixLLMContainer(t *testing.T) {
	baseURL := integrationBaseURL()
	healthURL := healthURLFromChatCompletionsURL(baseURL)

	// §11.4.3 per-environment-topology dispatch: probe reachability BEFORE
	// asserting anything: SKIP-with-reason when the container topology is absent
	// from this environment, never a fake PASS and never a false FAIL.
	probeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	healthReq, err := http.NewRequestWithContext(probeCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		t.Fatalf("failed to build health-check request: %v", err)
	}
	healthResp, err := http.DefaultClient.Do(healthReq)
	if err != nil {
		// Any transport-level failure (connection refused, timeout, DNS failure) at
		// this reachability probe means the container topology is absent from this
		// environment — an honest §11.4.3 SKIP, never a fake PASS.
		t.Skipf("SKIP §11.4.3: live HelixLLM llama.cpp container unreachable at %s (%v) — "+
			"this environment has no running container for this integration test; "+
			"start it per the T-ai session recipe (ghcr.io/ggml-org/llama.cpp:server, "+
			"Qwen2.5-1.5B-Instruct-Q4_K_M.gguf) and re-run to exercise this path", healthURL, err)
		return
	}
	_ = healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Skipf("SKIP §11.4.3: live HelixLLM container health check returned %d at %s", healthResp.StatusCode, healthURL)
		return
	}

	client := llmclient.NewGenericClient("helixllm-local-it", "local-no-auth-required", baseURL, "qwen2.5-1.5b-instruct")

	ctx, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel2()
	content, tokensUsed, err := client.Complete(ctx, "", 32, 0, "Say the single word: pong")
	if err != nil {
		t.Fatalf("real completion request against the live container failed: %v", err)
	}

	// Real, non-empty content came back from a genuine local LLM inference — this is
	// the anti-bluff proof (§11.4.5/§11.4.69/§11.4.107) that the adapter really talks
	// to a real model, not a fixture.
	if strings.TrimSpace(content) == "" {
		t.Fatal("FABRICATION-CLASS DEFECT: live container returned an empty completion")
	}
	if tokensUsed <= 0 {
		t.Fatalf("expected positive token usage from the live container, got %d", tokensUsed)
	}
	t.Logf("live HelixLLM completion evidence: content=%q tokensUsed=%d", content, tokensUsed)
}
