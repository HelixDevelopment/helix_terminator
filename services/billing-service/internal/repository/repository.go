package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/helixdevelopment/billing-service/internal/model"
)

// Repository handles billing data access
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

// CreateSubscription creates a new subscription
func (r *Repository) CreateSubscription(ctx context.Context, sub *model.Subscription) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := `
		INSERT INTO subscriptions (id, org_id, plan_id, status, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query, sub.ID, sub.OrgID, sub.PlanID, sub.Status, sub.StartedAt)
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
		SELECT id, org_id, plan_id, status, started_at, ends_at, canceled_at, created_at, updated_at
		FROM subscriptions WHERE id = $1
	`
	var sub model.Subscription
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&sub.ID, &sub.OrgID, &sub.PlanID, &sub.Status, &sub.StartedAt, &sub.EndsAt, &sub.CanceledAt, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, err
	}
	return &sub, nil
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
		SELECT id, org_id, plan_id, status, started_at, ends_at, canceled_at, created_at, updated_at
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
		if err := rows.Scan(
			&sub.ID, &sub.OrgID, &sub.PlanID, &sub.Status, &sub.StartedAt, &sub.EndsAt, &sub.CanceledAt, &sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		subs = append(subs, &sub)
	}
	return subs, total, rows.Err()
}

// UpdateSubscription updates a subscription
func (r *Repository) UpdateSubscription(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
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

	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE id = $%d", joinSetClauses(setClauses), argIdx)
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found")
	}
	return nil
}

// CancelSubscription cancels a subscription
func (r *Repository) CancelSubscription(ctx context.Context, id uuid.UUID) error {
	if err := r.checkPool(); err != nil {
		return err
	}
	query := "UPDATE subscriptions SET status = 'canceled', canceled_at = NOW(), updated_at = NOW() WHERE id = $1"
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found")
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
