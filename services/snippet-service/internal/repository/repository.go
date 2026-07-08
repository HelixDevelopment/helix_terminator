package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/helixdevelopment/snippet-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles snippet data access
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

// CreateSnippet creates a new snippet
func (r *Repository) CreateSnippet(ctx context.Context, snippet *model.Snippet) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO snippets (id, org_id, created_by, name, content, language, tags, description, is_public, usage_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, snippet.ID, snippet.OrgID, snippet.CreatedBy, snippet.Name, snippet.Content, snippet.Language, snippet.Tags, snippet.Description, snippet.IsPublic, snippet.UsageCount)
	if err != nil {
		return fmt.Errorf("failed to create snippet: %w", err)
	}
	return nil
}

// GetSnippetByID retrieves a snippet by ID
func (r *Repository) GetSnippetByID(ctx context.Context, id uuid.UUID) (*model.Snippet, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, org_id, created_by, name, content, language, tags, description, is_public, usage_count, created_at, updated_at
		FROM snippets WHERE id = $1
	`
	var snippet model.Snippet
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&snippet.ID, &snippet.OrgID, &snippet.CreatedBy, &snippet.Name, &snippet.Content, &snippet.Language, &snippet.Tags, &snippet.Description, &snippet.IsPublic, &snippet.UsageCount, &snippet.CreatedAt, &snippet.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("snippet not found")
		}
		return nil, err
	}
	return &snippet, nil
}

// ListSnippets retrieves snippets with filtering
func (r *Repository) ListSnippets(ctx context.Context, orgID *uuid.UUID, createdBy uuid.UUID, language string, isPublic *bool, limit, offset int) ([]*model.Snippet, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	whereClause := "1=1"
	var args []interface{}
	argIdx := 1

	if orgID != nil {
		whereClause += fmt.Sprintf(" AND org_id = $%d", argIdx)
		args = append(args, orgID)
		argIdx++
	}
	if createdBy != uuid.Nil {
		whereClause += fmt.Sprintf(" AND created_by = $%d", argIdx)
		args = append(args, createdBy)
		argIdx++
	}
	if language != "" {
		whereClause += fmt.Sprintf(" AND language = $%d", argIdx)
		args = append(args, language)
		argIdx++
	}
	if isPublic != nil {
		whereClause += fmt.Sprintf(" AND is_public = $%d", argIdx)
		args = append(args, *isPublic)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM snippets WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, org_id, created_by, name, content, language, tags, description, is_public, usage_count, created_at, updated_at
		FROM snippets WHERE %s
		ORDER BY updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var snippets []*model.Snippet
	for rows.Next() {
		var snippet model.Snippet
		if err := rows.Scan(
			&snippet.ID, &snippet.OrgID, &snippet.CreatedBy, &snippet.Name, &snippet.Content, &snippet.Language, &snippet.Tags, &snippet.Description, &snippet.IsPublic, &snippet.UsageCount, &snippet.CreatedAt, &snippet.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		snippets = append(snippets, &snippet)
	}
	return snippets, total, rows.Err()
}

// UpdateSnippet updates a snippet
func (r *Repository) UpdateSnippet(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE snippets SET name = $2, content = $3, language = $4, tags = $5, description = $6, is_public = $7, updated_at = NOW() WHERE id = $1"
	_, err := r.pool.Exec(ctx, query, id, updates["name"], updates["content"], updates["language"], updates["tags"], updates["description"], updates["is_public"])
	if err != nil {
		return fmt.Errorf("failed to update snippet: %w", err)
	}
	return nil
}

// DeleteSnippet deletes a snippet
func (r *Repository) DeleteSnippet(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "DELETE FROM snippets WHERE id = $1"
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete snippet: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("snippet not found")
	}
	return nil
}

// IncrementUsage increments the usage count of a snippet
func (r *Repository) IncrementUsage(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE snippets SET usage_count = usage_count + 1, updated_at = NOW() WHERE id = $1"
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}
	return nil
}
