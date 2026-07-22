package billing

// Internal (white-box) unit tests for StripeProvider. Constitution
// §11.4.27(A): a hand-written fake of the narrow stripeClient interface
// is permitted here because this is a unit-test source file — every
// other test type (integration/e2e/stress/chaos) that exercises
// StripeProvider MUST do so against the real Stripe API (see
// stripe_provider_integration_test.go, //go:build integration).
//
// The webhook-signature tests below deliberately do NOT fake Stripe's
// signature algorithm — they compute a real HMAC-SHA256 signature via
// the Stripe SDK's own exported webhook.ComputeSignature and verify
// StripeProvider.VerifyWebhook accepts it, and that a tampered payload
// is rejected. That is a real cryptographic proof, not a bluff.

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

// fakeStripeClient is a hand-written test double for the unexported
// stripeClient interface — permitted here per §11.4.27(A) (unit-test
// source only). It records every call it receives so tests can assert
// StripeProvider constructed the RIGHT parameters, not just that SOME
// call happened.
type fakeStripeClient struct {
	createCustomerFn       func(ctx context.Context, params *stripe.CustomerCreateParams) (*stripe.Customer, error)
	createSubscriptionFn   func(ctx context.Context, params *stripe.SubscriptionCreateParams) (*stripe.Subscription, error)
	retrieveSubscriptionFn func(ctx context.Context, id string) (*stripe.Subscription, error)
	updateSubscriptionFn   func(ctx context.Context, id string, params *stripe.SubscriptionUpdateParams) (*stripe.Subscription, error)
	cancelSubscriptionFn   func(ctx context.Context, id string, params *stripe.SubscriptionCancelParams) (*stripe.Subscription, error)

	createCustomerCalls     []*stripe.CustomerCreateParams
	createSubscriptionCalls []*stripe.SubscriptionCreateParams
	updateSubscriptionCalls []*stripe.SubscriptionUpdateParams
}

func (f *fakeStripeClient) CreateCustomer(ctx context.Context, params *stripe.CustomerCreateParams) (*stripe.Customer, error) {
	f.createCustomerCalls = append(f.createCustomerCalls, params)
	return f.createCustomerFn(ctx, params)
}

func (f *fakeStripeClient) CreateSubscription(ctx context.Context, params *stripe.SubscriptionCreateParams) (*stripe.Subscription, error) {
	f.createSubscriptionCalls = append(f.createSubscriptionCalls, params)
	return f.createSubscriptionFn(ctx, params)
}

func (f *fakeStripeClient) RetrieveSubscription(ctx context.Context, id string) (*stripe.Subscription, error) {
	return f.retrieveSubscriptionFn(ctx, id)
}

func (f *fakeStripeClient) UpdateSubscription(ctx context.Context, id string, params *stripe.SubscriptionUpdateParams) (*stripe.Subscription, error) {
	f.updateSubscriptionCalls = append(f.updateSubscriptionCalls, params)
	return f.updateSubscriptionFn(ctx, id, params)
}

func (f *fakeStripeClient) CancelSubscription(ctx context.Context, id string, params *stripe.SubscriptionCancelParams) (*stripe.Subscription, error) {
	return f.cancelSubscriptionFn(ctx, id, params)
}

func newTestProvider(fc *fakeStripeClient, webhookSecret string) *StripeProvider {
	return &StripeProvider{client: fc, webhookSecret: webhookSecret}
}

// TestStripeProvider_CreateSubscription_CreatesCustomerWhenAbsent proves
// CreateSubscription creates a real customer (via the client) when no
// ExistingCustomerID is supplied, and uses the returned customer id to
// create the subscription.
func TestStripeProvider_CreateSubscription_CreatesCustomerWhenAbsent(t *testing.T) {
	fc := &fakeStripeClient{
		createCustomerFn: func(ctx context.Context, params *stripe.CustomerCreateParams) (*stripe.Customer, error) {
			return &stripe.Customer{ID: "cus_new123"}, nil
		},
		createSubscriptionFn: func(ctx context.Context, params *stripe.SubscriptionCreateParams) (*stripe.Subscription, error) {
			if params.Customer == nil || *params.Customer != "cus_new123" {
				t.Fatalf("expected subscription create to target the newly-created customer, got %#v", params.Customer)
			}
			if len(params.Items) != 1 || params.Items[0].Price == nil || *params.Items[0].Price != "price_abc" {
				t.Fatalf("expected a single item with price_abc, got %#v", params.Items)
			}
			return &stripe.Subscription{ID: "sub_new123", Status: stripe.SubscriptionStatusActive}, nil
		},
	}
	p := newTestProvider(fc, "")

	result, err := p.CreateSubscription(context.Background(), CreateSubscriptionInput{
		OrgID:   "org-1",
		PriceID: "price_abc",
	})
	if err != nil {
		t.Fatalf("CreateSubscription returned error: %v", err)
	}
	if result.ExternalCustomerID != "cus_new123" {
		t.Errorf("expected ExternalCustomerID cus_new123, got %s", result.ExternalCustomerID)
	}
	if result.ExternalSubscriptionID != "sub_new123" {
		t.Errorf("expected ExternalSubscriptionID sub_new123, got %s", result.ExternalSubscriptionID)
	}
	if result.Status != "active" {
		t.Errorf("expected Status active, got %s", result.Status)
	}
	if len(fc.createCustomerCalls) != 1 {
		t.Errorf("expected exactly 1 customer-create call, got %d", len(fc.createCustomerCalls))
	}
}

// TestStripeProvider_CreateSubscription_ReusesExistingCustomer proves
// CreateSubscription does NOT create a duplicate customer when
// ExistingCustomerID is supplied.
func TestStripeProvider_CreateSubscription_ReusesExistingCustomer(t *testing.T) {
	fc := &fakeStripeClient{
		createCustomerFn: func(ctx context.Context, params *stripe.CustomerCreateParams) (*stripe.Customer, error) {
			t.Fatal("customer creation must NOT be called when ExistingCustomerID is supplied")
			return nil, nil
		},
		createSubscriptionFn: func(ctx context.Context, params *stripe.SubscriptionCreateParams) (*stripe.Subscription, error) {
			if params.Customer == nil || *params.Customer != "cus_existing" {
				t.Fatalf("expected subscription create to target the existing customer, got %#v", params.Customer)
			}
			return &stripe.Subscription{ID: "sub_x", Status: stripe.SubscriptionStatusActive}, nil
		},
	}
	p := newTestProvider(fc, "")

	result, err := p.CreateSubscription(context.Background(), CreateSubscriptionInput{
		OrgID:              "org-1",
		PriceID:            "price_abc",
		ExistingCustomerID: "cus_existing",
	})
	if err != nil {
		t.Fatalf("CreateSubscription returned error: %v", err)
	}
	if result.ExternalCustomerID != "cus_existing" {
		t.Errorf("expected ExternalCustomerID cus_existing, got %s", result.ExternalCustomerID)
	}
	if len(fc.createCustomerCalls) != 0 {
		t.Errorf("expected 0 customer-create calls, got %d", len(fc.createCustomerCalls))
	}
}

// TestStripeProvider_CreateSubscription_HonestStatusPassthrough proves
// a non-active Stripe status (e.g. "incomplete") is passed through
// verbatim, never silently upgraded to "active" — the exact class of
// fabrication this package exists to prevent.
func TestStripeProvider_CreateSubscription_HonestStatusPassthrough(t *testing.T) {
	fc := &fakeStripeClient{
		createCustomerFn: func(ctx context.Context, params *stripe.CustomerCreateParams) (*stripe.Customer, error) {
			return &stripe.Customer{ID: "cus_1"}, nil
		},
		createSubscriptionFn: func(ctx context.Context, params *stripe.SubscriptionCreateParams) (*stripe.Subscription, error) {
			return &stripe.Subscription{ID: "sub_1", Status: stripe.SubscriptionStatusIncomplete}, nil
		},
	}
	p := newTestProvider(fc, "")

	result, err := p.CreateSubscription(context.Background(), CreateSubscriptionInput{OrgID: "org-1", PriceID: "price_abc"})
	if err != nil {
		t.Fatalf("CreateSubscription returned error: %v", err)
	}
	if result.Status != "incomplete" {
		t.Fatalf("expected honest status passthrough 'incomplete', got %q — a bluff would silently report 'active'", result.Status)
	}
}

// TestStripeProvider_CreateSubscription_PropagatesProcessorError proves
// a real processor-side failure (e.g. invalid price) surfaces as an
// error rather than being swallowed into a fabricated success.
func TestStripeProvider_CreateSubscription_PropagatesProcessorError(t *testing.T) {
	wantErr := errors.New("resource_missing: no such price")
	fc := &fakeStripeClient{
		createCustomerFn: func(ctx context.Context, params *stripe.CustomerCreateParams) (*stripe.Customer, error) {
			return &stripe.Customer{ID: "cus_1"}, nil
		},
		createSubscriptionFn: func(ctx context.Context, params *stripe.SubscriptionCreateParams) (*stripe.Subscription, error) {
			return nil, wantErr
		},
	}
	p := newTestProvider(fc, "")

	_, err := p.CreateSubscription(context.Background(), CreateSubscriptionInput{OrgID: "org-1", PriceID: "price_bad"})
	if err == nil {
		t.Fatal("expected an error when the processor rejects the subscription create, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped processor error, got: %v", err)
	}
}

// TestStripeProvider_UpdateSubscription_ReplacesExistingItem proves
// UpdateSubscription targets the EXISTING subscription item's id
// (retrieved first) rather than blindly appending a bare price, which
// would add a second item instead of changing the plan (see the
// UpdateSubscription doc comment in stripe_provider.go for the Stripe
// API semantics this guards against).
func TestStripeProvider_UpdateSubscription_ReplacesExistingItem(t *testing.T) {
	fc := &fakeStripeClient{
		retrieveSubscriptionFn: func(ctx context.Context, id string) (*stripe.Subscription, error) {
			if id != "sub_1" {
				t.Fatalf("expected retrieve for sub_1, got %s", id)
			}
			return &stripe.Subscription{
				ID: "sub_1",
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{{ID: "si_existing"}},
				},
			}, nil
		},
		updateSubscriptionFn: func(ctx context.Context, id string, params *stripe.SubscriptionUpdateParams) (*stripe.Subscription, error) {
			if id != "sub_1" {
				t.Fatalf("expected update for sub_1, got %s", id)
			}
			if len(params.Items) != 1 {
				t.Fatalf("expected exactly 1 item in update params, got %d", len(params.Items))
			}
			item := params.Items[0]
			if item.ID == nil || *item.ID != "si_existing" {
				t.Fatalf("expected update to target existing item id si_existing, got %#v — a bare price would ADD a second item instead of replacing the price", item.ID)
			}
			if item.Price == nil || *item.Price != "price_new" {
				t.Fatalf("expected new price price_new, got %#v", item.Price)
			}
			return &stripe.Subscription{ID: "sub_1", Status: stripe.SubscriptionStatusActive, Customer: &stripe.Customer{ID: "cus_1"}}, nil
		},
	}
	p := newTestProvider(fc, "")

	result, err := p.UpdateSubscription(context.Background(), UpdateSubscriptionInput{
		ExternalSubscriptionID: "sub_1",
		NewPriceID:             "price_new",
	})
	if err != nil {
		t.Fatalf("UpdateSubscription returned error: %v", err)
	}
	if result.Status != "active" {
		t.Errorf("expected status active, got %s", result.Status)
	}
}

// TestStripeProvider_CancelSubscription_CallsProcessor proves
// CancelSubscription calls the real processor cancel endpoint and
// returns its real resulting status.
func TestStripeProvider_CancelSubscription_CallsProcessor(t *testing.T) {
	fc := &fakeStripeClient{
		cancelSubscriptionFn: func(ctx context.Context, id string, params *stripe.SubscriptionCancelParams) (*stripe.Subscription, error) {
			if id != "sub_1" {
				t.Fatalf("expected cancel for sub_1, got %s", id)
			}
			return &stripe.Subscription{ID: "sub_1", Status: stripe.SubscriptionStatusCanceled, Customer: &stripe.Customer{ID: "cus_1"}}, nil
		},
	}
	p := newTestProvider(fc, "")

	result, err := p.CancelSubscription(context.Background(), CancelSubscriptionInput{ExternalSubscriptionID: "sub_1"})
	if err != nil {
		t.Fatalf("CancelSubscription returned error: %v", err)
	}
	if result.Status != "canceled" {
		t.Errorf("expected status canceled, got %s", result.Status)
	}
}

// ---------------------------------------------------------------------
// VerifyWebhook — real cryptographic signature verification, no fakes.
// ---------------------------------------------------------------------

// TestStripeProvider_VerifyWebhook_AcceptsGenuineSignature computes a
// REAL Stripe webhook signature (via the SDK's own exported
// webhook.ComputeSignature — the same algorithm Stripe's servers use)
// and proves StripeProvider.VerifyWebhook accepts it. This is a real
// cryptographic round-trip, not a stubbed comparison.
func TestStripeProvider_VerifyWebhook_AcceptsGenuineSignature(t *testing.T) {
	const secret = "whsec_test_secret_0123456789"
	payload := []byte(`{"id":"evt_test123","object":"event","type":"customer.subscription.updated"}`)

	now := time.Now()
	sig := webhook.ComputeSignature(now, payload, secret)
	header := "t=" + strconv.FormatInt(now.Unix(), 10) + ",v1=" + hex.EncodeToString(sig)

	p := newTestProvider(&fakeStripeClient{}, secret)
	event, err := p.VerifyWebhook(payload, header)
	if err != nil {
		t.Fatalf("expected genuine signature to verify, got error: %v", err)
	}
	if event.ID != "evt_test123" {
		t.Errorf("expected event id evt_test123, got %s", event.ID)
	}
	if event.Type != "customer.subscription.updated" {
		t.Errorf("expected event type customer.subscription.updated, got %s", event.Type)
	}
}

// TestStripeProvider_VerifyWebhook_RejectsTamperedPayload proves a
// payload that does NOT match its signature (tampered after signing)
// is rejected — the exact scenario webhook verification exists to
// catch (an attacker POSTing a forged event to the webhook endpoint).
func TestStripeProvider_VerifyWebhook_RejectsTamperedPayload(t *testing.T) {
	const secret = "whsec_test_secret_0123456789"
	original := []byte(`{"id":"evt_test123","object":"event","type":"customer.subscription.updated"}`)
	tampered := []byte(`{"id":"evt_test123","object":"event","type":"customer.subscription.deleted"}`)

	now := time.Now()
	sig := webhook.ComputeSignature(now, original, secret) // signed over the ORIGINAL payload
	header := "t=" + strconv.FormatInt(now.Unix(), 10) + ",v1=" + hex.EncodeToString(sig)

	p := newTestProvider(&fakeStripeClient{}, secret)
	_, err := p.VerifyWebhook(tampered, header) // verified against the TAMPERED payload
	if err == nil {
		t.Fatal("expected tampered payload to be rejected, got no error")
	}
}

// TestStripeProvider_VerifyWebhook_NoSecretConfiguredFailsClosed proves
// a deployment with no STRIPE_WEBHOOK_SECRET configured never trusts
// ANY payload, however well-formed the signature header looks — fails
// closed, never open.
func TestStripeProvider_VerifyWebhook_NoSecretConfiguredFailsClosed(t *testing.T) {
	p := newTestProvider(&fakeStripeClient{}, "") // no webhook secret configured

	_, err := p.VerifyWebhook([]byte(`{"id":"evt_x"}`), "t=1,v1=deadbeef")
	if err == nil {
		t.Fatal("expected VerifyWebhook to fail closed when no webhook secret is configured")
	}
}
