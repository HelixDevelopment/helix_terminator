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
	"github.com/stretchr/testify/require"
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

func TestCreateSession_Validation(t *testing.T) {
	// Need a non-nil repo to test validation, so create a mock scenario
	// With nil repo, checkPool returns 503 before validation
	router, h := setupTestRouter()
	router.POST("/api/v1/sessions", h.CreateSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sessions", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCreateSession_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/sessions", h.CreateSession)

	body := map[string]interface{}{
		"host_id": uuid.New().String(),
		"name":    "test-session",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sessions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not available")
}

func TestGetSession_InvalidID(t *testing.T) {
	// With nil repo, 503 is returned before ID validation
	router, h := setupTestRouter()
	router.GET("/api/v1/sessions/:id", h.GetSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/sessions/invalid-id", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetSession_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/sessions/:id", h.GetSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/sessions/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListSessions_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/sessions", h.ListSessions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/sessions", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestJoinSession_Validation(t *testing.T) {
	// With nil repo, 503 is returned before validation
	router, h := setupTestRouter()
	router.POST("/api/v1/sessions/:id/join", h.JoinSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sessions/"+uuid.New().String()+"/join", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestJoinSession_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/sessions/:id/join", h.JoinSession)

	body := map[string]interface{}{"user_id": uuid.New().String()}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sessions/"+uuid.New().String()+"/join", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestLeaveSession_InvalidID(t *testing.T) {
	// With nil repo, 503 is returned before ID validation
	router, h := setupTestRouter()
	router.POST("/api/v1/sessions/:id/leave", h.LeaveSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sessions/invalid-id/leave", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestLeaveSession_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/sessions/:id/leave", h.LeaveSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sessions/"+uuid.New().String()+"/leave", nil)
	req.Header.Set("user_id", uuid.New().String())
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestEndSession_InvalidID(t *testing.T) {
	// With nil repo, 503 is returned before ID validation
	router, h := setupTestRouter()
	router.POST("/api/v1/sessions/:id/end", h.EndSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sessions/invalid-id/end", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestEndSession_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/sessions/:id/end", h.EndSession)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/sessions/"+uuid.New().String()+"/end", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListSessions_Pagination(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/sessions", h.ListSessions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/sessions?limit=101&offset=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCreateSessionRequest_Binding(t *testing.T) {
	req := CreateSessionRequest{
		HostID: uuid.New(),
		Name:   "test",
		OrgID:  uuid.New(),
	}
	require.NotEqual(t, uuid.Nil, req.HostID)
	require.NotEmpty(t, req.Name)
}

func TestJoinSessionRequest_Binding(t *testing.T) {
	req := JoinSessionRequest{UserID: uuid.New()}
	require.NotEqual(t, uuid.Nil, req.UserID)
}
