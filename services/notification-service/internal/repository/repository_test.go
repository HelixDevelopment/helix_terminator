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
	require.NoError(t, repo.MarkRead(ctx, notification.ID))

	retrieved, err := repo.GetNotificationByID(ctx, notification.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.ReadAt)
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
	require.NoError(t, repo.DeleteNotification(ctx, notification.ID))

	_, err := repo.GetNotificationByID(ctx, notification.ID)
	assert.Error(t, err)
}

func TestPreference(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	pref := &model.NotificationPreference{
		UserID:  userID,
		Channel: "email",
		Enabled: true,
		Types:   []string{"info", "warning"},
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
