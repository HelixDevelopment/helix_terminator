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

func TestCreateForward_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/forwards", h.CreateForward)

	body := map[string]interface{}{
		"hostId":     uuid.New().String(),
		"localPort":  8080,
		"remotePort": 80,
		"remoteHost": "localhost",
		"protocol":   "tcp",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/forwards", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not available")
}

func TestGetForward_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/forwards/:id", h.GetForward)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/forwards/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListForwards_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/forwards", h.ListForwards)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/forwards", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateForward_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.PUT("/api/v1/forwards/:id", h.UpdateForward)

	body := map[string]interface{}{"status": "inactive"}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/forwards/"+uuid.New().String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteForward_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.DELETE("/api/v1/forwards/:id", h.DeleteForward)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/forwards/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListForwards_Pagination(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/forwards", h.ListForwards)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/forwards?limit=101&offset=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
