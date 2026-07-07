package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/vault-service/internal/repository"
)

// TestNew_ReturnsNonNilRepository is a real (non-bluff) construction test.
// The previous version of this file was a stub (`assert.True(t, true)`)
// that asserted nothing about the repository package — replaced per
// queue#4 / §11.4.27 (mocks/stubs/placeholders permitted only in unit
// tests, and even a unit test must assert something real). Real Postgres
// integration tests proving encryption-at-rest live in
// repository_integration_test.go (build tag `integration`).
func TestNew_ReturnsNonNilRepository(t *testing.T) {
	repo := repository.New(nil)
	require.NotNil(t, repo)
}

// TestPing_NilPool proves Ping() genuinely detects a missing database
// connection rather than silently reporting healthy.
func TestPing_NilPool(t *testing.T) {
	repo := repository.New(nil)
	err := repo.Ping(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is nil")
}
