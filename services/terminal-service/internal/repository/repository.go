package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/terminal-service/internal/model"
)

// Repository handles database operations for terminal service.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new Repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Ping verifies connectivity.
func (r *Repository) Ping(ctx context.Context) error {
	if r.pool == nil {
		return fmt.Errorf("database not connected")
	}
	return r.pool.Ping(ctx)
}

// CreateSession creates a new terminal session.
func (r *Repository) CreateSession(ctx context.Context, session *model.TerminalSession) error {
	query := `
		INSERT INTO terminal_sessions (id, user_id, host_id, ssh_session_id, status, started_at, ended_at, duration_ms, cols, rows, shell_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.HostID, session.SSHSessionID, session.Status,
		session.StartedAt, session.EndedAt, session.DurationMs, session.Cols, session.Rows,
		session.ShellType, session.CreatedAt, session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSessionByID retrieves a terminal session by ID.
func (r *Repository) GetSessionByID(ctx context.Context, id uuid.UUID) (*model.TerminalSession, error) {
	query := `
		SELECT id, user_id, host_id, ssh_session_id, status, started_at, ended_at, duration_ms, cols, rows, shell_type, created_at, updated_at
		FROM terminal_sessions
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)

	session := &model.TerminalSession{}
	err := row.Scan(
		&session.ID, &session.UserID, &session.HostID, &session.SSHSessionID, &session.Status,
		&session.StartedAt, &session.EndedAt, &session.DurationMs, &session.Cols, &session.Rows,
		&session.ShellType, &session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return session, nil
}

// ListSessions retrieves terminal sessions with optional filtering.
func (r *Repository) ListSessions(ctx context.Context, userID, hostID, status string, limit, offset int) ([]*model.TerminalSession, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("database not connected")
	}

	conditions := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if userID != "" {
		uid, err := uuid.Parse(userID)
		if err != nil {
			return nil, fmt.Errorf("invalid user_id: %w", err)
		}
		conditions += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, uid)
		argIdx++
	}
	if hostID != "" {
		hid, err := uuid.Parse(hostID)
		if err != nil {
			return nil, fmt.Errorf("invalid host_id: %w", err)
		}
		conditions += fmt.Sprintf(" AND host_id = $%d", argIdx)
		args = append(args, hid)
		argIdx++
	}
	if status != "" {
		conditions += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, host_id, ssh_session_id, status, started_at, ended_at, duration_ms, cols, rows, shell_type, created_at, updated_at
		FROM terminal_sessions
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, conditions, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*model.TerminalSession
	for rows.Next() {
		session := &model.TerminalSession{}
		err := rows.Scan(
			&session.ID, &session.UserID, &session.HostID, &session.SSHSessionID, &session.Status,
			&session.StartedAt, &session.EndedAt, &session.DurationMs, &session.Cols, &session.Rows,
			&session.ShellType, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// UpdateSession updates a terminal session.
func (r *Repository) UpdateSession(ctx context.Context, session *model.TerminalSession) error {
	query := `
		UPDATE terminal_sessions
		SET status = $2, cols = $3, rows = $4, shell_type = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query,
		session.ID, session.Status, session.Cols, session.Rows, session.ShellType, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// CloseSession closes a terminal session and sets its duration.
func (r *Repository) CloseSession(ctx context.Context, id uuid.UUID, durationMs int) error {
	now := time.Now().UTC()
	query := `
		UPDATE terminal_sessions
		SET status = 'closed', ended_at = $2, duration_ms = $3, updated_at = $4
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, now, durationMs, now)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}
	return nil
}

// CreateOutput inserts a terminal output chunk.
func (r *Repository) CreateOutput(ctx context.Context, output *model.TerminalOutput) error {
	query := `
		INSERT INTO terminal_outputs (id, session_id, output_type, data, timestamp, sequence_num)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		output.ID, output.SessionID, output.OutputType, output.Data, output.Timestamp, output.SequenceNum,
	)
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}
	return nil
}

// GetOutputs retrieves terminal output chunks for a session.
func (r *Repository) GetOutputs(ctx context.Context, sessionID uuid.UUID, afterSequence, limit int) ([]*model.TerminalOutput, error) {
	query := `
		SELECT id, session_id, output_type, data, timestamp, sequence_num
		FROM terminal_outputs
		WHERE session_id = $1 AND sequence_num > $2
		ORDER BY sequence_num ASC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, query, sessionID, afterSequence, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get outputs: %w", err)
	}
	defer rows.Close()

	var outputs []*model.TerminalOutput
	for rows.Next() {
		output := &model.TerminalOutput{}
		err := rows.Scan(
			&output.ID, &output.SessionID, &output.OutputType, &output.Data, &output.Timestamp, &output.SequenceNum,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan output: %w", err)
		}
		outputs = append(outputs, output)
	}

	return outputs, nil
}

// CreateRecording inserts a terminal recording record.
func (r *Repository) CreateRecording(ctx context.Context, recording *model.TerminalRecording) error {
	query := `
		INSERT INTO terminal_recordings (id, session_id, format, file_path, file_size, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		recording.ID, recording.SessionID, recording.Format, recording.FilePath,
		recording.FileSize, recording.DurationMs, recording.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create recording: %w", err)
	}
	return nil
}

// GetRecordingBySessionID retrieves a recording by session ID.
func (r *Repository) GetRecordingBySessionID(ctx context.Context, sessionID uuid.UUID) (*model.TerminalRecording, error) {
	query := `
		SELECT id, session_id, format, file_path, file_size, duration_ms, created_at
		FROM terminal_recordings
		WHERE session_id = $1
	`
	row := r.pool.QueryRow(ctx, query, sessionID)

	recording := &model.TerminalRecording{}
	err := row.Scan(
		&recording.ID, &recording.SessionID, &recording.Format, &recording.FilePath,
		&recording.FileSize, &recording.DurationMs, &recording.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("recording not found")
		}
		return nil, fmt.Errorf("failed to get recording: %w", err)
	}
	return recording, nil
}

// CountSessions returns the total count of sessions for a user.
func (r *Repository) CountSessions(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM terminal_sessions WHERE user_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	return count, nil
}
