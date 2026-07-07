package server_test

// T15 production blocker (server-level wiring proof): server.New wires
// its JWTManager through loadJWTManager (internal/server/server.go),
// which resolves JWT_PRIVATE_KEY/JWT_PUBLIC_KEY/ENVIRONMENT exactly as
// documented in docs/guides/JWT_KEY_PROVISIONING.md. This file drives
// that resolution ONLY through server.New's public API (no build tag,
// no real database required - DATABASE_URL stays unset so server.New
// takes its existing, already-legitimate in-memory-mode fallback,
// isolating this test from Postgres/podman availability). No real key
// material is committed anywhere; every key here is generated fresh
// in-test.

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/helixdevelopment/auth-service/internal/server"
)

type recordingLogger struct {
	lines []string
}

func (l *recordingLogger) Printf(format string, v ...interface{}) {
	l.lines = append(l.lines, format)
}
func (l *recordingLogger) Println(v ...interface{}) {}

func (l *recordingLogger) contains(substr string) bool {
	for _, line := range l.lines {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

func clearJWTEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_PRIVATE_KEY", "")
	t.Setenv("JWT_PUBLIC_KEY", "")
	t.Setenv("ENVIRONMENT", "")
}

// TestServerNew_NoKeyNoProductionFlag_FallsBackToEphemeralWithLoudWarning
// covers the dev/test fallback path: no JWT_PRIVATE_KEY, no
// ENVIRONMENT=production - server.New must still succeed (this is the
// path today's existing integration test suite already relies on) AND
// must log a loud, unmistakable warning rather than silently proceeding.
func TestServerNew_NoKeyNoProductionFlag_FallsBackToEphemeralWithLoudWarning(t *testing.T) {
	clearJWTEnv(t)

	logger := &recordingLogger{}
	srv, err := server.New(logger)
	if err != nil {
		t.Fatalf("server.New failed with no JWT_PRIVATE_KEY and no ENVIRONMENT set (expected ephemeral fallback): %v", err)
	}
	if srv.JWTManager() == nil {
		t.Fatal("server.New returned a nil JWTManager on the ephemeral fallback path")
	}
	if !logger.contains("WARNING: ephemeral JWT key") {
		t.Fatalf("expected a loud ephemeral-key warning to be logged, got log lines: %v", logger.lines)
	}
}

// TestServerNew_ProductionModeWithNoKey_FailsClosed is the fail-closed
// proof: ENVIRONMENT=production with no JWT_PRIVATE_KEY MUST refuse to
// start rather than silently mint an ephemeral, cross-instance-unusable
// signing key - the exact production defect this fix closes.
func TestServerNew_ProductionModeWithNoKey_FailsClosed(t *testing.T) {
	clearJWTEnv(t)
	t.Setenv("ENVIRONMENT", "production")

	if srv, err := server.New(&recordingLogger{}); err == nil {
		t.Fatalf("server.New unexpectedly succeeded with ENVIRONMENT=production and no JWT_PRIVATE_KEY "+
			"(got server=%v); want a fatal, descriptive error", srv)
	}
}

// TestServerNew_ProvisionedKey_UsesExactKeyMaterial proves the
// production wiring end-to-end through the public API: with a real
// (test-generated) JWT_PRIVATE_KEY/JWT_PUBLIC_KEY pair provisioned via
// the environment, server.New's JWTManager signs with EXACTLY that key
// - not a substitute ephemeral one - verified by checking its public
// key matches the provisioned pair byte-for-byte.
func TestServerNew_ProvisionedKey_UsesExactKeyMaterial(t *testing.T) {
	clearJWTEnv(t)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey (test-only keypair) failed: %v", err)
	}
	privB64 := base64.StdEncoding.EncodeToString(priv)
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	t.Setenv("JWT_PRIVATE_KEY", privB64)
	t.Setenv("JWT_PUBLIC_KEY", pubB64)
	// Also set ENVIRONMENT=production: proves the fail-closed guard does
	// NOT block a genuinely well-provisioned production deployment.
	t.Setenv("ENVIRONMENT", "production")

	srv, err := server.New(&recordingLogger{})
	if err != nil {
		t.Fatalf("server.New failed with a valid provisioned JWT_PRIVATE_KEY/JWT_PUBLIC_KEY pair "+
			"under ENVIRONMENT=production: %v", err)
	}

	gotPub := srv.JWTManager().PublicKey()
	if !ed25519PublicKeyEqual(gotPub, pub) {
		t.Fatalf("server.New's JWTManager public key does not match the provisioned JWT_PUBLIC_KEY - "+
			"got %x, want %x", []byte(gotPub), []byte(pub))
	}

	// Round-trip through the exact manager this server would validate
	// requests against - the same manager instance /me and every
	// authenticated route use.
	token, _, err := srv.JWTManager().GenerateAccessToken("wiring-user", "wiring-org", "wiring@example.com", "user", "wiring-session", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken on the provisioned-key manager failed: %v", err)
	}
	if _, err := srv.JWTManager().ValidateToken(token); err != nil {
		t.Fatalf("ValidateToken on the provisioned-key manager failed on its own freshly-issued token: %v", err)
	}
}

// TestServerNew_MismatchedProvisionedKeys_FailsClosed proves server.New
// propagates NewJWTManagerFromKey's consistency-check failure rather
// than silently falling back to an ephemeral key when the operator
// provisions an internally-inconsistent key pair.
func TestServerNew_MismatchedProvisionedKeys_FailsClosed(t *testing.T) {
	clearJWTEnv(t)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey (private half) failed: %v", err)
	}
	unrelatedPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey (unrelated public half) failed: %v", err)
	}

	t.Setenv("JWT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(priv))
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(unrelatedPub))

	if srv, err := server.New(&recordingLogger{}); err == nil {
		t.Fatalf("server.New unexpectedly succeeded with a mismatched JWT_PRIVATE_KEY/JWT_PUBLIC_KEY "+
			"pair (got server=%v); want a fatal, descriptive error", srv)
	}
}

func ed25519PublicKeyEqual(a, b ed25519.PublicKey) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
