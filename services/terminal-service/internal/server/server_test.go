package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/terminal-service/internal/server"
)

// setupTestServer builds a real terminal-service Server in degraded mode
// (no DATABASE_URL set, matching server.New's documented no-DB fallback
// path) so these tests exercise the REAL router + middleware wiring without
// requiring a live database.
func setupTestServer(t *testing.T) *server.Server {
	t.Helper()
	t.Setenv("DATABASE_URL", "")
	s, err := server.New(nil)
	require.NoError(t, err)
	require.NotNil(t, s)
	return s
}

// TestHealthzRoutesRespondOK is a real, falsifiable regression guard for
// server.New's health-endpoint wiring (r.GET("/healthz", ...) /
// "/healthz/ready" / "/healthz/live" in server.go). RED-capable: if any of
// these routes were dropped from registration, this test would receive a
// gin 404 (StatusNotFound) instead of StatusOK and fail.
func TestHealthzRoutesRespondOK(t *testing.T) {
	s := setupTestServer(t)

	for _, path := range []string{"/healthz", "/healthz/ready", "/healthz/live"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, path, nil)
			s.Router().ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "route %s must be registered and respond 200", path)
		})
	}
}

// TestCORSMiddlewareReflectsAllowedOrigin is a real, falsifiable test of
// server.go's corsMiddleware + isAllowedOrigin + parseCORSAllowedOrigins:
// with CORS_ALLOWED_ORIGINS set to a specific origin, a preflight OPTIONS
// request from that EXACT origin must be reflected back in
// Access-Control-Allow-Origin. RED-capable: if isAllowedOrigin's comparison
// were broken (e.g. always false, or a substring match instead of an exact
// match), this header would be empty and the assertion would fail.
func TestCORSMiddlewareReflectsAllowedOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.helixterminator.example")
	s := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", "https://app.helixterminator.example")
	req.Header.Set("Access-Control-Request-Method", "GET")

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://app.helixterminator.example", w.Header().Get("Access-Control-Allow-Origin"),
		"an OPTIONS preflight from an explicitly allowed origin must be reflected back")
}

// TestCORSMiddlewareRejectsUnknownOrigin is the negation of the above: an
// origin NOT in CORS_ALLOWED_ORIGINS must NOT be granted
// Access-Control-Allow-Origin. RED-capable: if isAllowedOrigin degraded to
// "allow everything" (e.g. a stray `return true`), this assertion of
// emptiness would fail.
func TestCORSMiddlewareRejectsUnknownOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.helixterminator.example")
	s := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "GET")

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"),
		"an origin outside CORS_ALLOWED_ORIGINS must never be granted Access-Control-Allow-Origin")
}

// TestRequestIDMiddlewarePreservesClientSuppliedID is a real test of
// server.go's requestIDMiddleware: when the caller supplies an
// X-Request-ID header, the response must echo back that EXACT value (not
// generate a fresh one). RED-capable: if the middleware always generated a
// new ID regardless of the incoming header, the echoed value would differ
// from the client-supplied one and this equality assertion would fail.
func TestRequestIDMiddlewarePreservesClientSuppliedID(t *testing.T) {
	s := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/live", nil)
	req.Header.Set("X-Request-ID", "terminal-test-request-id-42")

	s.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "terminal-test-request-id-42", w.Header().Get("X-Request-ID"))
}
