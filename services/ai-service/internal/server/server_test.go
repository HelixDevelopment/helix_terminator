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

// minWriteTimeoutMargin mirrors server.MinWriteTimeoutMargin — kept as a local
// alias (not a re-declaration) so a future accidental drift between the two values
// is impossible: this file always asserts against the exported constant
// ValidateTimeoutInvariant itself uses at runtime.
const minWriteTimeoutMargin = server.MinWriteTimeoutMargin

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

	// Proof the process-startup check (cmd/ai-service/main.go calls
	// server.ValidateTimeoutInvariant with these SAME two values) agrees with the
	// manual assertions above — a misconfigured deploy relies on this function, not
	// on this test, to catch the truncation regression at runtime.
	if err := server.ValidateTimeoutInvariant(writeTimeout, llmTimeout); err != nil {
		t.Fatalf("ValidateTimeoutInvariant rejected the default configuration it should accept: %v", err)
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

// TestValidateTimeoutInvariant_RejectsReintroducedTruncationBug is the RED-baseline
// (§11.4.115 polarity) regression guard for the process-startup check itself: it
// reproduces, directly against the values, the EXACT misconfiguration class the T8-x
// finding discovered (AI_LLM_TIMEOUT raised without a matching AI_HTTP_WRITE_TIMEOUT
// raise) and asserts ValidateTimeoutInvariant refuses it with a descriptive error —
// so cmd/ai-service/main.go's log.Fatal on this error path is proven to actually
// fire for the defect class it exists to catch, not merely to compile.
func TestValidateTimeoutInvariant_RejectsReintroducedTruncationBug(t *testing.T) {
	cases := []struct {
		name         string
		writeTimeout time.Duration
		llmTimeout   time.Duration
		wantErr      bool
	}{
		{
			name:         "llm_timeout_raised_past_write_timeout_reintroduces_T8x",
			writeTimeout: 30 * time.Second,
			llmTimeout:   45 * time.Second, // > writeTimeout: the exact T8-x shape
			wantErr:      true,
		},
		{
			name:         "margin_below_minimum_still_rejected",
			writeTimeout: 50 * time.Second,
			llmTimeout:   45 * time.Second, // margin 5s < MinWriteTimeoutMargin (10s)
			wantErr:      true,
		},
		{
			name:         "equal_timeouts_rejected_zero_margin",
			writeTimeout: 45 * time.Second,
			llmTimeout:   45 * time.Second,
			wantErr:      true,
		},
		{
			name:         "exact_minimum_margin_accepted",
			writeTimeout: 55 * time.Second,
			llmTimeout:   45 * time.Second, // margin == MinWriteTimeoutMargin (10s)
			wantErr:      false,
		},
		{
			name:         "comfortable_margin_accepted",
			writeTimeout: 150 * time.Second,
			llmTimeout:   90 * time.Second, // production defaults
			wantErr:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := server.ValidateTimeoutInvariant(tc.writeTimeout, tc.llmTimeout)
			if tc.wantErr && err == nil {
				t.Fatalf("ValidateTimeoutInvariant(%s, %s) = nil error, want a rejection — "+
					"this margin silently reintroduces the T8-x HTTP-response-truncation defect",
					tc.writeTimeout, tc.llmTimeout)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateTimeoutInvariant(%s, %s) = %v, want nil (this margin is safe)",
					tc.writeTimeout, tc.llmTimeout, err)
			}
		})
	}
}
