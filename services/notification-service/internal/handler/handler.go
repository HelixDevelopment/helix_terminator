package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/notification-service/internal/model"
	"github.com/helixdevelopment/notification-service/internal/repository"
)

// Handler holds notification service handlers
type Handler struct {
	repo *repository.Repository
}

// New returns a new Handler with dependencies
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateNotification handles POST /api/v1/notifications
func (h *Handler) CreateNotification(c *gin.Context) {
	var req model.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var orgID *uuid.UUID
	if req.OrgID != "" {
		parsed, err := uuid.Parse(req.OrgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
			return
		}
		orgID = &parsed
	}

	status := req.Status
	if status == "" {
		status = "pending"
	}

	notification := &model.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		OrgID:     orgID,
		Type:      req.Type,
		Title:     req.Title,
		Message:   req.Message,
		Data:      req.Data,
		Channel:   req.Channel,
		Status:    status,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := h.repo.CreateNotification(c.Request.Context(), notification); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toNotificationResponse(notification))
}

// ListNotifications handles GET /api/v1/notifications
func (h *Handler) ListNotifications(c *gin.Context) {
	var req model.ListNotificationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var orgID *uuid.UUID
	if req.OrgID != "" {
		parsed, err := uuid.Parse(req.OrgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
			return
		}
		orgID = &parsed
	}

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	notifications, total, err := h.repo.ListNotifications(c.Request.Context(), userID, orgID, req.Status, req.Channel, limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	var responses []model.NotificationResponse
	for _, n := range notifications {
		responses = append(responses, toNotificationResponse(n))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   responses,
		"total":  total,
		"limit":  limit,
		"offset": req.Offset,
	})
}

// GetNotification handles GET /api/v1/notifications/:id
func (h *Handler) GetNotification(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	notification, err := h.repo.GetNotificationByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toNotificationResponse(notification))
}

// MarkRead handles POST /api/v1/notifications/:id/read
func (h *Handler) MarkRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.repo.MarkRead(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

// MarkAllRead handles POST /api/v1/notifications/read-all
func (h *Handler) MarkAllRead(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.repo.MarkAllRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all notifications marked as read"})
}

// DeleteNotification handles DELETE /api/v1/notifications/:id
func (h *Handler) DeleteNotification(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteNotification(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CountUnread handles GET /api/v1/notifications/unread-count
func (h *Handler) CountUnread(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	count, err := h.repo.CountUnread(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetPreference handles GET /api/v1/notifications/preferences
func (h *Handler) GetPreference(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	channel := c.Query("channel")
	if channel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel is required"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	pref, err := h.repo.GetPreference(c.Request.Context(), userID, channel)
	if err != nil {
		if err.Error() == "preference not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toPreferenceResponse(pref))
}

// UpdatePreference handles PUT /api/v1/notifications/preferences
func (h *Handler) UpdatePreference(c *gin.Context) {
	var req model.UpdatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	pref := &model.NotificationPreference{
		UserID:  userID,
		Channel: req.Channel,
		Enabled: req.Enabled,
		Types:   req.Types,
	}

	if err := h.repo.UpdatePreference(c.Request.Context(), pref); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toPreferenceResponse(pref))
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "notification-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"service": "notification-service",
			"error":   "database not connected",
		})
		return
	}

	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"service": "notification-service",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"service":   "notification-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func toNotificationResponse(n *model.Notification) model.NotificationResponse {
	var data json.RawMessage
	if n.Data != nil {
		data = json.RawMessage(n.Data)
	}
	return model.NotificationResponse{
		ID:        n.ID,
		UserID:    n.UserID,
		OrgID:     n.OrgID,
		Type:      n.Type,
		Title:     n.Title,
		Message:   n.Message,
		Data:      data,
		Channel:   n.Channel,
		Status:    n.Status,
		ReadAt:    n.ReadAt,
		SentAt:    n.SentAt,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}
}

func toPreferenceResponse(p *model.NotificationPreference) model.PreferenceResponse {
	return model.PreferenceResponse{
		UserID:    p.UserID,
		Channel:   p.Channel,
		Enabled:   p.Enabled,
		Types:     p.Types,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
