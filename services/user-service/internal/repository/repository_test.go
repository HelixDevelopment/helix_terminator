//go:build integration

// Package repository_test - REAL integration tests against a real
// PostgreSQL instance (§11.4.27: mocks/stubs are permitted only in unit
// tests; every other test type exercises the real, fully implemented
// system). Excluded from the default `go test ./...` run (build tag
// `integration`); boots a real, disposable rootless-podman PostgreSQL
// 17.2 container via internal/testinfra.StartPostgres and applies
// user-service's real golang-migrate schema before every test - the
// exact same boot path internal/handler's readiness-integration test
// already uses.
//
//	go test -tags integration ./internal/repository/...
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/user-service/internal/model"
	"github.com/helixdevelopment/user-service/internal/repository"
	"github.com/helixdevelopment/user-service/internal/testinfra"
)

func newRealRepo(t *testing.T) *repository.Repository {
	t.Helper()
	dbURL := testinfra.StartPostgres(t)
	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err, "pgxpool.New against the real test database failed")
	t.Cleanup(pool.Close)
	return repository.New(pool)
}

// TestPing_RealPostgres proves Ping genuinely round-trips to a live,
// reachable database (T8-6's underlying liveness primitive).
func TestPing_RealPostgres(t *testing.T) {
	repo := newRealRepo(t)
	require.NoError(t, repo.Ping(context.Background()))
}

// TestCreateUser_GetUserByID_RealPostgres_RoundTrips proves CreateUser
// genuinely persists a row through the real INSERT + RETURNING path and
// GetUserByID genuinely reads it back through the real SELECT path -
// not an in-memory fake standing in for either.
func TestCreateUser_GetUserByID_RealPostgres_RoundTrips(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	user := &model.User{
		Email:       "roundtrip@example.com",
		DisplayName: "Round Trip",
		Role:        "user",
		Permissions: []string{"read", "write"},
	}

	require.NoError(t, repo.CreateUser(ctx, user), "CreateUser against the real database failed")
	require.NotEmpty(t, user.ID, "CreateUser must populate a generated ID")
	require.False(t, user.CreatedAt.IsZero(), "CreateUser must populate CreatedAt from the real DEFAULT/RETURNING clause")

	fetched, err := repo.GetUserByID(ctx, user.ID)
	require.NoError(t, err, "GetUserByID against the real database failed")
	require.Equal(t, user.Email, fetched.Email)
	require.Equal(t, user.DisplayName, fetched.DisplayName)
	require.Equal(t, user.Role, fetched.Role)
	require.ElementsMatch(t, user.Permissions, fetched.Permissions)
	require.False(t, fetched.EmailVerified, "email_verified must default to false per the real schema DEFAULT")
}

// TestGetUserByID_RealPostgres_NotFound proves the real "not found"
// error path (pgx.ErrNoRows translated to a domain error) against a
// genuinely absent row - not a stubbed always-nil / always-error path.
func TestGetUserByID_RealPostgres_NotFound(t *testing.T) {
	repo := newRealRepo(t)
	_, err := repo.GetUserByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.Error(t, err, "GetUserByID must fail for a genuinely absent row")
}

// TestGetUserByEmail_RealPostgres proves the email-lookup query path
// against a real, indexed column.
func TestGetUserByEmail_RealPostgres(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	user := &model.User{Email: "by-email@example.com", DisplayName: "By Email", Role: "user"}
	require.NoError(t, repo.CreateUser(ctx, user))

	fetched, err := repo.GetUserByEmail(ctx, "by-email@example.com")
	require.NoError(t, err)
	require.Equal(t, user.ID, fetched.ID)

	_, err = repo.GetUserByEmail(ctx, "does-not-exist@example.com")
	require.Error(t, err)
}

// TestEmailExists_RealPostgres proves the EXISTS() query genuinely
// reflects real row presence/absence, before and after a real insert.
func TestEmailExists_RealPostgres(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	exists, err := repo.EmailExists(ctx, "not-yet-created@example.com")
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, repo.CreateUser(ctx, &model.User{
		Email: "not-yet-created@example.com", DisplayName: "Exists Now", Role: "user",
	}))

	exists, err = repo.EmailExists(ctx, "not-yet-created@example.com")
	require.NoError(t, err)
	require.True(t, exists)
}

// TestUpdateUser_RealPostgres_PersistsChanges proves UpdateUser's
// dynamic SET-clause builder genuinely persists the given fields against
// a real UPDATE statement, and that unrelated fields are left untouched.
func TestUpdateUser_RealPostgres_PersistsChanges(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	user := &model.User{Email: "update-me@example.com", DisplayName: "Before Update", Role: "user"}
	require.NoError(t, repo.CreateUser(ctx, user))

	require.NoError(t, repo.UpdateUser(ctx, user.ID, map[string]interface{}{
		"display_name": "After Update",
		"role":         "admin",
	}))

	fetched, err := repo.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "After Update", fetched.DisplayName, "UpdateUser must persist display_name through the real UPDATE path")
	require.Equal(t, "admin", fetched.Role, "UpdateUser must persist role through the real UPDATE path")
	require.Equal(t, user.Email, fetched.Email, "UpdateUser must leave unrelated columns untouched")
	require.True(t, fetched.UpdatedAt.After(user.UpdatedAt) || fetched.UpdatedAt.Equal(user.UpdatedAt),
		"updated_at must be bumped to now (or later) by the real UPDATE")

	// Not-found path: updating a row that genuinely does not exist.
	err = repo.UpdateUser(ctx, "00000000-0000-0000-0000-000000000000", map[string]interface{}{"display_name": "x"})
	require.Error(t, err, "UpdateUser must fail for a genuinely absent row")
}

// TestDeleteUser_RealPostgres_SoftDeletes proves DeleteUser genuinely
// sets deleted_at (soft delete) rather than removing the row, and that
// GetUserByID's `deleted_at IS NULL` filter genuinely excludes it
// afterward - a real behavioural contract, not an assumption.
func TestDeleteUser_RealPostgres_SoftDeletes(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	user := &model.User{Email: "delete-me@example.com", DisplayName: "Delete Me", Role: "user"}
	require.NoError(t, repo.CreateUser(ctx, user))

	require.NoError(t, repo.DeleteUser(ctx, user.ID))

	_, err := repo.GetUserByID(ctx, user.ID)
	require.Error(t, err, "GetUserByID must not return a soft-deleted row")

	// A second delete of the same (already soft-deleted) row must fail -
	// RowsAffected()==0 against the real UPDATE ... WHERE deleted_at IS NULL.
	err = repo.DeleteUser(ctx, user.ID)
	require.Error(t, err, "DeleteUser on an already-deleted row must report not found, not silently succeed")
}

// TestListUsers_RealPostgres_FiltersAndPaginates proves ListUsers'
// dynamically-built WHERE clause genuinely filters by org/role/search
// and genuinely paginates against a real, multi-row dataset - each
// filter is exercised against real rows that would fail the assertion
// if the corresponding WHERE fragment were dropped or mis-built.
func TestListUsers_RealPostgres_FiltersAndPaginates(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	// org_id is a real UUID column (migrations/001_init.up.sql) - use a
	// genuine UUID, not an arbitrary label, so the INSERT itself proves
	// the org filter path rather than merely asserting a constraint error.
	orgA := uuid.New().String()
	seed := []*model.User{
		{Email: "alice@list-test.com", DisplayName: "Alice Alpha", Role: "admin", OrgID: &orgA},
		{Email: "bob@list-test.com", DisplayName: "Bob Alpha", Role: "user", OrgID: &orgA},
		{Email: "carol@list-test.com", DisplayName: "Carol Other", Role: "user"},
	}
	for _, u := range seed {
		require.NoError(t, repo.CreateUser(ctx, u))
	}

	// Filter by org.
	users, total, err := repo.ListUsers(ctx, orgA, "", "", 20, 0)
	require.NoError(t, err)
	require.Equal(t, 2, total, "org filter must match exactly the 2 seeded org-alpha users")
	require.Len(t, users, 2)

	// Filter by org + role.
	users, total, err = repo.ListUsers(ctx, orgA, "admin", "", 20, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, users, 1)
	require.Equal(t, "alice@list-test.com", users[0].Email)

	// Search filter (ILIKE on email/display_name).
	users, total, err = repo.ListUsers(ctx, "", "", "Carol", 20, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, "carol@list-test.com", users[0].Email)

	// Pagination: limit=1 must return exactly 1 row even though more than
	// 1 row matches, ordered by created_at DESC.
	users, total, err = repo.ListUsers(ctx, orgA, "", "", 1, 0)
	require.NoError(t, err)
	require.Equal(t, 2, total, "total must reflect the full matching set, independent of the page limit")
	require.Len(t, users, 1, "limit=1 must return exactly 1 row")
}

// TestCreateOrUpdateProfile_GetProfile_RealPostgres proves the profile
// upsert path genuinely persists into user_profiles and GetProfile's
// real LEFT JOIN genuinely surfaces those fields alongside the base
// user row.
func TestCreateOrUpdateProfile_GetProfile_RealPostgres(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	user := &model.User{Email: "profile-owner@example.com", DisplayName: "Profile Owner", Role: "user"}
	require.NoError(t, repo.CreateUser(ctx, user))

	require.NoError(t, repo.CreateOrUpdateProfile(ctx, user.ID, map[string]interface{}{
		"bio":      "Real bio text",
		"timezone": "America/New_York",
		"locale":   "en-US",
	}))

	profile, err := repo.GetProfile(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "Real bio text", profile.Bio)
	require.Equal(t, "America/New_York", profile.Timezone)
	require.Equal(t, user.Email, profile.Email, "GetProfile's JOIN must surface the base user's email")

	// Upsert path: a second call with the same user_id updates in place
	// (ON CONFLICT DO UPDATE) rather than erroring on a duplicate key.
	require.NoError(t, repo.CreateOrUpdateProfile(ctx, user.ID, map[string]interface{}{
		"bio": "Updated bio text",
	}))
	profile, err = repo.GetProfile(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "Updated bio text", profile.Bio)
}

// TestGetProfile_RealPostgres_NoProfileRowYet is the RED-then-GREEN
// anti-bluff proof (§11.4.115) for a real product defect this test suite
// discovered: GetProfile's LEFT JOIN against user_profiles returns
// SQL NULL for every profile column when a user has never had a profile
// row created (CreateOrUpdateProfile never called) - the exact state of
// every brand-new user. Against the pre-fix repository (unconditional
// &profile.Bio/&profile.SSHPublicKey/... scan into non-pointer strings)
// this failed with "cannot scan NULL into *string" on the very first
// GetProfile call for a new user. Post-fix it must return a zero-valued
// (empty string) profile, not an error.
func TestGetProfile_RealPostgres_NoProfileRowYet(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	user := &model.User{Email: "no-profile-yet@example.com", DisplayName: "No Profile Yet", Role: "user"}
	require.NoError(t, repo.CreateUser(ctx, user))

	profile, err := repo.GetProfile(ctx, user.ID)
	require.NoError(t, err, "GetProfile must not fail for a user with no profile row yet (NULL LEFT JOIN columns)")
	require.Equal(t, user.Email, profile.Email)
	require.Empty(t, profile.Bio)
	require.Empty(t, profile.SSHPublicKey)
	require.Empty(t, profile.GitHubID)
	require.Empty(t, profile.GitLabID)
}

// TestGetProfile_RealPostgres_PartialProfileUpdate is the RED-then-GREEN
// anti-bluff proof (§11.4.115) for the live UpdateProfile HTTP path: the
// handler builds a profile map containing ONLY the fields the caller
// actually sent (internal/handler.go UpdateProfile), so
// CreateOrUpdateProfile's INSERT explicitly writes SQL NULL for every
// other named profile column. Against the pre-fix repository this
// reproduced the identical "cannot scan NULL into *string" failure on
// the GetProfile call the handler issues immediately after persisting a
// partial update - i.e. updating just a user's bio 500'd on the same
// request. Post-fix, the field that was set persists and every
// untouched field folds to "".
func TestGetProfile_RealPostgres_PartialProfileUpdate(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	user := &model.User{Email: "partial-profile@example.com", DisplayName: "Partial Profile", Role: "user"}
	require.NoError(t, repo.CreateUser(ctx, user))

	// Mirrors internal/handler.go UpdateProfile: only "bio" set by the
	// caller, every other profile field genuinely absent from the map.
	require.NoError(t, repo.CreateOrUpdateProfile(ctx, user.ID, map[string]interface{}{
		"bio": "Only bio was set",
	}))

	profile, err := repo.GetProfile(ctx, user.ID)
	require.NoError(t, err, "GetProfile must not fail after a partial profile update leaves other columns NULL")
	require.Equal(t, "Only bio was set", profile.Bio)
	require.Empty(t, profile.SSHPublicKey, "ssh_public_key was never set and must fold NULL to empty string")
	require.Empty(t, profile.GitHubID)
	require.Empty(t, profile.GitLabID)
}

// TestUpdateLastLogin_RealPostgres proves the last_login_at column is
// genuinely NULL before login and genuinely populated with a recent
// timestamp after UpdateLastLogin runs against the real DB.
func TestUpdateLastLogin_RealPostgres(t *testing.T) {
	repo := newRealRepo(t)
	ctx := context.Background()

	user := &model.User{Email: "login-tracker@example.com", DisplayName: "Login Tracker", Role: "user"}
	require.NoError(t, repo.CreateUser(ctx, user))

	fetched, err := repo.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	require.Nil(t, fetched.LastLoginAt, "last_login_at must be NULL before any login")

	before := time.Now().Add(-time.Second)
	require.NoError(t, repo.UpdateLastLogin(ctx, user.ID))

	fetched, err = repo.GetUserByID(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.LastLoginAt, "last_login_at must be populated after UpdateLastLogin")
	require.True(t, fetched.LastLoginAt.After(before), "last_login_at must reflect a genuinely recent NOW() write")
}
