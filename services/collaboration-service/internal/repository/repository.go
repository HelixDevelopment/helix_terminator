package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/helixdevelopment/collaboration-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles collaboration data access
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

// CreateSession creates a new collaboration session
func (r *Repository) CreateSession(ctx context.Context, session *model.CollaborationSession) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO collaboration_sessions (id, host_id, created_by, org_id, name, status, participants, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, session.ID, session.HostID, session.CreatedBy, session.OrgID, session.Name, session.Status, session.Participants)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSessionByID retrieves a session by ID
func (r *Repository) GetSessionByID(ctx context.Context, id uuid.UUID) (*model.CollaborationSession, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, host_id, created_by, org_id, name, status, participants, created_at, updated_at, ended_at
		FROM collaboration_sessions WHERE id = $1 AND ended_at IS NULL
	`
	var session model.CollaborationSession
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&session.ID, &session.HostID, &session.CreatedBy, &session.OrgID, &session.Name, &session.Status, &session.Participants, &session.CreatedAt, &session.UpdatedAt, &session.EndedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}
	return &session, nil
}

// ListSessions retrieves sessions with filtering
func (r *Repository) ListSessions(ctx context.Context, hostID, createdBy uuid.UUID, limit, offset int) ([]*model.CollaborationSession, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	whereClause := "ended_at IS NULL"
	var args []interface{}
	argIdx := 1

	if hostID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND host_id = $%d", argIdx)
		args = append(args, hostID)
		argIdx++
	}
	if createdBy != uuid.Nil {
		whereClause += fmt.Sprintf(" AND created_by = $%d", argIdx)
		args = append(args, createdBy)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM collaboration_sessions WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, host_id, created_by, org_id, name, status, participants, created_at, updated_at
		FROM collaboration_sessions WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []*model.CollaborationSession
	for rows.Next() {
		var session model.CollaborationSession
		if err := rows.Scan(
			&session.ID, &session.HostID, &session.CreatedBy, &session.OrgID, &session.Name, &session.Status, &session.Participants, &session.CreatedAt, &session.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, &session)
	}
	return sessions, total, rows.Err()
}

// JoinSession adds a participant to a session
func (r *Repository) JoinSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE collaboration_sessions
		SET participants = array_append(participants, $2), updated_at = NOW()
		WHERE id = $1 AND ended_at IS NULL AND NOT ($2 = ANY(participants))
	`
	_, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to join session: %w", err)
	}
	return nil
}

// LeaveSession removes a participant from a session
func (r *Repository) LeaveSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE collaboration_sessions
		SET participants = array_remove(participants, $2), updated_at = NOW()
		WHERE id = $1 AND ended_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to leave session: %w", err)
	}
	return nil
}

// EndSession marks a session as ended
func (r *Repository) EndSession(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE collaboration_sessions SET status = 'ended', ended_at = NOW(), updated_at = NOW() WHERE id = $1 AND ended_at IS NULL"
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found")
	}
	return nil
}
