package billing_test

import (
	"testing"

	"github.com/helixdevelopment/billing-service/internal/billing"
)

// TestNewProviderFromEnv_AbsentKeyReturnsNilNotError is the honest-501
// RED->GREEN proof at the lowest layer: with STRIPE_SECRET_KEY unset,
// NewProviderFromEnv MUST return (nil, nil) — nil provider, no error —
// never construct a StripeProvider and never fabricate any kind of
// "configured" signal. internal/handler and internal/server rely on
// this nil to trigger the honest 501 response (see
// internal/handler/handler_test.go TestCreateSubscription_NoProvider_Returns501).
func TestNewProviderFromEnv_AbsentKeyReturnsNilNotError(t *testing.T) {
	t.Setenv(billing.EnvStripeSecretKey, "")
	t.Setenv(billing.EnvStripeWebhookSecret, "")

	p, err := billing.NewProviderFromEnv()
	if err != nil {
		t.Fatalf("expected no error when STRIPE_SECRET_KEY is unset, got: %v", err)
	}
	if p != nil {
		t.Fatalf("expected nil provider when STRIPE_SECRET_KEY is unset, got: %#v", p)
	}
}

// TestNewProviderFromEnv_PresentKeyReturnsStripeProvider proves the
// other half of the honest feature-flag: when STRIPE_SECRET_KEY IS
// present, NewProviderFromEnv returns a real, non-nil *StripeProvider
// (construction alone requires no network call — the Stripe SDK client
// is a pure in-memory value until a method is invoked, so this
// assertion is deterministic and network-free).
func TestNewProviderFromEnv_PresentKeyReturnsStripeProvider(t *testing.T) {
	t.Setenv(billing.EnvStripeSecretKey, "sk_test_fake_key_for_construction_only")
	t.Setenv(billing.EnvStripeWebhookSecret, "whsec_fake_secret_for_construction_only")

	p, err := billing.NewProviderFromEnv()
	if err != nil {
		t.Fatalf("expected no error when STRIPE_SECRET_KEY is set, got: %v", err)
	}
	if p == nil {
		t.Fatal("expected a non-nil provider when STRIPE_SECRET_KEY is set")
	}
	if p.Name() != "stripe" {
		t.Fatalf("expected provider name %q, got %q", "stripe", p.Name())
	}
	if _, ok := p.(*billing.StripeProvider); !ok {
		t.Fatalf("expected *billing.StripeProvider, got %T", p)
	}
}
