package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/audit-service/internal/handler"
	"github.com/helixdevelopment/audit-service/internal/model"
	"github.com/helixdevelopment/audit-service/internal/repository"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestHealthCheck(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/healthz", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
	assert.Equal(t, "audit-service", resp["service"])
}

func TestReadinessCheckNoDB(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/healthz/ready", h.ReadinessCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["ready"])
	assert.Equal(t, "audit-service", resp["service"])
}

func TestCreateAuditLogValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.POST("/api/v1/audit/logs", h.CreateAuditLog)

	body := map[string]interface{}{
		"action": "invalid-action",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/audit/logs", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAuditLogNoDB(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.POST("/api/v1/audit/logs", h.CreateAuditLog)

	body := model.CreateAuditLogRequest{
		Action:       model.ActionCreate,
		ResourceType: model.ResourceTypeUser,
		Severity:     model.SeverityInfo,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/audit/logs", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Should fail because DB is not connected
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListAuditLogsValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/api/v1/audit/logs", h.ListAuditLogs)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit/logs?limit=abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListAuditLogsNoDB(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/api/v1/audit/logs", h.ListAuditLogs)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAuditLogInvalidID(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/api/v1/audit/logs/:id", h.GetAuditLog)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit/logs/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAuditLogNoDB(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/api/v1/audit/logs/:id", h.GetAuditLog)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit/logs/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCountByActionNoDB(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/api/v1/audit/stats/actions", h.CountByAction)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit/stats/actions", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCountByResourceTypeNoDB(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/api/v1/audit/stats/resources", h.CountByResourceType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit/stats/resources", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCountByActionValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/api/v1/audit/stats/actions", h.CountByAction)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit/stats/actions?start=invalid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCountByResourceTypeValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/api/v1/audit/stats/resources", h.CountByResourceType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit/stats/resources?end=invalid", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAuditLogWithDetails(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.POST("/api/v1/audit/logs", h.CreateAuditLog)

	body := model.CreateAuditLogRequest{
		Action:       model.ActionCreate,
		ResourceType: model.ResourceTypeWorkspace,
		Severity:     model.SeverityWarning,
		Details: map[string]interface{}{
			"workspaceName": "test-workspace",
			"ownerId":       uuid.New().String(),
		},
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent/1.0",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/audit/logs", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListAuditLogsWithFilters(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(repository.New(nil))
	r.GET("/api/v1/audit/logs", h.ListAuditLogs)

	orgID := uuid.New()
	userID := uuid.New()
	start := url.QueryEscape(time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339))
	end := url.QueryEscape(time.Now().UTC().Format(time.RFC3339))

	urlStr := "/api/v1/audit/logs?org_id=" + orgID.String() +
		"&user_id=" + userID.String() +
		"&action=create" +
		"&resource_type=user" +
		"&severity=info" +
		"&start=" + start +
		"&end=" + end +
		"&limit=10&offset=0"

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", urlStr, nil)
	r.ServeHTTP(w, req)

	// Still fails because no DB, but time parsing should succeed now
	if w.Code != http.StatusInternalServerError {
		t.Logf("Unexpected status: %d, body: %s", w.Code, w.Body.String())
	}
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
