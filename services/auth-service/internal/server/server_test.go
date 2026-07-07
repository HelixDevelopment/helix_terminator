//go:build integration

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/helixdevelopment/auth-service/internal/crypto"
	"github.com/helixdevelopment/auth-service/internal/model"
	"github.com/helixdevelopment/auth-service/internal/server"
	"github.com/helixdevelopment/auth-service/internal/testinfra"
)

// testLogger adapts *testing.T to server.Logger.
type testLogger struct{ t *testing.T }

func (l *testLogger) Printf(format string, v ...interface{}) { l.t.Logf(format, v...) }
func (l *testLogger) Println(v ...interface{})               { l.t.Log(v...) }

// newTestServer boots a real, disposable PostgreSQL 17.2 container (via
// rootless podman), points DATABASE_URL at it, and constructs the REAL
// auth-service server via server.New - the exact construction path
// cmd/auth-service/main.go uses, including the real migrations.Run
// schema-apply step. The returned httptest.Server is a genuine
// net/http server bound to a real TCP socket serving the real gin
// Router() - every request in this file travels real HTTP, not an
// in-process fake transport. Per §11.4.27 no mocks/stubs are used.
func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dbURL := testinfra.StartPostgres(t)
	t.Setenv("DATABASE_URL", dbURL)

	srv, err := server.New(&testLogger{t: t})
	if err != nil {
		t.Fatalf("server.New failed against real database %q: %v", dbURL, err)
	}

	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return ts, dbURL
}

func postJSON(t *testing.T, client *http.Client, url string, body interface{}, bearer string) (int, map[string]interface{}) {
	t.Helper()
	return doJSON(t, client, http.MethodPost, url, body, bearer)
}

func doJSON(t *testing.T, client *http.Client, method, url string, body interface{}, bearer string) (int, map[string]interface{}) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal(request body) failed: %v", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("http.NewRequest(%s %s) failed: %v", method, url, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: reading response body failed: %v", method, url, err)
	}

	var parsed map[string]interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("%s %s: response body is not valid JSON: %v\nbody: %s", method, url, err, raw)
		}
	}
	return resp.StatusCode, parsed
}

// assertPasswordHashedInDB independently re-verifies the required
// queue#4 proof directly against the real running PostgreSQL instance
// via a raw SQL connection (deliberately NOT reusing the repository
// package's code path, for an independent cross-check): the persisted
// users.password_hash column is a genuine, verifiable Argon2id hash of
// the registered password, never the plaintext.
func assertPasswordHashedInDB(t *testing.T, dbURL, email, plainPassword string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("pgx.Connect for independent DB-state assertion failed: %v", err)
	}
	defer conn.Close(ctx)

	var passwordHash string
	err = conn.QueryRow(ctx, `SELECT password_hash FROM users WHERE email = $1`, email).Scan(&passwordHash)
	if err != nil {
		t.Fatalf("raw SQL SELECT password_hash for %q failed: %v", email, err)
	}

	if passwordHash == plainPassword {
		t.Fatalf("users.password_hash for %q stores the PLAINTEXT password verbatim - critical security defect", email)
	}
	if !strings.HasPrefix(passwordHash, "$argon2id$") {
		t.Fatalf("users.password_hash for %q = %q, want a real $argon2id$... hash", email, passwordHash)
	}

	hasher := crypto.NewPasswordHasher()
	ok, err := hasher.VerifyPassword(plainPassword, passwordHash)
	if err != nil {
		t.Fatalf("VerifyPassword against the real DB row's hash errored: %v", err)
	}
	if !ok {
		t.Fatal("the real DB row's password_hash does not verify against the plaintext password that was registered")
	}
}

// TestFullAuthJourney_RegisterLoginUseRefreshLogout_Integration drives
// the complete real user journey over real HTTP against the real
// auth-service server and a real PostgreSQL instance: register a user,
// log in, use the access token against a protected route, refresh it,
// use the refreshed token, then log out and prove the token no longer
// works. This replaces the 4 pre-existing t.Skip("TODO") integration
// stubs with a genuine end-to-end proof (queue#4, §11.4.27).
func TestFullAuthJourney_RegisterLoginUseRefreshLogout_Integration(t *testing.T) {
	ts, dbURL := newTestServer(t)
	client := ts.Client()

	email := fmt.Sprintf("journey-%d@example.com", time.Now().UnixNano())
	password := "a-genuinely-long-password-987"

	// 1. Register.
	status, body := postJSON(t, client, ts.URL+"/register", model.RegisterRequest{
		Email:       email,
		Password:    password,
		DisplayName: "Journey User",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("POST /register status = %d, want 201; body=%v", status, body)
	}
	registerAccessToken, _ := body["accessToken"].(string)
	registerRefreshToken, _ := body["refreshToken"].(string)
	if registerAccessToken == "" || registerRefreshToken == "" {
		t.Fatalf("POST /register did not return real access/refresh tokens: %v", body)
	}
	userObj, _ := body["user"].(map[string]interface{})
	if userObj == nil || userObj["email"] != email {
		t.Fatalf("POST /register response user object missing/incorrect: %v", body)
	}
	if _, leaked := userObj["passwordHash"]; leaked {
		t.Fatalf("POST /register response leaks a passwordHash field: %v", userObj)
	}

	// Real DB-state assertion (independent raw-SQL cross-check): the
	// user row exists and its password is stored HASHED, not plaintext.
	assertPasswordHashedInDB(t, dbURL, email, password)

	// 2. Login (a second, independent authentication - proves login
	// works standalone, not merely as a side effect of register).
	status, body = postJSON(t, client, ts.URL+"/login", model.LoginRequest{
		Email:      email,
		Password:   password,
		DeviceID:   "integration-test-device",
		DeviceName: "CI Runner",
	}, "")
	if status != http.StatusOK {
		t.Fatalf("POST /login status = %d, want 200; body=%v", status, body)
	}
	accessToken, _ := body["accessToken"].(string)
	refreshToken, _ := body["refreshToken"].(string)
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("POST /login did not return real access/refresh tokens: %v", body)
	}
	if accessToken == registerAccessToken {
		t.Fatal("login issued the identical access token as register - tokens must be per-session")
	}

	// 3. Use it: call the authenticated /me route with the real bearer token.
	status, body = doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, accessToken)
	if status != http.StatusOK {
		t.Fatalf("GET /me with a fresh login token status = %d, want 200; body=%v", status, body)
	}
	if body["userId"] == nil || body["userId"] == "" {
		t.Fatalf("GET /me did not resolve a userId from the bearer token: %v", body)
	}

	// 4. Refresh: exchange the refresh token for a new access token.
	status, body = postJSON(t, client, ts.URL+"/refresh", model.RefreshRequest{RefreshToken: refreshToken}, "")
	if status != http.StatusOK {
		t.Fatalf("POST /refresh status = %d, want 200; body=%v", status, body)
	}
	refreshedAccessToken, _ := body["accessToken"].(string)
	if refreshedAccessToken == "" {
		t.Fatalf("POST /refresh did not return a real access token: %v", body)
	}
	if refreshedAccessToken == accessToken {
		t.Fatal("POST /refresh returned the SAME access token instead of minting a new one")
	}

	// The refreshed token must genuinely work against the protected
	// route - proves the session's access-token-hash was correctly
	// rebound to the new token server-side.
	status, body = doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, refreshedAccessToken)
	if status != http.StatusOK {
		t.Fatalf("GET /me with the refreshed token status = %d, want 200; body=%v", status, body)
	}

	// The now-superseded pre-refresh access token must no longer work,
	// since the session's revocation-lookup key moved to the new hash.
	status, _ = doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, accessToken)
	if status != http.StatusUnauthorized {
		t.Fatalf("GET /me with the pre-refresh access token status = %d, want 401 (superseded by refresh)", status)
	}

	// 5. Logout using the refreshed (currently-active) token.
	status, body = doJSON(t, client, http.MethodPost, ts.URL+"/logout", model.LogoutRequest{AllSessions: true}, refreshedAccessToken)
	if status != http.StatusNoContent {
		t.Fatalf("POST /logout status = %d, want 204; body=%v", status, body)
	}

	// 6. Real behavioural proof of logout: the just-logged-out token
	// must now be REJECTED. This exercises the same
	// "replayed-after-logout" property the dedicated security battery
	// (TestSecurityTokenRejectionBattery_Integration) proves in
	// isolation for a fresh user.
	status, body = doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, refreshedAccessToken)
	if status != http.StatusUnauthorized {
		t.Fatalf("GET /me with a token replayed after /logout status = %d, want 401; body=%v", status, body)
	}
}

// TestSecurityTokenRejectionBattery_Integration is the queue#4 security
// test: a battery of real HTTP requests against the real running
// server proving each class of invalid bearer token is genuinely
// rejected with 401 - not merely "would be rejected in theory".
func TestSecurityTokenRejectionBattery_Integration(t *testing.T) {
	ts, _ := newTestServer(t)
	client := ts.Client()

	email := fmt.Sprintf("security-%d@example.com", time.Now().UnixNano())
	password := "another-genuinely-long-password-321"

	status, body := postJSON(t, client, ts.URL+"/register", model.RegisterRequest{
		Email: email, Password: password, DisplayName: "Security Test User",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("setup: POST /register status = %d, want 201; body=%v", status, body)
	}
	accessToken, _ := body["accessToken"].(string)
	if accessToken == "" {
		t.Fatalf("setup: no access token returned: %v", body)
	}

	// Sanity: the freshly-issued token DOES work before the battery
	// runs, so every rejection below is meaningful and not a
	// pre-existing, broken-for-everyone 401.
	if status, _ := doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, accessToken); status != http.StatusOK {
		t.Fatalf("setup: fresh access token was rejected before any battery case ran, status = %d, want 200", status)
	}

	t.Run("missing_auth_header_rejected", func(t *testing.T) {
		status, body := doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, "")
		if status != http.StatusUnauthorized {
			t.Fatalf("GET /me with no Authorization header status = %d, want 401; body=%v", status, body)
		}
	})

	t.Run("tampered_signature_rejected", func(t *testing.T) {
		parts := strings.Split(accessToken, ".")
		if len(parts) != 3 {
			t.Fatalf("access token is not a well-formed JWT (want 3 segments), got %d", len(parts))
		}
		tampered := parts[0] + "." + parts[1] + "." + tamperSegment(parts[2])

		status, body := doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, tampered)
		if status != http.StatusUnauthorized {
			t.Fatalf("GET /me with a tampered-signature token status = %d, want 401; body=%v", status, body)
		}
	})

	t.Run("wrong_signing_key_rejected", func(t *testing.T) {
		// A well-formed, internally self-consistent token signed by a
		// completely different Ed25519 key than the server's - proves
		// the server verifies against ITS OWN key, not merely
		// "is this a validly-shaped JWT".
		foreignManager, err := crypto.NewJWTManager()
		if err != nil {
			t.Fatalf("crypto.NewJWTManager() failed: %v", err)
		}
		foreignToken, _, err := foreignManager.GenerateAccessToken("forged-user-id", "", email, "user", "forged-session-id", nil)
		if err != nil {
			t.Fatalf("foreignManager.GenerateAccessToken failed: %v", err)
		}

		status, body := doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, foreignToken)
		if status != http.StatusUnauthorized {
			t.Fatalf("GET /me with a token signed by a foreign key status = %d, want 401; body=%v", status, body)
		}
	})

	t.Run("expired_token_rejected", func(t *testing.T) {
		mgr, err := crypto.NewJWTManager()
		if err != nil {
			t.Fatalf("crypto.NewJWTManager() failed: %v", err)
		}
		expired, err := mgr.GenerateAccessTokenWithExpiry(
			"expired-user-id", "", email, "user", "expired-session-id", nil,
			time.Now().UTC().Add(-1*time.Hour),
		)
		if err != nil {
			t.Fatalf("GenerateAccessTokenWithExpiry failed: %v", err)
		}

		status, body := doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, expired)
		if status != http.StatusUnauthorized {
			t.Fatalf("GET /me with an expired token status = %d, want 401; body=%v", status, body)
		}
	})

	// This case runs last: it revokes accessToken, which every earlier
	// subtest depended on remaining valid.
	t.Run("replayed_after_logout_token_rejected", func(t *testing.T) {
		status, body := doJSON(t, client, http.MethodPost, ts.URL+"/logout", model.LogoutRequest{AllSessions: true}, accessToken)
		if status != http.StatusNoContent {
			t.Fatalf("setup: POST /logout status = %d, want 204; body=%v", status, body)
		}

		status, body = doJSON(t, client, http.MethodGet, ts.URL+"/me", nil, accessToken)
		if status != http.StatusUnauthorized {
			t.Fatalf("GET /me with a token replayed after /logout status = %d, want 401; body=%v", status, body)
		}
	})
}

// tamperSegment flips the first byte of a JWT segment to a different
// valid character, guaranteeing a byte-level change that invalidates
// the base64url-encoded signature it belongs to.
func tamperSegment(seg string) string {
	if len(seg) == 0 {
		return "x"
	}
	b := []byte(seg)
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	return string(b)
}
