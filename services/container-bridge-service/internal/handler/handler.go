package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	ctrruntime "digital.vasic.containers/pkg/runtime"
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

// CreateBridge creates a new container bridge. It NEVER writes a fabricated
// "active" status: the container is actually brought up (attach-and-start an
// existing container by req.ContainerID, or run a brand-new one from
// req.Image) and the persisted Status is derived from the REAL, runtime-
// confirmed container state per §11.4.108 (SOURCE/DB-committed status ==
// RUNTIME-CONFIRMED status, never assumed).
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

	ctx := c.Request.Context()
	if h.backend == nil || !h.backend.IsAvailable(ctx) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "container runtime not available"})
		return
	}

	containerID, status, err := h.bringUp(ctx, req.ContainerID, req.Image, req.Ports)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	hostID, _ := uuid.Parse(req.HostID)
	bridge := &model.ContainerBridge{
		ID:          uuid.New(),
		HostID:      hostID,
		ContainerID: containerID,
		Name:        req.Name,
		Image:       req.Image,
		Status:      status,
		Ports:       req.Ports,
	}
	if err := h.repo.CreateBridge(ctx, bridge); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bridge)
}

// bringUp ensures a real container backs this bridge and returns its
// resolved container ID plus its REAL, runtime-confirmed status. It never
// returns model.ContainerBridgeStatusActive unless h.backend.Status
// genuinely reports the container running.
//
// If containerID already names an existing container (attach case), it is
// started if not already running. Otherwise a brand-new container is run
// from image, named containerID (the client-supplied logical name).
func (h *Handler) bringUp(
	ctx context.Context, containerID, image string, ports []string,
) (string, string, error) {
	if containerID != "" {
		if st, err := h.backend.Status(ctx, containerID); err == nil {
			if st.State != ctrruntime.StateRunning {
				if startErr := h.backend.Start(ctx, containerID); startErr != nil {
					return "", "", fmt.Errorf(
						"start existing container %s: %w", containerID, startErr)
				}
				st, err = h.backend.Status(ctx, containerID)
				if err != nil {
					return "", "", fmt.Errorf(
						"status after start %s: %w", containerID, err)
				}
			}
			return containerID, containerrt.StatusFromState(st.State), nil
		}
	}

	if image == "" {
		return "", "", fmt.Errorf(
			"no existing container %q and no image given to create one from", containerID)
	}
	name := containerID
	if name == "" {
		name = "bridge-" + uuid.New().String()
	}
	newID, err := h.backend.RunFromImage(ctx, name, image, ports)
	if err != nil {
		return "", "", fmt.Errorf("create container from image %s: %w", image, err)
	}
	st, err := h.backend.Status(ctx, newID)
	if err != nil {
		return "", "", fmt.Errorf("status after create %s: %w", newID, err)
	}
	if st.State != ctrruntime.StateRunning {
		return "", "", fmt.Errorf(
			"container %s did not reach running state (state=%s)", newID, st.State)
	}
	return newID, containerrt.StatusFromState(st.State), nil
}

// reconcile overwrites bridge.Status (in the response only) with the REAL
// status the container runtime reports, and best-effort persists the
// correction so subsequent reads do not keep recomputing from a stale row.
// A bridge whose container has disappeared entirely is honestly inactive,
// never left reporting a stale "active".
func (h *Handler) reconcile(ctx context.Context, bridge *model.ContainerBridge) {
	if h.backend == nil || bridge == nil || bridge.ContainerID == "" {
		return
	}
	if !h.backend.IsAvailable(ctx) {
		return
	}
	var real string
	if st, err := h.backend.Status(ctx, bridge.ContainerID); err != nil {
		real = model.ContainerBridgeStatusInactive
	} else {
		real = containerrt.StatusFromState(st.State)
	}
	if real == bridge.Status {
		return
	}
	bridge.Status = real
	// Best-effort: a failed write here does not invalidate this response,
	// which already carries the honest, reconciled status.
	_ = h.repo.UpdateBridge(ctx, bridge.ID, map[string]interface{}{
		"name":   bridge.Name,
		"image":  bridge.Image,
		"status": real,
		"ports":  bridge.Ports,
	})
}

// GetBridge retrieves a bridge by ID. Its Status is reconciled against the
// REAL container runtime state before being returned — a stopped/removed
// container must never still read "active" (§11.4.108).
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
	ctx := c.Request.Context()
	bridge, err := h.repo.GetBridgeByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.reconcile(ctx, bridge)
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
	ctx := c.Request.Context()
	bridges, total, err := h.repo.ListBridges(ctx, hostID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, bridge := range bridges {
		h.reconcile(ctx, bridge)
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

// DeleteBridge actually Stops+Removes the backing container via the runtime
// (best-effort — a container already stopped/removed manually must not block
// deleting the bridge record) before deleting the row.
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

	ctx := c.Request.Context()
	if bridge, getErr := h.repo.GetBridgeByID(ctx, id); getErr == nil &&
		bridge.ContainerID != "" && h.backend != nil && h.backend.IsAvailable(ctx) {
		_ = h.backend.Stop(ctx, bridge.ContainerID, ctrruntime.WithStopTimeout(10*time.Second))
		_ = h.backend.Remove(ctx, bridge.ContainerID,
			ctrruntime.WithForceRemove(true), ctrruntime.WithRemoveVolumes(true))
	}

	if err := h.repo.DeleteBridge(ctx, id); err != nil {
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
