package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/vault-service/internal/model"
)

// Repository defines the interface for vault persistence operations.
type Repository interface {
	CreateSecret(ctx context.Context, secret *model.Secret) error
	GetSecretByID(ctx context.Context, id uuid.UUID) (*model.Secret, error)
	ListSecrets(ctx context.Context, userID, orgID uuid.UUID, secretType model.SecretType, tags []string, limit, offset int) ([]*model.Secret, error)
	UpdateSecret(ctx context.Context, secret *model.Secret) error
	DeleteSecret(ctx context.Context, id uuid.UUID) error
	CreateSecretVersion(ctx context.Context, version *model.SecretVersion) error
	GetSecretVersions(ctx context.Context, secretID uuid.UUID, limit int) ([]*model.SecretVersion, error)
	CountSecrets(ctx context.Context, userID, orgID uuid.UUID) (int, error)
	Ping(ctx context.Context) error
}

// Handler holds vault service handlers.
type Handler struct {
	repo Repository
}

// New returns a new Handler with dependencies.
func New(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// callerUserIDHeader is the request header conveying the caller's
// authenticated tenant identity. It is the SAME header
// server.tenantIsolationMiddleware validates for the secret-ID-scoped
// routes (GET/PUT/DELETE/rotate/versions) — the collection-level routes
// (ListSecrets, CreateSecret) have no target secret ID to check ownership
// against before the handler runs, so the handler itself is the
// authoritative point where the caller's identity MUST be derived and
// enforced, instead of trusting any caller-supplied user_id in the query
// string or JSON body (real IDOR / broken-object-level-authorization
// otherwise: T7).
const callerUserIDHeader = "X-User-ID"

// CallerUserID extracts and parses the caller's authenticated tenant
// identity from the X-User-ID header. Exported so server-layer middleware
// (tenantIsolationMiddleware and the collection-route caller-identity
// guard) share this exact parsing logic with the handlers that consume it,
// rather than maintaining two independently-drifting implementations of
// "what counts as a valid caller identity" (§11.4.124 reuse-don't-duplicate).
func CallerUserID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.GetHeader(callerUserIDHeader))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// CreateSecret handles POST /api/v1/vault/secrets.
//
// Security (T7 IDOR fix): the secret's owner is ALWAYS the authenticated
// caller (X-User-ID), never a body-supplied user_id — a caller could
// otherwise create/own secrets under another tenant's user_id. If the body
// carries a user_id that differs from the caller, the request is rejected
// (safer than silently overriding it, per the same-object convention used
// elsewhere in this service).
func (h *Handler) CreateSecret(c *gin.Context) {
	callerID, ok := CallerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid X-User-ID"})
		return
	}

	var req model.CreateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.UserID != uuid.Nil && req.UserID != callerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id in request body must match the authenticated caller (X-User-ID)"})
		return
	}

	secret := &model.Secret{
		ID:             uuid.New(),
		UserID:         callerID,
		OrgID:          req.OrgID,
		Name:           req.Name,
		Type:           model.SecretType(req.Type),
		EncryptedValue: req.EncryptedValue,
		IV:             req.IV,
		Salt:           req.Salt,
		Metadata:       req.Metadata,
		Tags:           req.Tags,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := h.repo.CreateSecret(c.Request.Context(), secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create secret"})
		return
	}

	c.JSON(http.StatusCreated, model.ToSecretResponse(secret))
}

// GetSecret handles GET /api/v1/vault/secrets/:id.
func (h *Handler) GetSecret(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid secret id"})
		return
	}

	secret, err := h.repo.GetSecretByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
		return
	}

	c.JSON(http.StatusOK, model.ToSecretResponse(secret))
}

// ListSecrets handles GET /api/v1/vault/secrets.
//
// Security (T7 IDOR fix): the tenant scope is ALWAYS the authenticated
// caller (X-User-ID), never a client-supplied user_id query param — the
// repository treats an empty/zero user_id filter as "no filter", so
// previously a caller could omit user_id entirely and list EVERY tenant's
// secrets, or supply another tenant's user_id and list theirs. A
// user_id query param is now permitted ONLY when it equals the caller's
// own identity (redundant no-op); any other value is rejected outright.
func (h *Handler) ListSecrets(c *gin.Context) {
	callerID, ok := CallerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid X-User-ID"})
		return
	}

	var req model.ListSecretsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.UserID != "" {
		requested, err := uuid.Parse(req.UserID)
		if err != nil || requested != callerID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query parameter must match the authenticated caller (X-User-ID)"})
			return
		}
	}

	var orgID uuid.UUID
	if req.OrgID != "" {
		orgID, _ = uuid.Parse(req.OrgID)
	}

	var tags []string
	if req.Tags != "" {
		tags = []string{req.Tags}
	}

	secrets, err := h.repo.ListSecrets(c.Request.Context(), callerID, orgID, model.SecretType(req.Type), tags, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list secrets"})
		return
	}

	total, err := h.repo.CountSecrets(c.Request.Context(), callerID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count secrets"})
		return
	}

	resp := &model.ListSecretsResponse{
		Secrets: make([]*model.SecretResponse, 0, len(secrets)),
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}
	for _, s := range secrets {
		resp.Secrets = append(resp.Secrets, model.ToSecretResponse(s))
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateSecret handles PUT /api/v1/vault/secrets/:id.
func (h *Handler) UpdateSecret(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid secret id"})
		return
	}

	var req model.UpdateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	secret, err := h.repo.GetSecretByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
		return
	}

	if req.Name != "" {
		secret.Name = req.Name
	}
	if req.EncryptedValue != "" {
		secret.EncryptedValue = req.EncryptedValue
	}
	if req.IV != "" {
		secret.IV = req.IV
	}
	if req.Salt != "" {
		secret.Salt = req.Salt
	}
	if req.Metadata != nil {
		secret.Metadata = req.Metadata
	}
	if req.Tags != nil {
		secret.Tags = req.Tags
	}
	secret.UpdatedAt = time.Now().UTC()

	if err := h.repo.UpdateSecret(c.Request.Context(), secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update secret"})
		return
	}

	c.JSON(http.StatusOK, model.ToSecretResponse(secret))
}

// DeleteSecret handles DELETE /api/v1/vault/secrets/:id.
func (h *Handler) DeleteSecret(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid secret id"})
		return
	}

	if err := h.repo.DeleteSecret(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete secret"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetSecretVersions handles GET /api/v1/vault/secrets/:id/versions.
func (h *Handler) GetSecretVersions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid secret id"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if parsed, err := strconv.Atoi(limitStr); err == nil && parsed >= 1 {
		limit = parsed
	}
	if limit > 100 {
		limit = 100
	}

	versions, err := h.repo.GetSecretVersions(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get secret versions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// RotateSecret handles POST /api/v1/vault/secrets/:id/rotate.
func (h *Handler) RotateSecret(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid secret id"})
		return
	}

	var req struct {
		EncryptedValue string    `json:"encrypted_value" binding:"required"`
		IV             string    `json:"iv" binding:"required"`
		Salt           string    `json:"salt" binding:"required"`
		CreatedBy      uuid.UUID `json:"created_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	secret, err := h.repo.GetSecretByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
		return
	}

	// Create a version snapshot of the current secret
	version := &model.SecretVersion{
		ID:             uuid.New(),
		SecretID:       secret.ID,
		EncryptedValue: secret.EncryptedValue,
		IV:             secret.IV,
		Salt:           secret.Salt,
		CreatedBy:      req.CreatedBy,
		CreatedAt:      time.Now().UTC(),
	}
	if err := h.repo.CreateSecretVersion(c.Request.Context(), version); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create secret version"})
		return
	}

	// Update the secret with the new encrypted value
	secret.EncryptedValue = req.EncryptedValue
	secret.IV = req.IV
	secret.Salt = req.Salt
	secret.UpdatedAt = time.Now().UTC()

	if err := h.repo.UpdateSecret(c.Request.Context(), secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update secret after rotation"})
		return
	}

	c.JSON(http.StatusOK, model.ToSecretResponse(secret))
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "vault-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status including DB connectivity.
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"service": "vault-service",
			"error":   "repository not initialized",
		})
		return
	}

	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"service": "vault-service",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"service":   "vault-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
