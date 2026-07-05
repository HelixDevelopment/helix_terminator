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

	"github.com/helixdevelopment/workspace-service/internal/handler"
	"github.com/helixdevelopment/workspace-service/internal/model"
	"github.com/helixdevelopment/workspace-service/internal/repository"
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
	assert.Equal(t, "workspace-service", resp["service"])
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

func TestCreateWorkspaceValidation(t *testing.T) {
	r, h := setupTestRouter()
	// Simulate auth middleware setting userID/orgID
	r.Use(func(c *gin.Context) {
		c.Set("userID", "00000000-0000-0000-0000-000000000000")
		c.Set("orgID", "00000000-0000-0000-0000-000000000000")
		c.Next()
	})
	r.POST("/api/v1/workspaces", h.CreateWorkspace)

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "missing name",
			body: map[string]interface{}{
				"description": "test",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "valid request with defaults",
			body: map[string]interface{}{
				"name": "my-workspace",
			},
			wantStatus: http.StatusServiceUnavailable, // DB not available
		},
		{
			name: "valid request with all fields",
			body: map[string]interface{}{
				"name":        "full-workspace",
				"description": "A full workspace",
				"color":       "#00ff00",
				"icon":        "cloud",
				"tags":        []string{"prod", "us-east"},
			},
			wantStatus: http.StatusServiceUnavailable, // DB not available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestCreateWorkspaceWithHosts(t *testing.T) {
	r, h := setupTestRouter()
	r.Use(func(c *gin.Context) {
		c.Set("userID", "00000000-0000-0000-0000-000000000000")
		c.Set("orgID", "00000000-0000-0000-0000-000000000000")
		c.Next()
	})
	r.POST("/api/v1/workspaces", h.CreateWorkspace)

	body := model.CreateWorkspaceRequest{
		Name:    "test-workspace",
		Tags:    []string{"dev"},
		HostIDs: []string{"11111111-1111-1111-1111-111111111111"},
	}
	payload, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// DB not available, so service unavailable
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
