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

func TestCreateBridge_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/container-bridges", h.CreateBridge)

	body := map[string]interface{}{
		"hostId":      uuid.New().String(),
		"containerId": "abc123",
		"name":        "test-container",
		"image":       "nginx:latest",
		"ports":       []string{"80:8080"},
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/container-bridges", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not available")
}

func TestGetBridge_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/container-bridges/:id", h.GetBridge)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/container-bridges/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListBridges_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/container-bridges", h.ListBridges)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/container-bridges", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateBridge_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.PUT("/api/v1/container-bridges/:id", h.UpdateBridge)

	body := map[string]interface{}{"status": "inactive"}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/container-bridges/"+uuid.New().String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteBridge_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.DELETE("/api/v1/container-bridges/:id", h.DeleteBridge)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/container-bridges/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListBridges_Pagination(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/container-bridges", h.ListBridges)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/container-bridges?limit=101&offset=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
