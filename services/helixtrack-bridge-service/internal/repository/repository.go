package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles HelixTrack-bridge data access
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

// CreateBridge creates a new HelixTrack bridge
func (r *Repository) CreateBridge(ctx context.Context, bridge *model.HelixTrackBridge) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO helixtrack_bridges (id, integration_id, org_id, name, status, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, bridge.ID, bridge.IntegrationID, bridge.OrgID, bridge.Name, bridge.Status, bridge.Config)
	if err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}
	return nil
}

// GetBridgeByID retrieves a bridge by ID
func (r *Repository) GetBridgeByID(ctx context.Context, id uuid.UUID) (*model.HelixTrackBridge, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, integration_id, org_id, name, status, config, created_at, updated_at
		FROM helixtrack_bridges WHERE id = $1
	`
	var bridge model.HelixTrackBridge
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&bridge.ID, &bridge.IntegrationID, &bridge.OrgID, &bridge.Name, &bridge.Status, &bridge.Config, &bridge.CreatedAt, &bridge.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("bridge not found")
		}
		return nil, err
	}
	return &bridge, nil
}

// ListBridges retrieves bridges with filtering
func (r *Repository) ListBridges(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*model.HelixTrackBridge, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	whereClause := "1=1"
	var args []interface{}
	argIdx := 1

	if orgID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND org_id = $%d", argIdx)
		args = append(args, orgID)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM helixtrack_bridges WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, integration_id, org_id, name, status, config, created_at, updated_at
		FROM helixtrack_bridges WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var bridges []*model.HelixTrackBridge
	for rows.Next() {
		var bridge model.HelixTrackBridge
		if err := rows.Scan(
			&bridge.ID, &bridge.IntegrationID, &bridge.OrgID, &bridge.Name, &bridge.Status, &bridge.Config, &bridge.CreatedAt, &bridge.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		bridges = append(bridges, &bridge)
	}
	return bridges, total, rows.Err()
}

// UpdateBridge updates a bridge
func (r *Repository) UpdateBridge(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE helixtrack_bridges SET name = $2, status = $3, config = $4, updated_at = NOW() WHERE id = $1"
	_, err := r.pool.Exec(ctx, query, id, updates["name"], updates["status"], updates["config"])
	if err != nil {
		return fmt.Errorf("failed to update bridge: %w", err)
	}
	return nil
}

// DeleteBridge deletes a bridge
func (r *Repository) DeleteBridge(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "DELETE FROM helixtrack_bridges WHERE id = $1"
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete bridge: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("bridge not found")
	}
	return nil
}
