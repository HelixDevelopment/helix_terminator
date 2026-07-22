package repository_test

import (
	"context"
	"testing"

	"github.com/helixdevelopment/gateway-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE (§11.4.124 dead/unwired-code investigation, captured before this
// test was written): internal/repository is genuinely UNWIRED
// scaffolding, same status as internal/model (see model_test.go's
// identical note). `git log --follow` shows a single MVP scaffold
// commit; nothing in gateway-service imports this package
// (main.go wires internal/server directly, which owns its own
// upstream-health-check state - not this Repository interface).
// PostgresRepository.Ping is a literal stub (`return errors.New("not
// implemented")`, see repository.go) - that IS the real, current,
// shipped behaviour. This test proves that real behaviour honestly
// (so a future accidental "fix" that silently starts returning nil
// without wiring a real DB connection is caught), rather than
// papering over it with a tautological assertion or silently deleting
// the unwired package per §11.4.124/§11.4.122.
func TestNewPostgresRepository_ReturnsNonNil(t *testing.T) {
	repo := repository.NewPostgresRepository()
	require.NotNil(t, repo)

	// Compile-time + runtime proof PostgresRepository genuinely
	// implements the declared Repository interface.
	var _ repository.Repository = repo
}

// TestPostgresRepository_Ping_IsAnHonestNotImplementedStub proves the
// CURRENT real behaviour: Ping does not silently return success without
// ever touching a database - it fails closed with a real, non-nil error
// (never a fabricated "connected" result), which is the correct interim
// contract for a repository with no injected *pgxpool.Pool per the
// "TODO: inject *sql.DB or *pgxpool.Pool" comment on the struct.
func TestPostgresRepository_Ping_IsAnHonestNotImplementedStub(t *testing.T) {
	repo := repository.NewPostgresRepository()
	err := repo.Ping(context.Background())
	assert.Error(t, err, "Ping must not silently claim success while genuinely unimplemented (no DB connection is ever injected)")
	assert.Contains(t, err.Error(), "not implemented")
}
