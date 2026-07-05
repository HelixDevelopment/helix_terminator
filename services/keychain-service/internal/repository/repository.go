package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/keychain-service/internal/model"
)

// Repository handles keychain data access
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

// CreateItem creates a new keychain item
func (r *Repository) CreateItem(ctx context.Context, item *model.KeychainItem) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO keychain_items (id, user_id, org_id, name, type, fingerprint, public_key, private_key, passphrase, metadata, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query,
		item.ID, item.UserID, item.OrgID, item.Name, item.Type, item.Fingerprint,
		item.PublicKey, item.PrivateKey, item.Passphrase, item.Metadata, item.Tags,
	)
	if err != nil {
		return fmt.Errorf("failed to create keychain item: %w", err)
	}
	return nil
}

// GetItemByID retrieves a keychain item by ID
func (r *Repository) GetItemByID(ctx context.Context, id uuid.UUID) (*model.KeychainItem, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, user_id, org_id, name, type, fingerprint, public_key, private_key, passphrase, metadata, tags, created_at, updated_at, deleted_at
		FROM keychain_items WHERE id = $1 AND deleted_at IS NULL
	`
	var item model.KeychainItem
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&item.ID, &item.UserID, &item.OrgID, &item.Name, &item.Type, &item.Fingerprint,
		&item.PublicKey, &item.PrivateKey, &item.Passphrase, &item.Metadata, &item.Tags,
		&item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("keychain item not found")
		}
		return nil, err
	}
	return &item, nil
}

// ListItems retrieves keychain items with filtering
func (r *Repository) ListItems(ctx context.Context, userID, orgID uuid.UUID, itemType string, limit, offset int) ([]*model.KeychainItem, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	whereClause := "deleted_at IS NULL"
	var args []interface{}
	argIdx := 1

	if userID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if orgID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND org_id = $%d", argIdx)
		args = append(args, orgID)
		argIdx++
	}
	if itemType != "" {
		whereClause += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, itemType)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM keychain_items WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, org_id, name, type, fingerprint, public_key, metadata, tags, created_at, updated_at
		FROM keychain_items WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*model.KeychainItem
	for rows.Next() {
		var item model.KeychainItem
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.OrgID, &item.Name, &item.Type, &item.Fingerprint,
			&item.PublicKey, &item.Metadata, &item.Tags, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, &item)
	}
	return items, total, rows.Err()
}

// UpdateItem updates a keychain item
func (r *Repository) UpdateItem(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	var setClauses []string
	var args []interface{}
	argIdx := 1
	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++
	args = append(args, id)

	query := fmt.Sprintf("UPDATE keychain_items SET %s WHERE id = $%d AND deleted_at IS NULL",
		joinSetClauses(setClauses), argIdx)
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("keychain item not found")
	}
	return nil
}

func joinSetClauses(clauses []string) string {
	result := ""
	for i, c := range clauses {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}

// DeleteItem soft-deletes a keychain item
func (r *Repository) DeleteItem(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE keychain_items SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL"
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("keychain item not found")
	}
	return nil
}
