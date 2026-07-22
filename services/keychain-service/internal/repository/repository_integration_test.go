//go:build integration

// Package repository_test — REAL integration tests against a real
// PostgreSQL instance (T10, §11.4.10 / §11.4.107 encryption-at-rest
// anti-bluff proof). Excluded from the default `go test ./...` run
// (build tag `integration`); requires a live Postgres reachable via
// DATABASE_URL, e.g.:
//
//	export DATABASE_URL="postgres://postgres:postgres@127.0.0.1:55491/keychain_service_test?sslmode=disable"
//	go test -tags integration ./internal/repository/...
//
// Unlike vault-service (a zero-knowledge store where the CLIENT
// encrypts before ever calling the repository), keychain-service
// receives plaintext private_key/passphrase over its own API and is
// responsible for encrypting them at rest itself — mirroring
// pki-service's server-side AES-256-GCM + PBKDF2 pattern
// (internal/crypto), keyed from KEYCHAIN_ENCRYPTION_KEY.
package repository_test

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/keychain-service/internal/model"
	"github.com/helixdevelopment/keychain-service/internal/repository"
)

// testEncKey is a passphrase supplied only by the test, standing in for the
// KEYCHAIN_ENCRYPTION_KEY environment variable a real deployment would set
// (§11.4.10 — never hardcoded in production source; this file is a test).
const testEncKey = "T10-test-only-encryption-key-2026-07-08-do-not-use-in-prod"

// mustConnectAndMigrate connects to the real Postgres pointed at by
// DATABASE_URL and applies keychain-service's real migration
// (migrations/001_init.up.sql) idempotently. Skips (does not fail) when
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

	// §11.4.102 root-cause FACT: this previously read "001_init.sql",
	// which has never existed on disk (the real migration files are
	// "001_init.up.sql" / "001_init.down.sql", see
	// services/keychain-service/migrations/) - so every test in this
	// file has always failed with "no such file or directory" before
	// ever reaching Postgres, whenever DATABASE_URL was actually set.
	// The T10 encryption-at-rest + T13 UpdateItem hardening real-Postgres
	// proofs this file documents have therefore never genuinely run.
	// Found via real-Postgres integration testing while covering T2.
	migrationPath := filepath.Join("..", "..", "migrations", "001_init.up.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err, "failed to read migrations/001_init.up.sql")

	_, err = pool.Exec(ctx, string(migrationSQL))
	require.NoError(t, err, "failed to apply real migration to real Postgres")

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// TestEncryptionAtRest_RealPostgres is the encryption-at-rest anti-bluff
// proof (T10). It:
//  1. Creates a keychain item with a KNOWN plaintext private_key +
//     passphrase through keychain-service's REAL repository layer against
//     a REAL Postgres instance.
//  2. Issues a REAL SQL query directly against the stored row and asserts
//     the persisted private_key / passphrase columns do NOT contain the
//     plaintext anywhere — i.e. genuine ciphertext, not a pass-through.
//  3. Fetches the item back through the real repository's real SQL SELECT
//     path (GetItemByID) and asserts the returned plaintext is IDENTICAL
//     to what was originally submitted — the round trip is lossless.
func TestEncryptionAtRest_RealPostgres(t *testing.T) {
	pool := mustConnectAndMigrate(t)
	repo, err := repository.New(pool, testEncKey)
	require.NoError(t, err, "repository.New with a non-empty encryption key must succeed")
	ctx := context.Background()

	const knownPrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nT10-PROOF-PLAINTEXT-DO-NOT-LEAK-9f3a7b21\n-----END OPENSSH PRIVATE KEY-----"
	const knownPassphrase = "s3cr3t-passphrase-T10-proof-do-not-leak"

	item := &model.KeychainItem{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Name:       "encryption-at-rest-proof",
		Type:       model.KeyTypeSSH,
		PublicKey:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI-not-secret",
		PrivateKey: knownPrivateKey,
		Passphrase: knownPassphrase,
		Metadata:   map[string]interface{}{"proof": "t10"},
		Tags:       []string{"encryption-at-rest-proof"},
	}

	require.NoError(t, repo.CreateItem(ctx, item), "CreateItem via the real repository failed")

	// --- Anti-bluff core assertion: query the ACTUAL DB row directly. ---
	var storedPrivateKey, storedPassphrase string
	err = pool.QueryRow(ctx,
		`SELECT private_key, passphrase FROM keychain_items WHERE id = $1`,
		item.ID,
	).Scan(&storedPrivateKey, &storedPassphrase)
	require.NoError(t, err, "failed to read back the raw stored row via a real SQL query")

	// PROOF (the encryption-at-rest anti-bluff assertion): the stored
	// columns do NOT equal or contain the plaintext anywhere, in any form.
	require.NotEqual(t, knownPrivateKey, storedPrivateKey,
		"CRITICAL: private_key stored VERBATIM as plaintext — encryption-at-rest is BROKEN")
	require.NotContains(t, storedPrivateKey, knownPrivateKey,
		"CRITICAL: plaintext private_key leaked into the stored column — encryption-at-rest is BROKEN")
	require.NotEqual(t, knownPassphrase, storedPassphrase,
		"CRITICAL: passphrase stored VERBATIM as plaintext — encryption-at-rest is BROKEN")
	require.NotContains(t, storedPassphrase, knownPassphrase,
		"CRITICAL: plaintext passphrase leaked into the stored column — encryption-at-rest is BROKEN")

	// PROOF: not merely a different string encoding of the same plaintext
	// (e.g. base64/hex/rot13) — decode the stored value and confirm the
	// plaintext is absent from the raw decoded bytes too.
	rawPrivateKeyBytes, decErr := base64.StdEncoding.DecodeString(storedPrivateKey)
	require.NoError(t, decErr, "stored private_key is not valid base64 ciphertext")
	require.NotContains(t, string(rawPrivateKeyBytes), knownPrivateKey,
		"CRITICAL: plaintext private_key leaked into the decoded ciphertext bytes — encryption-at-rest is BROKEN")
	rawPassphraseBytes, decErr := base64.StdEncoding.DecodeString(storedPassphrase)
	require.NoError(t, decErr, "stored passphrase is not valid base64 ciphertext")
	require.NotContains(t, string(rawPassphraseBytes), knownPassphrase,
		"CRITICAL: plaintext passphrase leaked into the decoded ciphertext bytes — encryption-at-rest is BROKEN")

	// Fetch through the real repository's real SQL SELECT path
	// (GetItemByID) and confirm the round trip through keychain-service's
	// persistence layer is lossless — decrypt-on-read returns the exact
	// original plaintext.
	fetched, err := repo.GetItemByID(ctx, item.ID)
	require.NoError(t, err)
	require.Equal(t, knownPrivateKey, fetched.PrivateKey,
		"decrypted private_key read back from the real DB row does not match the original plaintext")
	require.Equal(t, knownPassphrase, fetched.Passphrase,
		"decrypted passphrase read back from the real DB row does not match the original plaintext")
}

// TestEncryptionAtRest_WrongKeyFailsToDecrypt proves the ciphertext is
// genuinely bound to its key material — decrypting with the wrong
// encryption key must fail (AES-GCM authentication tag mismatch), not
// silently return garbage or (worse) the plaintext.
func TestEncryptionAtRest_WrongKeyFailsToDecrypt(t *testing.T) {
	pool := mustConnectAndMigrate(t)
	repo, err := repository.New(pool, testEncKey)
	require.NoError(t, err)
	ctx := context.Background()

	item := &model.KeychainItem{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Name:       "wrong-key-negative-proof",
		Type:       model.KeyTypeAPIKey,
		PrivateKey: "another-real-secret-value-for-negative-proof",
		Passphrase: "",
		Tags:       []string{},
	}
	require.NoError(t, repo.CreateItem(ctx, item))

	wrongRepo, err := repository.New(pool, "a-completely-different-wrong-key")
	require.NoError(t, err)

	_, err = wrongRepo.GetItemByID(ctx, item.ID)
	require.Error(t, err, "decrypting real ciphertext with the wrong key unexpectedly succeeded")
}

// TestNew_EmptyEncryptionKeyFailsClosed proves the repository refuses to
// be constructed without an encryption key — no silent plaintext
// fallback (§11.4.10 fail-closed requirement).
func TestNew_EmptyEncryptionKeyFailsClosed(t *testing.T) {
	pool := mustConnectAndMigrate(t)

	repo, err := repository.New(pool, "")
	require.Error(t, err, "repository.New with an empty encryption key must fail closed")
	require.Nil(t, repo)
}

// TestUpdateItem_RealPostgres_AllowedFieldsPersist_SecretsUntouched is the
// T13 real-database integration proof: it exercises the REAL UpdateItem
// path (allow-list hardened, this task) against a REAL Postgres — create
// an item (with a real private_key + passphrase, so T10 encryption-at-
// rest is engaged), update it through the real repository, and verify
// (1) the allowed-field update actually persisted, (2) the T10-encrypted
// secret columns are BYTE-FOR-BYTE unchanged by an update that never
// touches them, and still decrypt correctly, and (3) an UpdateItem call
// naming a disallowed field (including an attempt at "private_key") is
// REJECTED by the real repository, and the raw DB row proves nothing
// changed as a result.
func TestUpdateItem_RealPostgres_AllowedFieldsPersist_SecretsUntouched(t *testing.T) {
	pool := mustConnectAndMigrate(t)
	repo, err := repository.New(pool, testEncKey)
	require.NoError(t, err)
	ctx := context.Background()

	const originalPrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nT13-UPDATEITEM-PROOF-PLAINTEXT\n-----END OPENSSH PRIVATE KEY-----"
	const originalPassphrase = "t13-update-item-passphrase-do-not-leak"

	item := &model.KeychainItem{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Name:       "t13-update-item-real-pg",
		Type:       model.KeyTypeSSH,
		PublicKey:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI-original",
		PrivateKey: originalPrivateKey,
		Passphrase: originalPassphrase,
		Metadata:   map[string]interface{}{"phase": "before-update"},
		Tags:       []string{"t13-before"},
	}
	require.NoError(t, repo.CreateItem(ctx, item), "CreateItem via the real repository failed")

	// Capture the raw, still-encrypted DB columns BEFORE the update so we
	// can prove byte-for-byte they are untouched afterward.
	var storedPrivateKeyBefore, storedPassphraseBefore string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT private_key, passphrase FROM keychain_items WHERE id = $1`, item.ID,
	).Scan(&storedPrivateKeyBefore, &storedPassphraseBefore))

	// --- (1) Allowed-field update via the REAL, hardened UpdateItem path.
	err = repo.UpdateItem(ctx, item.ID, map[string]interface{}{
		"name": "t13-update-item-real-pg-RENAMED",
		"tags": []string{"t13-after"},
	})
	require.NoError(t, err, "UpdateItem with allow-listed fields must succeed against the real DB")

	updated, err := repo.GetItemByID(ctx, item.ID)
	require.NoError(t, err)
	require.Equal(t, "t13-update-item-real-pg-RENAMED", updated.Name,
		"the allowed-field update did not persist through the real UpdateItem path")
	require.Equal(t, []string{"t13-after"}, updated.Tags)

	// --- (2) T10 preservation: secrets untouched, still decrypt correctly.
	var storedPrivateKeyAfter, storedPassphraseAfter string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT private_key, passphrase FROM keychain_items WHERE id = $1`, item.ID,
	).Scan(&storedPrivateKeyAfter, &storedPassphraseAfter))
	require.Equal(t, storedPrivateKeyBefore, storedPrivateKeyAfter,
		"CRITICAL: UpdateItem altered the T10-encrypted private_key column bytes even though the update never named it")
	require.Equal(t, storedPassphraseBefore, storedPassphraseAfter,
		"CRITICAL: UpdateItem altered the T10-encrypted passphrase column bytes even though the update never named it")
	require.Equal(t, originalPrivateKey, updated.PrivateKey,
		"decrypted private_key changed after an UpdateItem call that never touched it")
	require.Equal(t, originalPassphrase, updated.Passphrase,
		"decrypted passphrase changed after an UpdateItem call that never touched it")

	// --- (3) The real repository REJECTS a disallowed-field update — even
	// an attempt to overwrite the T10-encrypted "private_key" column —
	// and the raw row proves nothing changed as a result.
	err = repo.UpdateItem(ctx, item.ID, map[string]interface{}{
		"private_key": "attacker-supplied-plaintext-should-never-land",
	})
	require.Error(t, err, "UpdateItem must reject an attempt to set the T10-encrypted private_key column via the real DB path")

	var storedPrivateKeyAfterRejectedAttempt string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT private_key FROM keychain_items WHERE id = $1`, item.ID,
	).Scan(&storedPrivateKeyAfterRejectedAttempt))
	require.Equal(t, storedPrivateKeyBefore, storedPrivateKeyAfterRejectedAttempt,
		"CRITICAL: a rejected UpdateItem call still mutated the private_key column in the real DB")

	err = repo.UpdateItem(ctx, item.ID, map[string]interface{}{
		"name = 'ignored', passphrase": "attacker-supplied-plaintext",
	})
	require.Error(t, err, "UpdateItem must reject the SQL-column-injection-shaped malicious key via the real DB path")
}
