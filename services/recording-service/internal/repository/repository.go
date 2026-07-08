package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/helixdevelopment/recording-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles recording data access
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

// CreateRecording creates a new recording
func (r *Repository) CreateRecording(ctx context.Context, recording *model.Recording) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO recordings (id, session_id, host_id, user_id, org_id, file_path, format, status, duration_sec, file_size_bytes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, recording.ID, recording.SessionID, recording.HostID, recording.UserID, recording.OrgID, recording.FilePath, recording.Format, recording.Status, recording.DurationSec, recording.FileSizeBytes)
	if err != nil {
		return fmt.Errorf("failed to create recording: %w", err)
	}
	return nil
}

// GetRecordingByID retrieves a recording by ID
func (r *Repository) GetRecordingByID(ctx context.Context, id uuid.UUID) (*model.Recording, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, session_id, host_id, user_id, org_id, file_path, format, status, duration_sec, file_size_bytes, created_at, updated_at
		FROM recordings WHERE id = $1
	`
	var recording model.Recording
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&recording.ID, &recording.SessionID, &recording.HostID, &recording.UserID, &recording.OrgID, &recording.FilePath, &recording.Format, &recording.Status, &recording.DurationSec, &recording.FileSizeBytes, &recording.CreatedAt, &recording.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("recording not found")
		}
		return nil, err
	}
	return &recording, nil
}

// ListRecordings retrieves recordings with filtering
func (r *Repository) ListRecordings(ctx context.Context, hostID, sessionID uuid.UUID, limit, offset int) ([]*model.Recording, int, error) {
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
	if sessionID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND session_id = $%d", argIdx)
		args = append(args, sessionID)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM recordings WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, session_id, host_id, user_id, org_id, file_path, format, status, duration_sec, file_size_bytes, created_at, updated_at
		FROM recordings WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var recordings []*model.Recording
	for rows.Next() {
		var recording model.Recording
		if err := rows.Scan(
			&recording.ID, &recording.SessionID, &recording.HostID, &recording.UserID, &recording.OrgID, &recording.FilePath, &recording.Format, &recording.Status, &recording.DurationSec, &recording.FileSizeBytes, &recording.CreatedAt, &recording.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		recordings = append(recordings, &recording)
	}
	return recordings, total, rows.Err()
}

// UpdateRecording updates a recording
func (r *Repository) UpdateRecording(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE recordings SET status = $2, duration_sec = $3, file_size_bytes = $4, updated_at = NOW() WHERE id = $1"
	_, err := r.pool.Exec(ctx, query, id, updates["status"], updates["duration_sec"], updates["file_size_bytes"])
	if err != nil {
		return fmt.Errorf("failed to update recording: %w", err)
	}
	return nil
}

// DeleteRecording deletes a recording
func (r *Repository) DeleteRecording(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "DELETE FROM recordings WHERE id = $1"
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete recording: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("recording not found")
	}
	return nil
}
