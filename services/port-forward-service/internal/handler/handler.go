package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"

	"github.com/helixdevelopment/port-forward-service/internal/forwarder"
	"github.com/helixdevelopment/port-forward-service/internal/model"
	"github.com/helixdevelopment/port-forward-service/internal/sshclient"
)

// ForwardRepository is the persistence contract the handler needs. It is
// satisfied by *repository.Repository in production and by lightweight
// fakes in tests (Constitution §11.4.98 — tests must be fully automated and
// re-runnable without a live database dependency for the tunnel-lifecycle
// paths).
type ForwardRepository interface {
	Ping(ctx context.Context) error
	CreateForward(ctx context.Context, forward *model.PortForward) error
	GetForwardByID(ctx context.Context, id uuid.UUID) (*model.PortForward, error)
	ListForwards(ctx context.Context, hostID uuid.UUID, limit, offset int) ([]*model.PortForward, int, error)
	UpdateForward(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	DeleteForward(ctx context.Context, id uuid.UUID) error
}

// Handler contains HTTP handlers for port-forward
type Handler struct {
	repo       ForwardRepository
	manager    *forwarder.Manager
	authorizer *forwarder.Authorizer
}

// New creates a new Handler
func New(repo ForwardRepository) *Handler {
	return &Handler{
		repo:       repo,
		manager:    forwarder.NewManager(),
		authorizer: forwarder.NewAuthorizerFromEnv(),
	}
}

// CreateForward creates a new port-forward CATALOG entry. It does NOT
// establish a tunnel — Status is set to "pending" (honest: no real tunnel
// exists yet). Use StartForward to really bring the tunnel up. High-blast-
// radius forward types (remote/-R, dynamic/-D) are gated here too
// (fail-fast) in addition to the authoritative check at Start time.
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

	forwardType := req.ForwardType
	if forwardType == "" {
		forwardType = model.ForwardTypeLocal
	}
	authType := req.AuthType
	if authType == "" {
		authType = model.AuthTypeKey
	}
	bindAddress := req.BindAddress
	if bindAddress == "" {
		bindAddress = "127.0.0.1"
	}
	sshPort := req.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	if forwardType == model.ForwardTypeLocal || forwardType == model.ForwardTypeRemote {
		if req.RemoteHost == "" || req.RemotePort == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": model.ErrMissingTarget.Error()})
			return
		}
	}

	if err := h.authorizer.Authorize(forwardType, req.SSHHost); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	hostID, _ := uuid.Parse(req.HostID)
	forward := &model.PortForward{
		ID:          uuid.New(),
		HostID:      hostID,
		ForwardType: forwardType,
		LocalPort:   req.LocalPort,
		RemotePort:  req.RemotePort,
		RemoteHost:  req.RemoteHost,
		Protocol:    req.Protocol,
		BindAddress: bindAddress,
		SSHHost:     req.SSHHost,
		SSHPort:     sshPort,
		SSHUsername: req.SSHUsername,
		AuthType:    authType,
		Status:      model.PortForwardStatusPending,
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

// UpdateForward updates a forward's editable metadata
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

// DeleteForward soft-deletes a forward. If a tunnel is currently running for
// it, the tunnel is really torn down first so nothing is orphaned.
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
	_ = h.manager.Stop(id) // best-effort: ErrTunnelNotRunning is fine here
	if err := h.repo.DeleteForward(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "forward deleted"})
}

// StartForward really establishes the SSH tunnel for a catalog entry: a
// real SSH connection is dialed and a real listener is bound (or, for
// remote forwards, really requested from the SSH server) BEFORE Status is
// ever reported as "active". The high-blast-radius gate is re-checked here
// too (defense in depth — the authoritative enforcement point, since this
// is where the real tunnel actually comes up).
func (h *Handler) StartForward(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid forward ID"})
		return
	}

	fwd, err := h.repo.GetForwardByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if err := h.authorizer.Authorize(fwd.ForwardType, fwd.SSHHost); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	var req model.StartForwardRequest
	// Body is optional for auth_type=agent.
	_ = c.ShouldBindJSON(&req)

	authMethod, err := authMethodFor(fwd.AuthType, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg := forwarder.Config{
		ID:          id,
		ForwardType: fwd.ForwardType,
		BindAddress: fwd.BindAddress,
		LocalPort:   fwd.LocalPort,
		RemoteHost:  fwd.RemoteHost,
		RemotePort:  fwd.RemotePort,
		SSHHost:     fwd.SSHHost,
		SSHPort:     fwd.SSHPort,
		SSHUsername: fwd.SSHUsername,
		AuthMethod:  authMethod,
	}

	tunnel, err := h.manager.Start(c.Request.Context(), cfg)
	if err != nil {
		// A failed Start MUST NOT report Active — record the real outcome.
		_ = h.repo.UpdateStatus(c.Request.Context(), id, model.PortForwardStatusError)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// The tunnel is genuinely up at this point: listener bound + SSH
	// connected (or, for remote forwards, the server accepted the
	// tcpip-forward request). Only now is Status allowed to become Active.
	if err := h.repo.UpdateStatus(c.Request.Context(), id, model.PortForwardStatusActive); err != nil {
		_ = h.manager.Stop(id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.StartForwardResponse{
		ID:           id,
		Status:       model.PortForwardStatusActive,
		BoundAddress: tunnel.Addr().String(),
	})
}

// StopForward really tears the tunnel down (closes the listener + SSH
// connection) and records the real resulting state.
func (h *Handler) StopForward(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid forward ID"})
		return
	}

	if err := h.manager.Stop(id); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.UpdateStatus(c.Request.Context(), id, model.PortForwardStatusStopped); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "forward stopped", "status": model.PortForwardStatusStopped})
}

// GetForwardMetrics returns REAL, live traffic counters for a running
// tunnel — never fabricated. 404 when no tunnel is currently running for
// the given ID.
func (h *Handler) GetForwardMetrics(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid forward ID"})
		return
	}
	tunnel, ok := h.manager.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": forwarder.ErrTunnelNotRunning.Error()})
		return
	}
	c.JSON(http.StatusOK, tunnel.Metrics())
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

// authMethodFor builds the real ssh.AuthMethod for the persisted authType,
// using the transient secret material from the Start request. Secrets are
// used once here and never stored (Constitution §11.4.10).
func authMethodFor(authType string, req model.StartForwardRequest) (ssh.AuthMethod, error) {
	switch authType {
	case model.AuthTypePassword:
		if req.Password == "" {
			return nil, model.ErrMissingCredential
		}
		return sshclient.AuthMethodFromPassword(req.Password), nil
	case model.AuthTypeKey:
		if req.PrivateKey == "" {
			return nil, model.ErrMissingCredential
		}
		return sshclient.AuthMethodFromKey(req.PrivateKey)
	case model.AuthTypeAgent:
		am := sshclient.AuthMethodFromAgent()
		if am == nil {
			return nil, model.ErrMissingCredential
		}
		return am, nil
	default:
		return nil, model.ErrMissingCredential
	}
}
