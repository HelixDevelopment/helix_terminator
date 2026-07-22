package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"
)

// stripeClient is the subset of *stripe.Client's surface StripeProvider
// depends on. Narrowing to an interface (rather than depending on
// *stripe.Client directly) lets unit tests substitute a fake backend at
// the Stripe SDK's own sanctioned extension point (stripe.Backend, via
// stripe.NewClient(key, stripe.WithBackends(...))) without ever faking
// StripeProvider itself — the provider's own logic (parameter
// construction, error wrapping, result mapping) still runs for real.
type stripeClient interface {
	CreateCustomer(ctx context.Context, params *stripe.CustomerCreateParams) (*stripe.Customer, error)
	CreateSubscription(ctx context.Context, params *stripe.SubscriptionCreateParams) (*stripe.Subscription, error)
	RetrieveSubscription(ctx context.Context, id string) (*stripe.Subscription, error)
	UpdateSubscription(ctx context.Context, id string, params *stripe.SubscriptionUpdateParams) (*stripe.Subscription, error)
	CancelSubscription(ctx context.Context, id string, params *stripe.SubscriptionCancelParams) (*stripe.Subscription, error)
}

// realStripeClient adapts *stripe.Client (github.com/stripe/stripe-go/v86)
// to the stripeClient interface.
type realStripeClient struct {
	sc *stripe.Client
}

func (r *realStripeClient) CreateCustomer(ctx context.Context, params *stripe.CustomerCreateParams) (*stripe.Customer, error) {
	return r.sc.V1Customers.Create(ctx, params)
}

func (r *realStripeClient) CreateSubscription(ctx context.Context, params *stripe.SubscriptionCreateParams) (*stripe.Subscription, error) {
	return r.sc.V1Subscriptions.Create(ctx, params)
}

func (r *realStripeClient) RetrieveSubscription(ctx context.Context, id string) (*stripe.Subscription, error) {
	return r.sc.V1Subscriptions.Retrieve(ctx, id, nil)
}

func (r *realStripeClient) UpdateSubscription(ctx context.Context, id string, params *stripe.SubscriptionUpdateParams) (*stripe.Subscription, error) {
	return r.sc.V1Subscriptions.Update(ctx, id, params)
}

func (r *realStripeClient) CancelSubscription(ctx context.Context, id string, params *stripe.SubscriptionCancelParams) (*stripe.Subscription, error) {
	return r.sc.V1Subscriptions.Cancel(ctx, id, params)
}

// StripeProvider is the real, network-calling PaymentProvider
// implementation backed by the official Stripe Go SDK
// (github.com/stripe/stripe-go/v86 — verified current major as of
// 2026-07-22, see docs/guides/BILLING.md "Sources verified" footer).
// It is constructed ONLY when STRIPE_SECRET_KEY is present in the
// process environment (see env.go, NewProviderFromEnv) — its mere
// existence in a running process IS the "a real processor is
// configured" signal internal/handler relies on for the honest
// feature-flag (§11.4 anti-bluff: present → real Stripe calls;
// absent → the process never constructs this type at all, and the
// handler layer reports 501, never a fabricated success).
type StripeProvider struct {
	client        stripeClient
	webhookSecret string
}

// NewStripeProvider constructs a StripeProvider from an already-created
// *stripe.Client and the webhook signing secret used to verify inbound
// Stripe webhook requests. webhookSecret may be empty — VerifyWebhook
// then always fails closed (never verifies a signature against an empty
// secret), which is the correct behaviour for a deployment that has not
// yet configured STRIPE_WEBHOOK_SECRET: webhook-driven reconciliation is
// simply unavailable, honestly, rather than silently accepting
// unverified payloads.
func NewStripeProvider(sc *stripe.Client, webhookSecret string) *StripeProvider {
	return &StripeProvider{client: &realStripeClient{sc: sc}, webhookSecret: webhookSecret}
}

// Name implements PaymentProvider.
func (p *StripeProvider) Name() string { return "stripe" }

// ErrCustomerRequired is returned by CreateSubscription when neither an
// existing processor customer id was supplied by the caller nor could
// one be created — should not occur in normal operation (customer
// creation is attempted automatically when ExistingCustomerID is
// empty), retained as a defensive, explicit failure mode rather than a
// nil-pointer panic.
var ErrCustomerRequired = errors.New("billing: stripe customer id required")

// CreateSubscription implements PaymentProvider. When in.ExistingCustomerID
// is empty it FIRST creates a real Stripe Customer (tagged with the
// caller's OrgID in customer metadata so the processor-side record is
// traceable back to the tenant that owns it), then creates the
// subscription against that customer. Both calls carry a
// processor-native idempotency key derived from in.IdempotencyKey so a
// caller-side retry of the exact same logical request can never create
// two processor-side customers or two processor-side subscriptions.
//
// The subscription is created with CollectionMethod "send_invoice" +
// DaysUntilDue 30 — Stripe finalizes and emails an invoice rather than
// attempting to auto-charge a card on file, which lets a subscription
// become genuinely "active" via the real API without first requiring a
// separate client-side card-collection (Stripe Elements/Checkout) flow.
// This mirrors billing-service's existing invoices table/endpoints
// (GetInvoice/ListInvoices), which already model invoice-based billing
// rather than instant-charge billing. See docs/guides/BILLING.md
// "Collection method" for the documented trade-off and how to switch to
// charge_automatically once a payment-method-collection flow exists.
func (p *StripeProvider) CreateSubscription(ctx context.Context, in CreateSubscriptionInput) (*SubscriptionResult, error) {
	if in.PriceID == "" {
		return nil, errors.New("billing: stripe: PriceID is required")
	}
	if in.OrgID == "" {
		return nil, errors.New("billing: stripe: OrgID is required")
	}

	customerID := in.ExistingCustomerID
	if customerID == "" {
		custParams := &stripe.CustomerCreateParams{
			Description: stripe.String("helix_terminator org " + in.OrgID),
			Metadata:    map[string]string{"org_id": in.OrgID},
		}
		if in.IdempotencyKey != "" {
			custParams.SetIdempotencyKey(in.IdempotencyKey + ":customer")
		}
		cust, err := p.client.CreateCustomer(ctx, custParams)
		if err != nil {
			return nil, fmt.Errorf("billing: stripe: create customer: %w", err)
		}
		if cust == nil || cust.ID == "" {
			return nil, ErrCustomerRequired
		}
		customerID = cust.ID
	}

	subParams := &stripe.SubscriptionCreateParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionCreateItemParams{
			{Price: stripe.String(in.PriceID)},
		},
		CollectionMethod: stripe.String(string(stripe.SubscriptionCollectionMethodSendInvoice)),
		DaysUntilDue:     stripe.Int64(30),
		Metadata:         map[string]string{"org_id": in.OrgID},
	}
	if in.IdempotencyKey != "" {
		subParams.SetIdempotencyKey(in.IdempotencyKey + ":subscription")
	}

	sub, err := p.client.CreateSubscription(ctx, subParams)
	if err != nil {
		return nil, fmt.Errorf("billing: stripe: create subscription: %w", err)
	}
	if sub == nil || sub.ID == "" {
		return nil, errors.New("billing: stripe: create subscription returned no id")
	}

	return &SubscriptionResult{
		ExternalSubscriptionID: sub.ID,
		ExternalCustomerID:     customerID,
		Status:                 string(sub.Status),
	}, nil
}

// UpdateSubscription implements PaymentProvider by changing the
// subscription's single price item to NewPriceID.
//
// Stripe's subscription-item update semantics require identifying WHICH
// item to change: passing only a bare Price with no item ID makes the
// API ADD a second item alongside the existing one rather than
// replacing it (see "Changing a subscription's price",
// https://docs.stripe.com/billing/subscriptions/change-price#changing —
// verified 2026-07-22, docs/guides/BILLING.md Sources-verified footer).
// StripeProvider always operates on the single-item subscriptions
// internal/handler creates, so it first retrieves the subscription to
// discover the existing item's id, then issues the update targeting
// that item id explicitly — this is what actually swaps the price
// rather than accumulating items.
func (p *StripeProvider) UpdateSubscription(ctx context.Context, in UpdateSubscriptionInput) (*SubscriptionResult, error) {
	if in.ExternalSubscriptionID == "" {
		return nil, errors.New("billing: stripe: ExternalSubscriptionID is required")
	}
	if in.NewPriceID == "" {
		return nil, errors.New("billing: stripe: NewPriceID is required")
	}

	current, err := p.client.RetrieveSubscription(ctx, in.ExternalSubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("billing: stripe: retrieve subscription %s before update: %w", in.ExternalSubscriptionID, err)
	}
	if current == nil || current.Items == nil || len(current.Items.Data) == 0 {
		return nil, fmt.Errorf("billing: stripe: subscription %s has no items to update", in.ExternalSubscriptionID)
	}
	existingItemID := current.Items.Data[0].ID

	params := &stripe.SubscriptionUpdateParams{
		Items: []*stripe.SubscriptionUpdateItemParams{
			{ID: stripe.String(existingItemID), Price: stripe.String(in.NewPriceID)},
		},
	}
	if in.IdempotencyKey != "" {
		params.SetIdempotencyKey(in.IdempotencyKey + ":update")
	}

	sub, err := p.client.UpdateSubscription(ctx, in.ExternalSubscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("billing: stripe: update subscription %s: %w", in.ExternalSubscriptionID, err)
	}

	return &SubscriptionResult{
		ExternalSubscriptionID: sub.ID,
		ExternalCustomerID:     externalCustomerID(sub),
		Status:                 string(sub.Status),
	}, nil
}

// CancelSubscription implements PaymentProvider.
func (p *StripeProvider) CancelSubscription(ctx context.Context, in CancelSubscriptionInput) (*SubscriptionResult, error) {
	if in.ExternalSubscriptionID == "" {
		return nil, errors.New("billing: stripe: ExternalSubscriptionID is required")
	}

	params := &stripe.SubscriptionCancelParams{}
	if in.IdempotencyKey != "" {
		params.SetIdempotencyKey(in.IdempotencyKey + ":cancel")
	}

	sub, err := p.client.CancelSubscription(ctx, in.ExternalSubscriptionID, params)
	if err != nil {
		return nil, fmt.Errorf("billing: stripe: cancel subscription %s: %w", in.ExternalSubscriptionID, err)
	}

	return &SubscriptionResult{
		ExternalSubscriptionID: sub.ID,
		ExternalCustomerID:     externalCustomerID(sub),
		Status:                 string(sub.Status),
	}, nil
}

// VerifyWebhook implements PaymentProvider using Stripe's own signature
// verification (github.com/stripe/stripe-go/v86/webhook.ConstructEventWithOptions):
// HMAC-SHA256 over the timestamped payload keyed by the webhook signing
// secret, with the standard 5-minute replay tolerance
// (webhook.DefaultTolerance). A payload whose signature does not
// verify, whose timestamp falls outside the tolerance window, or that
// arrives with no configured webhook secret, is rejected — never
// treated as trusted.
//
// IgnoreAPIVersionMismatch is set to true: the Stripe SDK's default
// (webhook.ConstructEvent) additionally rejects any event whose
// embedded api_version does not match the exact API version compiled
// into this SDK release (captured evidence: a real signature-valid
// event was rejected in testing with "received event with API version
// , but stripe-go 86.1.1 expects API version 2026-06-24.dahlia" — see
// docs/guides/BILLING.md "Webhook API version" for the full trade-off
// this documents). billing-service does not control the Stripe
// account's configured default API version (a platform-wide setting
// shared with every other Stripe integration on the account), so
// failing webhook delivery entirely on a version mismatch would be a
// self-inflicted outage for a mismatch that does not affect signature
// authenticity — only the shape of expanded nested objects. The
// signature check above is what proves the payload is genuinely from
// Stripe; the API-version check is a compatibility hint, not a trust
// boundary, so it is deliberately not fail-closed.
func (p *StripeProvider) VerifyWebhook(payload []byte, signatureHeader string) (*WebhookEvent, error) {
	if p.webhookSecret == "" {
		return nil, errors.New("billing: stripe: STRIPE_WEBHOOK_SECRET not configured — refusing to trust unverifiable webhook payload")
	}
	event, err := webhook.ConstructEventWithOptions(payload, signatureHeader, p.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, fmt.Errorf("billing: stripe: webhook signature verification failed: %w", err)
	}
	var objectRaw []byte
	if event.Data != nil {
		objectRaw = event.Data.Raw
	}
	return &WebhookEvent{ID: event.ID, Type: string(event.Type), ObjectRaw: objectRaw, Raw: payload}, nil
}

// ParseSubscriptionObject unmarshals a verified webhook event's nested
// object payload (WebhookEvent.ObjectRaw) for subscription-related
// event types (customer.subscription.created/updated/deleted) into the
// processor-agnostic (external subscription id, real status) pair a
// caller needs to reconcile its own locally-stored subscription state
// against what the processor now reports — the counterpart to
// CreateSubscription/UpdateSubscription/CancelSubscription's honest
// status passthrough, applied to processor-initiated changes (a failed
// payment auto-canceling a subscription, for example) the caller never
// itself requested.
func ParseSubscriptionObject(objectRaw []byte) (externalSubscriptionID, status string, err error) {
	var sub stripe.Subscription
	if err := json.Unmarshal(objectRaw, &sub); err != nil {
		return "", "", fmt.Errorf("billing: stripe: parse subscription webhook object: %w", err)
	}
	return sub.ID, string(sub.Status), nil
}

// externalCustomerID extracts the customer id from a *stripe.Subscription
// response. Stripe's Update/Cancel responses expand Customer as a
// *stripe.Customer object (possibly with only ID populated depending on
// expand options); this helper centralizes the nil-safety.
func externalCustomerID(sub *stripe.Subscription) string {
	if sub == nil || sub.Customer == nil {
		return ""
	}
	return sub.Customer.ID
}
