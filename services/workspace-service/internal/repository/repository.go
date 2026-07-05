package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/workspace-service/internal/model"
)

// Repository handles database operations for workspace service.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) checkPool() error {
	if r.pool == nil {
		return fmt.Errorf("database not connected")
	}
	return nil
}

// Ping verifies connectivity.
func (r *Repository) Ping(ctx context.Context) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	return r.pool.Ping(ctx)
}

// CreateWorkspace creates a new workspace.
func (r *Repository) CreateWorkspace(ctx context.Context, workspace *model.Workspace) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO workspaces (
			id, org_id, user_id, name, description, color, icon, tags, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		workspace.ID, workspace.OrgID, workspace.UserID, workspace.Name, workspace.Description,
		workspace.Color, workspace.Icon, workspace.Tags, workspace.CreatedAt, workspace.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	return nil
}

// GetWorkspaceByID retrieves a workspace by ID.
func (r *Repository) GetWorkspaceByID(ctx context.Context, id uuid.UUID) (*model.Workspace, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, org_id, user_id, name, description, color, icon, tags, created_at, updated_at, deleted_at
		FROM workspaces
		WHERE id = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, id)

	workspace := &model.Workspace{}
	err := row.Scan(
		&workspace.ID, &workspace.OrgID, &workspace.UserID, &workspace.Name, &workspace.Description,
		&workspace.Color, &workspace.Icon, &workspace.Tags, &workspace.CreatedAt, &workspace.UpdatedAt, &workspace.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workspace not found")
		}
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}
	return workspace, nil
}

// ListWorkspaces lists workspaces with optional filtering.
func (r *Repository) ListWorkspaces(ctx context.Context, orgID, userID uuid.UUID, tags []string, limit, offset int) ([]*model.Workspace, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if orgID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("org_id = $%d", argIdx))
		args = append(args, orgID)
		argIdx++
	}
	if userID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, userID)
		argIdx++
	}
	if len(tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, tags)
		argIdx++
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM workspaces WHERE %s`, joinConditions(conditions))
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count workspaces: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, org_id, user_id, name, description, color, icon, tags, created_at, updated_at, deleted_at
		FROM workspaces
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, joinConditions(conditions), argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []*model.Workspace
	for rows.Next() {
		workspace := &model.Workspace{}
		err := rows.Scan(
			&workspace.ID, &workspace.OrgID, &workspace.UserID, &workspace.Name, &workspace.Description,
			&workspace.Color, &workspace.Icon, &workspace.Tags, &workspace.CreatedAt, &workspace.UpdatedAt, &workspace.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan workspace: %w", err)
		}
		workspaces = append(workspaces, workspace)
	}

	return workspaces, total, nil
}

// UpdateWorkspace updates an existing workspace.
func (r *Repository) UpdateWorkspace(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now().UTC()

	setParts := []string{}
	args := []interface{}{id}
	argIdx := 2

	for key, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}

	query := fmt.Sprintf("UPDATE workspaces SET %s WHERE id = $1 AND deleted_at IS NULL", joinConditions(setParts))
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}
	return nil
}

// DeleteWorkspace performs a soft delete on a workspace.
func (r *Repository) DeleteWorkspace(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	now := time.Now().UTC()
	query := `
		UPDATE workspaces
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}
	return nil
}

// AddHost adds a host to a workspace.
func (r *Repository) AddHost(ctx context.Context, workspaceID, hostID, addedBy uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO workspace_hosts (workspace_id, host_id, added_at, added_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (workspace_id, host_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, workspaceID, hostID, time.Now().UTC(), addedBy)
	if err != nil {
		return fmt.Errorf("failed to add host to workspace: %w", err)
	}
	return nil
}

// RemoveHost removes a host from a workspace.
func (r *Repository) RemoveHost(ctx context.Context, workspaceID, hostID uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		DELETE FROM workspace_hosts
		WHERE workspace_id = $1 AND host_id = $2
	`
	_, err := r.pool.Exec(ctx, query, workspaceID, hostID)
	if err != nil {
		return fmt.Errorf("failed to remove host from workspace: %w", err)
	}
	return nil
}

// ListHosts returns the host IDs associated with a workspace.
func (r *Repository) ListHosts(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT host_id
		FROM workspace_hosts
		WHERE workspace_id = $1
		ORDER BY added_at DESC
	`
	rows, err := r.pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace hosts: %w", err)
	}
	defer rows.Close()

	var hostIDs []uuid.UUID
	for rows.Next() {
		var hostID uuid.UUID
		if err := rows.Scan(&hostID); err != nil {
			return nil, fmt.Errorf("failed to scan host_id: %w", err)
		}
		hostIDs = append(hostIDs, hostID)
	}

	return hostIDs, nil
}

// CountWorkspaces counts workspaces for a user/org.
func (r *Repository) CountWorkspaces(ctx context.Context, orgID, userID uuid.UUID) (int, error) {
	if err := r.checkPool(); err != nil {
		return 0, err
	}
	conditions := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if orgID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("org_id = $%d", argIdx))
		args = append(args, orgID)
		argIdx++
	}
	if userID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, userID)
		argIdx++
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM workspaces WHERE %s`, joinConditions(conditions))
	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count workspaces: %w", err)
	}
	return count, nil
}

func joinConditions(conditions []string) string {
	result := ""
	for i, c := range conditions {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}
