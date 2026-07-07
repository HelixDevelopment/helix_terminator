package crypto_test

// T15 production blocker: auth-service used to generate a fresh,
// ephemeral Ed25519 keypair on every process start (crypto.NewJWTManager
// called with no persisted key). A token signed by one process instance
// could never be verified by a DIFFERENT process instance - not
// gateway-service, not billing-service (both validate independently via
// a JWT_PUBLIC_KEY they load from their own environment), and not even
// auth-service itself after a restart. This file proves both halves of
// the fix with real cryptographic verification, never committing any
// real key material - every key here is generated fresh in-test.
//
// RED (below, TestCrossServiceValidation_EphemeralKeyFailsAcrossInstances):
// reproduces the underlying defect class on the pre-fix construction
// path (crypto.NewJWTManager with no persisted key) - a token signed by
// one manager instance is rejected by a public key independently loaded
// by a second instance, exactly mirroring how gateway-service/
// billing-service validate.
//
// GREEN (TestCrossServiceValidation_ProvisionedKeyValidatesAcrossIndependentlyLoadedPublicKey):
// proves the fix - crypto.NewJWTManagerFromKey, loaded from a base64
// (standard encoding) Ed25519 private key exactly as JWT_PRIVATE_KEY
// will be provisioned in production, signs a token that DOES verify
// against a public key independently decoded from the paired
// JWT_PUBLIC_KEY value, using gateway/billing's own validation logic
// mirrored below (jwt.ParseWithClaims + SigningMethodEd25519 check).

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/helixdevelopment/auth-service/internal/crypto"
)

// gatewayLikeClaims mirrors the Claims struct gateway-service and
// billing-service independently declare and validate against
// (services/gateway-service/internal/server/server.go,
// services/billing-service/internal/server/server.go) - a minimal
// UserID/OrgID projection of the full auth-service crypto.Claims, which
// is exactly what those services need and all they ever see, since the
// gateway forwards the original signed bearer token untouched.
type gatewayLikeClaims struct {
	UserID string `json:"userId"`
	OrgID  string `json:"orgId,omitempty"`
	jwt.RegisteredClaims
}

// verifyLikeGatewayOrBilling mirrors gateway-service's validateToken
// (services/gateway-service/internal/server/server.go) and
// billing-service's equivalent byte-for-byte: parse with the Ed25519
// signing-method guard, verify against an INDEPENDENTLY-supplied public
// key (never the signer's own in-process key), and return the decoded
// claims. This is the exact verification path a real gateway-service or
// billing-service process runs against a real, persisted JWT_PUBLIC_KEY.
func verifyLikeGatewayOrBilling(tokenString string, pub ed25519.PublicKey) (*gatewayLikeClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &gatewayLikeClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pub, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(*gatewayLikeClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}
	return claims, nil
}

// TestCrossServiceValidation_EphemeralKeyFailsAcrossInstances is the RED
// proof: it reproduces the production defect on the pre-fix
// construction path. auth-service's ephemeral crypto.NewJWTManager()
// generates a brand-new keypair every call - there is no persistence
// mechanism at all, so a second instance (modelling gateway-service, or
// auth-service itself after a restart) can never independently obtain
// the signer's current public key. Verifying the signed token against
// ANY independently-obtained public key - here, another freshly
// generated ephemeral manager's public key, which is the closest thing
// to "the public key gateway-service would have on hand" when no
// persisted key exists - MUST fail.
func TestCrossServiceValidation_EphemeralKeyFailsAcrossInstances(t *testing.T) {
	authInstance, err := crypto.NewJWTManager()
	if err != nil {
		t.Fatalf("crypto.NewJWTManager() (simulated auth-service instance) failed: %v", err)
	}

	token, _, err := authInstance.GenerateAccessToken("user-red-1", "org-red-1", "red@example.com", "user", "sess-red-1", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Simulate gateway-service/billing-service independently trying to
	// obtain "the auth-service public key" with no persisted key
	// available: the best it can do is its own independently generated
	// key, which is - by construction of the pre-fix design - NEVER the
	// same as auth's current ephemeral key.
	gatewayInstance, err := crypto.NewJWTManager()
	if err != nil {
		t.Fatalf("crypto.NewJWTManager() (simulated gateway-service instance) failed: %v", err)
	}

	if _, err := verifyLikeGatewayOrBilling(token, gatewayInstance.PublicKey()); err == nil {
		t.Fatal("RED assertion failed: a token signed by one ephemeral JWTManager instance " +
			"unexpectedly validated against a DIFFERENT instance's public key - the defect this " +
			"test exists to reproduce is not present, which means the pre-fix construction path " +
			"changed underneath this test")
	}

	// Same defect from a different angle: auth-service restarting (a
	// fresh crypto.NewJWTManager() call) also invalidates every token the
	// PREVIOUS instance issued, because there is still no persisted key.
	authInstanceAfterRestart, err := crypto.NewJWTManager()
	if err != nil {
		t.Fatalf("crypto.NewJWTManager() (simulated auth-service restart) failed: %v", err)
	}
	if _, err := verifyLikeGatewayOrBilling(token, authInstanceAfterRestart.PublicKey()); err == nil {
		t.Fatal("RED assertion failed: a token signed before an ephemeral-key restart " +
			"unexpectedly validated after the restart")
	}
}

// TestCrossServiceValidation_ProvisionedKeyValidatesAcrossIndependentlyLoadedPublicKey
// is the GREEN proof: crypto.NewJWTManagerFromKey loads a persisted
// Ed25519 private key exactly the way JWT_PRIVATE_KEY will be
// provisioned in production (base64 standard encoding, raw
// ed25519.PrivateKeySize bytes). A token it signs DOES validate against
// a public key INDEPENDENTLY decoded from the paired JWT_PUBLIC_KEY
// value - using gateway/billing's own mirrored verification logic -
// proving the cross-service/cross-restart validation gap is closed. The
// keypair is generated fresh in this test and never committed anywhere.
func TestCrossServiceValidation_ProvisionedKeyValidatesAcrossIndependentlyLoadedPublicKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey (test-only keypair) failed: %v", err)
	}
	privB64 := base64.StdEncoding.EncodeToString(priv)
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	// The "auth-service" side: constructed exactly the way
	// internal/server/server.go's loadJWTManager builds it from
	// JWT_PRIVATE_KEY (+ the paired JWT_PUBLIC_KEY consistency check).
	authInstance, err := crypto.NewJWTManagerFromKey(privB64, pubB64)
	if err != nil {
		t.Fatalf("NewJWTManagerFromKey failed with a valid, matching keypair: %v", err)
	}

	token, _, err := authInstance.GenerateAccessToken("user-green-1", "org-green-1", "green@example.com", "admin", "sess-green-1", []string{"read"})
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// The "gateway-service"/"billing-service" side: an INDEPENDENT
	// decode of the SAME base64 JWT_PUBLIC_KEY value - modelling a
	// completely separate process/service that never saw the private
	// key, exactly as services/gateway-service/internal/server/server.go
	// and services/billing-service/internal/server/server.go do today.
	independentRawPub, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil {
		t.Fatalf("base64 decode of JWT_PUBLIC_KEY (independent side) failed: %v", err)
	}
	if len(independentRawPub) != ed25519.PublicKeySize {
		t.Fatalf("decoded JWT_PUBLIC_KEY has wrong size: got %d, want %d", len(independentRawPub), ed25519.PublicKeySize)
	}
	independentPub := ed25519.PublicKey(independentRawPub)

	claims, err := verifyLikeGatewayOrBilling(token, independentPub)
	if err != nil {
		t.Fatalf("GREEN assertion failed: a token signed via the persisted-key manager did NOT "+
			"validate against an independently-loaded public key (mirroring gateway/billing's own "+
			"validation): %v", err)
	}
	if claims.UserID != "user-green-1" {
		t.Fatalf("independently-verified claims.UserID = %q, want %q", claims.UserID, "user-green-1")
	}
	if claims.OrgID != "org-green-1" {
		t.Fatalf("independently-verified claims.OrgID = %q, want %q", claims.OrgID, "org-green-1")
	}

	// The manager's own ValidateToken (the path auth-service's own
	// jwtValidationMiddleware uses) must also accept the token it just
	// issued, using ITS OWN in-memory public key - same key material,
	// different call path.
	if _, err := authInstance.ValidateToken(token); err != nil {
		t.Fatalf("authInstance.ValidateToken failed on its own freshly-issued token: %v", err)
	}
}

// TestNewJWTManagerFromKey_RejectsMismatchedPublicKey proves the
// fail-closed consistency check: a syntactically valid JWT_PUBLIC_KEY
// that does NOT correspond to the supplied JWT_PRIVATE_KEY must be
// rejected outright, never silently accepted (which would let a
// misconfigured deployment mint tokens no other service could ever
// validate - the exact class of outage this whole fix closes).
func TestNewJWTManagerFromKey_RejectsMismatchedPublicKey(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey (private half) failed: %v", err)
	}
	unrelatedPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey (unrelated public half) failed: %v", err)
	}

	privB64 := base64.StdEncoding.EncodeToString(priv)
	mismatchedPubB64 := base64.StdEncoding.EncodeToString(unrelatedPub)

	if _, err := crypto.NewJWTManagerFromKey(privB64, mismatchedPubB64); err == nil {
		t.Fatal("NewJWTManagerFromKey accepted a JWT_PUBLIC_KEY that does not match JWT_PRIVATE_KEY " +
			"- a mismatched provisioned key pair must be rejected, not silently accepted")
	}
}

// TestNewJWTManagerFromKey_RejectsMalformedInput is a table test over
// the malformed-input classes a real (mis)configured deployment could
// hand this function - none of them may be silently accepted.
func TestNewJWTManagerFromKey_RejectsMalformedInput(t *testing.T) {
	_, validPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey failed: %v", err)
	}
	validPrivB64 := base64.StdEncoding.EncodeToString(validPriv)

	cases := []struct {
		name    string
		privB64 string
		pubB64  string
	}{
		{name: "private key not base64", privB64: "not-valid-base64!!!", pubB64: ""},
		{name: "private key wrong size (too short)", privB64: base64.StdEncoding.EncodeToString([]byte("too-short")), pubB64: ""},
		{name: "public key not base64", privB64: validPrivB64, pubB64: "also-not-base64!!!"},
		{name: "public key wrong size", privB64: validPrivB64, pubB64: base64.StdEncoding.EncodeToString([]byte("too-short-for-a-public-key"))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := crypto.NewJWTManagerFromKey(tc.privB64, tc.pubB64); err == nil {
				t.Fatalf("NewJWTManagerFromKey(%q, %q) unexpectedly succeeded, want an error", tc.privB64, tc.pubB64)
			}
		})
	}
}
