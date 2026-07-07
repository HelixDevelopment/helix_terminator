//go:build integration

// Package repository_test — REAL integration tests against a real
// PostgreSQL instance (queue#4, §11.4.27, encryption-at-rest anti-bluff
// proof). These tests are excluded from the default `go test ./...` run
// (build tag `integration`) and require a live Postgres reachable via
// DATABASE_URL, e.g.:
//
//	export DATABASE_URL="postgres://postgres:pass@127.0.0.1:5432/vault_service_test?sslmode=disable"
//	go test -tags integration ./internal/repository/...
//
// vault-service is a ZERO-KNOWLEDGE store (see README.md "Zero-knowledge
// encrypted storage (client-side encryption)"): the repository/handler
// layers never see plaintext and never perform encryption/decryption
// themselves — the caller (here, the test, standing in for a real
// zero-knowledge client) encrypts before calling CreateSecret and
// decrypts after calling GetSecretByID. That contract is exactly what
// these tests exercise and prove.
package repository_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/vault-service/internal/model"
	"github.com/helixdevelopment/vault-service/internal/repository"
)

// deriveKey mimics a zero-knowledge client's key derivation: a real caller
// would use a strong KDF (e.g. Argon2id) — SHA-256(passphrase||salt) is
// used here only because it is stdlib-only and sufficient to prove genuine
// reversible AES-GCM ciphertext round-trips; the KDF strength itself is
// the CLIENT's responsibility, not vault-service's (vault-service never
// sees the passphrase or plaintext).
func deriveKey(passphrase, salt []byte) []byte {
	h := sha256.New()
	h.Write(passphrase)
	h.Write(salt)
	return h.Sum(nil) // 32 bytes -> AES-256
}

// encryptAESGCM performs REAL AES-256-GCM encryption, standing in for the
// zero-knowledge client that would run this in-browser/on-device before
// ever talking to vault-service.
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

// decryptAESGCM reverses encryptAESGCM, proving the exact bytes retrieved
// from the database decrypt back to the original plaintext.
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

// mustConnectAndMigrate connects to the real Postgres pointed at by
// DATABASE_URL and applies vault-service's real migration
// (migrations/001_init.sql) idempotently, so the test is self-sufficient
// against a freshly-created empty database. Skips (does not fail) when
// DATABASE_URL is unset — the correct §11.4.3 topology-appropriate
// behaviour for an integration test with no real target.
func mustConnectAndMigrate(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping real-Postgres integration test (§11.4.3)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to open pgxpool against DATABASE_URL")

	require.NoError(t, pool.Ping(ctx), "real Postgres at DATABASE_URL is not reachable")

	migrationPath := filepath.Join("..", "..", "migrations", "001_init.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err, "failed to read migrations/001_init.sql")

	_, err = pool.Exec(ctx, string(migrationSQL))
	require.NoError(t, err, "failed to apply real migration to real Postgres")

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// TestEncryptionAtRest_RealPostgres is the encryption-at-rest anti-bluff
// proof (queue#4). It:
//  1. Encrypts a known plaintext with REAL AES-256-GCM (simulating the
//     zero-knowledge client).
//  2. Stores it through vault-service's REAL repository layer against a
//     REAL Postgres 17.2 instance.
//  3. Issues a REAL SQL query directly against the stored row and asserts
//     the persisted encrypted_value column does NOT contain the plaintext
//     anywhere — i.e. it is genuine ciphertext, not a pass-through.
//  4. Decrypts the EXACT bytes read back from the database and asserts
//     they equal the original plaintext — proving the ciphertext is
//     real, reversible, and was not corrupted/mutated/truncated by
//     storage.
func TestEncryptionAtRest_RealPostgres(t *testing.T) {
	pool := mustConnectAndMigrate(t)
	repo := repository.New(pool)
	ctx := context.Background()

	const knownPlaintext = "s3cr3t-plaintext-CI-proof-9f3a7b21-do-not-leak-me"
	passphrase := []byte("zero-knowledge-test-passphrase-2026-07-07")

	ciphertextB64, ivB64, saltB64 := encryptAESGCM(t, []byte(knownPlaintext), passphrase)

	// Sanity: the ciphertext produced by the "client" must never equal or
	// contain the plaintext at the encryption stage itself (guards against
	// a broken/no-op cipher upstream of the real assertion below).
	require.NotContains(t, ciphertextB64, knownPlaintext)

	secret := &model.Secret{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		Name:           "encryption-at-rest-proof",
		Type:           model.SecretTypeAPIToken,
		EncryptedValue: ciphertextB64,
		IV:             ivB64,
		Salt:           saltB64,
		Metadata:       map[string]any{"proof": "queue4"},
		Tags:           []string{"encryption-at-rest-proof"},
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	require.NoError(t, repo.CreateSecret(ctx, secret), "CreateSecret via the real repository failed")

	// --- Anti-bluff core assertion: query the ACTUAL DB row directly. ---
	var storedEncryptedValue, storedIV, storedSalt string
	err := pool.QueryRow(ctx,
		`SELECT encrypted_value, iv, salt FROM secrets WHERE id = $1`,
		secret.ID,
	).Scan(&storedEncryptedValue, &storedIV, &storedSalt)
	require.NoError(t, err, "failed to read back the raw stored row via a real SQL query")

	// PROOF 1: the raw stored bytes are exactly the ciphertext we wrote —
	// storage did not corrupt/truncate/mutate them.
	require.Equal(t, ciphertextB64, storedEncryptedValue, "stored encrypted_value diverged from what was written")
	require.Equal(t, ivB64, storedIV)
	require.Equal(t, saltB64, storedSalt)

	// PROOF 2 (the encryption-at-rest anti-bluff assertion): the stored
	// column does NOT contain the plaintext anywhere, in any form.
	require.NotContains(t, storedEncryptedValue, knownPlaintext,
		"CRITICAL: plaintext leaked into the encrypted_value column — encryption-at-rest is BROKEN")
	require.False(t, strings.Contains(storedEncryptedValue, knownPlaintext))
	rawDecoded, decErr := base64.StdEncoding.DecodeString(storedEncryptedValue)
	require.NoError(t, decErr)
	require.NotContains(t, string(rawDecoded), knownPlaintext,
		"CRITICAL: plaintext leaked into the decoded ciphertext bytes — encryption-at-rest is BROKEN")

	// PROOF 3: fetch through the real repository's real SQL SELECT path
	// (GetSecretByID) and confirm it returns the identical ciphertext —
	// the round trip through vault-service's persistence layer is lossless.
	fetched, err := repo.GetSecretByID(ctx, secret.ID)
	require.NoError(t, err)
	require.Equal(t, ciphertextB64, fetched.EncryptedValue)
	require.Equal(t, ivB64, fetched.IV)
	require.Equal(t, saltB64, fetched.Salt)

	// PROOF 4: decrypt the EXACT bytes read back from the real database
	// row and confirm they round-trip to the original plaintext.
	decrypted := decryptAESGCM(t, fetched.EncryptedValue, fetched.IV, fetched.Salt, passphrase)
	require.Equal(t, knownPlaintext, string(decrypted),
		"decrypted ciphertext read back from the real DB row does not match the original plaintext")
}

// TestEncryptionAtRest_WrongKeyFailsToDecrypt proves the ciphertext is
// genuinely bound to its key material — decrypting with the wrong
// passphrase must fail (AES-GCM authentication tag mismatch), not
// silently return garbage or (worse) the plaintext.
func TestEncryptionAtRest_WrongKeyFailsToDecrypt(t *testing.T) {
	pool := mustConnectAndMigrate(t)
	repo := repository.New(pool)
	ctx := context.Background()

	const knownPlaintext = "another-real-secret-value-for-negative-proof"
	rightPassphrase := []byte("correct-passphrase")
	wrongPassphrase := []byte("wrong-passphrase")

	ciphertextB64, ivB64, saltB64 := encryptAESGCM(t, []byte(knownPlaintext), rightPassphrase)

	secret := &model.Secret{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		Name:           "wrong-key-negative-proof",
		Type:           model.SecretTypePassword,
		EncryptedValue: ciphertextB64,
		IV:             ivB64,
		Salt:           saltB64,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	require.NoError(t, repo.CreateSecret(ctx, secret))

	fetched, err := repo.GetSecretByID(ctx, secret.ID)
	require.NoError(t, err)

	// Attempting to decrypt with the wrong key must fail (GCM tag check),
	// proving the stored value is real authenticated ciphertext and not a
	// trivially-reversible encoding.
	ciphertext, _ := base64.StdEncoding.DecodeString(fetched.EncryptedValue)
	nonce, _ := base64.StdEncoding.DecodeString(fetched.IV)
	salt, _ := base64.StdEncoding.DecodeString(fetched.Salt)
	key := deriveKey(wrongPassphrase, salt)
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	_, decErr := gcm.Open(nil, nonce, ciphertext, nil)
	require.Error(t, decErr, "decrypting real ciphertext with the wrong key unexpectedly succeeded")
}

// TestCreateSecretVersion_RealPostgres proves rotation history is also
// stored as real ciphertext against the real secret_versions table.
func TestCreateSecretVersion_RealPostgres(t *testing.T) {
	pool := mustConnectAndMigrate(t)
	repo := repository.New(pool)
	ctx := context.Background()

	const plaintext = "rotated-secret-plaintext-v2"
	passphrase := []byte("rotation-test-passphrase")
	ciphertextB64, ivB64, saltB64 := encryptAESGCM(t, []byte(plaintext), passphrase)

	secret := &model.Secret{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		Name:           "rotation-proof",
		Type:           model.SecretTypeSSHKey,
		EncryptedValue: "original-ciphertext-placeholder",
		IV:             "original-iv",
		Salt:           "original-salt",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	require.NoError(t, repo.CreateSecret(ctx, secret))

	version := &model.SecretVersion{
		ID:             uuid.New(),
		SecretID:       secret.ID,
		EncryptedValue: ciphertextB64,
		IV:             ivB64,
		Salt:           saltB64,
		CreatedBy:      uuid.New(),
		CreatedAt:      time.Now().UTC(),
	}
	require.NoError(t, repo.CreateSecretVersion(ctx, version))

	var storedVal string
	err := pool.QueryRow(ctx, `SELECT encrypted_value FROM secret_versions WHERE id = $1`, version.ID).Scan(&storedVal)
	require.NoError(t, err)
	require.Equal(t, ciphertextB64, storedVal)
	require.NotContains(t, storedVal, plaintext)

	decrypted := decryptAESGCM(t, storedVal, ivB64, saltB64, passphrase)
	require.Equal(t, plaintext, string(decrypted))
}
