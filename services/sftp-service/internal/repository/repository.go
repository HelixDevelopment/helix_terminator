package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/helixdevelopment/sftp-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles SFTP data access
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

// CreateSession creates a new SFTP session
func (r *Repository) CreateSession(ctx context.Context, session *model.SFTPSession) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO sftp_sessions (id, host_id, user_id, remote_path, local_path, direction, status, bytes_transferred, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, session.ID, session.HostID, session.UserID, session.RemotePath, session.LocalPath, session.Direction, session.Status, session.BytesTransferred)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSessionByID retrieves a session by ID
func (r *Repository) GetSessionByID(ctx context.Context, id uuid.UUID) (*model.SFTPSession, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, host_id, user_id, remote_path, local_path, direction, status, bytes_transferred, created_at, updated_at, completed_at
		FROM sftp_sessions WHERE id = $1
	`
	var session model.SFTPSession
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&session.ID, &session.HostID, &session.UserID, &session.RemotePath, &session.LocalPath, &session.Direction, &session.Status, &session.BytesTransferred, &session.CreatedAt, &session.UpdatedAt, &session.CompletedAt,
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
func (r *Repository) ListSessions(ctx context.Context, hostID uuid.UUID, status string, limit, offset int) ([]*model.SFTPSession, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	whereClause := "1=1"
	var args []interface{}
	argIdx := 1

	if hostID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND host_id = $%d", argIdx)
		args = append(args, hostID)
		argIdx++
	}
	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sftp_sessions WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, host_id, user_id, remote_path, local_path, direction, status, bytes_transferred, created_at, updated_at, completed_at
		FROM sftp_sessions WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []*model.SFTPSession
	for rows.Next() {
		var session model.SFTPSession
		if err := rows.Scan(
			&session.ID, &session.HostID, &session.UserID, &session.RemotePath, &session.LocalPath, &session.Direction, &session.Status, &session.BytesTransferred, &session.CreatedAt, &session.UpdatedAt, &session.CompletedAt,
		); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, &session)
	}
	return sessions, total, rows.Err()
}

// UpdateSession updates a session
func (r *Repository) UpdateSession(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE sftp_sessions SET status = $2, bytes_transferred = $3, completed_at = $4, updated_at = NOW() WHERE id = $1"
	_, err := r.pool.Exec(ctx, query, id, updates["status"], updates["bytes_transferred"], updates["completed_at"])
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// DeleteSession deletes a session
func (r *Repository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "DELETE FROM sftp_sessions WHERE id = $1"
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("session not found")
	}
	return nil
}
