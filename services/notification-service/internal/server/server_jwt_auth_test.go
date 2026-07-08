package server_test

// T11 — notification-service authMiddleware / forwarded-gateway-JWT
// consistency (Constitution §11.4.102 investigation, §11.4.43/§11.4.115
// RED-then-GREEN).
//
// FORENSIC FINDING (confirmed by reading, not guessed — §11.4.6): the
// canonical service-to-service auth chain in this codebase is
// auth-service mints an Ed25519 (EdDSA) JWT with claims userId/orgId →
// gateway-service validates it AND forwards the caller's ORIGINAL signed
// "Authorization: Bearer <token>" header untouched to every proxied
// upstream (gateway-service's proxyTo: `proxyReq.Header =
// c.Request.Header.Clone()`, services/gateway-service/internal/server/
// server.go:1133 — it clones every header, including Authorization, and
// injects only X-Forwarded-For / X-Forwarded-Host / X-Gateway-Upstream /
// X-Request-ID; it NEVER sets an X-API-Key header) → billing-service (T12)
// independently re-validates the SAME token with the SAME JWT_PUBLIC_KEY
// (base64-std Ed25519 public key, SigningMethodEd25519, claims
// userId/orgId). gateway-service DOES proxy end-user traffic to
// notification-service this same way (services/gateway-service/internal/
// server/server.go:484-486: `api.POST("/notifications", ...)` →
// `s.proxyTo("notification-service", ...)`).
//
// Pre-fix, notification-service's authMiddleware ignored the forwarded
// Authorization header entirely and instead demanded a literal
// "X-API-Key" header matching NOTIFICATION_SERVICE_API_KEY — a header NO
// caller in the real request path (browser → gateway → notification-
// service) ever sends, because the gateway forwards the caller's JWT, not
// a service API key. The result: every real end-user notification request
// routed through the canonical gateway path was unconditionally rejected
// with 401, regardless of how valid the caller's auth-service-issued JWT
// was. This test reproduces that defect directly: a JWT signed with the
// SAME Ed25519 key shape gateway-service/billing-service use, presented
// exactly as the gateway forwards it, MUST be accepted (not 401) once
// notification-service is aligned with the canonical chain.
//
// RED (pre-fix): FAILS — request is rejected 401 because there is no
// X-API-Key header (a real caller never has NOTIFICATION_SERVICE_API_KEY;
// the caller only ever holds the auth-service-issued JWT). Confirmed via
// TestT11JWTAuth_RejectsRealGatewayJWT_PreFixOnly having failed against
// pre-fix commit HEAD — see agent report.
// GREEN (post-fix): notification-service loads JWT_PUBLIC_KEY exactly like
// billing-service, validates the SAME forwarded Ed25519 JWT, and lets the
// request through to the handler.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/server"
)

// gatewayClaims mirrors the SAME claim shape auth-service mints and
// gateway-service/billing-service validate (services/gateway-service/
// internal/server/server.go Claims, services/billing-service/internal/
// server/server.go Claims) — userId/orgId, EdDSA-signed.
type gatewayClaims struct {
	UserID string `json:"userId"`
	OrgID  string `json:"orgId,omitempty"`
	jwt.RegisteredClaims
}

// mustSetJWTPublicKey generates a real Ed25519 keypair, points
// notification-service's JWT_PUBLIC_KEY env var at the public half
// (exactly how gateway-service/billing-service are provisioned in
// production — same secret, same env var name), and returns a signer
// bound to the private half so tests can mint tokens exactly as
// auth-service would.
func mustSetJWTPublicKey(t *testing.T) func(userID, orgID string) string {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	prevKey, hadPrevKey := os.LookupEnv("JWT_PUBLIC_KEY")
	require.NoError(t, os.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(pub)))
	t.Cleanup(func() {
		if hadPrevKey {
			os.Setenv("JWT_PUBLIC_KEY", prevKey)
		} else {
			os.Unsetenv("JWT_PUBLIC_KEY")
		}
	})

	return func(userID, orgID string) string {
		claims := gatewayClaims{
			UserID: userID,
			OrgID:  orgID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		signed, err := tok.SignedString(priv)
		require.NoError(t, err)
		return signed
	}
}

// TestT11JWTAuth_AcceptsRealGatewayForwardedJWT is the T11 anti-bluff proof.
// It presents notification-service with EXACTLY what gateway-service
// forwards for a real, valid, auth-service-issued Ed25519 JWT — an
// "Authorization: Bearer <token>" header, no X-API-Key header at all
// (because the gateway never adds one; see proxyTo). Pre-fix this 401s.
// Post-fix it MUST NOT 401 — the whole point of the canonical JWT chain
// is that a caller who successfully authenticated at the edge is
// recognised at every downstream hop.
func TestT11JWTAuth_AcceptsRealGatewayForwardedJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sign := mustSetJWTPublicKey(t)

	srv, err := server.New(nil)
	require.NoError(t, err)

	userID := uuid.New().String()
	token := sign(userID, "")

	payload := map[string]interface{}{
		"userId":  userID,
		"type":    "info",
		"title":   "T11 real JWT",
		"message": "real gateway-forwarded JWT must be accepted",
		"channel": "in_app",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// This is EXACTLY what gateway-service's proxyTo forwards for a real
	// end-user request: the original signed Authorization bearer header,
	// cloned verbatim, and NOTHING else auth-related (no X-API-Key —
	// proxyTo never sets one).
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	// No DATABASE_URL is wired in this test (server.New(nil) with no env),
	// so the request correctly falls through auth to hit the in-memory
	// repository path and may still fail later (e.g. 503) — the ONLY thing
	// this test asserts is that AUTH itself did not reject it: a real,
	// validly-signed, gateway-forwarded JWT must never be turned away with
	// 401/403 by notification-service.
	assert.NotEqual(t, http.StatusUnauthorized, w.Code,
		"a valid forwarded gateway JWT (Authorization: Bearer, Ed25519/EdDSA, "+
			"validated against JWT_PUBLIC_KEY) must be accepted by notification-service "+
			"the same way gateway-service/billing-service accept it; body: %s", w.Body.String())
	assert.NotEqual(t, http.StatusForbidden, w.Code, "body: %s", w.Body.String())
}

// TestT11JWTAuth_RejectsMissingToken proves the fixed middleware still
// fails CLOSED: no Authorization header at all must still be 401.
func TestT11JWTAuth_RejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mustSetJWTPublicKey(t)

	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications?user_id="+uuid.New().String(), nil)
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "a request with no Authorization header must be rejected")
}

// TestT11JWTAuth_RejectsWrongKeySignedToken proves a token signed by a
// DIFFERENT Ed25519 key (i.e. not auth-service's real key) is rejected —
// signature validation must be real, not a no-op.
func TestT11JWTAuth_RejectsWrongKeySignedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mustSetJWTPublicKey(t) // sets the SERVER's trusted public key

	// Sign with a DIFFERENT, unrelated keypair the server never trusted.
	_, forgedPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	claims := gatewayClaims{
		UserID: uuid.New().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	forged, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(forgedPriv)
	require.NoError(t, err)

	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications?user_id="+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+forged)
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "a token signed by an untrusted key must be rejected")
}

// TestT11JWTAuth_FailsClosedWhenJWTPublicKeyUnconfigured proves the
// service still fails CLOSED (never open) if JWT_PUBLIC_KEY is unset,
// mirroring billing-service's same invariant.
func TestT11JWTAuth_FailsClosedWhenJWTPublicKeyUnconfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prevKey, hadPrevKey := os.LookupEnv("JWT_PUBLIC_KEY")
	os.Unsetenv("JWT_PUBLIC_KEY")
	t.Cleanup(func() {
		if hadPrevKey {
			os.Setenv("JWT_PUBLIC_KEY", prevKey)
		}
	})

	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications?user_id="+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer anything.at.all")
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"an unconfigured JWT_PUBLIC_KEY must fail closed, never fail open")
}
