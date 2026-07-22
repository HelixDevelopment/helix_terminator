// Package repository (white-box) — T13 SQL-injection-shaped hardening
// tests for UpdateItem's query builder.
//
// §11.4.102 investigation FACT (captured before any fix was written):
// Repository.UpdateItem previously built its SET clause with
//
//	setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
//
// where key came DIRECTLY from the caller-supplied
// map[string]interface{} — the SQL COLUMN NAME was string-interpolated
// with zero validation or allow-listing. The ONLY caller,
// Handler.UpdateItem (internal/handler/handler.go), builds that map
// using four hardcoded Go string literals ("name", "public_key",
// "metadata", "tags") — never a client-supplied field name — and
// UpdateKeychainItemRequest (internal/model/model.go) has no
// pass-through field-name mechanism, so the defect was NOT reachable
// from any exported HTTP entry point. The gRPC service defined in
// api/proto/keychain_service/v1/keychain_service.proto declares no RPCs beyond HealthCheck,
// so there is no gRPC path either. CONCLUSION (§11.4.6, no guessing):
// this was a LATENT SQL-shape defect, not an exploitable one, at the
// time of this fix — but Repository.UpdateItem is exported and took a
// raw map, so any future caller forwarding a client-controlled field
// name into that map would have turned it into a live SQL-injection
// primitive (e.g. a key such as `name = 'x', passphrase` would append
// an attacker-chosen SECOND column — including the T10-encrypted
// "passphrase" column — onto the SET clause, bound to the attacker's
// own value).
package repository

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// preFixBuildUpdateQuery reproduces — VERBATIM in shape — the query-
// building algorithm that shipped in Repository.UpdateItem before this
// T13 fix (see the FACT above): the SQL column name comes straight from
// the map key with no validation whatsoever. It exists ONLY in this test
// file, as the RED baseline, so the defect can be demonstrated and
// proven without needing a live database connection (Repository.
// UpdateItem itself gates on checkPool() and cannot be driven without a
// real *pgxpool.Pool).
func preFixBuildUpdateQuery(updates map[string]interface{}, id uuid.UUID) (string, []interface{}) {
	var setClauses []string
	var args []interface{}
	argIdx := 1
	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}
	query := fmt.Sprintf("UPDATE keychain_items SET %s WHERE id = $%d AND deleted_at IS NULL",
		joinSetClauses(setClauses), argIdx)
	return query, args
}

// TestRED_PreFixLogic_ColumnNameInterpolatedUnsanitized proves the
// PRE-FIX defect was real: given an attacker-shaped map key, the old
// algorithm spliced it verbatim into the generated SQL statement with no
// rejection, no escaping, no allow-list check. This is the RED evidence
// (§11.4.43 / §11.4.115) — it is expected to PASS, and its passing IS the
// proof the vulnerability existed in the code shape that shipped.
func TestRED_PreFixLogic_ColumnNameInterpolatedUnsanitized(t *testing.T) {
	// A key shaped to append a SECOND, attacker-chosen assignment onto
	// the SET clause — in a real exploit this could target the
	// T10-encrypted "passphrase" or "private_key" column, or any other
	// column, bound to the attacker's own supplied value.
	const maliciousKey = `name = 'ignored', passphrase`

	query, _ := preFixBuildUpdateQuery(map[string]interface{}{maliciousKey: "attacker-controlled-value"}, uuid.New())

	require.Contains(t, query, maliciousKey,
		"RED (T13, pre-fix behaviour): the attacker-supplied map key must appear VERBATIM, unsanitized, "+
			"in the generated SQL — this reproduces the exact SQL-column-injection shape that existed "+
			"before the allow-list hardening")
}

// TestGREEN_BuildUpdateQuery_AllowListRejectsUnknownKeys is the GREEN
// counterpart: the FIXED buildUpdateQuery (internal/repository/
// repository.go) must reject the identical malicious key with an error
// and produce NO SQL at all — the opposite of the RED test above.
func TestGREEN_BuildUpdateQuery_AllowListRejectsUnknownKeys(t *testing.T) {
	const maliciousKey = `name = 'ignored', passphrase`

	query, args, err := buildUpdateQuery(map[string]interface{}{maliciousKey: "attacker-controlled-value"}, uuid.New())

	require.Error(t, err, "GREEN (T13): an unknown/malicious update field must be rejected")
	require.Contains(t, err.Error(), maliciousKey, "rejection error should name the offending field")
	require.Empty(t, query, "no SQL may be generated when an update field fails the allow-list check")
	require.Nil(t, args)
}

// TestGREEN_BuildUpdateQuery_UnknownPlainFieldRejected proves rejection
// is not limited to SQL-shaped keys — ANY field not in the static
// allow-list is rejected, including an innocuous-looking but unknown
// field name (defense against typos AND against a future caller that
// forwards an arbitrary but legitimate-looking client field).
func TestGREEN_BuildUpdateQuery_UnknownPlainFieldRejected(t *testing.T) {
	query, args, err := buildUpdateQuery(map[string]interface{}{"nickname": "not-a-real-column"}, uuid.New())

	require.Error(t, err)
	require.Contains(t, err.Error(), "nickname")
	require.Empty(t, query)
	require.Nil(t, args)
}

// TestGREEN_BuildUpdateQuery_SecretColumnsAreNeverInTheAllowList is the
// explicit T10-preservation proof: UpdateItem must NEVER be able to
// write "private_key" or "passphrase" — those columns are only ever
// written, encrypted, via CreateItem, and only ever read, decrypted, via
// GetItemByID (internal/crypto). UpdateItem performs no encryption, so
// if it could write those columns it would silently persist PLAINTEXT
// into an encrypted-at-rest column, breaking the T10 contract. This test
// proves the allow-list makes that structurally impossible, not just a
// caller-discipline convention.
func TestGREEN_BuildUpdateQuery_SecretColumnsAreNeverInTheAllowList(t *testing.T) {
	for _, secretField := range []string{"private_key", "passphrase"} {
		t.Run(secretField, func(t *testing.T) {
			query, args, err := buildUpdateQuery(map[string]interface{}{secretField: "plaintext-attempt"}, uuid.New())

			require.Error(t, err, "UpdateItem must reject any attempt to set the T10-encrypted %q column", secretField)
			require.Empty(t, query)
			require.Nil(t, args)
		})
	}

	// Also assert directly against the allow-list table itself, so a
	// future edit that accidentally adds a secret column is caught even
	// if the rejection-path test above is ever weakened.
	_, privateKeyAllowed := allowedUpdateColumns["private_key"]
	_, passphraseAllowed := allowedUpdateColumns["passphrase"]
	require.False(t, privateKeyAllowed, "private_key must never be in UpdateItem's allow-list (T10)")
	require.False(t, passphraseAllowed, "passphrase must never be in UpdateItem's allow-list (T10)")
}

// TestGREEN_BuildUpdateQuery_AllowedFieldsProduceExpectedSQL proves the
// happy path still works exactly as before for every field the real
// handler actually uses, with values fully parameterized.
func TestGREEN_BuildUpdateQuery_AllowedFieldsProduceExpectedSQL(t *testing.T) {
	id := uuid.New()

	query, args, err := buildUpdateQuery(map[string]interface{}{
		"name":       "new-name",
		"public_key": "ssh-ed25519 AAAA...",
		"metadata":   map[string]interface{}{"k": "v"},
		"tags":       []string{"a", "b"},
	}, id)

	require.NoError(t, err)
	require.NotEmpty(t, query)
	require.Contains(t, query, "metadata = $")
	require.Contains(t, query, "name = $")
	require.Contains(t, query, "public_key = $")
	require.Contains(t, query, "tags = $")
	require.Contains(t, query, "updated_at = $")
	require.Contains(t, query, "WHERE id = $")
	require.Contains(t, query, "AND deleted_at IS NULL")
	// 4 field values + updated_at + id = 6 positional args.
	require.Len(t, args, 6)
	require.Equal(t, id, args[len(args)-1])
}

// TestGREEN_BuildUpdateQuery_EmptyUpdatesIsNoOp preserves the existing
// no-op contract (mirrors UpdateItem's own len(updates) == 0 guard).
func TestGREEN_BuildUpdateQuery_EmptyUpdatesIsNoOp(t *testing.T) {
	query, args, err := buildUpdateQuery(map[string]interface{}{}, uuid.New())

	require.NoError(t, err)
	require.Empty(t, query)
	require.Nil(t, args)
}
