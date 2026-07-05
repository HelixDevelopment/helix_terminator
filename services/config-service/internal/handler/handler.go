package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/config-service/internal/model"
	"github.com/helixdevelopment/config-service/internal/repository"
)

// Handler holds config service handlers.
type Handler struct {
	repo *repository.Repository
}

// New returns a new Handler with dependencies.
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateConfig handles POST /api/v1/configs.
func (h *Handler) CreateConfig(c *gin.Context) {
	var req model.CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse optional scope_id
	var scopeID *uuid.UUID
	if req.ScopeID != nil && *req.ScopeID != "" {
		parsed, err := uuid.Parse(*req.ScopeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope_id"})
			return
		}
		scopeID = &parsed
	}

	// Validate scope_id requirement based on scope
	if req.Scope == "org" || req.Scope == "user" {
		if scopeID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "scope_id is required for org and user scopes"})
			return
		}
	}

	config := &model.Config{
		ID:          uuid.New(),
		Scope:       req.Scope,
		ScopeID:     scopeID,
		Key:         req.Key,
		Value:       req.Value,
		ValueType:   req.ValueType,
		Description: req.Description,
		IsSecret:    req.IsSecret,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := h.repo.CreateConfig(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create config"})
		return
	}

	c.JSON(http.StatusCreated, config.ToResponse())
}

// ListConfigs handles GET /api/v1/configs.
func (h *Handler) ListConfigs(c *gin.Context) {
	var req model.ListConfigsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var scopeID *uuid.UUID
	if req.ScopeID != "" {
		parsed, err := uuid.Parse(req.ScopeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope_id"})
			return
		}
		scopeID = &parsed
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	configs, total, err := h.repo.ListConfigs(c.Request.Context(), req.Scope, scopeID, req.Search, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list configs"})
		return
	}

	resp := &model.ListConfigsResponse{
		Configs: make([]*model.ConfigResponse, 0, len(configs)),
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}
	for _, config := range configs {
		resp.Configs = append(resp.Configs, config.ToResponse())
	}

	c.JSON(http.StatusOK, resp)
}

// GetConfig handles GET /api/v1/configs/:id.
func (h *Handler) GetConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	config, err := h.repo.GetConfigByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "config not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	c.JSON(http.StatusOK, config.ToResponse())
}

// GetConfigByKey handles GET /api/v1/configs/by-key.
func (h *Handler) GetConfigByKey(c *gin.Context) {
	scope := c.Query("scope")
	key := c.Query("key")
	if scope == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scope and key are required"})
		return
	}

	var scopeID *uuid.UUID
	scopeIDStr := c.Query("scope_id")
	if scopeIDStr != "" {
		parsed, err := uuid.Parse(scopeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope_id"})
			return
		}
		scopeID = &parsed
	}

	config, err := h.repo.GetConfigByKey(c.Request.Context(), scope, scopeID, key)
	if err != nil {
		if err.Error() == "config not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	c.JSON(http.StatusOK, config.ToResponse())
}

// UpdateConfig handles PUT /api/v1/configs/:id.
func (h *Handler) UpdateConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req model.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Value != nil {
		updates["value"] = *req.Value
	}
	if req.ValueType != nil {
		updates["value_type"] = *req.ValueType
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.IsSecret != nil {
		updates["is_secret"] = *req.IsSecret
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	if err := h.repo.UpdateConfig(c.Request.Context(), id, updates); err != nil {
		if err.Error() == "no updates provided" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update config"})
		return
	}

	config, err := h.repo.GetConfigByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch updated config"})
		return
	}

	c.JSON(http.StatusOK, config.ToResponse())
}

// DeleteConfig handles DELETE /api/v1/configs/:id.
func (h *Handler) DeleteConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteConfig(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete config"})
		return
	}

	c.Status(http.StatusNoContent)
}

// BulkCreateConfigs handles POST /api/v1/configs/bulk.
func (h *Handler) BulkCreateConfigs(c *gin.Context) {
	var reqs []model.CreateConfigRequest
	if err := c.ShouldBindJSON(&reqs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(reqs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no configs provided"})
		return
	}

	configs := make([]*model.Config, 0, len(reqs))
	now := time.Now().UTC()
	for _, req := range reqs {
		var scopeID *uuid.UUID
		if req.ScopeID != nil && *req.ScopeID != "" {
			parsed, err := uuid.Parse(*req.ScopeID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scope_id in bulk request"})
				return
			}
			scopeID = &parsed
		}
		if req.Scope == "org" || req.Scope == "user" {
			if scopeID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "scope_id is required for org and user scopes"})
				return
			}
		}
		configs = append(configs, &model.Config{
			ID:          uuid.New(),
			Scope:       req.Scope,
			ScopeID:     scopeID,
			Key:         req.Key,
			Value:       req.Value,
			ValueType:   req.ValueType,
			Description: req.Description,
			IsSecret:    req.IsSecret,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	if err := h.repo.BulkCreateConfigs(c.Request.Context(), configs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to bulk create configs"})
		return
	}

	resp := make([]*model.ConfigResponse, 0, len(configs))
	for _, config := range configs {
		resp = append(resp, config.ToResponse())
	}
	c.JSON(http.StatusCreated, resp)
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "config-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status (503 if no DB).
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":     false,
			"service":   "config-service",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":     false,
			"service":   "config-service",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"service":   "config-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
