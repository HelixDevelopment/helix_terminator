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

func TestCreateSession_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/sftp/sessions", h.CreateSession)

	body := map[string]interface{}{
		"hostId":     uuid.New().String(),
		"remotePath": "/remote/file.txt",
		"localPath":  "/local/file.txt",
		"direction":  "download",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sftp/sessions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not available")
}

func TestGetSession_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/sftp/sessions/:id", h.GetSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/sftp/sessions/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListSessions_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/sftp/sessions", h.ListSessions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/sftp/sessions", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateSession_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.PUT("/api/v1/sftp/sessions/:id", h.UpdateSession)

	body := map[string]interface{}{"status": "completed", "bytesTransferred": 1024}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/sftp/sessions/"+uuid.New().String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteSession_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.DELETE("/api/v1/sftp/sessions/:id", h.DeleteSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/sftp/sessions/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListSessions_Pagination(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/sftp/sessions", h.ListSessions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/sftp/sessions?limit=101&offset=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
