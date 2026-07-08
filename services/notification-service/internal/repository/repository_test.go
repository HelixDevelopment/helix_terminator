package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/model"
	"github.com/helixdevelopment/notification-service/internal/repository"
)

// TestNilRepository_NeverPanics_AlwaysHonestError is a regression guard for
// an availability bug the security-hardening audit surfaced: server.New()
// leaves h.repo as a genuinely NIL *repository.Repository when
// DATABASE_URL is unset (in-memory mode), and every repo method's first
// call was checkPool() dereferencing r.pool on that nil receiver — a
// runtime nil-pointer-dereference panic on the very first request to any
// repo-backed route (e.g. POST /api/v1/notifications) in that mode. A repo
// method MUST degrade to the honest "database not connected" error, never
// crash the request goroutine.
func TestNilRepository_NeverPanics_AlwaysHonestError(t *testing.T) {
	var repo *repository.Repository // deliberately nil, mirrors server.New()'s in-memory-mode state
	ctx := context.Background()

	assert.NotPanics(t, func() {
		err := repo.CreateNotification(ctx, &model.Notification{ID: uuid.New()})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database not connected")
	})

	assert.NotPanics(t, func() {
		_, err := repo.GetNotificationByID(ctx, uuid.New())
		require.Error(t, err)
	})

	assert.NotPanics(t, func() {
		_, _, err := repo.ListNotifications(ctx, uuid.New(), nil, "", "", 20, 0)
		require.Error(t, err)
	})

	assert.NotPanics(t, func() {
		err := repo.Ping(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database not connected")
	})
}

func setupTestDB(t *testing.T) *repository.Repository {
	dbURL := "postgres://postgres:postgres@localhost:5432/notification_test?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}

	ctx := context.Background()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database not available: %v", err)
	}

	// Clean tables
	_, _ = pool.Exec(ctx, "DELETE FROM notification_preferences")
	_, _ = pool.Exec(ctx, "DELETE FROM notifications")

	return repository.New(pool)
}

func TestCreateAndGetNotification(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	notification := &model.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      "info",
		Title:     "Test Title",
		Message:   "Test Message",
		Channel:   "in_app",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := repo.CreateNotification(ctx, notification)
	require.NoError(t, err)

	retrieved, err := repo.GetNotificationByID(ctx, notification.ID)
	require.NoError(t, err)
	assert.Equal(t, notification.ID, retrieved.ID)
	assert.Equal(t, notification.Title, retrieved.Title)
}

func TestGetNotificationNotFound(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.GetNotificationByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListNotifications(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	for i := 0; i < 3; i++ {
		n := &model.Notification{
			ID:        uuid.New(),
			UserID:    userID,
			Type:      "info",
			Title:     "Title",
			Message:   "Message",
			Channel:   "in_app",
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.CreateNotification(ctx, n))
	}

	notifications, total, err := repo.ListNotifications(ctx, userID, nil, "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, notifications, 3)
}

func TestMarkRead(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	notification := &model.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      "info",
		Title:     "Test",
		Message:   "Test",
		Channel:   "in_app",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	require.NoError(t, repo.CreateNotification(ctx, notification))
	require.NoError(t, repo.MarkRead(ctx, notification.ID, userID))

	retrieved, err := repo.GetNotificationByID(ctx, notification.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.ReadAt)
}

// TestMarkRead_CrossUserMutationAffectsZeroRows is the repository-level
// SQL-scoping proof for the T18 follow-up (Constitution §11.4.134
// independent-review finding): MarkRead's UPDATE filters on BOTH id AND
// user_id, so a caller whose userID does not own the target row can
// NEVER flip read_at on it — even calling the repository method
// directly (bypassing the handler's fetch-then-compare entirely)
// affects ZERO rows. Against the pre-fix SQL (`WHERE id = $1` only) this
// exact call would have affected 1 row and silently marked the OTHER
// user's notification read — that pre-fix failure is the RED baseline
// captured for this fix (Constitution §11.4.115): temporarily reverting
// the WHERE clause to `WHERE id = $1` and re-running this test
// reproduces the defect (the assert.Error below fails because the
// mismatched-user call succeeds and read_at flips); restoring the
// `AND user_id = $2` clause makes it GREEN again.
func TestMarkRead_CrossUserMutationAffectsZeroRows(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	owner := uuid.New()
	attacker := uuid.New()
	notification := &model.Notification{
		ID:        uuid.New(),
		UserID:    owner,
		Type:      "info",
		Title:     "Test",
		Message:   "Test",
		Channel:   "in_app",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.CreateNotification(ctx, notification))

	// The attacker's userID does not own this notification — the SQL-level
	// scope MUST reject the mutation with zero rows affected (surfaced as
	// "notification not found"), never silently succeed against the wrong
	// user's row.
	err := repo.MarkRead(ctx, notification.ID, attacker)
	require.Error(t, err, "MarkRead with a mismatched userID must fail — affecting 1 row here would be the T18 IDOR regression")
	assert.Contains(t, err.Error(), "notification not found")

	// Positive control: read_at MUST remain untouched — proving the
	// rejected call above genuinely affected zero rows, not merely that
	// the error string happened to match.
	retrieved, err := repo.GetNotificationByID(ctx, notification.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved.ReadAt, "attacker's cross-user MarkRead call must not have flipped read_at")

	// Sanity: the real owner's MarkRead call still succeeds against the
	// SAME row — proves the fix scopes correctly, not that it locks
	// everyone out.
	require.NoError(t, repo.MarkRead(ctx, notification.ID, owner))
	retrieved, err = repo.GetNotificationByID(ctx, notification.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.ReadAt, "the owning user's MarkRead call must still succeed")
}

func TestMarkAllRead(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	for i := 0; i < 2; i++ {
		n := &model.Notification{
			ID:        uuid.New(),
			UserID:    userID,
			Type:      "info",
			Title:     "Test",
			Message:   "Test",
			Channel:   "in_app",
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.CreateNotification(ctx, n))
	}

	require.NoError(t, repo.MarkAllRead(ctx, userID))

	count, err := repo.CountUnread(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestDeleteNotification(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	notification := &model.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      "info",
		Title:     "Test",
		Message:   "Test",
		Channel:   "in_app",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	require.NoError(t, repo.CreateNotification(ctx, notification))
	require.NoError(t, repo.DeleteNotification(ctx, notification.ID, userID))

	_, err := repo.GetNotificationByID(ctx, notification.ID)
	assert.Error(t, err)
}

// TestDeleteNotification_CrossUserMutationAffectsZeroRows is the
// repository-level SQL-scoping proof for DeleteNotification, mirroring
// TestMarkRead_CrossUserMutationAffectsZeroRows above. Against the
// pre-fix SQL (`DELETE FROM notifications WHERE id = $1` only) this
// exact call would have affected 1 row and silently deleted the OTHER
// user's notification — that pre-fix failure is the RED baseline
// captured for this fix (Constitution §11.4.115): temporarily reverting
// the WHERE clause to `WHERE id = $1` and re-running this test
// reproduces the defect (the row disappears for the mismatched-user
// call); restoring the `AND user_id = $2` clause makes it GREEN again.
func TestDeleteNotification_CrossUserMutationAffectsZeroRows(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	owner := uuid.New()
	attacker := uuid.New()
	notification := &model.Notification{
		ID:        uuid.New(),
		UserID:    owner,
		Type:      "info",
		Title:     "Test",
		Message:   "Test",
		Channel:   "in_app",
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, repo.CreateNotification(ctx, notification))

	err := repo.DeleteNotification(ctx, notification.ID, attacker)
	require.Error(t, err, "DeleteNotification with a mismatched userID must fail — affecting 1 row here would be the T18 IDOR regression")
	assert.Contains(t, err.Error(), "notification not found")

	// Positive control: the row MUST still exist — proving the rejected
	// call above genuinely affected zero rows.
	_, err = repo.GetNotificationByID(ctx, notification.ID)
	require.NoError(t, err, "attacker's cross-user DeleteNotification call must not have deleted the row")

	// Sanity: the real owner's DeleteNotification call still succeeds
	// against the SAME row.
	require.NoError(t, repo.DeleteNotification(ctx, notification.ID, owner))
	_, err = repo.GetNotificationByID(ctx, notification.ID)
	assert.Error(t, err, "the owning user's DeleteNotification call must still succeed")
}

func TestPreference(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	pref := &model.NotificationPreference{
		UserID:    userID,
		Channel:   "email",
		Enabled:   true,
		Types:     []string{"info", "warning"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := repo.UpdatePreference(ctx, pref)
	require.NoError(t, err)

	retrieved, err := repo.GetPreference(ctx, userID, "email")
	require.NoError(t, err)
	assert.Equal(t, pref.Channel, retrieved.Channel)
	assert.Equal(t, pref.Enabled, retrieved.Enabled)
	assert.Equal(t, pref.Types, retrieved.Types)
}

func TestGetPreferenceNotFound(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.GetPreference(ctx, uuid.New(), "email")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCountUnread(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	for i := 0; i < 5; i++ {
		n := &model.Notification{
			ID:        uuid.New(),
			UserID:    userID,
			Type:      "info",
			Title:     "Test",
			Message:   "Test",
			Channel:   "in_app",
			Status:    "pending",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, repo.CreateNotification(ctx, n))
	}

	count, err := repo.CountUnread(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestCheckPool(t *testing.T) {
	repo := repository.New(nil)
	ctx := context.Background()

	err := repo.CreateNotification(ctx, &model.Notification{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}

func TestPing(t *testing.T) {
	dbURL := "postgres://postgres:postgres@localhost:5432/notification_test?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	ctx := context.Background()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database not available: %v", err)
	}

	repo := repository.New(pool)
	require.NoError(t, repo.Ping(ctx))
}
