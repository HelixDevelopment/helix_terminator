package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/health-service/internal/checker"
	"github.com/helixdevelopment/health-service/internal/handler"
	"github.com/helixdevelopment/health-service/internal/model"
)

func setupRouter(endpoints map[string]string) (*gin.Engine, *handler.Handler) {
	gin.SetMode(gin.TestMode)
	chk := checker.New(endpoints, 5*time.Second)
	h := handler.New(chk)
	r := gin.New()
	return r, h
}

func TestHealthCheck(t *testing.T) {
	r, h := setupRouter(nil)
	r.GET("/healthz", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "health-service")
}

func TestReadinessCheck(t *testing.T) {
	r, h := setupRouter(nil)
	r.GET("/healthz/ready", h.ReadinessCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}

func TestGetSystemHealth(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()

	endpoints := map[string]string{
		"svc1": good.URL,
	}

	r, h := setupRouter(endpoints)
	r.GET("/api/v1/health/system", h.GetSystemHealth)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health/system", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.SystemHealth
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, model.StatusHealthy, resp.OverallStatus)
	assert.Len(t, resp.Services, 1)
}

func TestGetSystemHealth_NoChecker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(nil)
	r := gin.New()
	r.GET("/api/v1/health/system", h.GetSystemHealth)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health/system", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "not configured")
}

func TestGetServiceHealth(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()

	endpoints := map[string]string{
		"svc1": good.URL,
	}

	r, h := setupRouter(endpoints)
	r.GET("/api/v1/health/services/:name", h.GetServiceHealth)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health/services/svc1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ServiceHealth
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "svc1", resp.Name)
	assert.Equal(t, model.StatusHealthy, resp.Status)
}

func TestGetServiceHealth_Unknown(t *testing.T) {
	r, h := setupRouter(map[string]string{})
	r.GET("/api/v1/health/services/:name", h.GetServiceHealth)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health/services/unknown", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp model.ServiceHealth
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, model.StatusUnhealthy, resp.Status)
}

func TestGetServiceHealth_EmptyName(t *testing.T) {
	r, h := setupRouter(map[string]string{})
	r.GET("/api/v1/health/services/:name", h.GetServiceHealth)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health/services/", nil)
	r.ServeHTTP(w, req)

	// Gin does not match /services/ with an empty :name param; it returns 404.
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRunHealthCheck(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()

	endpoints := map[string]string{
		"svc1": good.URL,
		"svc2": good.URL,
	}

	r, h := setupRouter(endpoints)
	r.POST("/api/v1/health/check", h.RunHealthCheck)

	body, _ := json.Marshal(model.HealthCheckRequest{Services: []string{"svc1"}})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/health/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.HealthCheckResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, model.StatusHealthy, resp.Status)
	assert.Len(t, resp.Services, 1)
}

func TestRunHealthCheck_NoChecker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(nil)
	r := gin.New()
	r.POST("/api/v1/health/check", h.RunHealthCheck)

	body, _ := json.Marshal(model.HealthCheckRequest{Services: []string{"svc1"}})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/health/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestRunHealthCheck_InvalidBody(t *testing.T) {
	r, h := setupRouter(map[string]string{})
	r.POST("/api/v1/health/check", h.RunHealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/health/check", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRunHealthCheck_EmptyServices(t *testing.T) {
	r, h := setupRouter(map[string]string{})
	r.POST("/api/v1/health/check", h.RunHealthCheck)

	body, _ := json.Marshal(model.HealthCheckRequest{Services: []string{}})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/health/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
