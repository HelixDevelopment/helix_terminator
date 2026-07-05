package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/host-service/internal/model"
)

// Repository handles database operations for host service.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new repository.
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

// CreateHost creates a new host.
func (r *Repository) CreateHost(ctx context.Context, host *model.Host) error {
	query := `
		INSERT INTO hosts (
			id, user_id, org_id, name, hostname, port, username, auth_type,
			vault_secret_id, connection_params, tags, connection_status, last_error, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.pool.Exec(ctx, query,
		host.ID, host.UserID, host.OrgID, host.Name, host.Hostname, host.Port, host.Username, host.AuthType,
		host.VaultSecretID, host.ConnectionParams, host.Tags, host.ConnectionStatus, host.LastError,
		host.CreatedAt, host.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create host: %w", err)
	}
	return nil
}

// GetHostByID retrieves a host by ID.
func (r *Repository) GetHostByID(ctx context.Context, id uuid.UUID) (*model.Host, error) {
	query := `
		SELECT id, user_id, org_id, name, hostname, port, username, auth_type,
		       vault_secret_id, connection_params, tags, last_connected_at,
		       connection_status, last_error, created_at, updated_at, deleted_at
		FROM hosts
		WHERE id = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, id)

	host := &model.Host{}
	err := row.Scan(
		&host.ID, &host.UserID, &host.OrgID, &host.Name, &host.Hostname, &host.Port, &host.Username, &host.AuthType,
		&host.VaultSecretID, &host.ConnectionParams, &host.Tags, &host.LastConnectedAt,
		&host.ConnectionStatus, &host.LastError, &host.CreatedAt, &host.UpdatedAt, &host.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("host not found")
		}
		return nil, fmt.Errorf("failed to get host: %w", err)
	}
	return host, nil
}

// ListHosts lists hosts with optional filtering.
func (r *Repository) ListHosts(ctx context.Context, userID, orgID uuid.UUID, tags []string, status string, limit, offset int) ([]*model.Host, error) {
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if userID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, userID)
		argIdx++
	}
	if orgID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("org_id = $%d", argIdx))
		args = append(args, orgID)
		argIdx++
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("connection_status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if len(tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, tags)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, org_id, name, hostname, port, username, auth_type,
		       vault_secret_id, connection_params, tags, last_connected_at,
		       connection_status, last_error, created_at, updated_at, deleted_at
		FROM hosts
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, joinConditions(conditions), argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list hosts: %w", err)
	}
	defer rows.Close()

	var hosts []*model.Host
	for rows.Next() {
		host := &model.Host{}
		err := rows.Scan(
			&host.ID, &host.UserID, &host.OrgID, &host.Name, &host.Hostname, &host.Port, &host.Username, &host.AuthType,
			&host.VaultSecretID, &host.ConnectionParams, &host.Tags, &host.LastConnectedAt,
			&host.ConnectionStatus, &host.LastError, &host.CreatedAt, &host.UpdatedAt, &host.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan host: %w", err)
		}
		hosts = append(hosts, host)
	}

	return hosts, nil
}

// UpdateHost updates an existing host.
func (r *Repository) UpdateHost(ctx context.Context, host *model.Host) error {
	query := `
		UPDATE hosts
		SET name = $2, hostname = $3, port = $4, username = $5, auth_type = $6,
		    vault_secret_id = $7, connection_params = $8, tags = $9, updated_at = $10
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query,
		host.ID, host.Name, host.Hostname, host.Port, host.Username, host.AuthType,
		host.VaultSecretID, host.ConnectionParams, host.Tags, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to update host: %w", err)
	}
	return nil
}

// DeleteHost performs a soft delete on a host.
func (r *Repository) DeleteHost(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE hosts
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to delete host: %w", err)
	}
	return nil
}

// UpdateConnectionStatus updates the connection status and last error of a host.
func (r *Repository) UpdateConnectionStatus(ctx context.Context, id uuid.UUID, status model.ConnectionStatus, errorMsg string) error {
	query := `
		UPDATE hosts
		SET connection_status = $2, last_error = $3, last_connected_at = $4, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, query, id, status, errorMsg, now)
	if err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}
	return nil
}

// CreateConnectionLog creates a new connection log entry.
func (r *Repository) CreateConnectionLog(ctx context.Context, log *model.HostConnectionLog) error {
	query := `
		INSERT INTO host_connection_logs (id, host_id, event_type, details, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, log.ID, log.HostID, log.EventType, log.Details, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create connection log: %w", err)
	}
	return nil
}

// GetConnectionLogs retrieves connection logs for a host.
func (r *Repository) GetConnectionLogs(ctx context.Context, hostID uuid.UUID, limit int) ([]*model.HostConnectionLog, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, host_id, event_type, details, created_at
		FROM host_connection_logs
		WHERE host_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, hostID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.HostConnectionLog
	for rows.Next() {
		logEntry := &model.HostConnectionLog{}
		err := rows.Scan(
			&logEntry.ID, &logEntry.HostID, &logEntry.EventType, &logEntry.Details, &logEntry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection log: %w", err)
		}
		logs = append(logs, logEntry)
	}

	return logs, nil
}

// CountHosts counts hosts for a user/org.
func (r *Repository) CountHosts(ctx context.Context, userID, orgID uuid.UUID) (int, error) {
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if userID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, userID)
		argIdx++
	}
	if orgID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("org_id = $%d", argIdx))
		args = append(args, orgID)
		argIdx++
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM hosts WHERE %s`, joinConditions(conditions))
	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count hosts: %w", err)
	}
	return count, nil
}

func joinConditions(conditions []string) string {
	result := ""
	for i, c := range conditions {
		if i > 0 {
			result += " AND "
		}
		result += c
	}
	return result
}
