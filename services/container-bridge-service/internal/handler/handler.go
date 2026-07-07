package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/container-bridge-service/internal/containerrt"
	"github.com/helixdevelopment/container-bridge-service/internal/model"
)

// BridgeStore is the persistence surface Handler needs. It is satisfied
// structurally by *repository.Repository (the real, Postgres-backed store)
// and by any test fake, without repository.go needing to change or the
// handler package needing to import it directly.
type BridgeStore interface {
	CreateBridge(ctx context.Context, bridge *model.ContainerBridge) error
	GetBridgeByID(ctx context.Context, id uuid.UUID) (*model.ContainerBridge, error)
	ListBridges(ctx context.Context, hostID uuid.UUID, limit, offset int) ([]*model.ContainerBridge, int, error)
	UpdateBridge(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	DeleteBridge(ctx context.Context, id uuid.UUID) error
	Ping(ctx context.Context) error
}

// Handler contains HTTP handlers for container-bridge
type Handler struct {
	repo    BridgeStore
	backend containerrt.Backend
}

// New creates a new Handler. backend may be nil when no supported container
// runtime was detected at startup; every route that needs it degrades to an
// honest 503 rather than fabricating container state.
func New(repo BridgeStore, backend containerrt.Backend) *Handler {
	return &Handler{repo: repo, backend: backend}
}

// CreateBridge creates a new container bridge
func (h *Handler) CreateBridge(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var req model.CreateContainerBridgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hostID, _ := uuid.Parse(req.HostID)
	bridge := &model.ContainerBridge{
		ID:          uuid.New(),
		HostID:      hostID,
		ContainerID: req.ContainerID,
		Name:        req.Name,
		Image:       req.Image,
		Status:      model.ContainerBridgeStatusActive,
		Ports:       req.Ports,
	}
	if err := h.repo.CreateBridge(c.Request.Context(), bridge); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bridge)
}

// GetBridge retrieves a bridge by ID
func (h *Handler) GetBridge(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bridge ID"})
		return
	}
	bridge, err := h.repo.GetBridgeByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bridge)
}

// ListBridges retrieves bridges with filtering
func (h *Handler) ListBridges(c *gin.Context) {
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
	bridges, total, err := h.repo.ListBridges(c.Request.Context(), hostID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   bridges,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateBridge updates a bridge
func (h *Handler) UpdateBridge(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bridge ID"})
		return
	}
	var req model.UpdateContainerBridgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{
		"name":   req.Name,
		"image":  req.Image,
		"status": req.Status,
		"ports":  req.Ports,
	}
	if err := h.repo.UpdateBridge(c.Request.Context(), id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bridge updated"})
}

// DeleteBridge deletes a bridge
func (h *Handler) DeleteBridge(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bridge ID"})
		return
	}
	if err := h.repo.DeleteBridge(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bridge deleted"})
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
