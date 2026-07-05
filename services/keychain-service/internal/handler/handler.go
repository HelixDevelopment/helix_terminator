package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/keychain-service/internal/model"
	"github.com/helixdevelopment/keychain-service/internal/repository"
)

// Handler holds keychain service handlers
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateItem handles keychain item creation
func (h *Handler) CreateItem(c *gin.Context) {
	var req model.CreateKeychainItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr, _ := c.Get("userID")
	var userID uuid.UUID
	if userIDStr != nil {
		userID, _ = uuid.Parse(userIDStr.(string))
	}

	item := &model.KeychainItem{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       req.Name,
		Type:       model.KeyType(req.Type),
		PublicKey:  req.PublicKey,
		PrivateKey: req.PrivateKey,
		Passphrase: req.Passphrase,
		Metadata:   req.Metadata,
		Tags:       req.Tags,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := h.repo.CreateItem(c.Request.Context(), item); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create keychain item"})
		return
	}

	c.JSON(http.StatusCreated, toItemResponse(item))
}

// GetItem handles retrieving a keychain item by ID
func (h *Handler) GetItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	item, err := h.repo.GetItemByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "keychain item not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "keychain item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get keychain item"})
		return
	}
	c.JSON(http.StatusOK, toItemResponse(item))
}

// ListItems handles listing keychain items with filtering
func (h *Handler) ListItems(c *gin.Context) {
	var req model.ListKeychainItemsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	var userID, orgID uuid.UUID
	if req.UserID != "" {
		userID, _ = uuid.Parse(req.UserID)
	}
	if req.OrgID != "" {
		orgID, _ = uuid.Parse(req.OrgID)
	}

	items, total, err := h.repo.ListItems(c.Request.Context(), userID, orgID, req.Type, req.Limit, req.Offset)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list keychain items"})
		return
	}

	resp := &model.ListKeychainItemsResponse{
		Items:  make([]*model.KeychainItemResponse, len(items)),
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}
	for i, item := range items {
		resp.Items[i] = toItemResponse(item)
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateItem handles updating a keychain item
func (h *Handler) UpdateItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req model.UpdateKeychainItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.PublicKey != nil {
		updates["public_key"] = *req.PublicKey
	}
	if req.Metadata != nil {
		updates["metadata"] = req.Metadata
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}

	if err := h.repo.UpdateItem(c.Request.Context(), id, updates); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "keychain item not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "keychain item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update keychain item"})
		return
	}

	item, err := h.repo.GetItemByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated item"})
		return
	}
	c.JSON(http.StatusOK, toItemResponse(item))
}

// DeleteItem handles soft-deleting a keychain item
func (h *Handler) DeleteItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	if err := h.repo.DeleteItem(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "keychain item not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "keychain item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete keychain item"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "keychain-service", "timestamp": time.Now().UTC()})
}

// ReadinessCheck returns service readiness status
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "database not available"})
		return
	}
	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "keychain-service"})
}

func toItemResponse(item *model.KeychainItem) *model.KeychainItemResponse {
	return &model.KeychainItemResponse{
		ID:          item.ID,
		UserID:      item.UserID,
		OrgID:       item.OrgID,
		Name:        item.Name,
		Type:        string(item.Type),
		Fingerprint: item.Fingerprint,
		PublicKey:   item.PublicKey,
		Metadata:    item.Metadata,
		Tags:        item.Tags,
		CreatedAt:   item.CreatedAt,
	}
}
