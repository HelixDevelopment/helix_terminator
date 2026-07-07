package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/vault-service/internal/server"
)

// The previous version of this file was a stub (`assert.True(t, true)`)
// that asserted nothing about the server package — replaced per queue#4 /
// §11.4.27. These are real (DB-independent) unit tests that exercise the
// server's new access-control middleware directly over its real router
// via httptest. The corresponding real-Postgres, real-server, real-SQL-row
// security tests (proving another tenant genuinely cannot read a secret)
// live in server_integration_test.go (build tag `integration`).

func newTestServer(t *testing.T) *server.Server {
	t.Helper()
	srv, err := server.New(nil)
	require.NoError(t, err)
	return srv
}

func TestHealthCheck_NoAuthRequired(t *testing.T) {
	srv := newTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	srv.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_RejectsMissingAPIKey(t *testing.T) {
	t.Setenv("VAULT_SERVICE_API_KEY", "test-service-key-12345")
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "X-API-Key")
}

func TestAuthMiddleware_RejectsWrongAPIKey(t *testing.T) {
	t.Setenv("VAULT_SERVICE_API_KEY", "test-service-key-12345")
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	req.Header.Set("X-API-Key", "totally-wrong-key")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_FailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("VAULT_SERVICE_API_KEY", "")
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	req.Header.Set("X-API-Key", "anything-at-all")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"an unconfigured VAULT_SERVICE_API_KEY must fail closed, never fail open")
}

func TestAuthMiddleware_AllowsCorrectAPIKeyThroughToHandler(t *testing.T) {
	t.Setenv("VAULT_SERVICE_API_KEY", "test-service-key-12345")
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	req.Header.Set("X-API-Key", "test-service-key-12345")
	srv.Router().ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusUnauthorized, w.Code,
		"a correct X-API-Key must not be rejected by the auth middleware")
}

func TestTenantIsolationMiddleware_RejectsMissingUserID(t *testing.T) {
	t.Setenv("VAULT_SERVICE_API_KEY", "test-service-key-12345")
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+uuid.New().String(), nil)
	req.Header.Set("X-API-Key", "test-service-key-12345")
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "X-User-ID")
}

func TestTenantIsolationMiddleware_RejectsMalformedUserID(t *testing.T) {
	t.Setenv("VAULT_SERVICE_API_KEY", "test-service-key-12345")
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+uuid.New().String(), nil)
	req.Header.Set("X-API-Key", "test-service-key-12345")
	req.Header.Set("X-User-ID", "not-a-uuid")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
