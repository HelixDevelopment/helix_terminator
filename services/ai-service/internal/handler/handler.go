package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/ai-service/internal/model"
)

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
	repo Repository
	llm  LLMClient
}

// New creates a new Handler. llm may be nil only for call sites that never invoke
// CreateRequest (e.g. health-check-only wiring in tests) — CreateRequest treats a nil
// llm as a real provider failure (Status: "failed"), never as a silent fabricated
// success, so passing nil never risks resurrecting the fabricated-"pending" bluff.
func New(repo Repository, llm LLMClient) *Handler {
	return &Handler{repo: repo, llm: llm}
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
		content, tokensUsed, err := h.llm.Complete(c.Request.Context(), req.Model, req.MaxTokens, req.Temperature, req.Prompt)
		if err != nil {
			aiReq.Status = "failed"
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
