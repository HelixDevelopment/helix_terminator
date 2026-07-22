package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/helixdevelopment/billing-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles billing data access
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new Repository
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// checkPool reports the honest "database not connected" error for
// EITHER a Repository constructed with a nil pool (repository.New(nil))
// OR a nil *Repository itself (a caller that passed a literal nil in
// place of a *repository.Repository — e.g. handler.New(nil) across
// several test files). Constitution §11.4.108 (pre-existing, unrelated
// defect discovered while validating this stream's own changes, per
// §11.4.124 investigate-before-touching): before this fix, a nil
// *Repository receiver made this method dereference a nil pointer
// (`r.pool` on a nil `r`) and PANIC rather than return the honest
// error every OTHER repository method already expects checkPool to
// produce — captured evidence: `go test -tags chaos ./internal/handler/`
// crashed with "invalid memory address or nil pointer dereference"
// inside checkPool via ListSubscriptions/UpdateSubscription BEFORE this
// fix, using the exact handler.New(nil) construction
// internal/handler/handler_test.go's TestHealthCheck (etc.) already
// relies on elsewhere in this package without incident (those callers
// never happened to reach a repo method — this one does).
func (r *Repository) checkPool() error {
	if r == nil || r.pool == nil {
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

// CreateSubscription creates a new subscription. sub.Provider,
// sub.ExternalSubscriptionID and sub.ExternalCustomerID (migration
// 002_payment_provider) are persisted verbatim — the caller
// (internal/handler) is responsible for populating them from a REAL
// PaymentProvider result (or leaving them at their zero values / "none"
// only in the defensive fallback path a configured provider makes
// structurally unreachable; see internal/billing/provider.go).
func (r *Repository) CreateSubscription(ctx context.Context, sub *model.Subscription) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	provider := sub.Provider
	if provider == "" {
		provider = "none"
	}
	query := `
		INSERT INTO subscriptions (id, org_id, plan_id, status, started_at, provider, external_subscription_id, external_customer_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, sub.ID, sub.OrgID, sub.PlanID, sub.Status, sub.StartedAt,
		provider, nullableString(sub.ExternalSubscriptionID), nullableString(sub.ExternalCustomerID))
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	return nil
}

// GetSubscriptionByID retrieves a subscription by ID
func (r *Repository) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, org_id, plan_id, status, started_at, ends_at, canceled_at, created_at, updated_at,
		       provider, external_subscription_id, external_customer_id
		FROM subscriptions WHERE id = $1
	`
	var sub model.Subscription
	var externalSubID, externalCustID sql.NullString
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&sub.ID, &sub.OrgID, &sub.PlanID, &sub.Status, &sub.StartedAt, &sub.EndsAt, &sub.CanceledAt, &sub.CreatedAt, &sub.UpdatedAt,
		&sub.Provider, &externalSubID, &externalCustID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, err
	}
	sub.ExternalSubscriptionID = externalSubID.String
	sub.ExternalCustomerID = externalCustID.String
	return &sub, nil
}

// GetLatestExternalCustomerID looks up the most-recently-created
// processor-side customer id this org already has on file for the
// given provider (e.g. "stripe"), so CreateSubscription can reuse an
// existing customer record instead of creating a duplicate one for
// every subscription the same org creates. Returns ("", nil) — not an
// error — when the org has no prior subscription with that provider.
func (r *Repository) GetLatestExternalCustomerID(ctx context.Context, orgID uuid.UUID, provider string) (string, error) {
	if err := r.checkPool(); err != nil {
		return "", err
	}
	query := `
		SELECT external_customer_id FROM subscriptions
		WHERE org_id = $1 AND provider = $2 AND external_customer_id IS NOT NULL
		ORDER BY created_at DESC
		LIMIT 1
	`
	var customerID sql.NullString
	err := r.pool.QueryRow(ctx, query, orgID, provider).Scan(&customerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return customerID.String, nil
}

// ListSubscriptions retrieves subscriptions with filtering
func (r *Repository) ListSubscriptions(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*model.Subscription, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	whereClause := "1=1"
	var args []interface{}
	argIdx := 1

	if orgID != uuid.Nil {
		whereClause += fmt.Sprintf(" AND org_id = $%d", argIdx)
		args = append(args, orgID)
		argIdx++
	}
	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM subscriptions WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, org_id, plan_id, status, started_at, ends_at, canceled_at, created_at, updated_at,
		       provider, external_subscription_id, external_customer_id
		FROM subscriptions WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var sub model.Subscription
		var externalSubID, externalCustID sql.NullString
		if err := rows.Scan(
			&sub.ID, &sub.OrgID, &sub.PlanID, &sub.Status, &sub.StartedAt, &sub.EndsAt, &sub.CanceledAt, &sub.CreatedAt, &sub.UpdatedAt,
			&sub.Provider, &externalSubID, &externalCustID,
		); err != nil {
			return nil, 0, err
		}
		sub.ExternalSubscriptionID = externalSubID.String
		sub.ExternalCustomerID = externalCustID.String
		subs = append(subs, &sub)
	}
	return subs, total, rows.Err()
}

// UpdateSubscription updates a subscription, scoped to the owning org
// (T14). The WHERE clause filters on BOTH id AND org_id so a caller can
// never mutate another tenant's subscription — even if the caller's own
// ownership check at the handler layer were ever bypassed or raced, the
// UPDATE itself cannot touch a row belonging to a different org. A
// mismatch (row exists but belongs to a different org, or the row does
// not exist at all) produces the identical zero-rows-affected outcome,
// so the caller (internal/handler) can return the same "subscription
// not found" response either way — no existence oracle.
func (r *Repository) UpdateSubscription(ctx context.Context, id, orgID uuid.UUID, updates map[string]interface{}) error {
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
	idIdx := argIdx
	argIdx++
	args = append(args, orgID)
	orgIdx := argIdx

	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE id = $%d AND org_id = $%d", joinSetClauses(setClauses), idIdx, orgIdx)
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found")
	}
	return nil
}

// CancelSubscription cancels a subscription, scoped to the owning org
// (T14) — same org_id-scoped WHERE clause rationale as UpdateSubscription.
// status is the REAL status string returned by the configured
// PaymentProvider's CancelSubscription call (Constitution §11.4 —
// persisting a hardcoded 'canceled' literal regardless of what the
// processor actually reported would be the exact class of fabrication
// this service is being fixed for; Stripe, for example, can return
// "canceled" immediately or leave a subscription "active" with
// cancel_at_period_end set, depending on how the cancel was requested).
func (r *Repository) CancelSubscription(ctx context.Context, id, orgID uuid.UUID, status string) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE subscriptions SET status = $3, canceled_at = NOW(), updated_at = NOW() WHERE id = $1 AND org_id = $2"
	result, err := r.pool.Exec(ctx, query, id, orgID, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found")
	}
	return nil
}

// UpdateSubscriptionStatusByExternalID reconciles a subscription's
// locally-stored status from a REAL event a configured PaymentProvider
// reported (Constitution §11.4 — webhook-driven reconciliation, see
// internal/handler.StripeWebhook + internal/billing.ParseSubscriptionObject).
// Deliberately NOT org-scoped: webhook events are authenticated by
// their processor signature (verified before this is ever called), not
// by a caller-supplied org id, and (provider, external_subscription_id)
// is unique per processor subscription. A processor subscription id
// this billing-service instance has no matching row for is a no-op,
// not an error — the event may belong to a different environment or a
// subscription this instance never created.
func (r *Repository) UpdateSubscriptionStatusByExternalID(ctx context.Context, provider, externalSubscriptionID, status string) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	if externalSubscriptionID == "" || status == "" {
		return nil
	}
	query := "UPDATE subscriptions SET status = $3, updated_at = NOW() WHERE provider = $1 AND external_subscription_id = $2"
	_, err := r.pool.Exec(ctx, query, provider, externalSubscriptionID, status)
	if err != nil {
		return fmt.Errorf("failed to reconcile subscription status from webhook: %w", err)
	}
	return nil
}

// CreateInvoice creates a new invoice
func (r *Repository) CreateInvoice(ctx context.Context, inv *model.Invoice) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO invoices (id, org_id, subscription_id, amount_cents, currency, status, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, inv.ID, inv.OrgID, inv.SubscriptionID, inv.AmountCents, inv.Currency, inv.Status, inv.DueDate)
	if err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}
	return nil
}

// GetInvoiceByID retrieves an invoice by ID
func (r *Repository) GetInvoiceByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error) {
	if err := r.checkPool(); err != nil {
		return nil, err
	}
	query := `
		SELECT id, org_id, subscription_id, amount_cents, currency, status, due_date, paid_at, created_at, updated_at
		FROM invoices WHERE id = $1
	`
	var inv model.Invoice
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.OrgID, &inv.SubscriptionID, &inv.AmountCents, &inv.Currency, &inv.Status, &inv.DueDate, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("invoice not found")
		}
		return nil, err
	}
	return &inv, nil
}

// ListInvoices retrieves invoices for an org
func (r *Repository) ListInvoices(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*model.Invoice, int, error) {
	if err := r.checkPool(); err != nil {
		return nil, 0, err
	}
	countQuery := "SELECT COUNT(*) FROM invoices WHERE org_id = $1"
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, orgID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, org_id, subscription_id, amount_cents, currency, status, due_date, paid_at, created_at, updated_at
		FROM invoices WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var invoices []*model.Invoice
	for rows.Next() {
		var inv model.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.OrgID, &inv.SubscriptionID, &inv.AmountCents, &inv.Currency, &inv.Status, &inv.DueDate, &inv.PaidAt, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		invoices = append(invoices, &inv)
	}
	return invoices, total, rows.Err()
}

// RecordUsage creates a usage record
func (r *Repository) RecordUsage(ctx context.Context, usage *model.UsageRecord) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO usage_records (id, org_id, resource_type, quantity, period_start, period_end, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.pool.Exec(ctx, query, usage.ID, usage.OrgID, usage.ResourceType, usage.Quantity, usage.PeriodStart, usage.PeriodEnd)
	if err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
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

// nullableString converts an empty Go string to SQL NULL (rather than
// an empty-string value) for the nullable external_subscription_id /
// external_customer_id columns — an empty processor-side id is not a
// meaningful value, it means "no processor was involved", which the
// column's NULL-ness should represent honestly.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
