package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/org-service/internal/model"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := New(nil)
	r := gin.New()
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	return r
}

func TestHealthCheck(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
	assert.Contains(t, w.Body.String(), "org-service")
}

func TestReadinessCheck_NoDB(t *testing.T) {
	router := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["ready"].(bool))
}

func TestCreateOrg_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil)
	r := gin.New()
	r.POST("/api/v1/orgs", h.CreateOrg)

	// Missing name
	body := map[string]interface{}{"slug": "test-org"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/orgs", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")

	// Missing slug
	body = map[string]interface{}{"name": "Test Org"}
	jsonBody, _ = json.Marshal(body)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/orgs", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestCreateOrg_DBUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil)
	r := gin.New()
	r.POST("/api/v1/orgs", h.CreateOrg)

	body := model.CreateOrgRequest{
		Name: "Test Org",
		Slug: "test-org",
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/orgs", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not available")
}

func TestGetOrg_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil)
	r := gin.New()
	r.GET("/api/v1/orgs/:id", h.GetOrg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/orgs/invalid-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid organization id")
}

func TestGetOrgBySlug_DBUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := New(nil)
	r := gin.New()
	r.GET("/api/v1/orgs/by-slug/:slug", h.GetOrgBySlug)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/orgs/by-slug/test-slug", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not connected")
}
