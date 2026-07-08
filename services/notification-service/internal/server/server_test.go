package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
// open endpoint into a potential spam / SSRF-amplification relay).
//
// T11 (this batch): the auth MECHANISM itself was replaced. It previously
// demanded a literal "X-API-Key" header matching NOTIFICATION_SERVICE_API_KEY
// — a header no real caller in the canonical request path (browser →
// gateway-service → notification-service) ever sends, because
// gateway-service's proxyTo forwards the caller's original signed
// "Authorization: Bearer <Ed25519-JWT>" header untouched and never injects
// an X-API-Key (services/gateway-service/internal/server/server.go:
// 1133 `proxyReq.Header = c.Request.Header.Clone()`). Every real end-user
// notification request routed through the gateway was therefore
// unconditionally rejected pre-fix, regardless of how valid the caller's
// auth-service-issued JWT was. The tests below (and the dedicated
// server_jwt_auth_test.go, which additionally proves a real forwarded
// gateway JWT is now accepted) now exercise the SAME canonical Ed25519
// JWT_PUBLIC_KEY chain gateway-service/billing-service validate, mirroring
// their authMiddleware test conventions.

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

func TestAuthMiddleware_RejectsMissingBearerToken(t *testing.T) {
	mustSetJWTPublicKey(t)
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(createNotificationBody()))
	req.Header.Set("Content-Type", "application/json")
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "authorization")
}

func TestAuthMiddleware_RejectsTokenSignedByUntrustedKey(t *testing.T) {
	sign := mustSetJWTPublicKey(t)
	_ = sign // the server's trusted key is set; sign with a DIFFERENT key below
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(createNotificationBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer totally.wrong.token")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_FailsClosedWhenJWTPublicKeyUnconfigured(t *testing.T) {
	prevKey, hadPrevKey := os.LookupEnv("JWT_PUBLIC_KEY")
	os.Unsetenv("JWT_PUBLIC_KEY")
	t.Cleanup(func() {
		if hadPrevKey {
			os.Setenv("JWT_PUBLIC_KEY", prevKey)
		}
	})
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(createNotificationBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer anything-at-all")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"an unconfigured JWT_PUBLIC_KEY must fail closed, never fail open")
}

func TestAuthMiddleware_AllowsValidJWTThroughToHandler(t *testing.T) {
	sign := mustSetJWTPublicKey(t)
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	token := sign(uuid.New().String(), "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(createNotificationBody()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	// No database is wired in this test (server.New(nil) with no
	// DATABASE_URL), so the request correctly falls past auth to the
	// in-memory repository path — the key point is it is NOT 401/403: a
	// valid, correctly-signed JWT let the request reach the handler.
	assert.NotEqual(t, http.StatusUnauthorized, w.Code,
		"a valid JWT must not be rejected by the auth middleware; body: %s", w.Body.String())
	assert.NotEqual(t, http.StatusForbidden, w.Code,
		"a valid JWT must not be rejected by the auth middleware; body: %s", w.Body.String())
}

func TestAuthMiddleware_AppliesToEveryNotificationRoute(t *testing.T) {
	mustSetJWTPublicKey(t)
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
				"%s %s must require a valid bearer JWT like every other /api/v1/notifications route", rt.method, rt.path)
		})
	}
}

func TestHealthEndpoints_NoAuthRequired(t *testing.T) {
	mustSetJWTPublicKey(t)
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	for _, path := range []string{"/healthz", "/healthz/live", "/healthz/ready"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		srv.Router().ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusUnauthorized, w.Code, "%s must never require auth", path)
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
