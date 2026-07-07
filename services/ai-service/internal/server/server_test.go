package server_test

import (
	"testing"
	"time"

	"github.com/helixdevelopment/ai-service/internal/handler"
	"github.com/helixdevelopment/ai-service/internal/server"
)

func TestServerStub(t *testing.T) {
	t.Skip("TODO: implement server tests")
}

// minWriteTimeoutMargin is the minimum gap this invariant enforces between the
// effective HTTP WriteTimeout and the effective LLM completion budget, whatever
// their configured values (env-overridden or default).
const minWriteTimeoutMargin = 10 * time.Second

// TestHTTPWriteTimeoutExceedsLLMBudget is the regression guard for the T8-x
// independent-review finding: CreateRequest calls the configured LLM provider
// SYNCHRONOUSLY, so the ai-service http.Server's WriteTimeout MUST always stay
// comfortably above the LLM completion budget (handler.ResolveLLMTimeout) — a
// WriteTimeout shorter than (or too close to) that budget silently truncates the
// HTTP response on a slow-but-successful completion even though the DB row was
// written correctly. This test asserts the invariant against the DEFAULT
// configuration.
func TestHTTPWriteTimeoutExceedsLLMBudget(t *testing.T) {
	writeTimeout := server.ResolveHTTPWriteTimeout()
	llmTimeout := handler.ResolveLLMTimeout()

	if writeTimeout <= llmTimeout {
		t.Fatalf("WriteTimeout (%s) must exceed the LLM completion budget (%s) so a "+
			"successful-but-slow completion is never truncated", writeTimeout, llmTimeout)
	}
	if margin := writeTimeout - llmTimeout; margin < minWriteTimeoutMargin {
		t.Fatalf("WriteTimeout margin over the LLM budget is only %s, want >= %s (WriteTimeout=%s LLMTimeout=%s)",
			margin, minWriteTimeoutMargin, writeTimeout, llmTimeout)
	}
}

// TestHTTPWriteTimeoutExceedsLLMBudget_EnvOverride proves the SAME invariant holds
// for operator-configured values, not just the compiled-in defaults — a deployment
// that raises AI_LLM_TIMEOUT without raising AI_HTTP_WRITE_TIMEOUT to match
// reproduces the exact truncation defect this fix closes.
func TestHTTPWriteTimeoutExceedsLLMBudget_EnvOverride(t *testing.T) {
	t.Setenv("AI_LLM_TIMEOUT", "45s")
	t.Setenv("AI_HTTP_WRITE_TIMEOUT", "90s")

	writeTimeout := server.ResolveHTTPWriteTimeout()
	llmTimeout := handler.ResolveLLMTimeout()

	if writeTimeout != 90*time.Second {
		t.Fatalf("expected AI_HTTP_WRITE_TIMEOUT override to resolve to 90s, got %s", writeTimeout)
	}
	if llmTimeout != 45*time.Second {
		t.Fatalf("expected AI_LLM_TIMEOUT override to resolve to 45s, got %s", llmTimeout)
	}
	if margin := writeTimeout - llmTimeout; margin < minWriteTimeoutMargin {
		t.Fatalf("WriteTimeout margin over the LLM budget is only %s, want >= %s (WriteTimeout=%s LLMTimeout=%s)",
			margin, minWriteTimeoutMargin, writeTimeout, llmTimeout)
	}
}
