package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/ai-service/internal/model"
)

// Repository handles AI data access
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

// CreateRequest creates a new AI request. The caller (handler.CreateRequest) resolves
// the real LLM completion SYNCHRONOUSLY before calling this method, so req.Response /
// req.TokensUsed / req.Status already carry the real terminal outcome ("completed" or
// "failed") — this INSERT persists them in the same round trip rather than writing a
// placeholder row and updating it later (§11.4.108: no code path ever persists the
// fabricated "pending" this method used to write unconditionally).
func (r *Repository) CreateRequest(ctx context.Context, req *model.AIRequest) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO ai_requests (id, user_id, org_id, prompt, context, model, max_tokens, temperature, status, response, tokens_used, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, req.ID, req.UserID, req.OrgID, req.Prompt, req.Context, req.Model, req.MaxTokens, req.Temperature, req.Status, req.Response, req.TokensUsed)
	if err != nil {
		return fmt.Errorf("failed to create AI request: %w", err)
	}
	return nil
}

// GetRequestByID retrieves an AI request by ID
func (r *Repository) GetRequestByID(ctx context.Context, id uuid.UUID) (*model.AIRequest, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, user_id, org_id, prompt, context, model, max_tokens, temperature, status, response, tokens_used, created_at, updated_at
		FROM ai_requests WHERE id = $1
	`
	var req model.AIRequest
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&req.ID, &req.UserID, &req.OrgID, &req.Prompt, &req.Context, &req.Model, &req.MaxTokens, &req.Temperature, &req.Status, &req.Response, &req.TokensUsed, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("AI request not found")
		}
		return nil, err
	}
	return &req, nil
}

// ListRequests retrieves AI requests with filtering
func (r *Repository) ListRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.AIRequest, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	countQuery := "SELECT COUNT(*) FROM ai_requests WHERE user_id = $1"
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, org_id, prompt, context, model, max_tokens, temperature, status, response, tokens_used, created_at, updated_at
		FROM ai_requests WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reqs []*model.AIRequest
	for rows.Next() {
		var req model.AIRequest
		if err := rows.Scan(
			&req.ID, &req.UserID, &req.OrgID, &req.Prompt, &req.Context, &req.Model, &req.MaxTokens, &req.Temperature, &req.Status, &req.Response, &req.TokensUsed, &req.CreatedAt, &req.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		reqs = append(reqs, &req)
	}
	return reqs, total, rows.Err()
}

// UpdateResponse updates the AI response
func (r *Repository) UpdateResponse(ctx context.Context, id uuid.UUID, response string, tokensUsed int) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE ai_requests SET response = $2, tokens_used = $3, status = 'completed', updated_at = NOW() WHERE id = $1"
	_, err := r.pool.Exec(ctx, query, id, response, tokensUsed)
	if err != nil {
		return fmt.Errorf("failed to update AI response: %w", err)
	}
	return nil
}
