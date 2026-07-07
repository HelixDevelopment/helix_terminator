//go:build integration

// Package server_test — REAL integration + security tests against a real
// PostgreSQL instance and the REAL vault-service HTTP server (queue#4,
// §11.4.27). Excluded from the default `go test ./...` run (build tag
// `integration`). Requires:
//
//	export DATABASE_URL="postgres://postgres:pass@127.0.0.1:5432/vault_service_test?sslmode=disable"
//	go test -tags integration ./internal/server/...
package server_test

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/vault-service/internal/model"
	"github.com/helixdevelopment/vault-service/internal/server"
)

const testAPIKey = "queue4-integration-test-api-key"

// nopLogger discards log output so integration test runs stay quiet.
type nopLogger struct{}

func (nopLogger) Printf(format string, v ...interface{}) {}
func (nopLogger) Println(v ...interface{})               {}

// mustConnectAndMigrate connects to the real Postgres pointed at by
// DATABASE_URL and applies vault-service's real migration idempotently.
// Skips (does not fail) when DATABASE_URL is unset (§11.4.3).
func mustConnectAndMigrate(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping real-Postgres real-server integration test (§11.4.3)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	require.NoError(t, pool.Ping(ctx), "real Postgres at DATABASE_URL is not reachable")

	migrationPath := filepath.Join("..", "..", "migrations", "001_init.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(migrationSQL))
	require.NoError(t, err)

	t.Cleanup(func() { pool.Close() })
	return pool
}

// newRealServer boots the REAL vault-service HTTP server wired to the
// REAL Postgres via DATABASE_URL, exactly as cmd/vault-service/main.go
// does at process start.
func newRealServer(t *testing.T) *server.Server {
	t.Helper()
	t.Setenv("VAULT_SERVICE_API_KEY", testAPIKey)
	srv, err := server.New(nopLogger{})
	require.NoError(t, err)
	return srv
}

func deriveKey(passphrase, salt []byte) []byte {
	h := sha256.New()
	h.Write(passphrase)
	h.Write(salt)
	return h.Sum(nil)
}

func encryptAESGCM(t *testing.T, plaintext, passphrase []byte) (ciphertextB64, ivB64, saltB64 string) {
	t.Helper()
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	require.NoError(t, err)
	key := deriveKey(passphrase, salt)
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(nonce),
		base64.StdEncoding.EncodeToString(salt)
}

func decryptAESGCM(t *testing.T, ciphertextB64, ivB64, saltB64 string, passphrase []byte) []byte {
	t.Helper()
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	require.NoError(t, err)
	nonce, err := base64.StdEncoding.DecodeString(ivB64)
	require.NoError(t, err)
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	require.NoError(t, err)
	key := deriveKey(passphrase, salt)
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	require.NoError(t, err)
	return plaintext
}

// createSecretViaRealHTTP POSTs a real encrypted secret through the real
// server and returns the created secret's ID.
func createSecretViaRealHTTP(t *testing.T, srv *server.Server, userID uuid.UUID, ciphertextB64, ivB64, saltB64 string) uuid.UUID {
	t.Helper()
	body := map[string]any{
		"user_id":         userID.String(),
		"name":            "queue4-http-proof",
		"type":            "api_token",
		"encrypted_value": ciphertextB64,
		"iv":              ivB64,
		"salt":            saltB64,
	}
	buf, err := json.Marshal(body)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/vault/secrets", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testAPIKey)
	// T7: CreateSecret derives the secret's owner from the authenticated
	// caller (X-User-ID), which must match the body's user_id.
	req.Header.Set("X-User-ID", userID.String())
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "CreateSecret via real HTTP server failed: %s", w.Body.String())

	var resp model.SecretResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.ID
}

// TestEncryptionAtRest_ThroughRealHTTPServer is the encryption-at-rest
// anti-bluff proof driven through the REAL HTTP server (handler + real
// repository + real Postgres 17.2), not just the repository layer.
func TestEncryptionAtRest_ThroughRealHTTPServer(t *testing.T) {
	pool := mustConnectAndMigrate(t)
	srv := newRealServer(t)

	const knownPlaintext = "http-server-proof-plaintext-do-not-leak-me-4f8c"
	passphrase := []byte("http-server-integration-passphrase")
	ciphertextB64, ivB64, saltB64 := encryptAESGCM(t, []byte(knownPlaintext), passphrase)
	require.NotContains(t, ciphertextB64, knownPlaintext)

	userID := uuid.New()
	secretID := createSecretViaRealHTTP(t, srv, userID, ciphertextB64, ivB64, saltB64)

	// --- Anti-bluff core assertion: query the ACTUAL DB row directly,
	// bypassing the service entirely, to see exactly what physically
	// landed on disk. ---
	ctx := context.Background()
	var storedEncryptedValue, storedIV, storedSalt string
	err := pool.QueryRow(ctx,
		`SELECT encrypted_value, iv, salt FROM secrets WHERE id = $1`, secretID,
	).Scan(&storedEncryptedValue, &storedIV, &storedSalt)
	require.NoError(t, err)

	require.Equal(t, ciphertextB64, storedEncryptedValue)
	require.NotContains(t, storedEncryptedValue, knownPlaintext,
		"CRITICAL: plaintext leaked into the encrypted_value column via the real HTTP path")

	// The API response must NEVER echo the ciphertext/iv/salt fields back
	// (SecretResponse omits them via json:"-") — confirm the zero-knowledge
	// contract holds over the wire too.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+secretID.String(), nil)
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("X-User-ID", userID.String())
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotContains(t, w.Body.String(), knownPlaintext)
	require.NotContains(t, w.Body.String(), ciphertextB64,
		"the HTTP response must never expose the raw ciphertext")

	// Decrypt the exact bytes read back from the real DB row.
	decrypted := decryptAESGCM(t, storedEncryptedValue, storedIV, storedSalt, passphrase)
	require.Equal(t, knownPlaintext, string(decrypted))
}

// TestSecurity_MissingAPIKeyRejected proves the real server rejects any
// vault request lacking service authentication.
func TestSecurity_MissingAPIKeyRejected(t *testing.T) {
	mustConnectAndMigrate(t)
	srv := newRealServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+uuid.New().String(), nil)
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestSecurity_WrongAPIKeyRejected proves a mismatched service API key is
// rejected, not merely a missing one.
func TestSecurity_WrongAPIKeyRejected(t *testing.T) {
	mustConnectAndMigrate(t)
	srv := newRealServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+uuid.New().String(), nil)
	req.Header.Set("X-API-Key", "not-the-real-key")
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestSecurity_AnotherTenantCannotReadSecret is the core access-control
// denial proof: two real tenants, two real secrets, real HTTP requests
// against the real server + real Postgres — tenant B must NOT be able to
// read tenant A's secret, and vice versa.
func TestSecurity_AnotherTenantCannotReadSecret(t *testing.T) {
	mustConnectAndMigrate(t)
	srv := newRealServer(t)

	tenantA := uuid.New()
	tenantB := uuid.New()

	ctA, ivA, saltA := encryptAESGCM(t, []byte("tenant-A-only-secret"), []byte("tenant-a-key"))
	secretAID := createSecretViaRealHTTP(t, srv, tenantA, ctA, ivA, saltA)

	// Tenant B, correctly authenticated as ITSELF, attempts to read
	// tenant A's secret by ID.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+secretAID.String(), nil)
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("X-User-ID", tenantB.String())
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code,
		"tenant B was able to read tenant A's secret — access control is BROKEN")

	// Sanity: tenant A, reading its OWN secret with the same API key, DOES
	// succeed — proving the denial above is genuine tenant isolation, not
	// a generally-broken endpoint.
	wOwner := httptest.NewRecorder()
	reqOwner, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+secretAID.String(), nil)
	reqOwner.Header.Set("X-API-Key", testAPIKey)
	reqOwner.Header.Set("X-User-ID", tenantA.String())
	srv.Router().ServeHTTP(wOwner, reqOwner)
	require.Equal(t, http.StatusOK, wOwner.Code,
		"the secret's real owner must still be able to read it")
}

// TestSecurity_AnotherTenantCannotDeleteOrRotateSecret extends the tenant
// isolation proof to the other secret-ID-scoped mutating routes.
func TestSecurity_AnotherTenantCannotDeleteOrRotateSecret(t *testing.T) {
	mustConnectAndMigrate(t)
	srv := newRealServer(t)

	tenantA := uuid.New()
	tenantB := uuid.New()

	ctA, ivA, saltA := encryptAESGCM(t, []byte("tenant-A-rotate-delete-secret"), []byte("tenant-a-key-2"))
	secretAID := createSecretViaRealHTTP(t, srv, tenantA, ctA, ivA, saltA)

	// Tenant B attempts to rotate tenant A's secret.
	rotateBody, _ := json.Marshal(map[string]any{
		"encrypted_value": "attacker-supplied-ciphertext",
		"iv":              "attacker-iv",
		"salt":            "attacker-salt",
		"created_by":      tenantB.String(),
	})
	wRotate := httptest.NewRecorder()
	reqRotate, _ := http.NewRequest(http.MethodPost, "/api/v1/vault/secrets/"+secretAID.String()+"/rotate", bytes.NewReader(rotateBody))
	reqRotate.Header.Set("Content-Type", "application/json")
	reqRotate.Header.Set("X-API-Key", testAPIKey)
	reqRotate.Header.Set("X-User-ID", tenantB.String())
	srv.Router().ServeHTTP(wRotate, reqRotate)
	require.Equal(t, http.StatusNotFound, wRotate.Code,
		"tenant B was able to rotate tenant A's secret — access control is BROKEN")

	// Tenant B attempts to delete tenant A's secret.
	wDelete := httptest.NewRecorder()
	reqDelete, _ := http.NewRequest(http.MethodDelete, "/api/v1/vault/secrets/"+secretAID.String(), nil)
	reqDelete.Header.Set("X-API-Key", testAPIKey)
	reqDelete.Header.Set("X-User-ID", tenantB.String())
	srv.Router().ServeHTTP(wDelete, reqDelete)
	require.Equal(t, http.StatusNotFound, wDelete.Code,
		"tenant B was able to delete tenant A's secret — access control is BROKEN")

	// Confirm the secret genuinely survived intact (was not soft-deleted)
	// by having its real owner fetch it successfully afterwards.
	wOwner := httptest.NewRecorder()
	reqOwner, _ := http.NewRequest(http.MethodGet, "/api/v1/vault/secrets/"+secretAID.String(), nil)
	reqOwner.Header.Set("X-API-Key", testAPIKey)
	reqOwner.Header.Set("X-User-ID", tenantA.String())
	srv.Router().ServeHTTP(wOwner, reqOwner)
	require.Equal(t, http.StatusOK, wOwner.Code,
		"tenant A's secret must have survived tenant B's rejected delete/rotate attempts")
}

// listSecretsViaRealHTTP GETs /api/v1/vault/secrets as callerID through the
// real server and returns the decoded response.
func listSecretsViaRealHTTP(t *testing.T, srv *server.Server, callerID uuid.UUID, extraQuery string) (*http.Response, model.ListSecretsResponse) {
	t.Helper()
	url := "/api/v1/vault/secrets"
	if extraQuery != "" {
		url += "?" + extraQuery
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("X-User-ID", callerID.String())
	srv.Router().ServeHTTP(w, req)

	resp := w.Result()
	var decoded model.ListSecretsResponse
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &decoded))
	}
	return resp, decoded
}

// TestSecurity_AnotherTenantCannotListSecrets is the T7 IDOR proof for
// ListSecrets: two real tenants, two real secrets, real HTTP requests
// against the real server + real Postgres. Tenant B must NEVER see tenant
// A's secret — not by passing tenant A's user_id as a query parameter, and
// not by omitting user_id (the pre-fix behaviour returned EVERY tenant's
// secrets when no filter was supplied, since the repository treats a
// zero-value user_id as "no filter").
func TestSecurity_AnotherTenantCannotListSecrets(t *testing.T) {
	mustConnectAndMigrate(t)
	srv := newRealServer(t)

	tenantA := uuid.New()
	tenantB := uuid.New()

	ctA, ivA, saltA := encryptAESGCM(t, []byte("tenant-A-list-secret"), []byte("tenant-a-list-key"))
	secretAID := createSecretViaRealHTTP(t, srv, tenantA, ctA, ivA, saltA)

	ctB, ivB, saltB := encryptAESGCM(t, []byte("tenant-B-list-secret"), []byte("tenant-b-list-key"))
	secretBID := createSecretViaRealHTTP(t, srv, tenantB, ctB, ivB, saltB)

	// Attack 1: tenant B, correctly authenticated as ITSELF, tries to list
	// secrets by passing tenant A's user_id as the query parameter.
	wMismatch, _ := listSecretsViaRealHTTP(t, srv, tenantB, "user_id="+tenantA.String())
	require.Equal(t, http.StatusBadRequest, wMismatch.StatusCode,
		"tenant B was able to pass tenant A's user_id to ListSecrets — access control is BROKEN")

	// Attack 2 (the original bug): tenant B lists with NO user_id query
	// param at all. Pre-fix, an empty/zero user_id fell through to the
	// repository's "no filter" behaviour and returned every tenant's
	// secrets, including tenant A's. Post-fix it must return ONLY tenant
	// B's own secret.
	wList, listResp := listSecretsViaRealHTTP(t, srv, tenantB, "")
	require.Equal(t, http.StatusOK, wList.StatusCode)
	for _, s := range listResp.Secrets {
		require.NotEqual(t, secretAID, s.ID, "tenant B's list leaked tenant A's secret — IDOR is present")
		require.Equal(t, tenantB, s.UserID, "every secret in tenant B's list must belong to tenant B")
	}
	foundOwnB := false
	for _, s := range listResp.Secrets {
		if s.ID == secretBID {
			foundOwnB = true
		}
	}
	require.True(t, foundOwnB, "tenant B's own secret must appear in tenant B's own list")

	// Sanity: tenant A's own list still works, is scoped to tenant A only,
	// and contains its own secret — proving the denials above are genuine
	// tenant isolation, not a generally-broken endpoint.
	wOwner, ownerResp := listSecretsViaRealHTTP(t, srv, tenantA, "")
	require.Equal(t, http.StatusOK, wOwner.StatusCode)
	foundOwnA := false
	for _, s := range ownerResp.Secrets {
		require.NotEqual(t, secretBID, s.ID, "tenant A's list leaked tenant B's secret")
		if s.ID == secretAID {
			foundOwnA = true
		}
	}
	require.True(t, foundOwnA, "tenant A's own secret must appear in tenant A's own list")
}

// TestSecurity_AnotherTenantCannotCreateSecretForVictim is the T7 IDOR
// proof for CreateSecret: a real caller authenticated as tenant B cannot
// create/own a secret under tenant A's identity merely by putting tenant
// A's user_id in the request body.
func TestSecurity_AnotherTenantCannotCreateSecretForVictim(t *testing.T) {
	pool := mustConnectAndMigrate(t)
	srv := newRealServer(t)

	tenantA := uuid.New()
	tenantB := uuid.New()

	ct, iv, salt := encryptAESGCM(t, []byte("attacker-attempted-secret"), []byte("attacker-key"))
	body, err := json.Marshal(map[string]any{
		"user_id":         tenantA.String(), // attacker-supplied victim tenant
		"name":            "stolen-secret",
		"type":            "api_token",
		"encrypted_value": ct,
		"iv":              iv,
		"salt":            salt,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/vault/secrets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("X-User-ID", tenantB.String()) // real authenticated caller
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code,
		"tenant B was able to create a secret owned by tenant A — access control is BROKEN")

	// Confirm no row was ever persisted under tenant A's ownership for this
	// attempt — the rejection is real, not just an API-shape mismatch.
	ctx := context.Background()
	var count int
	errCount := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM secrets WHERE user_id = $1 AND name = $2`, tenantA, "stolen-secret",
	).Scan(&count)
	require.NoError(t, errCount)
	require.Equal(t, 0, count, "no secret must have been persisted under tenant A's ownership")

	// Sanity: tenant B creating a secret for ITSELF still works.
	secretBID := createSecretViaRealHTTP(t, srv, tenantB, ct, iv, salt)
	require.NotEqual(t, uuid.Nil, secretBID)
}
