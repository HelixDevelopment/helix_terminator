package server_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/vault-service/internal/server"
)

// The previous version of this file was a stub (`assert.True(t, true)`)
// that asserted nothing about the server package — replaced per queue#4 /
// §11.4.27. These are real (DB-independent) unit tests that exercise the
// server's access-control middleware directly over its real router via
// httptest. The corresponding real-Postgres, real-server, real-SQL-row
// security tests (proving another tenant genuinely cannot read a secret)
// live in server_integration_test.go (build tag `integration`).
//
// T19: authMiddleware previously demanded a service-to-service X-API-Key
// header (VAULT_SERVICE_API_KEY) that the gateway NEVER sends — the
// gateway forwards the caller's own Ed25519-signed Authorization bearer
// JWT through to upstream services untouched (T11's notification-service
// finding generalised to vault-service). Every real gateway-routed request
// was therefore rejected with 401 "missing or invalid X-API-Key". These
// tests were rewritten from the X-API-Key mechanism to the canonical
// Ed25519 JWT_PUBLIC_KEY chain (services/billing-service/internal/server/
// server.go / services/gateway-service/internal/server/server_test.go),
// using an in-test-generated Ed25519 keypair — never a committed key
// (§11.4.10).

var testPublicKey ed25519.PublicKey
var testPrivateKey ed25519.PrivateKey

func init() {
	var err error
	testPublicKey, testPrivateKey, err = ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
}

func newTestServer(t *testing.T) *server.Server {
	t.Helper()
	srv, err := server.New(nil)
	require.NoError(t, err)
	return srv
}

// generateTestToken signs a Claims token with the test private key. Passing
// a different signing key (see TestAuthMiddleware_RejectsTokenSignedByUntrustedKey)
// proves the middleware genuinely validates the signature against
// JWT_PUBLIC_KEY rather than merely checking token shape.
func generateTestToken(t *testing.T, signingKey ed25519.PrivateKey, userID, orgID string) string {
	t.Helper()
	claims := server.Claims{
		UserID: userID,
		OrgID:  orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenString, err := token.SignedString(signingKey)
	require.NoError(t, err)
	return tokenString
}

func TestHealthCheck_NoAuthRequired(t *testing.T) {
	srv := newTestServer(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	srv.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAuthMiddleware_RejectsMissingToken proves the T19 fix's fail-closed
// posture: no Authorization header at all is rejected exactly as it was
// under the old X-API-Key mechanism (equivalent negative-space coverage),
// just for the new mechanism.
func TestAuthMiddleware_RejectsMissingToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "authorization header")
}

func TestAuthMiddleware_RejectsMalformedAuthHeader(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	req.Header.Set("Authorization", "totally-not-a-bearer-token")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_RejectsInvalidToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_RejectsTokenSignedByUntrustedKey proves the middleware
// validates the JWT's Ed25519 signature against the configured
// JWT_PUBLIC_KEY, not just its shape — a token signed by a DIFFERENT
// (attacker-controlled) keypair must be rejected fail-closed.
func TestAuthMiddleware_RejectsTokenSignedByUntrustedKey(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	_, untrustedPrivateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	forgedToken := generateTestToken(t, untrustedPrivateKey, uuid.New().String(), uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+forgedToken)
	req.Header.Set("X-User-ID", uuid.New().String())
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"a token signed by an untrusted key must be rejected even though it is well-formed")
}

func TestAuthMiddleware_FailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", "")
	srv := newTestServer(t)

	token := generateTestToken(t, testPrivateKey, uuid.New().String(), uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"an unconfigured JWT_PUBLIC_KEY must fail closed, never fail open")
}

// TestAuthMiddleware_AcceptsRealGatewayForwardedJWT is the T19 confirmation
// test: it reproduces exactly what the gateway sends on every real
// gateway-routed request — the caller's own Ed25519-signed Authorization
// bearer JWT, untouched — and proves the fixed middleware lets it through.
// Pre-fix, this exact request (no X-API-Key header, only Authorization)
// was unconditionally rejected 401 "missing or invalid X-API-Key" by the
// old authMiddleware; see the RED-phase evidence in the T19 report (the
// original X-API-Key-only middleware, temporarily restored via `git stash`,
// rejects this same request with 401).
func TestAuthMiddleware_AcceptsRealGatewayForwardedJWT(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	token := generateTestToken(t, testPrivateKey, uuid.New().String(), uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	// ListSecrets is a tenant-scoped collection route (T7): it additionally
	// requires a valid caller identity, independent of the JWT check this
	// test targets (out of scope for T19 — preserved unchanged).
	req.Header.Set("X-User-ID", uuid.New().String())
	srv.Router().ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusUnauthorized, w.Code,
		"a real gateway-forwarded Ed25519 JWT must not be rejected by the auth middleware")
}

// TestRequireCallerIdentityMiddleware_RejectsMissingUserID proves the T7 fix:
// the collection-level routes (ListSecrets, CreateSecret) reject a caller
// that presents no X-User-ID at all, closing the gap where ListSecrets
// previously trusted an absent/caller-supplied user_id query parameter.
func TestRequireCallerIdentityMiddleware_RejectsMissingUserID(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)
	token := generateTestToken(t, testPrivateKey, uuid.New().String(), uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "X-User-ID")
}

// TestRequireCallerIdentityMiddleware_RejectsMalformedUserID mirrors the
// tenant-isolation middleware's malformed-header rejection for the
// collection-level routes.
func TestRequireCallerIdentityMiddleware_RejectsMalformedUserID(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)
	token := generateTestToken(t, testPrivateKey, uuid.New().String(), uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/vault/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-User-ID", "not-a-uuid")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantIsolationMiddleware_RejectsMissingUserID(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)
	token := generateTestToken(t, testPrivateKey, uuid.New().String(), uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "X-User-ID")
}

func TestTenantIsolationMiddleware_RejectsMalformedUserID(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)
	token := generateTestToken(t, testPrivateKey, uuid.New().String(), uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-User-ID", "not-a-uuid")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
