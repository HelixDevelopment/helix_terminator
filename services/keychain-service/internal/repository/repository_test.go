package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/helixdevelopment/keychain-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Real, non-DB unit tests of Repository's construction + fail-closed
// contract. Real-database CRUD + T10 encryption-at-rest proofs already
// live in repository_integration_test.go (build tag "integration",
// against a real Postgres via internal/testinfra); white-box
// SQL-injection-shaped hardening for UpdateItem's query builder already
// lives in repository_internal_test.go. This file's scope is the
// New() constructor contract and the checkPool() nil-pool guard every
// Repository method delegates to - genuine behaviour exercised without
// requiring a live database connection.

// TestNew_EmptyEncryptionKeyFailsClosed_NoDB proves New refuses to
// construct a Repository without an encryption key (§11.4.10
// fail-closed requirement - never a silent plaintext fallback), without
// requiring a live database (the "integration"-tagged
// repository_integration_test.go proves the identical contract again
// end-to-end against a real Postgres-backed Repository; this is the
// fast, always-run unit-level version of the same guarantee).
func TestNew_EmptyEncryptionKeyFailsClosed_NoDB(t *testing.T) {
	repo, err := repository.New(nil, "")
	require.Error(t, err)
	require.Nil(t, repo)
	assert.Contains(t, err.Error(), "encryption key")
}

// TestNew_NonEmptyEncryptionKeySucceedsEvenWithNilPool proves New's
// validation is scoped to the encryption key only - a nil pool (no DB
// wired yet) is accepted by the constructor; connectivity failures
// surface later, from Ping/CRUD calls via checkPool(), not from New.
func TestNew_NonEmptyEncryptionKeySucceedsEvenWithNilPool(t *testing.T) {
	repo, err := repository.New(nil, "a-non-empty-test-key")
	require.NoError(t, err)
	require.NotNil(t, repo)
}

// TestRepository_NilPool_FailsClosedRatherThanPanicking proves every
// pool-dependent method genuinely returns the checkPool() "database not
// connected" error for a Repository constructed with a nil pool -
// rather than nil-pointer-dereferencing on *pgxpool.Pool, which would
// crash the calling goroutine instead of returning a handleable error.
func TestRepository_NilPool_FailsClosedRatherThanPanicking(t *testing.T) {
	repo, err := repository.New(nil, "a-non-empty-test-key")
	require.NoError(t, err)

	pingErr := repo.Ping(context.Background())
	require.Error(t, pingErr, "Ping on a nil-pool Repository must fail closed, not panic")
	assert.Contains(t, pingErr.Error(), "not connected")

	createErr := repo.CreateItem(context.Background(), nil)
	require.Error(t, createErr, "CreateItem on a nil-pool Repository must fail closed, not panic")
	assert.Contains(t, createErr.Error(), "not connected")

	deleteErr := repo.DeleteItem(context.Background(), uuid.Nil)
	require.Error(t, deleteErr, "DeleteItem on a nil-pool Repository must fail closed, not panic")
	assert.Contains(t, deleteErr.Error(), "not connected")
}
