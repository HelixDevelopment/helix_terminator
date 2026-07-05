package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/config-service/internal/model"
)

// Repository handles database operations for config service.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// checkPool verifies the pool is initialized.
func (r *Repository) checkPool() error {
	if r.pool == nil {
		return fmt.Errorf("database not connected")
	}
	return nil
}

// CreateConfig creates a new config entry.
func (r *Repository) CreateConfig(ctx context.Context, config *model.Config) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO configs (id, scope, scope_id, key, value, value_type, description, is_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		config.ID, config.Scope, config.ScopeID, config.Key, config.Value,
		config.ValueType, config.Description, config.IsSecret, config.CreatedAt, config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}
	return nil
}

// GetConfigByID retrieves a config by its ID.
func (r *Repository) GetConfigByID(ctx context.Context, id uuid.UUID) (*model.Config, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, scope, scope_id, key, value, value_type, description, is_secret, created_at, updated_at, deleted_at
		FROM configs
		WHERE id = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, id)
	config := &model.Config{}
	err := row.Scan(
		&config.ID, &config.Scope, &config.ScopeID, &config.Key, &config.Value,
		&config.ValueType, &config.Description, &config.IsSecret,
		&config.CreatedAt, &config.UpdatedAt, &config.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("config not found")
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return config, nil
}

// GetConfigByKey retrieves a config by scope, scopeID, and key.
func (r *Repository) GetConfigByKey(ctx context.Context, scope string, scopeID *uuid.UUID, key string) (*model.Config, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, scope, scope_id, key, value, value_type, description, is_secret, created_at, updated_at, deleted_at
		FROM configs
		WHERE scope = $1 AND scope_id IS NOT DISTINCT FROM $2 AND key = $3 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, scope, scopeID, key)
	config := &model.Config{}
	err := row.Scan(
		&config.ID, &config.Scope, &config.ScopeID, &config.Key, &config.Value,
		&config.ValueType, &config.Description, &config.IsSecret,
		&config.CreatedAt, &config.UpdatedAt, &config.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("config not found")
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return config, nil
}

// ListConfigs lists configs with optional filtering, search, and pagination.
func (r *Repository) ListConfigs(ctx context.Context, scope string, scopeID *uuid.UUID, search string, limit, offset int) ([]*model.Config, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}

	whereParts := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if scope != "" {
		whereParts = append(whereParts, fmt.Sprintf("scope = $%d", argIdx))
		args = append(args, scope)
		argIdx++
	}
	if scopeID != nil {
		whereParts = append(whereParts, fmt.Sprintf("scope_id = $%d", argIdx))
		args = append(args, scopeID)
		argIdx++
	} else if scope != "" {
		// When scope is provided but no scopeID, include rows where scope_id IS NULL
		// This is important for global scope queries
		whereParts = append(whereParts, "scope_id IS NULL")
	}
	if search != "" {
		whereParts = append(whereParts, fmt.Sprintf("(key ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	whereClause := strings.Join(whereParts, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM configs WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count configs: %w", err)
	}

	// Fetch rows
	query := fmt.Sprintf(`
		SELECT id, scope, scope_id, key, value, value_type, description, is_secret, created_at, updated_at, deleted_at
		FROM configs
		WHERE %s
		ORDER BY key ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list configs: %w", err)
	}
	defer rows.Close()

	var configs []*model.Config
	for rows.Next() {
		config := &model.Config{}
		err := rows.Scan(
			&config.ID, &config.Scope, &config.ScopeID, &config.Key, &config.Value,
			&config.ValueType, &config.Description, &config.IsSecret,
			&config.CreatedAt, &config.UpdatedAt, &config.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan config: %w", err)
		}
		configs = append(configs, config)
	}

	return configs, total, nil
}

// UpdateConfig updates a config by ID with the given fields.
func (r *Repository) UpdateConfig(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}

	setParts := []string{}
	args := []interface{}{}
	argIdx := 1

	for col, val := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, val)
		argIdx++
	}
	// Always update updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf("UPDATE configs SET %s WHERE id = $%d AND deleted_at IS NULL", strings.Join(setParts, ", "), argIdx)

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	return nil
}

// DeleteConfig soft-deletes a config by ID.
func (r *Repository) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE configs
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	return nil
}

// BulkCreateConfigs inserts multiple configs in a single transaction.
func (r *Repository) BulkCreateConfigs(ctx context.Context, configs []*model.Config) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	if len(configs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO configs (id, scope, scope_id, key, value, value_type, description, is_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	for _, config := range configs {
		batch.Queue(query,
			config.ID, config.Scope, config.ScopeID, config.Key, config.Value,
			config.ValueType, config.Description, config.IsSecret, config.CreatedAt, config.UpdatedAt,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(configs); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to bulk create configs: %w", err)
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("failed to close batch: %w", err)
	}
	return nil
}

// CountConfigs returns the total number of configs for a given scope and optional scopeID.
func (r *Repository) CountConfigs(ctx context.Context, scope string, scopeID *uuid.UUID) (int, error) {
	if err := r.checkPool(); err != nil {
		return 0, err
	}
	query := `SELECT COUNT(*) FROM configs WHERE scope = $1 AND scope_id IS NOT DISTINCT FROM $2 AND deleted_at IS NULL`
	var count int
	if err := r.pool.QueryRow(ctx, query, scope, scopeID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count configs: %w", err)
	}
	return count, nil
}

// Ping verifies database connectivity.
func (r *Repository) Ping(ctx context.Context) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	return r.pool.Ping(ctx)
}
