package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() (*gin.Engine, *Handler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := New(nil)
	router.GET("/healthz", h.HealthCheck)
	router.GET("/healthz/ready", h.ReadinessCheck)
	return router, h
}

func TestHealthCheck(t *testing.T) {
	router, _ := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestReadinessCheck_NoDB(t *testing.T) {
	router, _ := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz/ready", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "not ready")
}

func TestCreateEvent_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/analytics/events", h.CreateEvent)

	body := map[string]interface{}{
		"eventType": "session",
		"payload":   map[string]string{"action": "login"},
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/analytics/events", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not available")
}

func TestGetEvent_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/analytics/events/:id", h.GetEvent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/events/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListEvents_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/analytics/events", h.ListEvents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/events", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCountByEventType_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/analytics/stats/event-types", h.CountByEventType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/stats/event-types", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListEvents_Pagination(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/analytics/events", h.ListEvents)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/events?limit=101&offset=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
