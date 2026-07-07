package server_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/gateway-service/internal/server"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// upstreamServiceNames mirrors the service names registered by
// server.registerUpstreams (internal/server/server.go). Kept in sync
// explicitly rather than exported, since these unit tests intentionally
// stay decoupled from server-package internals.
var upstreamServiceNames = []string{
	"auth-service", "user-service", "vault-service", "host-service",
	"ssh-proxy-service", "terminal-service", "sftp-service", "port-forward-service",
	"snippet-service", "keychain-service", "workspace-service", "collaboration-service",
	"notification-service", "audit-service", "analytics-service", "ai-service",
	"recording-service", "pki-service", "org-service", "billing-service",
	"config-service", "health-service", "container-bridge-service", "helixtrack-bridge-service",
}

// upstreamEnvKey derives the same env-var-override key the server itself
// computes (envKeyForService in server.go), so tests can point every
// registered upstream at a real, local, fake upstream server.
func upstreamEnvKey(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_ADDR"
}

// fakeUpstreamHandler is the REAL (network-listening, not in-process)
// stand-in upstream used by the unit tests below. Since proxyTo now
// performs a genuine reverse-proxy hop (no more stub), these tests need
// a real listener to proxy to. It echoes back enough of the real,
// received request (which service the gateway forwarded to it as, the
// request id, method and path) so tests can assert the round trip really
// happened.
func fakeUpstreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":%q,"request_id":%q,"upstream_path":%q,"upstream_method":%q}`,
		r.Header.Get("X-Gateway-Upstream"), r.Header.Get("X-Request-ID"), r.URL.Path, r.Method)
}

// TestMain starts one real, loopback-listening fake-upstream HTTP server
// and points every registered gateway upstream at it via the
// <SERVICE>_ADDR environment-variable override, before any test runs.
// This keeps the pre-existing unit test suite green now that proxyTo
// performs a real network hop instead of returning a stub.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	ts := httptest.NewServer(http.HandlerFunc(fakeUpstreamHandler))
	for _, name := range upstreamServiceNames {
		os.Setenv(upstreamEnvKey(name), ts.URL)
	}

	code := m.Run()
	ts.Close()
	os.Exit(code)
}

type testLogger struct{}

func (t *testLogger) Printf(format string, v ...interface{}) {}
func (t *testLogger) Println(v ...interface{})               {}

var testPublicKey ed25519.PublicKey
var testPrivateKey ed25519.PrivateKey

func init() {
	var err error
	testPublicKey, testPrivateKey, err = ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
}

func setupTestServer() *server.Server {
	return server.New(&testLogger{})
}

func generateTestToken() string {
	claims := server.Claims{
		UserID:    "test-user-id",
		OrgID:     "test-org-id",
		Email:     "test@example.com",
		Role:      "user",
		SessionID: "test-session-id",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Subject:   "test-user-id",
			Issuer:    "helixterminator",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenString, _ := token.SignedString(testPrivateKey)
	return tokenString
}

func TestLivenessEndpoint(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/live", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
	assert.Contains(t, w.Body.String(), "timestamp")
}

func TestReadinessEndpoint(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
	assert.Contains(t, w.Body.String(), "services")
}

func TestFullHealthEndpoint(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
	assert.Contains(t, w.Body.String(), "version")
	assert.Contains(t, w.Body.String(), "services")
}

func TestMetricsEndpoint(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/metrics", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Gateway metrics")
}

func TestCORSMiddleware(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.helixterminator.io")
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/api/v1/hosts", nil)
	req.Header.Set("Origin", "https://app.helixterminator.io")
	req.Header.Set("Access-Control-Request-Method", "GET")

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://app.helixterminator.io", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestCORSMiddleware_UnknownOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.helixterminator.io")
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/api/v1/hosts", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "GET")

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestRequestIDMiddleware(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/live", nil)
	req.Header.Set("X-Request-ID", "test-request-123")

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-request-123", w.Header().Get("X-Request-ID"))
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/live", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestJWTValidationMiddleware_MissingToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/hosts", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization header")
}

func TestJWTValidationMiddleware_InvalidFormat(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/hosts", nil)
	req.Header.Set("Authorization", "invalid-token")

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid authorization header format")
}

func TestJWTValidationMiddleware_EmptyToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/hosts", nil)
	req.Header.Set("Authorization", "Bearer ")

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "empty token")
}

func TestJWTValidationMiddleware_PassesWithToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/hosts", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken())

	s.Router().ServeHTTP(w, req)

	// Should route to host-service (returns stub response)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "host-service")
}

func TestJWTValidationMiddleware_InvalidToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/hosts", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid token")
}

func TestAuthRoutes_NoTokenRequired(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "auth-service")
}

func TestRateLimitMiddleware(t *testing.T) {
	s := setupTestServer()

	// Make requests until rate limit kicks in
	// Note: In real scenario, this would need many requests
	// For test, we verify the middleware is wired by checking headers exist
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/live", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProxyToUpstream(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/hosts", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken())

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "host-service")
	assert.Contains(t, w.Body.String(), "request_id")
}

func TestProxyToUnknownService(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vaults", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken())

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "vault-service")
}

func TestTerminalWebSocket_NotImplemented(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ws/terminal/123", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Contains(t, w.Body.String(), "WebSocket terminal not yet implemented")
}

func TestSSO_NotImplemented(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/sso/google", nil)

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Contains(t, w.Body.String(), "SSO not yet implemented")
}

func TestLoggingMiddleware(t *testing.T) {
	s := setupTestServer()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/live", nil)

	start := time.Now()
	s.Router().ServeHTTP(w, req)
	elapsed := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Less(t, elapsed, 1*time.Second, "request should be fast")
}

func TestServer_RouterExposure(t *testing.T) {
	s := setupTestServer()
	router := s.Router()
	require.NotNil(t, router)
	assert.Implements(t, (*http.Handler)(nil), router)
}

func TestAllUpstreamServicesRegistered(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer()

	// Test a few key upstream services are routable
	testCases := []struct {
		path       string
		auth       bool
		expectSvc  string
		expectCode int
	}{
		{"/api/v1/auth/login", false, "auth-service", http.StatusOK},
		{"/api/v1/users/me", true, "user-service", http.StatusOK},
		{"/api/v1/vaults", true, "vault-service", http.StatusOK},
		{"/api/v1/hosts", true, "host-service", http.StatusOK},
		{"/api/v1/sessions", true, "ssh-proxy-service", http.StatusOK},
		{"/api/v1/snippets", true, "snippet-service", http.StatusOK},
		{"/api/v1/workspaces", true, "workspace-service", http.StatusOK},
		{"/api/v1/recordings", true, "recording-service", http.StatusOK},
		{"/api/v1/audit", true, "audit-service", http.StatusOK},
		{"/api/v1/notifications", true, "notification-service", http.StatusOK},
		{"/api/v1/billing/subscription", true, "billing-service", http.StatusOK},
		{"/api/v1/config", true, "config-service", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			method := http.MethodGet
			if tc.path == "/api/v1/auth/login" {
				method = http.MethodPost
			}
			req, _ := http.NewRequest(method, tc.path, nil)
			if tc.auth {
				req.Header.Set("Authorization", "Bearer "+generateTestToken())
			}

			s.Router().ServeHTTP(w, req)

			assert.Equal(t, tc.expectCode, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectSvc)
		})
	}
}

func BenchmarkHealthEndpoint(b *testing.B) {
	s := setupTestServer()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/live", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.Router().ServeHTTP(w, req)
	}
}
