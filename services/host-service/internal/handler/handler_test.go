package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/host-service/internal/handler"
	"github.com/helixdevelopment/host-service/internal/model"
	"github.com/helixdevelopment/host-service/internal/repository"
)

func setupTestRouter() (*gin.Engine, *handler.Handler) {
	gin.SetMode(gin.TestMode)
	repo := repository.New(nil)
	h := handler.New(repo)
	r := gin.New()
	return r, h
}

func TestHealthCheck(t *testing.T) {
	r, h := setupTestRouter()
	r.GET("/healthz", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
	assert.Equal(t, "host-service", resp["service"])
}

func TestReadinessCheck(t *testing.T) {
	r, h := setupTestRouter()
	r.GET("/healthz/ready", h.ReadinessCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	// Without DB, readiness should return 503 because repo.Ping fails
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, false, resp["ready"])
}

func TestCreateHostValidation(t *testing.T) {
	r, h := setupTestRouter()
	// Simulate auth middleware setting userID/orgID
	r.Use(func(c *gin.Context) {
		c.Set("userID", "00000000-0000-0000-0000-000000000000")
		c.Set("orgID", "00000000-0000-0000-0000-000000000000")
		c.Next()
	})
	r.POST("/api/v1/hosts", h.CreateHost)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "missing name",
			body: map[string]interface{}{
				"hostname": "192.168.1.1",
				"username": "admin",
				"auth_type": "password",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing hostname",
			body: map[string]interface{}{
				"name": "my-host",
				"username": "admin",
				"auth_type": "password",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing username",
			body: map[string]interface{}{
				"name": "my-host",
				"hostname": "192.168.1.1",
				"auth_type": "password",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid auth_type",
			body: map[string]interface{}{
				"name": "my-host",
				"hostname": "192.168.1.1",
				"username": "admin",
				"auth_type": "invalid",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "valid request with defaults",
			body: map[string]interface{}{
				"name": "my-host",
				"hostname": "192.168.1.1",
				"username": "admin",
				"auth_type": "password",
			},
			wantStatus: http.StatusServiceUnavailable, // DB not available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/hosts", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestCreateHostWithPort(t *testing.T) {
	r, h := setupTestRouter()
	r.Use(func(c *gin.Context) {
		c.Set("userID", "00000000-0000-0000-0000-000000000000")
		c.Set("orgID", "00000000-0000-0000-0000-000000000000")
		c.Next()
	})
	r.POST("/api/v1/hosts", h.CreateHost)

	body := model.CreateHostRequest{
		Name:     "test-host",
		Hostname: "10.0.0.1",
		Port:     2222,
		Username: "root",
		AuthType: model.AuthTypeKey,
		Tags:     []string{"prod", "us-east"},
	}
	payload, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/hosts", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// DB not available, so service unavailable
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
