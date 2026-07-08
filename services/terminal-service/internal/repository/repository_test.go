package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/terminal-service/internal/repository"
)

// TestNewWithNilPoolDegradesHonestly is a real, falsifiable regression guard
// for repository.New / repository.Repository.Ping's degraded-mode contract:
// when constructed with a nil pool (the exact shape server.New produces when
// DATABASE_URL is unset or the connection attempt failed, see
// internal/server/server.go), Ping MUST return an honest "database not
// connected" error rather than panicking on a nil pgxpool.Pool dereference or
// silently reporting success. This is RED-capable: removing the nil-pool
// guard in Repository.Ping would either panic (nil pointer dereference on
// r.pool.Ping) or, if a bogus success were substituted instead, would make
// this assertion fail because it requires a genuine error to be returned.
func TestNewWithNilPoolDegradesHonestly(t *testing.T) {
	repo := repository.New(nil)
	require.NotNil(t, repo, "New must always return a non-nil Repository, even in degraded (no-pool) mode")

	err := repo.Ping(context.Background())
	require.Error(t, err, "Ping on a repository with no database pool must return an honest error, never a silent nil (fabricated success)")
	assert.Contains(t, err.Error(), "database not connected",
		"Ping's degraded-mode error must clearly identify the cause (no pool), not an unrelated or generic message")
}

// TestListSessionsWithNilPoolDegradesHonestly proves the same degraded-mode
// contract holds for ListSessions specifically (a distinct method with its
// own explicit nil-pool guard in repository.go, separate from Ping's). RED-
// capable: if that guard were removed, this call would panic on a nil
// *pgxpool.Pool method call instead of returning the expected error, failing
// the test either way.
func TestListSessionsWithNilPoolDegradesHonestly(t *testing.T) {
	repo := repository.New(nil)

	sessions, err := repo.ListSessions(context.Background(), "", "", "", 10, 0)
	require.Error(t, err, "ListSessions with no database pool must return an honest error")
	assert.Nil(t, sessions, "no sessions should be returned when the database is not connected")
	assert.Contains(t, err.Error(), "database not connected")
}
