package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/helixdevelopment/keychain-service/internal/crypto"
	"github.com/helixdevelopment/keychain-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles keychain data access
type Repository struct {
	pool   *pgxpool.Pool
	encKey string
}

// New creates a new Repository. encKey is the encryption-at-rest key used
// to protect private_key + passphrase (§11.4.10 / T10) — it MUST be
// non-empty; New fails closed (returns an error, never a repository that
// would silently store plaintext) when it is not. Production callers MUST
// source it from the KEYCHAIN_ENCRYPTION_KEY environment variable (never
// hardcoded, §11.4.10); tests supply a test-only key.
func New(pool *pgxpool.Pool, encKey string) (*Repository, error) {
	if encKey == "" {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}
	return &Repository{pool: pool, encKey: encKey}, nil
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

// CreateItem creates a new keychain item. private_key and passphrase are
// encrypted at rest (§11.4.10 / T10) — the caller's item struct is left
// untouched (still holds plaintext in memory); only the values written to
// the database are ciphertext.
func (r *Repository) CreateItem(ctx context.Context, item *model.KeychainItem) error {
	if err := r.checkPool(); err != nil {
		return err
	}

	encPrivateKey, err := crypto.Encrypt(item.PrivateKey, r.encKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %w", err)
	}
	encPassphrase, err := crypto.Encrypt(item.Passphrase, r.encKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt passphrase: %w", err)
	}

	query := `
		INSERT INTO keychain_items (id, user_id, org_id, name, type, fingerprint, public_key, private_key, passphrase, metadata, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
	`
	_, err = r.pool.Exec(ctx, query,
		item.ID, item.UserID, item.OrgID, item.Name, item.Type, item.Fingerprint,
		item.PublicKey, encPrivateKey, encPassphrase, item.Metadata, item.Tags,
	)
	if err != nil {
		return fmt.Errorf("failed to create keychain item: %w", err)
	}
	return nil
}

// GetItemByID retrieves a keychain item by ID. private_key and passphrase
// are decrypted on read (§11.4.10 / T10) so the returned item carries
// plaintext, matching CreateItem's contract.
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

	item.PrivateKey, err = crypto.Decrypt(item.PrivateKey, r.encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}
	item.Passphrase, err = crypto.Decrypt(item.Passphrase, r.encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt passphrase: %w", err)
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

// UpdateItem updates a keychain item. The SQL SET-clause column names are
// resolved exclusively via buildUpdateQuery's static allow-list (T13,
// §11.4 SQL-injection-shaped hardening) — see buildUpdateQuery for the
// full rationale.
func (r *Repository) UpdateItem(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	query, args, err := buildUpdateQuery(updates, id)
	if err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("keychain item not found")
	}
	return nil
}

// buildUpdateQuery builds the parameterized UPDATE statement for
// UpdateItem. It is split out from UpdateItem so the SQL-shape logic is
// unit-testable without a live database connection (T13).
//
// SECURITY (T13, §11.4 SQL-injection-shaped hardening): earlier revisions
// of this function took the caller-supplied map key and interpolated it
// DIRECTLY into the SET clause via fmt.Sprintf("%s = $%d", key, argIdx) —
// i.e. the SQL COLUMN NAME came straight from a Go map key with zero
// validation. Investigation (§11.4.102) established as FACT that the
// only current caller, Handler.UpdateItem, builds that map using four
// hardcoded Go string literals ("name", "public_key", "metadata",
// "tags") — never a client-supplied field name — so the defect was NOT
// exploitable through any exported HTTP or gRPC entry point at the time
// of this fix (the gRPC service in api/proto/keychain_service/v1/keychain_service.proto
// defines no RPCs beyond HealthCheck). It was nonetheless a LATENT
// SQL-shape defect: Repository.UpdateItem is an exported method that
// accepts a raw map[string]interface{}, and any future caller that ever
// forwarded a client-controlled field name into that map (a generic
// PATCH endpoint, an admin tool, a new gRPC RPC) would have turned it
// into a live SQL-injection primitive with no code-level defense.
//
// The fix: column names are now resolved ONLY through the static,
// hardcoded allowedUpdateColumns table below — never through the
// caller's map key directly — so a column name can NEVER be
// attacker-influenced regardless of what the caller passes in. Any key
// not present in the allow-list is REJECTED with an error (fail-closed,
// §11.4.6) rather than silently dropped or passed through. Values remain
// fully parameterized ($1, $2, ...) exactly as before.
//
// T10 preservation: allowedUpdateColumns deliberately EXCLUDES
// "private_key" and "passphrase" — the two columns CreateItem/
// GetItemByID encrypt/decrypt at rest (T10). UpdateItem itself performs
// no encryption, so — before this fix, exactly as after — it must never
// be able to write those columns; the allow-list now makes that a
// structural guarantee instead of a caller-discipline convention: any
// future caller attempting to set "private_key" or "passphrase" via
// UpdateItem is rejected outright, rather than silently writing
// plaintext into an encrypted-at-rest column.
func buildUpdateQuery(updates map[string]interface{}, id uuid.UUID) (string, []interface{}, error) {
	if len(updates) == 0 {
		return "", nil, nil
	}

	// Deterministic key order (map iteration order is randomized in Go)
	// so the generated query string is stable and testable.
	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var setClauses []string
	var args []interface{}
	argIdx := 1
	for _, key := range keys {
		column, ok := allowedUpdateColumns[key]
		if !ok {
			return "", nil, fmt.Errorf("invalid update field %q: not in allow-list", key)
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", column, argIdx))
		args = append(args, updates[key])
		argIdx++
	}
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++
	args = append(args, id)

	query := fmt.Sprintf("UPDATE keychain_items SET %s WHERE id = $%d AND deleted_at IS NULL",
		joinSetClauses(setClauses), argIdx)
	return query, args, nil
}

// allowedUpdateColumns is the CLOSED, static allow-list mapping every
// logical field name UpdateItem accepts to its exact, hardcoded SQL
// column name (T13). This is the ONLY source of column names
// buildUpdateQuery ever emits into a SET clause — caller-supplied map
// keys are used purely as lookup keys into this table, never
// interpolated into SQL themselves. Deliberately EXCLUDED: "private_key"
// and "passphrase" (T10 encrypted-at-rest secret columns — see
// buildUpdateQuery's doc comment) and every identity/system-managed
// column (id, user_id, org_id, type, fingerprint, created_at,
// updated_at, deleted_at) which UpdateItem was never meant to touch.
var allowedUpdateColumns = map[string]string{
	"name":       "name",
	"public_key": "public_key",
	"metadata":   "metadata",
	"tags":       "tags",
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
