package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/workspace-service/internal/model"
	"github.com/helixdevelopment/workspace-service/internal/repository"
)

// Handler holds workspace service handlers.
type Handler struct {
	repo *repository.Repository
}

// New returns a new Handler with dependencies.
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateWorkspace handles POST /api/v1/workspaces.
func (h *Handler) CreateWorkspace(c *gin.Context) {
	var req model.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, _ := c.Get("userID")
	orgIDStr, _ := c.Get("orgID")
	var userID, orgID uuid.UUID
	if userIDStr != nil {
		userID, _ = uuid.Parse(userIDStr.(string))
	}
	if orgIDStr != nil {
		orgID, _ = uuid.Parse(orgIDStr.(string))
	}

	workspace := &model.Workspace{
		ID:          uuid.New(),
		OrgID:       orgID,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		Icon:        req.Icon,
		Tags:        req.Tags,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	if err := h.repo.CreateWorkspace(c.Request.Context(), workspace); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	// Add hosts if provided
	if len(req.HostIDs) > 0 {
		for _, hostID := range req.HostIDs {
			_ = h.repo.AddHost(c.Request.Context(), workspace.ID, hostID, userID)
		}
	}

	c.JSON(http.StatusCreated, model.WorkspaceResponse{Workspace: *workspace})
}

// ListWorkspaces handles GET /api/v1/workspaces.
func (h *Handler) ListWorkspaces(c *gin.Context) {
	var req model.ListWorkspacesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	var orgID, userID uuid.UUID
	if req.OrgID != "" {
		orgID, _ = uuid.Parse(req.OrgID)
	}
	if req.UserID != "" {
		userID, _ = uuid.Parse(req.UserID)
	}

	workspaces, total, err := h.repo.ListWorkspaces(c.Request.Context(), orgID, userID, req.Tags, req.Limit, req.Offset)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list workspaces"})
		return
	}

	c.JSON(http.StatusOK, model.ListWorkspacesResponse{
		Data:   workspaces,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
}

// GetWorkspace handles GET /api/v1/workspaces/:id.
func (h *Handler) GetWorkspace(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	workspace, err := h.repo.GetWorkspaceByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	// Load host IDs
	hostIDs, err := h.repo.ListHosts(c.Request.Context(), id)
	if err == nil {
		workspace.HostIDs = hostIDs
	}

	c.JSON(http.StatusOK, model.WorkspaceResponse{Workspace: *workspace})
}

// UpdateWorkspace handles PUT /api/v1/workspaces/:id.
func (h *Handler) UpdateWorkspace(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	var req model.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspace, err := h.repo.GetWorkspaceByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}

	if err := h.repo.UpdateWorkspace(c.Request.Context(), id, updates); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update workspace"})
		return
	}

	// Refresh workspace data
	workspace, _ = h.repo.GetWorkspaceByID(c.Request.Context(), id)
	if workspace == nil {
		workspace = &model.Workspace{ID: id}
	}

	// Handle host IDs replacement if provided
	if req.HostIDs != nil {
		existing, _ := h.repo.ListHosts(c.Request.Context(), id)
		existingMap := make(map[uuid.UUID]bool)
		for _, e := range existing {
			existingMap[e] = true
		}
		newMap := make(map[uuid.UUID]bool)
		for _, n := range req.HostIDs {
			newMap[n] = true
		}
		userIDStr, _ := c.Get("userID")
		var addedBy uuid.UUID
		if userIDStr != nil {
			addedBy, _ = uuid.Parse(userIDStr.(string))
		}
		for _, e := range existing {
			if !newMap[e] {
				_ = h.repo.RemoveHost(c.Request.Context(), id, e)
			}
		}
		for _, n := range req.HostIDs {
			if !existingMap[n] {
				_ = h.repo.AddHost(c.Request.Context(), id, n, addedBy)
			}
		}
		workspace.HostIDs, _ = h.repo.ListHosts(c.Request.Context(), id)
	} else {
		hostIDs, _ := h.repo.ListHosts(c.Request.Context(), id)
		workspace.HostIDs = hostIDs
	}

	c.JSON(http.StatusOK, model.WorkspaceResponse{Workspace: *workspace})
}

// DeleteWorkspace handles DELETE /api/v1/workspaces/:id.
func (h *Handler) DeleteWorkspace(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	if err := h.repo.DeleteWorkspace(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete workspace"})
		return
	}

	c.Status(http.StatusNoContent)
}

// AddHost handles POST /api/v1/workspaces/:id/hosts.
func (h *Handler) AddHost(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	var req model.AddHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, _ := c.Get("userID")
	var addedBy uuid.UUID
	if userIDStr != nil {
		addedBy, _ = uuid.Parse(userIDStr.(string))
	}

	if err := h.repo.AddHost(c.Request.Context(), workspaceID, req.HostID, addedBy); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add host"})
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveHost handles DELETE /api/v1/workspaces/:id/hosts/:host_id.
func (h *Handler) RemoveHost(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	hostID, err := uuid.Parse(c.Param("host_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	if err := h.repo.RemoveHost(c.Request.Context(), workspaceID, hostID); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove host"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListHosts handles GET /api/v1/workspaces/:id/hosts.
func (h *Handler) ListHosts(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	hostIDs, err := h.repo.ListHosts(c.Request.Context(), workspaceID)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list hosts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": hostIDs})
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "workspace-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status.
func (h *Handler) ReadinessCheck(c *gin.Context) {
	ready := true
	if h.repo == nil {
		ready = false
	} else if err := h.repo.Ping(c.Request.Context()); err != nil {
		ready = false
	}
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{
		"ready":     ready,
		"service":   "workspace-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
