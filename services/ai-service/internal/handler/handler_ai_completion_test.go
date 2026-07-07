package handler_test

// §11.4.115 RED-baseline-on-the-broken-artifact + polarity-switch: these tests
// reproduce the fabricated-"pending" defect (§11.4.108 T-ai) against the REAL
// handler.CreateRequest code path — a fake Repository + fake LLMClient are injected
// (§11.4.27(A) unit-test fake), but CreateRequest itself is the unmodified
// production code under test. On the pre-fix handler these tests FAIL (captured RED
// evidence below); after the fix they PASS (GREEN) with no change to the test source
// — the same test is both the bug-catcher and the regression guard.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/ai-service/internal/handler"
	"github.com/helixdevelopment/ai-service/internal/model"
)

// fakeRepo is an in-memory handler.Repository fake — no live Postgres required.
type fakeRepo struct {
	requests map[uuid.UUID]*model.AIRequest
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{requests: make(map[uuid.UUID]*model.AIRequest)}
}

func (f *fakeRepo) CreateRequest(ctx context.Context, req *model.AIRequest) error {
	stored := *req
	f.requests[req.ID] = &stored
	return nil
}

func (f *fakeRepo) GetRequestByID(ctx context.Context, id uuid.UUID) (*model.AIRequest, error) {
	req, ok := f.requests[id]
	if !ok {
		return nil, errNotFound
	}
	return req, nil
}

func (f *fakeRepo) ListRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.AIRequest, int, error) {
	return nil, 0, nil
}

func (f *fakeRepo) Ping(ctx context.Context) error { return nil }

type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

const errNotFound = sentinelErr("AI request not found")

// fakeLLM is a handler.LLMClient fake that records the exact call it received and
// returns a pre-configured result — proving CreateRequest genuinely invokes the real
// completion contract with the caller's prompt, rather than fabricating a response.
type fakeLLM struct {
	called      bool
	gotModel    string
	gotMaxTok   int
	gotTemp     float64
	gotPrompt   string
	content     string
	tokensUsed  int
	err         error
}

func (f *fakeLLM) Complete(ctx context.Context, model string, maxTokens int, temperature float64, prompt string) (string, int, error) {
	f.called = true
	f.gotModel = model
	f.gotMaxTok = maxTokens
	f.gotTemp = temperature
	f.gotPrompt = prompt
	return f.content, f.tokensUsed, f.err
}

func setupAIRouter(h *handler.Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/ai/requests", h.CreateRequest)
	return r
}

// TestCreateRequest_CallsRealLLM_NoFabricatedPending is the RED→GREEN test for the
// core defect: CreateRequest MUST synchronously call the configured LLMClient with
// the caller's prompt and persist the REAL completion + a terminal Status — never the
// unconditional "pending" + empty Response the pre-fix code wrote every time.
func TestCreateRequest_CallsRealLLM_NoFabricatedPending(t *testing.T) {
	repo := newFakeRepo()
	llm := &fakeLLM{content: "4", tokensUsed: 7}
	h := handler.New(repo, llm)
	r := setupAIRouter(h)

	body := model.CreateAIRequest{Prompt: "What is 2+2?", Model: "qwen-test"}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ai/requests", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}

	var resp model.AIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Proof #1: the LLMClient was ACTUALLY invoked with the caller's real prompt —
	// not skipped, not stubbed out, not bypassed.
	if !llm.called {
		t.Fatal("FABRICATION DEFECT: CreateRequest never called the configured LLMClient — " +
			"the request was accepted without any real completion attempt")
	}
	if llm.gotPrompt != "What is 2+2?" {
		t.Fatalf("LLMClient.Complete called with wrong prompt: got %q, want %q", llm.gotPrompt, "What is 2+2?")
	}

	// Proof #2: the terminal Status reflects the REAL outcome, never the hardcoded
	// "pending" placeholder.
	if resp.Status == "pending" {
		t.Fatalf("FABRICATION DEFECT: Status is still the hardcoded placeholder %q — "+
			"a real LLM call happened but its outcome was never persisted", resp.Status)
	}
	if resp.Status != "completed" {
		t.Fatalf("expected Status %q, got %q", "completed", resp.Status)
	}

	// Proof #3: Response + TokensUsed carry the REAL completion, not empty fabricated
	// zero-values.
	if resp.Response != "4" {
		t.Fatalf("FABRICATION DEFECT: Response is %q, want the real completion %q", resp.Response, "4")
	}
	if resp.TokensUsed != 7 {
		t.Fatalf("expected TokensUsed 7, got %d", resp.TokensUsed)
	}

	// Proof #4: the PERSISTED record (not just the JSON response) carries the real
	// values — a bug could fabricate the HTTP response while still persisting
	// "pending" underneath.
	stored, ok := repo.requests[resp.ID]
	if !ok {
		t.Fatalf("request %s was never persisted", resp.ID)
	}
	if stored.Status != "completed" || stored.Response != "4" || stored.TokensUsed != 7 {
		t.Fatalf("persisted record does not carry the real completion: status=%q response=%q tokens=%d",
			stored.Status, stored.Response, stored.TokensUsed)
	}
}

// TestCreateRequest_LLMProviderError_SetsFailedStatus proves a provider error is
// surfaced honestly (Status: "failed", no fabricated Response) — never silently
// swallowed back into "pending" or masked as a fake success.
func TestCreateRequest_LLMProviderError_SetsFailedStatus(t *testing.T) {
	repo := newFakeRepo()
	llm := &fakeLLM{err: errProviderUnreachable}
	h := handler.New(repo, llm)
	r := setupAIRouter(h)

	body := model.CreateAIRequest{Prompt: "hello", Model: "qwen-test"}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/ai/requests", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (request persisted with a failed-completion status), got %d body=%s", w.Code, w.Body.String())
	}

	var resp model.AIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !llm.called {
		t.Fatal("FABRICATION DEFECT: CreateRequest never called the configured LLMClient")
	}
	if resp.Status != "failed" {
		t.Fatalf("expected Status %q on provider error, got %q (Status must never stay \"pending\" or fake-succeed)", "failed", resp.Status)
	}
	if resp.Response != "" {
		t.Fatalf("expected empty Response on provider error, got fabricated content %q", resp.Response)
	}
}

const errProviderUnreachable = sentinelErr("provider unreachable")
