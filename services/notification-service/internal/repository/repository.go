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

// checkPool reports whether the repository has a usable connection pool. It
// is deliberately safe to call on a NIL *Repository receiver (r == nil) —
// that is exactly the state server.New() produces when DATABASE_URL is
// unset (see server.go: "var repo *repository.Repository" is never
// assigned via repository.New(...) in that path). Before this guard, every
// repository method's first line ("if err := r.checkPool(); err != nil")
// dereferenced r.pool on a nil r, causing a runtime nil-pointer-dereference
// panic on the very first request to any repo-backed route when the
// service starts without a database configured — an availability bug
// surfaced by the notification-service security-hardening audit (a route
// MUST degrade to an honest 503 "database not connected", never crash).
func (r *Repository) checkPool() error {
	if r == nil || r.pool == nil {
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
		INSERT INTO notifications (id, user_id, org_id, type, title, message, data, channel, target, status, read_at, sent_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.pool.Exec(ctx, query,
		notification.ID, notification.UserID, notification.OrgID, notification.Type,
		notification.Title, notification.Message, notification.Data, notification.Channel,
		notification.Target, notification.Status, notification.ReadAt, notification.SentAt,
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
		SELECT id, user_id, org_id, type, title, message, data, channel, target, status, read_at, sent_at, created_at, updated_at
		FROM notifications
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)

	notification := &model.Notification{}
	var target *string
	err := row.Scan(
		&notification.ID, &notification.UserID, &notification.OrgID, &notification.Type,
		&notification.Title, &notification.Message, &notification.Data, &notification.Channel,
		&target, &notification.Status, &notification.ReadAt, &notification.SentAt,
		&notification.CreatedAt, &notification.UpdatedAt,
	)
	if target != nil {
		notification.Target = *target
	}
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
		SELECT id, user_id, org_id, type, title, message, data, channel, target, status, read_at, sent_at, created_at, updated_at
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
		var target *string
		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.OrgID, &notification.Type,
			&notification.Title, &notification.Message, &notification.Data, &notification.Channel,
			&target, &notification.Status, &notification.ReadAt, &notification.SentAt,
			&notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		if target != nil {
			notification.Target = *target
		}
		notifications = append(notifications, notification)
	}

	return notifications, total, nil
}

// MarkRead marks a notification as read, scoped to the owning user (T18
// follow-up, Constitution §11.4.134 independent-review finding). The
// handler already does a fetch-then-compare ownership check before
// calling this method, but that check and this mutation are two
// separate statements — a defense-in-depth backstop belongs at the SQL
// layer too, mirroring billing-service's T12/T14 UpdateSubscription /
// CancelSubscription pattern (WHERE id = $1 AND org_id = $2). The WHERE
// clause here filters on BOTH id AND user_id so the UPDATE itself can
// NEVER affect a row belonging to a different user, even if the
// handler-level check were ever bypassed, raced (TOCTOU), or removed by
// a future refactor. A mismatch (row exists but belongs to a different
// user, or the row does not exist at all) produces the identical
// zero-rows-affected outcome, so the caller can return the same
// "notification not found" response either way — no existence oracle.
func (r *Repository) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE notifications
		SET read_at = $3, updated_at = $4
		WHERE id = $1 AND user_id = $2
	`
	now := time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, id, userID, now, now)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("notification not found")
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

// DeleteNotification deletes a notification by ID, scoped to the owning
// user — same defense-in-depth rationale and billing-service T12/T14
// mirroring as MarkRead above.
func (r *Repository) DeleteNotification(ctx context.Context, id, userID uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `DELETE FROM notifications WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("notification not found")
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
