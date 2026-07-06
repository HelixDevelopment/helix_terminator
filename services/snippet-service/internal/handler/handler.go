package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/snippet-service/internal/model"
	"github.com/helixdevelopment/snippet-service/internal/repository"
)

// Handler contains HTTP handlers for snippets
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateSnippet creates a new snippet
func (h *Handler) CreateSnippet(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var req model.CreateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)
	snippet := &model.Snippet{
		ID:          uuid.New(),
		CreatedBy:   userID,
		Name:        req.Name,
		Content:     req.Content,
		Language:    req.Language,
		Tags:        req.Tags,
		Description: req.Description,
		IsPublic:    req.IsPublic,
		UsageCount:  0,
	}
	if err := h.repo.CreateSnippet(c.Request.Context(), snippet); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, snippet)
}

// GetSnippet retrieves a snippet by ID
func (h *Handler) GetSnippet(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snippet ID"})
		return
	}
	snippet, err := h.repo.GetSnippetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	// Increment usage count on read
	_ = h.repo.IncrementUsage(c.Request.Context(), id)
	c.JSON(http.StatusOK, snippet)
}

// ListSnippets retrieves snippets with filtering
func (h *Handler) ListSnippets(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var orgID *uuid.UUID
	var createdBy uuid.UUID
	if oStr := c.Query("org_id"); oStr != "" {
		id, err := uuid.Parse(oStr)
		if err == nil {
			orgID = &id
		}
	}
	if cStr := c.Query("created_by"); cStr != "" {
		id, err := uuid.Parse(cStr)
		if err == nil {
			createdBy = id
		}
	}
	language := c.Query("language")
	var isPublic *bool
	if pStr := c.Query("is_public"); pStr != "" {
		b, _ := strconv.ParseBool(pStr)
		isPublic = &b
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	snippets, total, err := h.repo.ListSnippets(c.Request.Context(), orgID, createdBy, language, isPublic, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   snippets,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateSnippet updates a snippet
func (h *Handler) UpdateSnippet(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snippet ID"})
		return
	}
	var req model.UpdateSnippetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	isPublic := interface{}(nil)
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"content":     req.Content,
		"language":    req.Language,
		"tags":        req.Tags,
		"description": req.Description,
		"is_public":   isPublic,
	}
	if err := h.repo.UpdateSnippet(c.Request.Context(), id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "snippet updated"})
}

// DeleteSnippet deletes a snippet
func (h *Handler) DeleteSnippet(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snippet ID"})
		return
	}
	if err := h.repo.DeleteSnippet(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "snippet deleted"})
}

// HealthCheck returns service health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// ReadinessCheck returns readiness status
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
		return
	}
	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
