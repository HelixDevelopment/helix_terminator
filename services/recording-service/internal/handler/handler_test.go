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

func TestCreateRecording_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/recordings", h.CreateRecording)

	body := map[string]interface{}{
		"sessionId": uuid.New().String(),
		"hostId":    uuid.New().String(),
		"filePath":  "/recordings/session.cast",
		"format":    "asciinema",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/recordings", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not available")
}

func TestGetRecording_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/recordings/:id", h.GetRecording)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recordings/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListRecordings_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/recordings", h.ListRecordings)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recordings", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateRecording_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.PUT("/api/v1/recordings/:id", h.UpdateRecording)

	body := map[string]interface{}{"status": "completed", "durationSec": 120, "fileSizeBytes": 102400}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/recordings/"+uuid.New().String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteRecording_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.DELETE("/api/v1/recordings/:id", h.DeleteRecording)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/recordings/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListRecordings_Pagination(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/recordings", h.ListRecordings)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recordings?limit=101&offset=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
