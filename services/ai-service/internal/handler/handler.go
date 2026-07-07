package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/ai-service/internal/model"
)

// DefaultLLMTimeout bounds the local-LLM completion budget: CreateRequest wraps
// every h.llm.Complete call in a context.WithTimeout(DefaultLLMTimeout) (overridable
// via AI_LLM_TIMEOUT, see ResolveLLMTimeout) so a slow-but-SUCCESSFUL completion
// returns a clean context.DeadlineExceeded error — mapped to 504 Gateway Timeout by
// CreateRequest — well before the ai-service http.Server's WriteTimeout could
// silently truncate the in-flight HTTP response underneath the synchronous call.
//
// Independent-review finding (T8-x, §11.4.108): making CreateRequest synchronous
// exposed server.go's pre-existing 15s WriteTimeout, while the LLM
// generic.Provider's own httpClient allows up to 120s (generic.DefaultTimeout) and
// CreateAIRequest.MaxTokens allows up to 32000 — so a slow-but-successful
// completion (>15s) got its HTTP response truncated by WriteTimeout even though the
// DB row was written correctly (the client saw a broken connection, not the real
// completion). DefaultLLMTimeout MUST stay comfortably BELOW the ai-service
// http.Server's WriteTimeout — see internal/server.DefaultHTTPWriteTimeout +
// internal/server.ResolveHTTPWriteTimeout, and the paired invariant test
// internal/server.TestHTTPWriteTimeoutExceedsLLMBudget.
const DefaultLLMTimeout = 90 * time.Second

// llmTimeoutEnvVar overrides DefaultLLMTimeout — see ResolveLLMTimeout.
const llmTimeoutEnvVar = "AI_LLM_TIMEOUT"

// ResolveLLMTimeout reads AI_LLM_TIMEOUT (a Go duration string, e.g. "90s") and
// returns it when present and valid (> 0); otherwise returns DefaultLLMTimeout.
// Exported so internal/server's WriteTimeout invariant test can assert the two
// timeouts stay correctly ordered without duplicating the env-var name or default.
func ResolveLLMTimeout() time.Duration {
	if v := os.Getenv(llmTimeoutEnvVar); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return DefaultLLMTimeout
}

// Repository defines the persistence operations the handler depends on. Satisfied by
// *repository.Repository in production; tests inject a fake (§11.4.27(A) — unit tests
// fake the collaborator, never the codebase-under-test).
type Repository interface {
	CreateRequest(ctx context.Context, req *model.AIRequest) error
	GetRequestByID(ctx context.Context, id uuid.UUID) (*model.AIRequest, error)
	ListRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.AIRequest, int, error)
	Ping(ctx context.Context) error
}

// LLMClient is the minimal real-completion contract CreateRequest depends on. Satisfied
// by *llmclient.GenericClient (HelixLLM local llama.cpp backend) in production; tests
// inject a fake. The cloud tier (OpenAI/Anthropic API keys) is OPERATOR-BLOCKED and
// deliberately not represented here — this service only ever talks to the local tier.
type LLMClient interface {
	Complete(ctx context.Context, model string, maxTokens int, temperature float64, prompt string) (content string, tokensUsed int, err error)
}

// Handler holds AI service handlers
type Handler struct {
	repo       Repository
	llm        LLMClient
	llmTimeout time.Duration
}

// New creates a new Handler. llm may be nil only for call sites that never invoke
// CreateRequest (e.g. health-check-only wiring in tests) — CreateRequest treats a nil
// llm as a real provider failure (Status: "failed"), never as a silent fabricated
// success, so passing nil never risks resurrecting the fabricated-"pending" bluff.
// llmTimeout is resolved from AI_LLM_TIMEOUT (or DefaultLLMTimeout) at construction
// time — see ResolveLLMTimeout.
func New(repo Repository, llm LLMClient) *Handler {
	return &Handler{repo: repo, llm: llm, llmTimeout: ResolveLLMTimeout()}
}

// CreateRequest handles AI request creation. It SYNCHRONOUSLY calls the configured
// local LLM provider (h.llm) BEFORE persisting — closing the fabricated-"pending"
// PASS-bluff (§11.4/§11.4.108) where the API accepted a request, wrote a placeholder
// row, and never produced any real completion. Status always lands on a real
// terminal value: "completed" when the provider returns a completion, "failed" when
// the provider errors (or is unconfigured) — "pending" is never written.
func (h *Handler) CreateRequest(c *gin.Context) {
	var req model.CreateAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, _ := c.Get("userID")
	var userID uuid.UUID
	if userIDStr != nil {
		userID, _ = uuid.Parse(userIDStr.(string))
	}

	aiReq := &model.AIRequest{
		ID:          uuid.New(),
		UserID:      userID,
		Prompt:      req.Prompt,
		Context:     req.Context,
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if h.llm == nil {
		// No LLM client configured — an honest failure, never a silent fabricated
		// "pending"/"completed". Production wiring (cmd/ai-service/main.go) always
		// injects a real client; a nil llm only reaches here in a misconfigured
		// deployment or a test that intentionally exercises this path.
		aiReq.Status = "failed"
	} else {
		// Bound the completion call to h.llmTimeout (§DefaultLLMTimeout) so an
		// over-budget call returns a clean context.DeadlineExceeded error instead of
		// running until the HTTP server's WriteTimeout silently truncates the
		// response underneath this synchronous call (T8-x finding).
		llmCtx := c.Request.Context()
		if h.llmTimeout > 0 {
			var cancel context.CancelFunc
			llmCtx, cancel = context.WithTimeout(llmCtx, h.llmTimeout)
			defer cancel()
		}
		content, tokensUsed, err := h.llm.Complete(llmCtx, req.Model, req.MaxTokens, req.Temperature, req.Prompt)
		if err != nil {
			// §11.4.1/§11.4.6 — the LLM failure MUST be surfaced, never silently
			// dropped: it is the only diagnostic signal an operator has for why a
			// request landed with Status "failed" instead of a real completion.
			log.Printf("ai-service: LLM completion failed (request %s): %v", aiReq.ID, err)
			aiReq.Status = "failed"
			if errors.Is(err, context.DeadlineExceeded) {
				// Clean timeout: the completion exceeded its bounded budget. Persist
				// the honest "failed" row for the operator's audit trail (best
				// effort — a persistence error here is logged, not fatal to the
				// timeout response), then respond 504 so the caller can
				// distinguish "provider overloaded/too slow" from an ordinary
				// provider error (which stays 201 + Status "failed" below).
				if perr := h.repo.CreateRequest(c.Request.Context(), aiReq); perr != nil {
					log.Printf("ai-service: failed to persist timed-out request %s: %v", aiReq.ID, perr)
				}
				c.JSON(http.StatusGatewayTimeout, gin.H{"error": "LLM completion timed out"})
				return
			}
		} else {
			aiReq.Response = content
			aiReq.TokensUsed = tokensUsed
			aiReq.Status = "completed"
		}
	}

	if err := h.repo.CreateRequest(c.Request.Context(), aiReq); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create AI request"})
		return
	}

	c.JSON(http.StatusCreated, toAIResponse(aiReq))
}

// GetRequest handles retrieving an AI request by ID
func (h *Handler) GetRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	req, err := h.repo.GetRequestByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "AI request not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "AI request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get AI request"})
		return
	}
	c.JSON(http.StatusOK, toAIResponse(req))
}

// ListRequests handles listing AI requests
func (h *Handler) ListRequests(c *gin.Context) {
	userIDStr, _ := c.Get("userID")
	var userID uuid.UUID
	if userIDStr != nil {
		userID, _ = uuid.Parse(userIDStr.(string))
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	reqs, total, err := h.repo.ListRequests(c.Request.Context(), userID, limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list AI requests"})
		return
	}

	resp := &model.ListAIRequestsResponse{
		Items:  make([]*model.AIResponse, len(reqs)),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
	for i, req := range reqs {
		resp.Items[i] = toAIResponse(req)
	}
	c.JSON(http.StatusOK, resp)
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "ai-service", "timestamp": time.Now().UTC()})
}

// ReadinessCheck returns service readiness status
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "database not available"})
		return
	}
	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "ai-service"})
}

func toAIResponse(req *model.AIRequest) *model.AIResponse {
	return &model.AIResponse{
		ID:         req.ID,
		UserID:     req.UserID,
		OrgID:      req.OrgID,
		Prompt:     req.Prompt,
		Response:   req.Response,
		Model:      req.Model,
		TokensUsed: req.TokensUsed,
		Status:     req.Status,
		CreatedAt:  req.CreatedAt,
	}
}
