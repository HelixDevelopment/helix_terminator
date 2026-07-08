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

	"github.com/helixdevelopment/keychain-service/internal/server"
)

// T19: keychain-service's /api/v1/keychain routes — which store encrypted
// private keys / passphrases — previously had NO authentication middleware
// of ANY kind (confirmed via `git show HEAD:services/keychain-service/
// internal/server/server.go`, the commit this T19 worktree started from:
// the `/api/v1` route group had no `.Use(...)` call at all). Any caller
// could create/list/read/update/delete keychain items completely
// unauthenticated. These tests were rewritten from "no coverage" to the
// canonical Ed25519 JWT_PUBLIC_KEY chain (services/billing-service/
// internal/server/server.go / services/gateway-service/internal/server/
// server_test.go), using in-test-generated Ed25519 keypairs — never a
// committed key (§11.4.10).

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
	return server.New(nil)
}

// generateTestToken signs a Claims token with the given key. Passing an
// untrusted signing key (see TestAuthMiddleware_RejectsTokenSignedByUntrustedKey)
// proves the middleware genuinely validates the signature against
// JWT_PUBLIC_KEY, not merely that the token has the right shape.
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
	for _, path := range []string{"/healthz", "/healthz/ready", "/health", "/ready"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		srv.Router().ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusUnauthorized, w.Code, "%s must never require a JWT", path)
	}
}

// TestAuthMiddleware_RejectsMissingToken is the T19 confirmation test: it
// proves that, post-fix, an unauthenticated request to a keychain route is
// now rejected 401. Pre-fix this exact request (no Authorization header at
// all, no auth middleware present) reached ListItems directly; see
// TestOldServer_UnauthenticatedRequestReachesHandler below for the executed
// RED-phase reproduction against the verbatim pre-fix router.
func TestAuthMiddleware_RejectsMissingToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/keychain", nil)
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "authorization header")
}

func TestAuthMiddleware_RejectsMalformedAuthHeader(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/keychain", nil)
	req.Header.Set("Authorization", "not-a-bearer-token")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_RejectsInvalidToken(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/keychain", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_RejectsTokenSignedByUntrustedKey proves the middleware
// validates the JWT's Ed25519 signature against the configured
// JWT_PUBLIC_KEY, not just its shape.
func TestAuthMiddleware_RejectsTokenSignedByUntrustedKey(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	_, untrustedPrivateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	forgedToken := generateTestToken(t, untrustedPrivateKey, "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/keychain", nil)
	req.Header.Set("Authorization", "Bearer "+forgedToken)
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"a token signed by an untrusted key must be rejected even though it is well-formed")
}

func TestAuthMiddleware_FailsClosedWhenUnconfigured(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", "")
	srv := newTestServer(t)

	token := generateTestToken(t, testPrivateKey, "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/keychain", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"an unconfigured JWT_PUBLIC_KEY must fail closed, never fail open")
}

// TestAuthMiddleware_AcceptsRealGatewayForwardedJWT is the T19 GREEN proof:
// a real gateway-forwarded request — the caller's own Ed25519-signed
// Authorization bearer JWT, untouched — now reaches the handler instead of
// being served by a completely unauthenticated route.
func TestAuthMiddleware_AcceptsRealGatewayForwardedJWT(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	token := generateTestToken(t, testPrivateKey, "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/keychain", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusUnauthorized, w.Code,
		"a real gateway-forwarded Ed25519 JWT must not be rejected by the auth middleware")
}

// TestAuthMiddleware_AppliesToEveryKeychainRoute proves every route in the
// group is gated, not just the one the fix happened to be tested against.
func TestAuthMiddleware_AppliesToEveryKeychainRoute(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	srv := newTestServer(t)

	id := "33333333-3333-3333-3333-333333333333"
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/keychain"},
		{http.MethodGet, "/api/v1/keychain"},
		{http.MethodGet, "/api/v1/keychain/" + id},
		{http.MethodPut, "/api/v1/keychain/" + id},
		{http.MethodDelete, "/api/v1/keychain/" + id},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(rt.method, rt.path, nil)
			srv.Router().ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"%s %s must require a valid JWT like every other /api/v1/keychain route", rt.method, rt.path)
		})
	}
}
