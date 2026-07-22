package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/helixdevelopment/user-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles user data access
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new Repository
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, display_name, avatar_url, role, permissions, org_id, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	// The users.permissions column is NOT NULL DEFAULT '{}' (see
	// migrations/001_init.up.sql). A nil Go slice marshals to SQL NULL,
	// not an empty array, and pgx does not coerce it - so a caller that
	// omits Permissions would violate the NOT NULL constraint at INSERT
	// time. The HTTP handler (internal/handler.CreateUser) already
	// defends against this before calling CreateUser, but Repository is
	// an exported type any other caller (gRPC handler, CLI seed tool,
	// tests) can call directly - defend at the layer that actually owns
	// the schema contract, not only at the one caller that happens to
	// exist today (found via real-Postgres integration testing, T2).
	if user.Permissions == nil {
		user.Permissions = []string{}
	}
	return r.pool.QueryRow(ctx, query,
		user.ID, user.Email, user.DisplayName, user.AvatarURL, user.Role, user.Permissions, user.OrgID, user.EmailVerified,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, role, permissions, org_id, email_verified, last_login_at, created_at, updated_at, deleted_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`
	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL, &user.Role, &user.Permissions,
		&user.OrgID, &user.EmailVerified, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, role, permissions, org_id, email_verified, last_login_at, created_at, updated_at, deleted_at
		FROM users WHERE email = $1 AND deleted_at IS NULL
	`
	var user model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL, &user.Role, &user.Permissions,
		&user.OrgID, &user.EmailVerified, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// ListUsers retrieves users with optional filtering
func (r *Repository) ListUsers(ctx context.Context, orgID, role, search string, limit, offset int) ([]*model.User, int, error) {
	whereClause := "deleted_at IS NULL"
	var args []interface{}
	argIdx := 1

	if orgID != "" {
		whereClause += fmt.Sprintf(" AND org_id = $%d", argIdx)
		args = append(args, orgID)
		argIdx++
	}
	if role != "" {
		whereClause += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, role)
		argIdx++
	}
	if search != "" {
		whereClause += fmt.Sprintf(" AND (email ILIKE $%d OR display_name ILIKE $%d)", argIdx, argIdx+1)
		args = append(args, "%"+search+"%", "%"+search+"%")
		argIdx += 2
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, email, display_name, avatar_url, role, permissions, org_id, email_verified, last_login_at, created_at, updated_at
		FROM users WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID, &user.Email, &user.DisplayName, &user.AvatarURL, &user.Role, &user.Permissions,
			&user.OrgID, &user.EmailVerified, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, &user)
	}
	return users, total, rows.Err()
}

// UpdateUser updates a user
func (r *Repository) UpdateUser(ctx context.Context, id string, updates map[string]interface{}) error {
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

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), argIdx)
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// DeleteUser soft-deletes a user
func (r *Repository) DeleteUser(ctx context.Context, id string) error {
	query := "UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL"
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// EmailExists checks if an email is already registered
func (r *Repository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)"
	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}

// UpdateLastLogin updates the last login timestamp
func (r *Repository) UpdateLastLogin(ctx context.Context, id string) error {
	query := "UPDATE users SET last_login_at = NOW() WHERE id = $1 AND deleted_at IS NULL"
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CreateOrUpdateProfile creates or updates a user profile
func (r *Repository) CreateOrUpdateProfile(ctx context.Context, userID string, profile map[string]interface{}) error {
	query := `
		INSERT INTO user_profiles (user_id, bio, timezone, locale, preferences, ssh_public_key, github_id, gitlab_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			bio = COALESCE(EXCLUDED.bio, user_profiles.bio),
			timezone = COALESCE(EXCLUDED.timezone, user_profiles.timezone),
			locale = COALESCE(EXCLUDED.locale, user_profiles.locale),
			preferences = COALESCE(EXCLUDED.preferences, user_profiles.preferences),
			ssh_public_key = COALESCE(EXCLUDED.ssh_public_key, user_profiles.ssh_public_key),
			github_id = COALESCE(EXCLUDED.github_id, user_profiles.github_id),
			gitlab_id = COALESCE(EXCLUDED.gitlab_id, user_profiles.gitlab_id),
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, userID,
		profile["bio"], profile["timezone"], profile["locale"], profile["preferences"],
		profile["ssh_public_key"], profile["github_id"], profile["gitlab_id"],
	)
	return err
}

// GetProfile retrieves a user profile.
//
// The query LEFT JOINs user_profiles, so bio/timezone/locale/
// ssh_public_key/github_id/gitlab_id are genuinely SQL NULL - not just
// possible but the COMMON case - whenever a user has never had a
// profile row created for them (CreateOrUpdateProfile never called) or
// only had a PARTIAL profile written (UpdateProfile's handler builds a
// map containing only the fields the caller actually sent, and
// CreateOrUpdateProfile's INSERT explicitly binds every other named
// column to SQL NULL). model.UserProfile declares these as plain
// (non-pointer) strings, and pgx refuses to scan SQL NULL into a
// non-pointer string destination - so scan into nullable *string locals
// first and fold NULL to "" when populating the response (found via
// real-Postgres integration testing, T2: the previous unconditional
// &profile.Bio/&profile.SSHPublicKey/... scan 500'd on the very next
// GetProfile call after ANY partial profile update, and on the very
// first GetProfile for a brand-new user with no profile row at all).
func (r *Repository) GetProfile(ctx context.Context, userID string) (*model.UserProfile, error) {
	query := `
		SELECT u.id, u.email, u.display_name, u.avatar_url, u.role, u.permissions, u.org_id, u.email_verified, u.last_login_at, u.created_at, u.updated_at,
			p.bio, p.timezone, p.locale, p.preferences, p.ssh_public_key, p.github_id, p.gitlab_id
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`
	var profile model.UserProfile
	var bio, timezone, locale, sshPublicKey, githubID, gitlabID *string
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&profile.ID, &profile.Email, &profile.DisplayName, &profile.AvatarURL, &profile.Role, &profile.Permissions,
		&profile.OrgID, &profile.EmailVerified, &profile.LastLoginAt, &profile.CreatedAt, &profile.UpdatedAt,
		&bio, &timezone, &locale, &profile.Preferences, &sshPublicKey,
		&githubID, &gitlabID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	if bio != nil {
		profile.Bio = *bio
	}
	if timezone != nil {
		profile.Timezone = *timezone
	}
	if locale != nil {
		profile.Locale = *locale
	}
	if sshPublicKey != nil {
		profile.SSHPublicKey = *sshPublicKey
	}
	if githubID != nil {
		profile.GitHubID = *githubID
	}
	if gitlabID != nil {
		profile.GitLabID = *gitlabID
	}
	return &profile, nil
}

// Ping verifies the underlying database connection pool is genuinely
// reachable. It is the real DB-liveness check ReadinessCheck relies on
// (T8-6) - a pool that is closed, exhausted, or connected to a
// crashed/unreachable PostgreSQL instance returns a non-nil error here,
// which is what makes readiness reporting honest instead of a
// fabricated "status":"ready" regardless of DB state.
func (r *Repository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("repository has no database pool configured")
	}
	return r.pool.Ping(ctx)
}
