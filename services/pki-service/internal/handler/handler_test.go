package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/pki-service/internal/handler"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestHealthCheck(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil, "test-key")
	r.GET("/healthz", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
	assert.Equal(t, "pki-service", resp["service"])
}

func TestReadinessCheck_NoDB(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil, "test-key")
	r.GET("/healthz/ready", h.ReadinessCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["ready"])
}

func TestCreateCAValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil, "test-key")
	r.POST("/api/v1/pki/ca", h.CreateCA)

	body := map[string]interface{}{
		"name": "Test CA",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/pki/ca", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCAValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil, "test-key")
	r.GET("/api/v1/pki/ca/:id", h.GetCA)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/pki/ca/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRevokeCertValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil, "test-key")
	r.POST("/api/v1/pki/certs/:id/revoke", h.RevokeCert)

	body := map[string]interface{}{}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/pki/certs/550e8400-e29b-41d4-a716-446655440000/revoke", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
