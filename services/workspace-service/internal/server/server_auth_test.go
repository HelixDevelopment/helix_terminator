package server_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/workspace-service/internal/server"
)

// testPublicKey/testPrivateKey is a stable in-test keypair used to sign and
// verify real Ed25519 JWTs in the positive/negative-signature tests below.
var testPublicKey ed25519.PublicKey
var testPrivateKey ed25519.PrivateKey

func init() {
	var err error
	testPublicKey, testPrivateKey, err = ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
}

// generateTestToken signs a server.Claims token with the given key. Passing an
// untrusted signing key proves the middleware genuinely validates the signature
// against JWT_PUBLIC_KEY, not merely that the token has the right shape.
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

// T19 (§11.4.214): workspace-service's authMiddleware previously FAILED OPEN —
// a request with no Authorization header was injected a default
// "00000000-0000-0000-0000-000000000000" userID/orgID and passed through to the
// handler, so every /api/v1/workspaces route was reachable completely
// unauthenticated. These tests reproduce that hole (RED against the pre-fix
// code) and then guard the fail-closed behaviour (GREEN post-fix), matching the
// canonical Ed25519 JWT_PUBLIC_KEY chain the keychain/vault/notification
// services already enforce. In-test-generated Ed25519 keypairs only — never a
// committed key (§11.4.10).

// validTestPublicKeyB64 returns a freshly-generated, base64-std-encoded Ed25519
// public key suitable for JWT_PUBLIC_KEY. The private half is discarded — these
// stdlib-only reproduction tests never sign a token, they only prove that a
// missing/garbage credential is rejected once a key is configured.
func validTestPublicKeyB64(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(pub)
}

// TestAuthMiddleware_RejectsMissingToken is the load-bearing RED reproduction:
// an unauthenticated request (no Authorization header) MUST be rejected 401.
// Against the pre-fix fail-open middleware this request was served with a
// default userID/orgID injected — this test FAILS (non-401) on that code.
func TestAuthMiddleware_RejectsMissingToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", validTestPublicKeyB64(t))
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code,
		"an unauthenticated request (no Authorization header) must be rejected, never served with a default user")
	assert.Contains(t, w.Body.String(), "authorization header")
}

// TestAuthMiddleware_RejectsInvalidToken proves a garbage bearer token is
// rejected once a JWT_PUBLIC_KEY is configured. Pre-fix, any token was accepted.
func TestAuthMiddleware_RejectsInvalidToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", validTestPublicKeyB64(t))
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"a malformed/unverifiable bearer token must be rejected")
}

// TestAuthMiddleware_FailsClosedWhenUnconfigured proves that an unset
// JWT_PUBLIC_KEY fails CLOSED (401), never open. Pre-fix, the middleware never
// consulted a key at all and accepted any bearer token.
func TestAuthMiddleware_FailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", "")
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.Header.Set("Authorization", "Bearer any-token-shape")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"an unconfigured JWT_PUBLIC_KEY must fail closed, never fail open")
}

// TestAuthMiddleware_RejectsMalformedAuthHeader proves a non-bearer header is
// rejected.
func TestAuthMiddleware_RejectsMalformedAuthHeader(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", validTestPublicKeyB64(t))
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.Header.Set("Authorization", "not-a-bearer-token")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_RejectsTokenSignedByUntrustedKey proves the middleware
// validates the JWT's Ed25519 signature against the configured JWT_PUBLIC_KEY,
// not just its shape — a well-formed token signed by a different key is
// rejected.
func TestAuthMiddleware_RejectsTokenSignedByUntrustedKey(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv, err := server.New(nil)
	require.NoError(t, err)

	_, untrustedPrivateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	forgedToken := generateTestToken(t, untrustedPrivateKey, "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.Header.Set("Authorization", "Bearer "+forgedToken)
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"a token signed by an untrusted key must be rejected even though it is well-formed")
}

// TestAuthMiddleware_AcceptsRealGatewayForwardedJWT is the GREEN proof that the
// fail-closed change does not over-reject: a real gateway-forwarded request —
// the caller's own Ed25519-signed Authorization bearer JWT, untouched — is NOT
// rejected by the auth middleware.
func TestAuthMiddleware_AcceptsRealGatewayForwardedJWT(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv, err := server.New(nil)
	require.NoError(t, err)

	token := generateTestToken(t, testPrivateKey, "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusUnauthorized, w.Code,
		"a real gateway-forwarded Ed25519 JWT must not be rejected by the auth middleware")
}

// TestAuthMiddleware_AppliesToEveryWorkspaceRoute proves every route in the
// group is gated, not just the one the fix happened to be tested against.
func TestAuthMiddleware_AppliesToEveryWorkspaceRoute(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", validTestPublicKeyB64(t))
	srv, err := server.New(nil)
	require.NoError(t, err)

	id := "33333333-3333-3333-3333-333333333333"
	hostID := "44444444-4444-4444-4444-444444444444"
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/workspaces"},
		{http.MethodGet, "/api/v1/workspaces"},
		{http.MethodGet, "/api/v1/workspaces/" + id},
		{http.MethodPut, "/api/v1/workspaces/" + id},
		{http.MethodDelete, "/api/v1/workspaces/" + id},
		{http.MethodPost, "/api/v1/workspaces/" + id + "/hosts"},
		{http.MethodDelete, "/api/v1/workspaces/" + id + "/hosts/" + hostID},
		{http.MethodGet, "/api/v1/workspaces/" + id + "/hosts"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(rt.method, rt.path, nil)
			srv.Router().ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s %s must require a valid JWT like every other /api/v1/workspaces route", rt.method, rt.path)
		})
	}
}
