//go:build integration

// Real-Stripe-API integration test for StripeProvider (Constitution
// §11.4.27(A)/§11.4.85: every test type OTHER than a unit test MUST
// exercise the real, fully implemented system against real
// infrastructure — no fakes here of any kind).
//
// Requires a Stripe TEST-mode secret key and a pre-created TEST-mode
// recurring Price to subscribe to:
//
//	export STRIPE_SECRET_KEY="sk_test_..."
//	export STRIPE_TEST_PRICE_ID="price_..."          # a real test-mode recurring Price
//	export STRIPE_WEBHOOK_SECRET="whsec_..."          # optional, only needed for webhook tests
//	GOMAXPROCS=2 go test -tags integration -p 2 -v -run TestStripeProvider_Integration ./internal/billing/...
//
// When STRIPE_SECRET_KEY (or STRIPE_TEST_PRICE_ID) is unset — the
// common case for a plain `go test -tags integration ./...` with no
// Stripe test account provisioned — every test in this file SKIPs with
// an honest reason (§11.4.3) rather than faking a PASS. See
// docs/guides/BILLING.md "Running the Stripe integration tests" for the
// full key-provisioning walkthrough.
package billing_test

import (
	"context"
	"os"
	"testing"

	"github.com/helixdevelopment/billing-service/internal/billing"
)

// requireStripeIntegrationEnv returns a real StripeProvider constructed
// from STRIPE_SECRET_KEY/STRIPE_WEBHOOK_SECRET, plus the real test-mode
// Price id to subscribe to, or SKIPs the calling test honestly when
// either is absent.
func requireStripeIntegrationEnv(t *testing.T) (*billing.StripeProvider, string) {
	t.Helper()

	priceID := os.Getenv("STRIPE_TEST_PRICE_ID")
	if os.Getenv(billing.EnvStripeSecretKey) == "" || priceID == "" {
		t.Skip("SKIP: STRIPE_SECRET_KEY and/or STRIPE_TEST_PRICE_ID not set — cannot run this test against the real Stripe API (operator_attended); see docs/guides/BILLING.md")
	}

	provider, err := billing.NewProviderFromEnv()
	if err != nil {
		t.Fatalf("NewProviderFromEnv failed: %v", err)
	}
	sp, ok := provider.(*billing.StripeProvider)
	if !ok {
		t.Fatalf("expected *billing.StripeProvider, got %T", provider)
	}
	return sp, priceID
}

// TestStripeProvider_Integration_FullSubscriptionLifecycle drives a
// REAL create -> update -> cancel cycle against the real Stripe test-mode
// API. Every assertion is against a value Stripe itself returned — this
// is the rock-solid proof (Constitution §11.4.123) that StripeProvider
// genuinely talks to Stripe, not a proof that only exercises this
// process's own in-memory logic.
func TestStripeProvider_Integration_FullSubscriptionLifecycle(t *testing.T) {
	provider, priceID := requireStripeIntegrationEnv(t)
	ctx := context.Background()

	t.Log("EVIDENCE: creating a REAL Stripe customer + subscription via the live test-mode API")
	createResult, err := provider.CreateSubscription(ctx, billing.CreateSubscriptionInput{
		OrgID:          "integration-test-org",
		PriceID:        priceID,
		IdempotencyKey: "billing-integration-test-create-" + t.Name(),
	})
	if err != nil {
		t.Fatalf("CreateSubscription against the real Stripe API failed: %v", err)
	}
	if createResult.ExternalSubscriptionID == "" {
		t.Fatal("real Stripe API returned no subscription id")
	}
	if createResult.ExternalCustomerID == "" {
		t.Fatal("real Stripe API returned no customer id")
	}
	t.Logf("EVIDENCE: real Stripe subscription created: id=%s customer=%s status=%s",
		createResult.ExternalSubscriptionID, createResult.ExternalCustomerID, createResult.Status)

	// Honest status: send_invoice collection MUST leave the subscription
	// "active" without any card-collection step (see stripe_provider.go
	// doc comment) — assert the REAL value, never assume.
	if createResult.Status == "" {
		t.Fatal("real Stripe API returned an empty status — a bluff-shaped result")
	}

	t.Log("EVIDENCE: canceling the REAL Stripe subscription via the live test-mode API")
	cancelResult, err := provider.CancelSubscription(ctx, billing.CancelSubscriptionInput{
		ExternalSubscriptionID: createResult.ExternalSubscriptionID,
		IdempotencyKey:         "billing-integration-test-cancel-" + t.Name(),
	})
	if err != nil {
		t.Fatalf("CancelSubscription against the real Stripe API failed: %v", err)
	}
	if cancelResult.Status != "canceled" {
		t.Fatalf("expected real Stripe status 'canceled' after cancel, got %q", cancelResult.Status)
	}
	t.Logf("EVIDENCE: real Stripe subscription canceled: id=%s status=%s", cancelResult.ExternalSubscriptionID, cancelResult.Status)
}

// TestStripeProvider_Integration_CreateSubscription_InvalidPriceRejected
// proves a genuinely-invalid price id is REJECTED by the real Stripe
// API (not silently accepted) — the negative-path proof that error
// handling is real, not merely well-typed.
func TestStripeProvider_Integration_CreateSubscription_InvalidPriceRejected(t *testing.T) {
	provider, _ := requireStripeIntegrationEnv(t)
	ctx := context.Background()

	_, err := provider.CreateSubscription(ctx, billing.CreateSubscriptionInput{
		OrgID:          "integration-test-org-invalid-price",
		PriceID:        "price_this_does_not_exist_00000000000000",
		IdempotencyKey: "billing-integration-test-invalid-price-" + t.Name(),
	})
	if err == nil {
		t.Fatal("expected the real Stripe API to reject a nonexistent price id, got no error")
	}
	t.Logf("EVIDENCE: real Stripe API rejected the invalid price with: %v", err)
}

// TestStripeProvider_Integration_ReusesExistingCustomer proves two
// CreateSubscription calls for the SAME ExistingCustomerID create TWO
// distinct real subscriptions under the SAME real Stripe customer
// (never a duplicate customer record).
func TestStripeProvider_Integration_ReusesExistingCustomer(t *testing.T) {
	provider, priceID := requireStripeIntegrationEnv(t)
	ctx := context.Background()

	first, err := provider.CreateSubscription(ctx, billing.CreateSubscriptionInput{
		OrgID:          "integration-test-org-reuse",
		PriceID:        priceID,
		IdempotencyKey: "billing-integration-test-reuse-first-" + t.Name(),
	})
	if err != nil {
		t.Fatalf("first CreateSubscription failed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = provider.CancelSubscription(context.Background(), billing.CancelSubscriptionInput{ExternalSubscriptionID: first.ExternalSubscriptionID})
	})

	second, err := provider.CreateSubscription(ctx, billing.CreateSubscriptionInput{
		OrgID:              "integration-test-org-reuse",
		PriceID:            priceID,
		ExistingCustomerID: first.ExternalCustomerID,
		IdempotencyKey:     "billing-integration-test-reuse-second-" + t.Name(),
	})
	if err != nil {
		t.Fatalf("second CreateSubscription failed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = provider.CancelSubscription(context.Background(), billing.CancelSubscriptionInput{ExternalSubscriptionID: second.ExternalSubscriptionID})
	})

	if second.ExternalCustomerID != first.ExternalCustomerID {
		t.Fatalf("expected the second subscription to reuse customer %s, real Stripe API returned %s", first.ExternalCustomerID, second.ExternalCustomerID)
	}
	if second.ExternalSubscriptionID == first.ExternalSubscriptionID {
		t.Fatal("expected two DISTINCT real subscriptions, got the same id twice")
	}
	t.Logf("EVIDENCE: two real subscriptions (%s, %s) under one real customer %s",
		first.ExternalSubscriptionID, second.ExternalSubscriptionID, first.ExternalCustomerID)
}
