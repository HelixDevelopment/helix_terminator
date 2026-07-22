package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/org-service/internal/model"
)

// Repository handles database operations for org service.
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

// CreateOrg creates a new organization.
func (r *Repository) CreateOrg(ctx context.Context, org *model.Organization) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO organizations (id, name, slug, description, logo_url, owner_id, plan, settings, member_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		org.ID, org.Name, org.Slug, org.Description, org.LogoURL, org.OwnerID, org.Plan, org.Settings, org.MemberCount, org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}
	return nil
}

// GetOrgByID retrieves an organization by ID.
func (r *Repository) GetOrgByID(ctx context.Context, id uuid.UUID) (*model.Organization, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, name, slug, description, logo_url, owner_id, plan, settings, member_count, created_at, updated_at, deleted_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, id)

	org := &model.Organization{}
	err := row.Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &org.LogoURL, &org.OwnerID, &org.Plan, &org.Settings, &org.MemberCount,
		&org.CreatedAt, &org.UpdatedAt, &org.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("organization not found")
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return org, nil
}

// GetOrgBySlug retrieves an organization by slug.
func (r *Repository) GetOrgBySlug(ctx context.Context, slug string) (*model.Organization, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, name, slug, description, logo_url, owner_id, plan, settings, member_count, created_at, updated_at, deleted_at
		FROM organizations
		WHERE slug = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, slug)

	org := &model.Organization{}
	err := row.Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &org.LogoURL, &org.OwnerID, &org.Plan, &org.Settings, &org.MemberCount,
		&org.CreatedAt, &org.UpdatedAt, &org.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("organization not found")
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return org, nil
}

// ListOrgs lists organizations a user belongs to with optional search.
func (r *Repository) ListOrgs(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]*model.Organization, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}

	conditions := []string{"o.deleted_at IS NULL", "m.user_id = $1"}
	args := []interface{}{userID}
	argIdx := 2

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(o.name ILIKE $%d OR o.slug ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM organizations o
		JOIN memberships m ON o.id = m.org_id
		WHERE %s
	`, joinConditions(conditions))
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count organizations: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT o.id, o.name, o.slug, o.description, o.logo_url, o.owner_id, o.plan, o.settings, o.member_count, o.created_at, o.updated_at, o.deleted_at
		FROM organizations o
		JOIN memberships m ON o.id = m.org_id
		WHERE %s
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d
	`, joinConditions(conditions), argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*model.Organization
	for rows.Next() {
		org := &model.Organization{}
		err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Description, &org.LogoURL, &org.OwnerID, &org.Plan, &org.Settings, &org.MemberCount,
			&org.CreatedAt, &org.UpdatedAt, &org.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	return orgs, total, nil
}

// UpdateOrg updates an organization.
func (r *Repository) UpdateOrg(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now().UTC()

	setParts := []string{}
	args := []interface{}{}
	argIdx := 1
	for col, val := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, val)
		argIdx++
	}
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE organizations
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
	`, joinSetClauses(setParts), argIdx)
	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}
	return nil
}

// DeleteOrg performs a soft delete on an organization.
func (r *Repository) DeleteOrg(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	now := time.Now().UTC()
	query := `
		UPDATE organizations
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	return nil
}

// CreateTeam creates a new team.
func (r *Repository) CreateTeam(ctx context.Context, team *model.Team) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO teams (id, org_id, name, description, member_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		team.ID, team.OrgID, team.Name, team.Description, team.MemberCount, team.CreatedAt, team.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

// GetTeamByID retrieves a team by ID.
func (r *Repository) GetTeamByID(ctx context.Context, id uuid.UUID) (*model.Team, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, org_id, name, description, member_count, created_at, updated_at
		FROM teams
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)

	team := &model.Team{}
	err := row.Scan(
		&team.ID, &team.OrgID, &team.Name, &team.Description, &team.MemberCount, &team.CreatedAt, &team.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("team not found")
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	return team, nil
}

// ListTeams lists teams for an organization.
func (r *Repository) ListTeams(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*model.Team, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT id, org_id, name, description, member_count, created_at, updated_at
		FROM teams
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	defer rows.Close()

	var teams []*model.Team
	for rows.Next() {
		team := &model.Team{}
		err := rows.Scan(
			&team.ID, &team.OrgID, &team.Name, &team.Description, &team.MemberCount, &team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}
		teams = append(teams, team)
	}
	return teams, nil
}

// UpdateTeam updates a team.
func (r *Repository) UpdateTeam(ctx context.Context, team *model.Team) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE teams
		SET name = $2, description = $3, updated_at = $4
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, team.ID, team.Name, team.Description, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}
	return nil
}

// DeleteTeam deletes a team.
func (r *Repository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `DELETE FROM teams WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	return nil
}

// AddMember adds a membership.
func (r *Repository) AddMember(ctx context.Context, membership *model.Membership) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO memberships (id, org_id, user_id, team_id, role, invited_by, invited_at, joined_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		membership.ID, membership.OrgID, membership.UserID, membership.TeamID, membership.Role, membership.InvitedBy, membership.InvitedAt, membership.JoinedAt, membership.CreatedAt, membership.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}
	return nil
}

// GetMember retrieves a membership by org and user.
func (r *Repository) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*model.Membership, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, org_id, user_id, team_id, role, invited_by, invited_at, joined_at, created_at, updated_at
		FROM memberships
		WHERE org_id = $1 AND user_id = $2
	`
	row := r.pool.QueryRow(ctx, query, orgID, userID)

	m := &model.Membership{}
	err := row.Scan(
		&m.ID, &m.OrgID, &m.UserID, &m.TeamID, &m.Role, &m.InvitedBy, &m.InvitedAt, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("membership not found")
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return m, nil
}

// ListMembers lists memberships with optional filtering.
func (r *Repository) ListMembers(ctx context.Context, orgID uuid.UUID, teamID *uuid.UUID, role string, limit, offset int) ([]*model.Membership, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}

	conditions := []string{"org_id = $1"}
	args := []interface{}{orgID}
	argIdx := 2

	if teamID != nil {
		conditions = append(conditions, fmt.Sprintf("team_id = $%d", argIdx))
		args = append(args, *teamID)
		argIdx++
	}
	if role != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, role)
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM memberships
		WHERE %s
	`, joinConditions(conditions))
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count members: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, org_id, user_id, team_id, role, invited_by, invited_at, joined_at, created_at, updated_at
		FROM memberships
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, joinConditions(conditions), argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list members: %w", err)
	}
	defer rows.Close()

	var members []*model.Membership
	for rows.Next() {
		m := &model.Membership{}
		err := rows.Scan(
			&m.ID, &m.OrgID, &m.UserID, &m.TeamID, &m.Role, &m.InvitedBy, &m.InvitedAt, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, total, nil
}

// UpdateMember updates a membership.
func (r *Repository) UpdateMember(ctx context.Context, membership *model.Membership) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		UPDATE memberships
		SET role = $2, team_id = $3, updated_at = $4
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, membership.ID, membership.Role, membership.TeamID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}
	return nil
}

// RemoveMember removes a membership.
func (r *Repository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `DELETE FROM memberships WHERE org_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}
	return nil
}

// CountMembers counts members in an organization.
func (r *Repository) CountMembers(ctx context.Context, orgID uuid.UUID) (int, error) {
	if err := r.checkPool(); err != nil {
		return 0, err
	}
	query := `SELECT COUNT(*) FROM memberships WHERE org_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count members: %w", err)
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

// joinSetClauses joins UPDATE ... SET assignment fragments with ", "
// (SQL SET-clause syntax), distinct from joinConditions' " AND " join
// (SQL WHERE-clause syntax). Found via real-Postgres integration
// testing (T2): UpdateOrg previously reused joinConditions for its SET
// clause, producing invalid SQL like "SET name = $1 AND updated_at =
// $2" - a syntax error Postgres rejects, meaning every PUT
// /api/v1/orgs/:id request always failed with a 500 regardless of
// payload.
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
