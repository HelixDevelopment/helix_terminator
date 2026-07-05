package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/ssh-proxy-service/internal/model"
)

// Repository defines the persistence interface for SSH sessions and channels.
type Repository interface {
	Ping(ctx context.Context) error
	CreateSession(ctx context.Context, session *model.SSHSession) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*model.SSHSession, error)
	ListSessions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.SSHSession, error)
	UpdateSessionStatus(ctx context.Context, id uuid.UUID, status model.ConnectionStatus) error
	CreateChannel(ctx context.Context, channel *model.SSHChannel) error
	ListChannels(ctx context.Context, sessionID uuid.UUID) ([]*model.SSHChannel, error)
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Ping verifies connectivity.
func (r *PostgresRepository) Ping(ctx context.Context) error {
	if r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	return r.pool.Ping(ctx)
}

// CreateSession inserts a new SSH session.
func (r *PostgresRepository) CreateSession(ctx context.Context, session *model.SSHSession) error {
	query := `
		INSERT INTO ssh_sessions (id, user_id, host_id, host_address, username, auth_type, connection_status, connected_at, disconnected_at, last_activity_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.HostID, session.HostAddress, session.Username,
		session.AuthType, session.ConnectionStatus, session.ConnectedAt, session.DisconnectedAt,
		session.LastActivityAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSessionByID retrieves a session by its UUID.
func (r *PostgresRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*model.SSHSession, error) {
	query := `
		SELECT id, user_id, host_id, host_address, username, auth_type, connection_status, connected_at, disconnected_at, last_activity_at, created_at
		FROM ssh_sessions
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)

	session := &model.SSHSession{}
	err := row.Scan(
		&session.ID, &session.UserID, &session.HostID, &session.HostAddress, &session.Username,
		&session.AuthType, &session.ConnectionStatus, &session.ConnectedAt, &session.DisconnectedAt,
		&session.LastActivityAt, &session.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return session, nil
}

// ListSessions returns paginated sessions for a user.
func (r *PostgresRepository) ListSessions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.SSHSession, error) {
	query := `
		SELECT id, user_id, host_id, host_address, username, auth_type, connection_status, connected_at, disconnected_at, last_activity_at, created_at
		FROM ssh_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*model.SSHSession
	for rows.Next() {
		session := &model.SSHSession{}
		err := rows.Scan(
			&session.ID, &session.UserID, &session.HostID, &session.HostAddress, &session.Username,
			&session.AuthType, &session.ConnectionStatus, &session.ConnectedAt, &session.DisconnectedAt,
			&session.LastActivityAt, &session.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

// UpdateSessionStatus updates the connection status and timestamps.
func (r *PostgresRepository) UpdateSessionStatus(ctx context.Context, id uuid.UUID, status model.ConnectionStatus) error {
	now := time.Now().UTC()
	var query string
	var args []interface{}

	switch status {
	case model.StatusConnected:
		query = `UPDATE ssh_sessions SET connection_status = $2, connected_at = $3, last_activity_at = $3 WHERE id = $1`
		args = []interface{}{id, status, now}
	case model.StatusDisconnected, model.StatusError:
		query = `UPDATE ssh_sessions SET connection_status = $2, disconnected_at = $3 WHERE id = $1`
		args = []interface{}{id, status, now}
	default:
		query = `UPDATE ssh_sessions SET connection_status = $2, last_activity_at = $3 WHERE id = $1`
		args = []interface{}{id, status, now}
	}

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}
	return nil
}

// CreateChannel inserts a new SSH channel record.
func (r *PostgresRepository) CreateChannel(ctx context.Context, channel *model.SSHChannel) error {
	query := `
		INSERT INTO ssh_channels (id, session_id, channel_type, local_port, remote_port, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		channel.ID, channel.SessionID, channel.ChannelType, channel.LocalPort, channel.RemotePort, channel.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}
	return nil
}

// ListChannels returns all channels for a given session.
func (r *PostgresRepository) ListChannels(ctx context.Context, sessionID uuid.UUID) ([]*model.SSHChannel, error) {
	query := `
		SELECT id, session_id, channel_type, local_port, remote_port, created_at
		FROM ssh_channels
		WHERE session_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	defer rows.Close()

	var channels []*model.SSHChannel
	for rows.Next() {
		channel := &model.SSHChannel{}
		err := rows.Scan(
			&channel.ID, &channel.SessionID, &channel.ChannelType, &channel.LocalPort, &channel.RemotePort, &channel.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

// InMemoryRepository is an in-memory implementation of Repository for testing.
type InMemoryRepository struct {
	mu        sync.RWMutex
	sessions  map[uuid.UUID]*model.SSHSession
	channels  map[uuid.UUID]*model.SSHChannel
}

func (r *InMemoryRepository) Ping(ctx context.Context) error {
	return nil
}

func (r *InMemoryRepository) CreateSession(ctx context.Context, session *model.SSHSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sessions == nil {
		r.sessions = make(map[uuid.UUID]*model.SSHSession)
	}
	r.sessions[session.ID] = session
	return nil
}

func (r *InMemoryRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*model.SSHSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.sessions[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found")
}

func (r *InMemoryRepository) ListSessions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.SSHSession, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*model.SSHSession
	for _, s := range r.sessions {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	// simple offset/limit
	if offset > len(out) {
		return []*model.SSHSession{}, nil
	}
	end := offset + limit
	if end > len(out) || limit == 0 {
		end = len(out)
	}
	return out[offset:end], nil
}

func (r *InMemoryRepository) UpdateSessionStatus(ctx context.Context, id uuid.UUID, status model.ConnectionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.sessions[id]; ok {
		s.ConnectionStatus = status
		now := time.Now().UTC()
		switch status {
		case model.StatusConnected:
			s.ConnectedAt = &now
			s.LastActivityAt = &now
		case model.StatusDisconnected, model.StatusError:
			s.DisconnectedAt = &now
		default:
			s.LastActivityAt = &now
		}
		return nil
	}
	return fmt.Errorf("session not found")
}

func (r *InMemoryRepository) CreateChannel(ctx context.Context, channel *model.SSHChannel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.channels == nil {
		r.channels = make(map[uuid.UUID]*model.SSHChannel)
	}
	r.channels[channel.ID] = channel
	return nil
}

func (r *InMemoryRepository) ListChannels(ctx context.Context, sessionID uuid.UUID) ([]*model.SSHChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*model.SSHChannel
	for _, ch := range r.channels {
		if ch.SessionID == sessionID {
			out = append(out, ch)
		}
	}
	return out, nil
}
