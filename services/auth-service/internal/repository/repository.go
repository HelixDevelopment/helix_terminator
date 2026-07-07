package repository

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/auth-service/internal/model"
)

// Repository handles database operations for auth service
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new repository
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, display_name, role, mfa_enabled, mfa_method, mfa_secret, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.DisplayName, user.Role,
		user.MFAEnabled, user.MFAMethod, user.MFASecret, user.EmailVerified,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, role, mfa_enabled, mfa_method, mfa_secret,
		       email_verified, failed_login_attempts, locked_until, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, email)

	user := &model.User{}
	err := row.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.Role,
		&user.MFAEnabled, &user.MFAMethod, &user.MFASecret,
		&user.EmailVerified, &user.FailedLogins, &user.LockedUntil,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, role, mfa_enabled, mfa_method, mfa_secret,
		       email_verified, failed_login_attempts, locked_until, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, id)

	user := &model.User{}
	err := row.Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.Role,
		&user.MFAEnabled, &user.MFAMethod, &user.MFASecret,
		&user.EmailVerified, &user.FailedLogins, &user.LockedUntil,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// UpdateUser updates a user
func (r *Repository) UpdateUser(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET display_name = $2, role = $3, mfa_enabled = $4, mfa_method = $5, mfa_secret = $6,
		    email_verified = $7, failed_login_attempts = $8, locked_until = $9, updated_at = $10
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query,
		user.ID, user.DisplayName, user.Role, user.MFAEnabled, user.MFAMethod, user.MFASecret,
		user.EmailVerified, user.FailedLogins, user.LockedUntil, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// IncrementFailedLogins increments the failed login count for a user
func (r *Repository) IncrementFailedLogins(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE WHEN failed_login_attempts >= 4 THEN $2 ELSE locked_until END,
		    updated_at = $3
		WHERE id = $1
	`
	lockUntil := time.Now().UTC().Add(15 * time.Minute)
	_, err := r.pool.Exec(ctx, query, userID, lockUntil, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to increment failed logins: %w", err)
	}
	return nil
}

// ResetFailedLogins resets the failed login count
func (r *Repository) ResetFailedLogins(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET failed_login_attempts = 0, locked_until = NULL, updated_at = $2
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, userID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to reset failed logins: %w", err)
	}
	return nil
}

// CreateSession creates a new session
func (r *Repository) CreateSession(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO user_sessions (id, user_id, device_id, device_name, device_type, ip_address, user_agent,
		                          access_token_hash, refresh_token_hash, expires_at, last_active_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	// ip_address is a nullable INET column. Several call sites (e.g.
	// Register, VerifyMFA) do not have a client IP to record and leave
	// model.Session.IPAddress at its Go zero value (""), which is NOT a
	// valid INET literal - PostgreSQL rejects it with "invalid input
	// syntax for type inet". An unknown IP is correctly represented as
	// SQL NULL, not the empty string, so translate it here rather than
	// requiring every caller to remember to do so.
	var ipAddress interface{}
	if session.IPAddress != "" {
		ipAddress = session.IPAddress
	}
	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.DeviceID, session.DeviceName, session.DeviceType,
		ipAddress, session.UserAgent, session.AccessTokenHash, session.RefreshTokenHash,
		session.ExpiresAt, session.LastActiveAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSessionByTokenHash retrieves a session by access token hash
func (r *Repository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	query := `
		SELECT id, user_id, device_id, device_name, device_type, ip_address, user_agent,
		       access_token_hash, refresh_token_hash, expires_at, last_active_at, revoked_at, created_at
		FROM user_sessions
		WHERE access_token_hash = $1 AND revoked_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, tokenHash)

	// ip_address is a nullable Postgres INET column. pgx v5 decodes a
	// non-NULL inet value in binary format into net/netip.Prefix (NOT
	// a plain string - scanning into *string/**string only happens to
	// "work" for the trivial NULL case and errors with "cannot scan
	// inet ... in binary format into **string" for any real value), so
	// scan into a *netip.Prefix and fold the address portion into the
	// model's plain string field.
	session := &model.Session{}
	var ipPrefix *netip.Prefix
	err := row.Scan(
		&session.ID, &session.UserID, &session.DeviceID, &session.DeviceName, &session.DeviceType,
		&ipPrefix, &session.UserAgent, &session.AccessTokenHash, &session.RefreshTokenHash,
		&session.ExpiresAt, &session.LastActiveAt, &session.RevokedAt, &session.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if ipPrefix != nil {
		session.IPAddress = ipPrefix.Addr().String()
	}
	return session, nil
}

// UpdateSessionAccessTokenHash rebinds a session's revocation-lookup key
// to a freshly-minted access token hash. Called after a successful
// /refresh mints a new access token so that GetSessionByTokenHash keeps
// recognising the session by whichever access token the client is
// currently presenting, rather than only the one issued at login. Only
// updates non-revoked sessions - RowsAffected()==0 tells the caller the
// session was already revoked (e.g. by a prior /logout) or never
// existed, so a stolen/expired refresh token cannot mint a session-
// bound access token after logout.
func (r *Repository) UpdateSessionAccessTokenHash(ctx context.Context, sessionID uuid.UUID, newAccessTokenHash string) error {
	query := `
		UPDATE user_sessions
		SET access_token_hash = $2
		WHERE id = $1 AND revoked_at IS NULL
	`
	ct, err := r.pool.Exec(ctx, query, sessionID, newAccessTokenHash)
	if err != nil {
		return fmt.Errorf("failed to update session access token hash: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("session not found or already revoked")
	}
	return nil
}

// RevokeSession revokes a session
func (r *Repository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = $2
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, sessionID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	return nil
}

// RevokeAllUserSessions revokes all sessions for a user
func (r *Repository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE user_sessions
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, userID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to revoke all sessions: %w", err)
	}
	return nil
}

// ListActiveSessions lists all active sessions for a user
func (r *Repository) ListActiveSessions(ctx context.Context, userID uuid.UUID) ([]*model.Session, error) {
	query := `
		SELECT id, user_id, device_id, device_name, device_type, ip_address, user_agent,
		       expires_at, last_active_at, created_at
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > $2
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*model.Session
	for rows.Next() {
		session := &model.Session{}
		var ipPrefix *netip.Prefix
		err := rows.Scan(
			&session.ID, &session.UserID, &session.DeviceID, &session.DeviceName, &session.DeviceType,
			&ipPrefix, &session.UserAgent, &session.ExpiresAt, &session.LastActiveAt, &session.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		if ipPrefix != nil {
			session.IPAddress = ipPrefix.Addr().String()
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// UpdateSessionActivity updates the last active timestamp
func (r *Repository) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE user_sessions
		SET last_active_at = $2
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, sessionID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}
	return nil
}

// EmailExists checks if an email is already registered
func (r *Repository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email: %w", err)
	}
	return exists, nil
}

// Ping verifies the underlying database connection pool is genuinely
// reachable. It is the real DB-liveness check ReadinessCheck relies on
// (T8-6) - a pool that is closed, exhausted, or connected to a
// crashed/unreachable PostgreSQL instance returns a non-nil error here,
// which is what makes readiness reporting honest instead of a
// fabricated "ready:true" regardless of DB state.
func (r *Repository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("repository has no database pool configured")
	}
	return r.pool.Ping(ctx)
}
