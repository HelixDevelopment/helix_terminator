//go:build integration

package repository_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/auth-service/internal/crypto"
	"github.com/helixdevelopment/auth-service/internal/model"
	"github.com/helixdevelopment/auth-service/internal/repository"
	"github.com/helixdevelopment/auth-service/internal/testinfra"
)

// newTestRepository boots a real, disposable PostgreSQL 17.2 container
// (via rootless podman), applies the real embedded migrations against
// it (migrations/001_init.up.sql, through migrations.Run - the exact
// runner server.New calls at process startup), and returns a
// Repository wired to a real pgxpool connected to it. No mocks, no
// stubs, no in-memory fakes anywhere in this file - per §11.4.27.
func newTestRepository(t *testing.T) *repository.Repository {
	t.Helper()
	dbURL := testinfra.StartPostgres(t)

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New failed: %v", err)
	}
	t.Cleanup(pool.Close)

	return repository.New(pool)
}

func newTestUser(t *testing.T, email, plainPassword string) *model.User {
	t.Helper()
	hasher := crypto.NewPasswordHasher()
	hash, err := hasher.HashPassword(plainPassword)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	return &model.User{
		ID:            uuid.New(),
		Email:         email,
		PasswordHash:  hash,
		DisplayName:   "Repository Test User",
		Role:          "user",
		MFAEnabled:    false,
		EmailVerified: false,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
}

// TestCreateAndGetUser_PasswordStoredHashedNotPlaintext is the direct
// DB-layer proof required by queue#4: create a user through the real
// repository, then read the row back through a REAL SELECT against the
// REAL running PostgreSQL instance, and assert the persisted
// password_hash column is a genuine Argon2id hash - never the
// plaintext password.
func TestCreateAndGetUser_PasswordStoredHashedNotPlaintext(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	plainPassword := "correct-horse-battery-staple-42"
	email := "hashed-row-proof@example.com"
	user := newTestUser(t, email, plainPassword)

	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Independent real-row read-back (by email, then again by ID) -
	// both go through real SQL SELECTs against the real database.
	byEmail, err := repo.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if byEmail.ID != user.ID {
		t.Fatalf("GetUserByEmail returned ID %s, want %s", byEmail.ID, user.ID)
	}

	byID, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	for name, got := range map[string]*model.User{"GetUserByEmail": byEmail, "GetUserByID": byID} {
		if got.PasswordHash == plainPassword {
			t.Fatalf("%s: password_hash column stores the PLAINTEXT password verbatim - critical security defect", name)
		}
		if !strings.HasPrefix(got.PasswordHash, "$argon2id$") {
			t.Fatalf("%s: password_hash = %q, want a real $argon2id$... hash", name, got.PasswordHash)
		}
		if strings.Contains(got.PasswordHash, plainPassword) {
			t.Fatalf("%s: password_hash %q contains the plaintext password as a substring", name, got.PasswordHash)
		}

		// The stored hash must itself verify against the original
		// password via the real Argon2id verifier - proving it is a
		// correct, usable hash of that exact password, not garbage.
		hasher := crypto.NewPasswordHasher()
		ok, err := hasher.VerifyPassword(plainPassword, got.PasswordHash)
		if err != nil {
			t.Fatalf("%s: VerifyPassword against stored hash errored: %v", name, err)
		}
		if !ok {
			t.Fatalf("%s: stored hash does not verify against the original plaintext password", name)
		}

		ok, err = hasher.VerifyPassword("wrong-password-entirely", got.PasswordHash)
		if err != nil {
			t.Fatalf("%s: VerifyPassword(wrong password) errored: %v", name, err)
		}
		if ok {
			t.Fatalf("%s: stored hash incorrectly verified a wrong password", name)
		}
	}
}

func TestEmailExists(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	email := "exists-check@example.com"
	exists, err := repo.EmailExists(ctx, email)
	if err != nil {
		t.Fatalf("EmailExists (before create) failed: %v", err)
	}
	if exists {
		t.Fatal("EmailExists reported true before the user was ever created")
	}

	user := newTestUser(t, email, "irrelevant-password-value-1")
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	exists, err = repo.EmailExists(ctx, email)
	if err != nil {
		t.Fatalf("EmailExists (after create) failed: %v", err)
	}
	if !exists {
		t.Fatal("EmailExists reported false after the user was created")
	}
}

func TestSessionLifecycle_CreateLookupUpdateRevoke(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	user := newTestUser(t, "session-lifecycle@example.com", "some-long-password-value")
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	accessHash := crypto.HashToken("access-token-v1")
	refreshHash := crypto.HashToken("refresh-token-v1")
	session := &model.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		AccessTokenHash:  accessHash,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        time.Now().UTC().Add(time.Hour),
		LastActiveAt:     time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	got, err := repo.GetSessionByTokenHash(ctx, accessHash)
	if err != nil {
		t.Fatalf("GetSessionByTokenHash(accessHash) failed: %v", err)
	}
	if got.ID != session.ID {
		t.Fatalf("GetSessionByTokenHash returned session %s, want %s", got.ID, session.ID)
	}

	// Simulate a token refresh: rebind the session's access-token hash
	// to a new value; the old hash must stop resolving and the new one
	// must resolve.
	newAccessHash := crypto.HashToken("access-token-v2-after-refresh")
	if err := repo.UpdateSessionAccessTokenHash(ctx, session.ID, newAccessHash); err != nil {
		t.Fatalf("UpdateSessionAccessTokenHash failed: %v", err)
	}
	if _, err := repo.GetSessionByTokenHash(ctx, accessHash); err == nil {
		t.Fatal("GetSessionByTokenHash(old accessHash) succeeded after refresh rebind, want not-found")
	}
	got, err = repo.GetSessionByTokenHash(ctx, newAccessHash)
	if err != nil {
		t.Fatalf("GetSessionByTokenHash(newAccessHash) failed: %v", err)
	}
	if got.ID != session.ID {
		t.Fatalf("GetSessionByTokenHash(newAccessHash) returned session %s, want %s", got.ID, session.ID)
	}

	// Revoke and prove the session is genuinely gone from the
	// active-session lookup - this is the DB-layer half of the
	// "replayed-after-logout token rejected" security property.
	if err := repo.RevokeSession(ctx, session.ID); err != nil {
		t.Fatalf("RevokeSession failed: %v", err)
	}
	if _, err := repo.GetSessionByTokenHash(ctx, newAccessHash); err == nil {
		t.Fatal("GetSessionByTokenHash succeeded for a revoked session, want not-found")
	}

	// A revoked session's access-token hash can no longer be rebound
	// (e.g. by a stale refresh token race) - RowsAffected()==0 must
	// surface as an error.
	if err := repo.UpdateSessionAccessTokenHash(ctx, session.ID, crypto.HashToken("post-revoke-attempt")); err == nil {
		t.Fatal("UpdateSessionAccessTokenHash succeeded against an already-revoked session, want error")
	}
}

func TestRevokeAllUserSessions(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	user := newTestUser(t, "revoke-all@example.com", "some-long-password-value")
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	var hashes []string
	for i := 0; i < 3; i++ {
		hash := crypto.HashToken(uuid.NewString())
		hashes = append(hashes, hash)
		session := &model.Session{
			ID:               uuid.New(),
			UserID:           user.ID,
			AccessTokenHash:  hash,
			RefreshTokenHash: crypto.HashToken(uuid.NewString()),
			ExpiresAt:        time.Now().UTC().Add(time.Hour),
			LastActiveAt:     time.Now().UTC(),
			CreatedAt:        time.Now().UTC(),
		}
		if err := repo.CreateSession(ctx, session); err != nil {
			t.Fatalf("CreateSession[%d] failed: %v", i, err)
		}
	}

	if err := repo.RevokeAllUserSessions(ctx, user.ID); err != nil {
		t.Fatalf("RevokeAllUserSessions failed: %v", err)
	}

	for i, hash := range hashes {
		if _, err := repo.GetSessionByTokenHash(ctx, hash); err == nil {
			t.Fatalf("session[%d] still resolves as active after RevokeAllUserSessions", i)
		}
	}
}

func TestIncrementAndResetFailedLogins_LocksAccountAfterThreshold(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	user := newTestUser(t, "lockout@example.com", "some-long-password-value")
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// The lockout query locks once failed_login_attempts >= 4 BEFORE
	// this increment (i.e. becomes locked on the 5th failure). Drive
	// five real failures against the real row.
	for i := 0; i < 5; i++ {
		if err := repo.IncrementFailedLogins(ctx, user.ID); err != nil {
			t.Fatalf("IncrementFailedLogins[%d] failed: %v", i, err)
		}
	}

	locked, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID after failures failed: %v", err)
	}
	if locked.FailedLogins < 5 {
		t.Fatalf("FailedLogins = %d, want >= 5", locked.FailedLogins)
	}
	if locked.LockedUntil == nil || !locked.LockedUntil.After(time.Now().UTC()) {
		t.Fatalf("LockedUntil = %v, want a future timestamp after 5 failed logins", locked.LockedUntil)
	}

	if err := repo.ResetFailedLogins(ctx, user.ID); err != nil {
		t.Fatalf("ResetFailedLogins failed: %v", err)
	}
	reset, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID after reset failed: %v", err)
	}
	if reset.FailedLogins != 0 {
		t.Fatalf("FailedLogins after reset = %d, want 0", reset.FailedLogins)
	}
	if reset.LockedUntil != nil {
		t.Fatalf("LockedUntil after reset = %v, want nil", reset.LockedUntil)
	}
}
