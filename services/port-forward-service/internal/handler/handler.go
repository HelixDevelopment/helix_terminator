package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/port-forward-service/internal/model"
	"github.com/helixdevelopment/port-forward-service/internal/repository"
)

// Handler contains HTTP handlers for port-forward
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateForward creates a new port forward
func (h *Handler) CreateForward(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var req model.CreatePortForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hostID, _ := uuid.Parse(req.HostID)
	forward := &model.PortForward{
		ID:         uuid.New(),
		HostID:     hostID,
		LocalPort:  req.LocalPort,
		RemotePort: req.RemotePort,
		RemoteHost: req.RemoteHost,
		Protocol:   req.Protocol,
		Status:     model.PortForwardStatusActive,
	}
	if err := h.repo.CreateForward(c.Request.Context(), forward); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, forward)
}

// GetForward retrieves a forward by ID
func (h *Handler) GetForward(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid forward ID"})
		return
	}
	forward, err := h.repo.GetForwardByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, forward)
}

// ListForwards retrieves forwards with filtering
func (h *Handler) ListForwards(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var hostID uuid.UUID
	if hStr := c.Query("host_id"); hStr != "" {
		id, err := uuid.Parse(hStr)
		if err == nil {
			hostID = id
		}
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	forwards, total, err := h.repo.ListForwards(c.Request.Context(), hostID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   forwards,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateForward updates a forward
func (h *Handler) UpdateForward(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid forward ID"})
		return
	}
	var req model.UpdatePortForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{
		"local_port":  req.LocalPort,
		"remote_port": req.RemotePort,
		"remote_host": req.RemoteHost,
		"protocol":    req.Protocol,
		"status":      req.Status,
	}
	if err := h.repo.UpdateForward(c.Request.Context(), id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "forward updated"})
}

// DeleteForward soft-deletes a forward
func (h *Handler) DeleteForward(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid forward ID"})
		return
	}
	if err := h.repo.DeleteForward(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "forward deleted"})
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
