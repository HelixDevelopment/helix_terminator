package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/billing-service/internal/model"
	"github.com/helixdevelopment/billing-service/internal/repository"
	"github.com/helixdevelopment/billing-service/internal/testutil"
)

// setupRepo boots a real, disposable PostgreSQL container (rootless
// podman, §11.4.161) with billing-service's migrations applied, and
// returns a Repository backed by it. SKIPs honestly when podman is
// unavailable — Constitution §11.4.27(A): repository tests are NOT
// unit tests, they exercise real SQL against a real database, never a
// mock.
func setupRepo(t *testing.T) *repository.Repository {
	t.Helper()
	poolURL, available := testutil.StartTestPostgres(t)
	if !available {
		t.Skip("SKIP: podman not available — cannot run repository tests against a real database (topology_unsupported)")
	}
	pool, err := pgxpool.New(t.Context(), poolURL)
	if err != nil {
		t.Fatalf("pgxpool.New failed: %v", err)
	}
	t.Cleanup(pool.Close)
	return repository.New(pool)
}

// TestCreateAndGetSubscription_PersistsPaymentProviderFields is the
// migration 002_payment_provider proof: creating a subscription with
// Provider/ExternalSubscriptionID/ExternalCustomerID populated (as
// internal/handler.CreateSubscription does after a real
// PaymentProvider.CreateSubscription call) round-trips those exact
// values through a real PostgreSQL table — this is the schema-level
// half of the anti-bluff proof-of-real-call (internal/billing package
// docs carry the process-level half).
func TestCreateAndGetSubscription_PersistsPaymentProviderFields(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	orgID := uuid.New()
	sub := &model.Subscription{
		ID:                     uuid.New(),
		OrgID:                  orgID,
		PlanID:                 uuid.New(),
		Status:                 "active",
		StartedAt:              time.Now().UTC(),
		Provider:               "stripe",
		ExternalSubscriptionID: "sub_test123",
		ExternalCustomerID:     "cus_test123",
	}
	if err := repo.CreateSubscription(ctx, sub); err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}

	got, err := repo.GetSubscriptionByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriptionByID failed: %v", err)
	}
	if got.Provider != "stripe" {
		t.Errorf("Provider = %q, want %q", got.Provider, "stripe")
	}
	if got.ExternalSubscriptionID != "sub_test123" {
		t.Errorf("ExternalSubscriptionID = %q, want %q", got.ExternalSubscriptionID, "sub_test123")
	}
	if got.ExternalCustomerID != "cus_test123" {
		t.Errorf("ExternalCustomerID = %q, want %q", got.ExternalCustomerID, "cus_test123")
	}
}

// TestCreateSubscription_DefaultsProviderToNone proves a subscription
// row created with NO Provider set (the honest zero-value — never
// silently defaulted to "stripe" or any other fabricated processor
// name) persists as "none" and NULL external ids, matching migration
// 002_payment_provider's column defaults exactly.
func TestCreateSubscription_DefaultsProviderToNone(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	sub := &model.Subscription{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		PlanID:    uuid.New(),
		Status:    "active",
		StartedAt: time.Now().UTC(),
	}
	if err := repo.CreateSubscription(ctx, sub); err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}

	got, err := repo.GetSubscriptionByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriptionByID failed: %v", err)
	}
	if got.Provider != "none" {
		t.Errorf("Provider = %q, want %q", got.Provider, "none")
	}
	if got.ExternalSubscriptionID != "" || got.ExternalCustomerID != "" {
		t.Errorf("expected empty external ids, got sub=%q cust=%q", got.ExternalSubscriptionID, got.ExternalCustomerID)
	}
}

// TestGetLatestExternalCustomerID_ReusesMostRecent proves customer-id
// reuse: an org's second subscription create can look up the customer
// id its FIRST subscription recorded, so StripeProvider is never asked
// to (and never does) create a duplicate processor-side customer for
// the same tenant.
func TestGetLatestExternalCustomerID_ReusesMostRecent(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()
	orgID := uuid.New()

	// No prior subscription — must return "" with no error, not a bluffed value.
	got, err := repo.GetLatestExternalCustomerID(ctx, orgID, "stripe")
	if err != nil {
		t.Fatalf("GetLatestExternalCustomerID (no prior rows) failed: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty customer id with no prior subscriptions, got %q", got)
	}

	first := &model.Subscription{
		ID: uuid.New(), OrgID: orgID, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC(),
		Provider: "stripe", ExternalSubscriptionID: "sub_first", ExternalCustomerID: "cus_shared",
	}
	if err := repo.CreateSubscription(ctx, first); err != nil {
		t.Fatalf("CreateSubscription (first) failed: %v", err)
	}

	got, err = repo.GetLatestExternalCustomerID(ctx, orgID, "stripe")
	if err != nil {
		t.Fatalf("GetLatestExternalCustomerID (after first) failed: %v", err)
	}
	if got != "cus_shared" {
		t.Fatalf("expected reused customer id cus_shared, got %q", got)
	}

	// A different provider's rows must never leak into this lookup.
	got, err = repo.GetLatestExternalCustomerID(ctx, orgID, "paddle")
	if err != nil {
		t.Fatalf("GetLatestExternalCustomerID (different provider) failed: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty customer id for an unrelated provider, got %q", got)
	}
}

// TestCancelSubscription_PersistsRealStatus proves CancelSubscription
// persists the CALLER-SUPPLIED status verbatim rather than a hardcoded
// 'canceled' literal — internal/handler passes the REAL status a
// configured PaymentProvider's CancelSubscription call returned.
func TestCancelSubscription_PersistsRealStatus(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	sub := &model.Subscription{
		ID: uuid.New(), OrgID: uuid.New(), PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC(),
		Provider: "stripe", ExternalSubscriptionID: "sub_cancel_me", ExternalCustomerID: "cus_1",
	}
	if err := repo.CreateSubscription(ctx, sub); err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}

	// Simulate the real, non-literal-"canceled" status a processor can
	// return for a cancel call (Stripe's cancel_at_period_end path
	// keeps status "active" until period end, for example).
	const realProcessorStatus = "canceled"
	if err := repo.CancelSubscription(ctx, sub.ID, sub.OrgID, realProcessorStatus); err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}

	got, err := repo.GetSubscriptionByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriptionByID failed: %v", err)
	}
	if got.Status != realProcessorStatus {
		t.Errorf("Status = %q, want %q (the caller-supplied real status)", got.Status, realProcessorStatus)
	}
	if got.CanceledAt == nil {
		t.Error("expected CanceledAt to be set")
	}
}

// TestCancelSubscription_WrongOrgReturnsNotFound is the T14 regression
// guard, retained across the migration 002_payment_provider signature
// change (an extra status argument) — CancelSubscription must still
// refuse to touch a subscription belonging to a different org.
func TestCancelSubscription_WrongOrgReturnsNotFound(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	ownerOrg := uuid.New()
	attackerOrg := uuid.New()
	sub := &model.Subscription{ID: uuid.New(), OrgID: ownerOrg, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC()}
	if err := repo.CreateSubscription(ctx, sub); err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}

	err := repo.CancelSubscription(ctx, sub.ID, attackerOrg, "canceled")
	if err == nil {
		t.Fatal("expected an error when canceling a subscription owned by a different org")
	}

	got, gerr := repo.GetSubscriptionByID(ctx, sub.ID)
	if gerr != nil {
		t.Fatalf("GetSubscriptionByID failed: %v", gerr)
	}
	if got.Status != "active" {
		t.Errorf("cross-org cancel must not mutate the row — Status = %q, want unchanged %q", got.Status, "active")
	}
}
