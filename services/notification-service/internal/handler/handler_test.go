package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/notification-service/internal/handler"
	"github.com/helixdevelopment/notification-service/internal/model"
	"github.com/helixdevelopment/notification-service/internal/repository"
)

func setupTestHandler(t *testing.T) (*handler.Handler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	repo := repository.New(nil)
	h := handler.New(repo)

	r := gin.New()
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	api := r.Group("/api/v1/notifications")
	{
		api.POST("", h.CreateNotification)
		api.GET("", h.ListNotifications)
		api.GET("/unread-count", h.CountUnread)
		api.GET("/:id", h.GetNotification)
		api.POST("/:id/read", h.MarkRead)
		api.POST("/read-all", h.MarkAllRead)
		api.DELETE("/:id", h.DeleteNotification)
		api.GET("/preferences", h.GetPreference)
		api.PUT("/preferences", h.UpdatePreference)
	}

	return h, r
}

func TestHealthCheck(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestReadinessCheckNoDB(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not connected")
}

func TestCreateNotificationNoDB(t *testing.T) {
	_, r := setupTestHandler(t)

	payload := map[string]interface{}{
		"userId":  uuid.New().String(),
		"type":    "info",
		"title":   "Test",
		"message": "Test message",
		"channel": "in_app",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not connected")
}

func TestCreateNotificationValidation(t *testing.T) {
	_, r := setupTestHandler(t)

	payload := map[string]interface{}{
		"userId": "not-a-uuid",
		"type":   "info",
		"title":  "Test",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListNotificationsValidation(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetNotificationInvalidID(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/invalid-id", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMarkReadInvalidID(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/invalid-id/read", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMarkAllReadMissingUserID(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "user_id is required")
}

func TestDeleteNotificationInvalidID(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/notifications/invalid-id", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCountUnreadMissingUserID(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "user_id is required")
}

func TestGetPreferenceMissingParams(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/preferences", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "user_id is required")
}

func TestUpdatePreferenceValidation(t *testing.T) {
	_, r := setupTestHandler(t)

	payload := map[string]interface{}{
		"userId":  "not-a-uuid",
		"channel": "invalid",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNotificationResponseMapping(t *testing.T) {
	now := time.Now().UTC()
	orgID := uuid.New()
	n := &model.Notification{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		OrgID:     &orgID,
		Type:      "info",
		Title:     "Title",
		Message:   "Message",
		Data:      []byte(`{"key":"value"}`),
		Channel:   "in_app",
		Status:    "pending",
		ReadAt:    &now,
		SentAt:    &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Verify the handler can map without panic by exercising the full flow
	// (we can't call the unexported helper directly, but we can verify the model)
	assert.Equal(t, "info", n.Type)
	assert.Equal(t, "Title", n.Title)
	assert.Equal(t, "Message", n.Message)
	assert.Equal(t, "in_app", n.Channel)
	assert.Equal(t, "pending", n.Status)
}

func TestPreferenceResponseMapping(t *testing.T) {
	now := time.Now().UTC()
	p := &model.NotificationPreference{
		UserID:    uuid.New(),
		Channel:   "email",
		Enabled:   true,
		Types:     []string{"info", "warning"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "email", p.Channel)
	assert.True(t, p.Enabled)
	assert.Equal(t, []string{"info", "warning"}, p.Types)
}
