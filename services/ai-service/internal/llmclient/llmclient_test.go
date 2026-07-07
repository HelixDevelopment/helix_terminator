package llmclient_test

// Unit-level adapter tests per §11.4.27(A): mocks/fakes are permitted at the unit-test
// layer — here an httptest.Server standing in for the llama.cpp OpenAI-compatible
// endpoint, exercising the REAL generic.Provider (via llmclient.NewGenericClient),
// never a fake of GenericClient itself. The MANDATORY real-container integration test
// (§11.4.27) lives in llmclient_integration_test.go and hits the live HelixLLM
// llama.cpp server — this file only proves the adapter's request/response shaping and
// error propagation are correct against a controlled fixture server.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/helixdevelopment/ai-service/internal/llmclient"
)

func TestGenericClient_Complete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer placeholder-key" {
			t.Errorf("expected Authorization header with the configured apiKey, got %q", got)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		msgs, ok := body["messages"].([]interface{})
		if !ok || len(msgs) != 1 {
			t.Fatalf("expected exactly one message, got %#v", body["messages"])
		}
		msg, ok := msgs[0].(map[string]interface{})
		if !ok {
			t.Fatalf("message entry has unexpected shape: %#v", msgs[0])
		}
		if msg["role"] != "user" {
			t.Errorf("expected role \"user\", got %v", msg["role"])
		}
		if content, _ := msg["content"].(string); !strings.Contains(content, "hello llama") {
			t.Errorf("expected the real prompt forwarded as-is, got %q", content)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "cmpl-1",
			"object": "chat.completion",
			"model": "qwen-test",
			"choices": [{"index":0,"message":{"role":"assistant","content":"hi there"},"finish_reason":"stop"}],
			"usage": {"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8}
		}`))
	}))
	defer srv.Close()

	client := llmclient.NewGenericClient("test", "placeholder-key", srv.URL+"/v1/chat/completions", "qwen-test")
	content, tokensUsed, err := client.Complete(context.Background(), "", 0, 0, "hello llama")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "hi there" {
		t.Fatalf("expected content %q, got %q", "hi there", content)
	}
	if tokensUsed != 8 {
		t.Fatalf("expected tokensUsed 8, got %d", tokensUsed)
	}
}

func TestGenericClient_Complete_ProviderHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	client := llmclient.NewGenericClient("test", "placeholder-key", srv.URL+"/v1/chat/completions", "qwen-test")
	content, tokensUsed, err := client.Complete(context.Background(), "", 0, 0, "hello")
	if err == nil {
		t.Fatal("expected an error from a 500 provider response, got nil")
	}
	if content != "" || tokensUsed != 0 {
		t.Fatalf("expected zero-value content/tokens on error, got content=%q tokensUsed=%d", content, tokensUsed)
	}
}

func TestGenericClient_Complete_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	client := llmclient.NewGenericClient("test", "placeholder-key", srv.URL+"/v1/chat/completions", "qwen-test")
	if _, _, err := client.Complete(context.Background(), "", 0, 0, "hello"); err == nil {
		t.Fatal("expected an error parsing malformed JSON, got nil")
	}
}

func TestGenericClient_Complete_ModelOverridePassedThrough(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotModel, _ = body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	client := llmclient.NewGenericClient("test", "placeholder-key", srv.URL+"/v1/chat/completions", "default-model")
	if _, _, err := client.Complete(context.Background(), "caller-requested-model", 0, 0, "hi"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotModel != "caller-requested-model" {
		t.Fatalf("expected the caller-requested model to be forwarded, got %q", gotModel)
	}
}

func TestNewGenericClient_NilSafety(t *testing.T) {
	var client *llmclient.GenericClient
	_, _, err := client.Complete(context.Background(), "", 0, 0, "hi")
	if err == nil {
		t.Fatal("expected an error calling Complete on a nil *GenericClient, got nil")
	}
}
