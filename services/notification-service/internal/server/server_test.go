package server_test

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

	"github.com/helixdevelopment/notification-service/internal/server"
)

func TestServerNew(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)
	require.NotNil(t, srv)

	r := srv.Router()
	require.NotNil(t, r)
}

func TestHealthEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	r := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/healthz/live", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not connected")
}

// --- Authorization (Constitution §11.4 security-hardening finding #3:
// CreateNotification, and every other /api/v1/notifications route, MUST
// reject unauthenticated callers — the real email/webhook sinks turned an
// open endpoint into a potential spam / SSRF-amplification relay). These
// mirror vault-service's authMiddleware test suite (same X-API-Key /
// fail-closed / constant-time-compare pattern, project convention).

func createNotificationBody() []byte {
	payload := map[string]interface{}{
		"userId":  uuid.New().String(),
		"type":    "info",
		"title":   "Auth test",
		"message": "Auth test message",
		"channel": "in_app",
	}
	body, _ := json.Marshal(payload)
	return body
}

func TestAuthMiddleware_RejectsMissingAPIKey(t *testing.T) {
	t.Setenv("NOTIFICATION_SERVICE_API_KEY", "test-service-key-12345")
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(createNotificationBody()))
	req.Header.Set("Content-Type", "application/json")
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "X-API-Key")
}

func TestAuthMiddleware_RejectsWrongAPIKey(t *testing.T) {
	t.Setenv("NOTIFICATION_SERVICE_API_KEY", "test-service-key-12345")
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(createNotificationBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "totally-wrong-key")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_FailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("NOTIFICATION_SERVICE_API_KEY", "")
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(createNotificationBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "anything-at-all")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"an unconfigured NOTIFICATION_SERVICE_API_KEY must fail closed, never fail open")
}

func TestAuthMiddleware_AllowsCorrectAPIKeyThroughToHandler(t *testing.T) {
	t.Setenv("NOTIFICATION_SERVICE_API_KEY", "test-service-key-12345")
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(createNotificationBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-service-key-12345")
	srv.Router().ServeHTTP(w, req)

	// No database is wired in this test (server.New(nil) with no
	// DATABASE_URL), so the request correctly fails past auth with 503 —
	// the key point is it is NOT 401/403: the correct API key let the
	// request reach the handler.
	assert.NotEqual(t, http.StatusUnauthorized, w.Code,
		"a correct X-API-Key must not be rejected by the auth middleware")
	assert.NotEqual(t, http.StatusForbidden, w.Code,
		"a correct X-API-Key must not be rejected by the auth middleware")
}

func TestAuthMiddleware_AppliesToEveryNotificationRoute(t *testing.T) {
	t.Setenv("NOTIFICATION_SERVICE_API_KEY", "test-service-key-12345")
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	id := uuid.New().String()
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/notifications?user_id=" + id},
		{http.MethodGet, "/api/v1/notifications/" + id},
		{http.MethodPost, "/api/v1/notifications/" + id + "/read"},
		{http.MethodPost, "/api/v1/notifications/read-all?user_id=" + id},
		{http.MethodDelete, "/api/v1/notifications/" + id},
		{http.MethodGet, "/api/v1/notifications/unread-count?user_id=" + id},
		{http.MethodGet, "/api/v1/notifications/preferences?user_id=" + id + "&channel=email"},
		{http.MethodPut, "/api/v1/notifications/preferences"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(rt.method, rt.path, nil)
			srv.Router().ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s %s must require X-API-Key like every other /api/v1/notifications route", rt.method, rt.path)
		})
	}
}

func TestHealthEndpoints_NoAuthRequired(t *testing.T) {
	t.Setenv("NOTIFICATION_SERVICE_API_KEY", "test-service-key-12345")
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	for _, path := range []string{"/healthz", "/healthz/live", "/healthz/ready"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		srv.Router().ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusUnauthorized, w.Code, "%s must never require an API key", path)
	}
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	r := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	r := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-request-id", w.Header().Get("X-Request-ID"))
}
