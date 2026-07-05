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

// CreateSecret handles POST /api/v1/vault/secrets.
func (h *Handler) CreateSecret(c *gin.Context) {
	var req model.CreateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	secret := &model.Secret{
		ID:             uuid.New(),
		UserID:         req.UserID,
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
func (h *Handler) ListSecrets(c *gin.Context) {
	var req model.ListSecretsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID, orgID uuid.UUID
	if req.UserID != "" {
		userID, _ = uuid.Parse(req.UserID)
	}
	if req.OrgID != "" {
		orgID, _ = uuid.Parse(req.OrgID)
	}

	var tags []string
	if req.Tags != "" {
		tags = []string{req.Tags}
	}

	secrets, err := h.repo.ListSecrets(c.Request.Context(), userID, orgID, model.SecretType(req.Type), tags, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list secrets"})
		return
	}

	total, err := h.repo.CountSecrets(c.Request.Context(), userID, orgID)
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
		EncryptedValue string         `json:"encrypted_value" binding:"required"`
		IV             string         `json:"iv" binding:"required"`
		Salt           string         `json:"salt" binding:"required"`
		CreatedBy      uuid.UUID      `json:"created_by" binding:"required"`
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
