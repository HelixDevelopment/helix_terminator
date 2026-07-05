package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/vault-service/internal/model"
)

// Repository handles database operations for the vault service.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new Repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateSecret inserts a new secret into the database.
func (r *Repository) CreateSecret(ctx context.Context, secret *model.Secret) error {
	query := `
		INSERT INTO secrets (id, user_id, org_id, name, type, encrypted_value, iv, salt, metadata, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.pool.Exec(ctx, query,
		secret.ID, secret.UserID, secret.OrgID, secret.Name, secret.Type,
		secret.EncryptedValue, secret.IV, secret.Salt, secret.Metadata, secret.Tags,
		secret.CreatedAt, secret.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return nil
}

// GetSecretByID retrieves a secret by its ID, excluding soft-deleted records.
func (r *Repository) GetSecretByID(ctx context.Context, id uuid.UUID) (*model.Secret, error) {
	query := `
		SELECT id, user_id, org_id, name, type, encrypted_value, iv, salt, metadata, tags, created_at, updated_at, deleted_at
		FROM secrets
		WHERE id = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, id)

	secret := &model.Secret{}
	err := row.Scan(
		&secret.ID, &secret.UserID, &secret.OrgID, &secret.Name, &secret.Type,
		&secret.EncryptedValue, &secret.IV, &secret.Salt, &secret.Metadata, &secret.Tags,
		&secret.CreatedAt, &secret.UpdatedAt, &secret.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("secret not found")
		}
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}
	return secret, nil
}

// ListSecrets retrieves secrets with optional filtering by userID, orgID, type, and tags.
func (r *Repository) ListSecrets(ctx context.Context, userID, orgID uuid.UUID, secretType model.SecretType, tags []string, limit, offset int) ([]*model.Secret, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "deleted_at IS NULL")

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
	if secretType != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, secretType)
		argIdx++
	}
	if len(tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, tags)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, user_id, org_id, name, type, encrypted_value, iv, salt, metadata, tags, created_at, updated_at, deleted_at
		 FROM secrets
		 WHERE %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		strings.Join(conditions, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer rows.Close()

	var secrets []*model.Secret
	for rows.Next() {
		secret := &model.Secret{}
		err := rows.Scan(
			&secret.ID, &secret.UserID, &secret.OrgID, &secret.Name, &secret.Type,
			&secret.EncryptedValue, &secret.IV, &secret.Salt, &secret.Metadata, &secret.Tags,
			&secret.CreatedAt, &secret.UpdatedAt, &secret.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan secret: %w", err)
		}
		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// UpdateSecret updates an existing secret and sets updated_at to now.
func (r *Repository) UpdateSecret(ctx context.Context, secret *model.Secret) error {
	query := `
		UPDATE secrets
		SET name = $2, type = $3, encrypted_value = $4, iv = $5, salt = $6, metadata = $7, tags = $8, updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL
	`
	secret.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, query,
		secret.ID, secret.Name, secret.Type, secret.EncryptedValue, secret.IV,
		secret.Salt, secret.Metadata, secret.Tags, secret.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}
	return nil
}

// DeleteSecret performs a soft delete on a secret.
func (r *Repository) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE secrets
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

// CreateSecretVersion inserts a new version record for a secret.
func (r *Repository) CreateSecretVersion(ctx context.Context, version *model.SecretVersion) error {
	query := `
		INSERT INTO secret_versions (id, secret_id, encrypted_value, iv, salt, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		version.ID, version.SecretID, version.EncryptedValue, version.IV, version.Salt,
		version.CreatedBy, version.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create secret version: %w", err)
	}
	return nil
}

// GetSecretVersions retrieves historical versions for a secret, ordered by creation time descending.
func (r *Repository) GetSecretVersions(ctx context.Context, secretID uuid.UUID, limit int) ([]*model.SecretVersion, error) {
	query := `
		SELECT id, secret_id, encrypted_value, iv, salt, created_by, created_at
		FROM secret_versions
		WHERE secret_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, secretID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret versions: %w", err)
	}
	defer rows.Close()

	var versions []*model.SecretVersion
	for rows.Next() {
		version := &model.SecretVersion{}
		err := rows.Scan(
			&version.ID, &version.SecretID, &version.EncryptedValue, &version.IV, &version.Salt,
			&version.CreatedBy, &version.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan secret version: %w", err)
		}
		versions = append(versions, version)
	}

	return versions, nil
}

// CountSecrets returns the total number of non-deleted secrets for a user or org.
func (r *Repository) CountSecrets(ctx context.Context, userID, orgID uuid.UUID) (int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "deleted_at IS NULL")

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

	query := fmt.Sprintf("SELECT COUNT(*) FROM secrets WHERE %s", strings.Join(conditions, " AND "))
	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count secrets: %w", err)
	}
	return count, nil
}

// Ping verifies database connectivity.
func (r *Repository) Ping(ctx context.Context) error {
	if r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	return r.pool.Ping(ctx)
}
