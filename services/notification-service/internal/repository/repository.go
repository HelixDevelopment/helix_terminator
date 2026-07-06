package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/notification-service/internal/model"
)

// Repository handles database operations for notification service
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new repository
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) checkPool() error {
	if r.pool == nil {
		return fmt.Errorf("database not connected")
	}
	return nil
}

// CreateNotification creates a new notification
func (r *Repository) CreateNotification(ctx context.Context, notification *model.Notification) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO notifications (id, user_id, org_id, type, title, message, data, channel, status, read_at, sent_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.pool.Exec(ctx, query,
		notification.ID, notification.UserID, notification.OrgID, notification.Type,
		notification.Title, notification.Message, notification.Data, notification.Channel,
		notification.Status, notification.ReadAt, notification.SentAt,
		notification.CreatedAt, notification.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

// GetNotificationByID retrieves a notification by ID
func (r *Repository) GetNotificationByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, user_id, org_id, type, title, message, data, channel, status, read_at, sent_at, created_at, updated_at
		FROM notifications
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)

	notification := &model.Notification{}
	err := row.Scan(
		&notification.ID, &notification.UserID, &notification.OrgID, &notification.Type,
		&notification.Title, &notification.Message, &notification.Data, &notification.Channel,
		&notification.Status, &notification.ReadAt, &notification.SentAt,
		&notification.CreatedAt, &notification.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}
	return notification, nil
}

// ListNotifications lists notifications with optional filters
func (r *Repository) ListNotifications(ctx context.Context, userID uuid.UUID, orgID *uuid.UUID, status, channel string, limit, offset int) ([]*model.Notification, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}

	// Build count query
	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	countArgs := []interface{}{userID}
	argIdx := 2

	if orgID != nil {
		countQuery += fmt.Sprintf(" AND org_id = $%d", argIdx)
		countArgs = append(countArgs, *orgID)
		argIdx++
	}
	if status != "" {
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		countArgs = append(countArgs, status)
		argIdx++
	}
	if channel != "" {
		countQuery += fmt.Sprintf(" AND channel = $%d", argIdx)
		countArgs = append(countArgs, channel)
		argIdx++
	}

	var total int
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Build list query
	listQuery := `
		SELECT id, user_id, org_id, type, title, message, data, channel, status, read_at, sent_at, created_at, updated_at
		FROM notifications
		WHERE user_id = $1`
	listArgs := []interface{}{userID}
	argIdx = 2

	if orgID != nil {
		listQuery += fmt.Sprintf(" AND org_id = $%d", argIdx)
		listArgs = append(listArgs, *orgID)
		argIdx++
	}
	if status != "" {
		listQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		listArgs = append(listArgs, status)
		argIdx++
	}
	if channel != "" {
		listQuery += fmt.Sprintf(" AND channel = $%d", argIdx)
		listArgs = append(listArgs, channel)
		argIdx++
	}

	listQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*model.Notification
	for rows.Next() {
		notification := &model.Notification{}
		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.OrgID, &notification.Type,
			&notification.Title, &notification.Message, &notification.Data, &notification.Channel,
			&notification.Status, &notification.ReadAt, &notification.SentAt,
			&notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	return notifications, total, nil
}

// MarkRead marks a notification as read
func (r *Repository) MarkRead(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE notifications
		SET read_at = $2, updated_at = $3
		WHERE id = $1
	`
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, query, id, now, now)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	return nil
}

// MarkAllRead marks all notifications for a user as read
func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE notifications
		SET read_at = $2, updated_at = $3
		WHERE user_id = $1 AND read_at IS NULL
	`
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, query, userID, now, now)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return nil
}

// DeleteNotification deletes a notification by ID
func (r *Repository) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `DELETE FROM notifications WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}
	return nil
}

// GetPreference retrieves a user's notification preference for a channel
func (r *Repository) GetPreference(ctx context.Context, userID uuid.UUID, channel string) (*model.NotificationPreference, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT user_id, channel, enabled, types, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1 AND channel = $2
	`
	row := r.pool.QueryRow(ctx, query, userID, channel)

	pref := &model.NotificationPreference{}
	err := row.Scan(
		&pref.UserID, &pref.Channel, &pref.Enabled, &pref.Types,
		&pref.CreatedAt, &pref.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("preference not found")
		}
		return nil, fmt.Errorf("failed to get preference: %w", err)
	}
	return pref, nil
}

// UpdatePreference updates or creates a user's notification preference
func (r *Repository) UpdatePreference(ctx context.Context, pref *model.NotificationPreference) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO notification_preferences (user_id, channel, enabled, types, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, channel)
		DO UPDATE SET enabled = EXCLUDED.enabled, types = EXCLUDED.types, updated_at = EXCLUDED.updated_at
	`
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, query,
		pref.UserID, pref.Channel, pref.Enabled, pref.Types, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to update preference: %w", err)
	}
	return nil
}

// CountUnread counts unread notifications for a user
func (r *Repository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	if err := r.checkPool(); err != nil {
		return 0, err
	}
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`
	var count int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}
	return count, nil
}

// Ping verifies database connectivity
func (r *Repository) Ping(ctx context.Context) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	return r.pool.Ping(ctx)
}
