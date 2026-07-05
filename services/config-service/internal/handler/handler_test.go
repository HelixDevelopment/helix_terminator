package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/helixdevelopment/config-service/internal/handler"
	"github.com/helixdevelopment/config-service/internal/model"
	"github.com/helixdevelopment/config-service/internal/repository"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestHealthCheck(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.GET("/healthz", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
	assert.Equal(t, "config-service", resp["service"])
}

func TestReadinessCheck_NoDB(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
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

func TestCreateConfigValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.POST("/api/v1/configs", h.CreateConfig)

	body := model.CreateConfigRequest{
		Scope:     "invalid-scope",
		Key:       "",
		Value:     "",
		ValueType: "invalid-type",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/configs", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateConfigValidation_MissingScopeID(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.POST("/api/v1/configs", h.CreateConfig)

	body := model.CreateConfigRequest{
		Scope:     "org",
		Key:       "my-key",
		Value:     "my-value",
		ValueType: "string",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/configs", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "scope_id is required")
}

func TestCreateConfigValidation_GlobalNoScopeID(t *testing.T) {
	r := setupTestRouter()
	// Use a mock repo that returns DB error instead of nil to avoid nil panic
	repo := repository.New(nil)
	h := handler.New(repo)
	r.POST("/api/v1/configs", h.CreateConfig)

	body := model.CreateConfigRequest{
		Scope:     "global",
		Key:       "global-key",
		Value:     "global-value",
		ValueType: "string",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/configs", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Should fail because DB is not connected (repo nil), but validation passes
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetConfig_InvalidID(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.GET("/api/v1/configs/:id", h.GetConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/configs/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateConfig_NoFields(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.PUT("/api/v1/configs/:id", h.UpdateConfig)

	body := model.UpdateConfigRequest{}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/configs/550e8400-e29b-41d4-a716-446655440000", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "no fields to update")
}

func TestGetConfigByKey_MissingParams(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.GET("/api/v1/configs/by-key", h.GetConfigByKey)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/configs/by-key", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "scope and key are required")
}
