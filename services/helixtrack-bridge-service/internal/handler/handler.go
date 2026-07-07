package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/model"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/repository"
)

// Authenticator authenticates against a real HelixTrack Core instance (see
// internal/coreclient.Client, which satisfies this interface). Declared as a
// minimal interface here so handler tests can substitute a spy without a
// live Core server (unit-test layer, §11.4.27), while production wiring
// (cmd/helixtrack-bridge-service/main.go) injects the real coreclient.Client.
type Authenticator interface {
	EnsureAuthenticated(ctx context.Context) error
}

// Handler contains HTTP handlers for HelixTrack-bridge
type Handler struct {
	repo *repository.Repository
	core Authenticator
}

// New creates a new Handler. core MAY be nil (e.g. in tests that never reach
// CreateBridge's auth gate) but a nil core fails CLOSED — see authenticateCore.
func New(repo *repository.Repository, core Authenticator) *Handler {
	return &Handler{repo: repo, core: core}
}

// authenticateCore verifies (or refreshes) a real HelixTrack Core
// authentication before any bridge may be marked active. This is the
// anti-bluff fix for the fabricated-status defect (§11.4.108): CreateBridge
// previously set Status "active" unconditionally, without ever contacting a
// real Core. A nil Authenticator (misconfiguration) fails closed rather than
// fabricating success.
func (h *Handler) authenticateCore(ctx context.Context) error {
	if h.core == nil {
		return fmt.Errorf("helixtrack core client not configured")
	}
	return h.core.EnsureAuthenticated(ctx)
}

// CreateBridge creates a new HelixTrack bridge
func (h *Handler) CreateBridge(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var req model.CreateHelixTrackBridgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Anti-bluff gate (§11.4.108/§11.4.43/§11.4.115): Status MUST NEVER be
	// fabricated as "active" without a genuine authenticate call succeeding
	// against the running HelixTrack Core. Short-circuits BEFORE any DB
	// write is attempted.
	if err := h.authenticateCore(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": model.HelixTrackBridgeStatusError,
			"error":  err.Error(),
		})
		return
	}

	orgID, _ := uuid.Parse(req.OrgID)
	bridge := &model.HelixTrackBridge{
		ID:            uuid.New(),
		IntegrationID: req.IntegrationID,
		OrgID:         orgID,
		Name:          req.Name,
		Status:        model.HelixTrackBridgeStatusActive,
		Config:        req.Config,
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
	var orgID uuid.UUID
	if oStr := c.Query("org_id"); oStr != "" {
		id, err := uuid.Parse(oStr)
		if err == nil {
			orgID = id
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
	bridges, total, err := h.repo.ListBridges(c.Request.Context(), orgID, limit, offset)
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
	var req model.UpdateHelixTrackBridgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{
		"name":   req.Name,
		"status": req.Status,
		"config": req.Config,
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
