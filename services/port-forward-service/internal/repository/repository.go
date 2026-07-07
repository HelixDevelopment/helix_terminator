package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/helixdevelopment/port-forward-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles port-forward data access
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

// CreateForward creates a new port forward
func (r *Repository) CreateForward(ctx context.Context, forward *model.PortForward) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO port_forwards (id, host_id, forward_type, local_port, remote_port, remote_host, protocol, bind_address, ssh_host, ssh_port, ssh_username, auth_type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query,
		forward.ID, forward.HostID, forward.ForwardType, forward.LocalPort, forward.RemotePort, forward.RemoteHost,
		forward.Protocol, forward.BindAddress, forward.SSHHost, forward.SSHPort, forward.SSHUsername, forward.AuthType, forward.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to create forward: %w", err)
	}
	return nil
}

// GetForwardByID retrieves a forward by ID
func (r *Repository) GetForwardByID(ctx context.Context, id uuid.UUID) (*model.PortForward, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, host_id, forward_type, local_port, remote_port, remote_host, protocol, bind_address, ssh_host, ssh_port, ssh_username, auth_type, status, created_at, updated_at, deleted_at
		FROM port_forwards WHERE id = $1 AND deleted_at IS NULL
	`
	var forward model.PortForward
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&forward.ID, &forward.HostID, &forward.ForwardType, &forward.LocalPort, &forward.RemotePort, &forward.RemoteHost,
		&forward.Protocol, &forward.BindAddress, &forward.SSHHost, &forward.SSHPort, &forward.SSHUsername, &forward.AuthType,
		&forward.Status, &forward.CreatedAt, &forward.UpdatedAt, &forward.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("forward not found")
		}
		return nil, err
	}
	return &forward, nil
}

// ListForwards retrieves forwards with filtering
func (r *Repository) ListForwards(ctx context.Context, hostID uuid.UUID, limit, offset int) ([]*model.PortForward, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	whereClause := "deleted_at IS NULL"
	var args []interface{}
	argIdx := 1

	if hostID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND host_id = $%d", argIdx)
		args = append(args, hostID)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM port_forwards WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, host_id, forward_type, local_port, remote_port, remote_host, protocol, bind_address, ssh_host, ssh_port, ssh_username, auth_type, status, created_at, updated_at
		FROM port_forwards WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var forwards []*model.PortForward
	for rows.Next() {
		var forward model.PortForward
		if err := rows.Scan(
			&forward.ID, &forward.HostID, &forward.ForwardType, &forward.LocalPort, &forward.RemotePort, &forward.RemoteHost,
			&forward.Protocol, &forward.BindAddress, &forward.SSHHost, &forward.SSHPort, &forward.SSHUsername, &forward.AuthType,
			&forward.Status, &forward.CreatedAt, &forward.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		forwards = append(forwards, &forward)
	}
	return forwards, total, rows.Err()
}

// UpdateForward updates a forward's editable metadata (not the real tunnel
// lifecycle status — use UpdateStatus for that).
func (r *Repository) UpdateForward(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE port_forwards SET local_port = $2, remote_port = $3, remote_host = $4, protocol = $5, status = $6, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL"
	_, err := r.pool.Exec(ctx, query, id, updates["local_port"], updates["remote_port"], updates["remote_host"], updates["protocol"], updates["status"])
	if err != nil {
		return fmt.Errorf("failed to update forward: %w", err)
	}
	return nil
}

// UpdateStatus sets a forward's status to reflect REAL tunnel state (pending
// / active / stopped / error). It never touches the forward's other
// metadata columns.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE port_forwards SET status = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL"
	result, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update forward status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("forward not found")
	}
	return nil
}

// DeleteForward soft-deletes a forward
func (r *Repository) DeleteForward(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE port_forwards SET status = 'deleted', deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL"
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete forward: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("forward not found")
	}
	return nil
}
