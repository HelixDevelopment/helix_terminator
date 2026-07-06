package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/ai-service/internal/model"
	"github.com/helixdevelopment/ai-service/internal/repository"
)

// Handler holds AI service handlers
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateRequest handles AI request creation
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
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
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
