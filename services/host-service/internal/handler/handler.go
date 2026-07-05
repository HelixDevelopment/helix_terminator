package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/host-service/internal/model"
	"github.com/helixdevelopment/host-service/internal/repository"
)

// Handler holds host service handlers.
type Handler struct {
	repo *repository.Repository
}

// New returns a new Handler with dependencies.
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateHost handles POST /api/v1/hosts.
func (h *Handler) CreateHost(c *gin.Context) {
	var req model.CreateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Port == 0 {
		req.Port = 22
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

	host := &model.Host{
		ID:               uuid.New(),
		UserID:           userID,
		OrgID:            orgID,
		Name:             req.Name,
		Hostname:         req.Hostname,
		Port:             req.Port,
		Username:         req.Username,
		AuthType:         req.AuthType,
		VaultSecretID:    req.VaultSecretID,
		ConnectionParams: req.ConnectionParams,
		Tags:             req.Tags,
		ConnectionStatus: model.StatusUnknown,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	if err := h.repo.CreateHost(c.Request.Context(), host); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, model.HostResponse{Host: *host})
}

// GetHost handles GET /api/v1/hosts/:id.
func (h *Handler) GetHost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	host, err := h.repo.GetHostByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	c.JSON(http.StatusOK, model.HostResponse{Host: *host})
}

// ListHosts handles GET /api/v1/hosts.
func (h *Handler) ListHosts(c *gin.Context) {
	var req model.ListHostsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Limit == 0 {
		req.Limit = 20
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

	hosts, err := h.repo.ListHosts(c.Request.Context(), userID, orgID, req.Tags, req.Status, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list hosts"})
		return
	}

	count, err := h.repo.CountHosts(c.Request.Context(), userID, orgID)
	if err != nil {
		count = len(hosts)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   hosts,
		"total":  count,
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}

// UpdateHost handles PUT /api/v1/hosts/:id.
func (h *Handler) UpdateHost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	var req model.UpdateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	host, err := h.repo.GetHostByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	if req.Name != "" {
		host.Name = req.Name
	}
	if req.Hostname != "" {
		host.Hostname = req.Hostname
	}
	if req.Port != 0 {
		host.Port = req.Port
	}
	if req.Username != "" {
		host.Username = req.Username
	}
	if req.AuthType != "" {
		host.AuthType = req.AuthType
	}
	if req.VaultSecretID != nil {
		host.VaultSecretID = req.VaultSecretID
	}
	if req.ConnectionParams != nil {
		host.ConnectionParams = req.ConnectionParams
	}
	if req.Tags != nil {
		host.Tags = req.Tags
	}
	host.UpdatedAt = time.Now().UTC()

	if err := h.repo.UpdateHost(c.Request.Context(), host); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update host"})
		return
	}

	c.JSON(http.StatusOK, model.HostResponse{Host: *host})
}

// DeleteHost handles DELETE /api/v1/hosts/:id.
func (h *Handler) DeleteHost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	if err := h.repo.DeleteHost(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete host"})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestConnection handles POST /api/v1/hosts/:id/test-connection.
func (h *Handler) TestConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	_, err = h.repo.GetHostByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	c.JSON(http.StatusOK, model.TestConnectionResponse{
		Success: false,
		Message: "not implemented",
	})
}

// GetConnectionLogs handles GET /api/v1/hosts/:id/logs.
func (h *Handler) GetConnectionLogs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}

	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := h.repo.GetConnectionLogs(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get connection logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "host-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status.
func (h *Handler) ReadinessCheck(c *gin.Context) {
	ready := true
	if h.repo != nil {
		if err := h.repo.Ping(c.Request.Context()); err != nil {
			ready = false
		}
	}
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{
		"ready":     ready,
		"service":   "host-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
