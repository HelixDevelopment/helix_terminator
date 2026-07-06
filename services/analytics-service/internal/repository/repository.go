package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/analytics-service/internal/model"
)

// Repository handles analytics data access
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new Repository
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) checkPool() error {
	if r.pool == nil {
		return fmt.Errorf("database not connected")
	}
	return nil
}

// Ping verifies connectivity
func (r *Repository) Ping(ctx context.Context) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	return r.pool.Ping(ctx)
}

// CreateEvent creates a new analytics event
func (r *Repository) CreateEvent(ctx context.Context, event *model.AnalyticsEvent) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO analytics_events (id, org_id, user_id, host_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.pool.Exec(ctx, query, event.ID, event.OrgID, event.UserID, event.HostID, event.EventType, event.Payload)
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}
	return nil
}

// GetEventByID retrieves an event by ID
func (r *Repository) GetEventByID(ctx context.Context, id uuid.UUID) (*model.AnalyticsEvent, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, org_id, user_id, host_id, event_type, payload, created_at
		FROM analytics_events WHERE id = $1
	`
	var event model.AnalyticsEvent
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&event.ID, &event.OrgID, &event.UserID, &event.HostID, &event.EventType, &event.Payload, &event.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("event not found")
		}
		return nil, err
	}
	return &event, nil
}

// ListEvents retrieves events with filtering
func (r *Repository) ListEvents(ctx context.Context, orgID *uuid.UUID, eventType string, limit, offset int) ([]*model.AnalyticsEvent, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	whereClause := "1=1"
	var args []interface{}
	argIdx := 1

	if orgID != nil {
		whereClause += fmt.Sprintf(" AND org_id = $%d", argIdx)
		args = append(args, orgID)
		argIdx++
	}
	if eventType != "" {
		whereClause += fmt.Sprintf(" AND event_type = $%d", argIdx)
		args = append(args, eventType)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM analytics_events WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, org_id, user_id, host_id, event_type, payload, created_at
		FROM analytics_events WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*model.AnalyticsEvent
	for rows.Next() {
		var event model.AnalyticsEvent
		if err := rows.Scan(
			&event.ID, &event.OrgID, &event.UserID, &event.HostID, &event.EventType, &event.Payload, &event.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		events = append(events, &event)
	}
	return events, total, rows.Err()
}

// CountByEventType returns event counts grouped by type
func (r *Repository) CountByEventType(ctx context.Context, orgID *uuid.UUID) ([]*model.AnalyticsSummary, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	whereClause := "1=1"
	var args []interface{}
	if orgID != nil {
		whereClause += " AND org_id = $1"
		args = append(args, orgID)
	}
	query := fmt.Sprintf(`
		SELECT event_type, COUNT(*) as count
		FROM analytics_events
		WHERE %s
		GROUP BY event_type
	`, whereClause)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*model.AnalyticsSummary
	for rows.Next() {
		var summary model.AnalyticsSummary
		if err := rows.Scan(&summary.EventType, &summary.Count); err != nil {
			return nil, err
		}
		summaries = append(summaries, &summary)
	}
	return summaries, rows.Err()
}
