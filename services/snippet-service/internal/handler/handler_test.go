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

func TestCreateSnippet_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/snippets", h.CreateSnippet)

	body := map[string]interface{}{
		"name":     "test-snippet",
		"content":  "echo hello",
		"language": "bash",
		"tags":     []string{"test"},
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/snippets", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not available")
}

func TestGetSnippet_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/snippets/:id", h.GetSnippet)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/snippets/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListSnippets_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/snippets", h.ListSnippets)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/snippets", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateSnippet_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.PUT("/api/v1/snippets/:id", h.UpdateSnippet)

	body := map[string]interface{}{"name": "updated"}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/snippets/"+uuid.New().String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteSnippet_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.DELETE("/api/v1/snippets/:id", h.DeleteSnippet)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/snippets/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListSnippets_Pagination(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/snippets", h.ListSnippets)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/snippets?limit=101&offset=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
