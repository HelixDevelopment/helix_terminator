// Package billing defines billing-service's payment-processor abstraction
// (Constitution §11.4 anti-bluff covenant) and its concrete
// implementations.
//
// THE BLUFF THIS PACKAGE CLOSES: prior to this package's introduction,
// internal/handler.CreateSubscription persisted a new subscription row
// with Status:"active" unconditionally — no payment processor was ever
// contacted, so "active" was a fabricated success with nothing behind
// it (§11.4 PASS-bluff at the product layer: the row claimed the org
// had a working paid subscription when no money had moved and no
// processor had agreed to anything). PaymentProvider is the seam that
// makes that fabrication structurally impossible: every subscription
// lifecycle mutation (create/update/cancel) MUST go through a
// PaymentProvider implementation that talks to a real processor, and
// the handler layer honestly reports 501 "payments provider not
// configured" when none is wired — never a fabricated success.
//
// PaymentProvider is intentionally processor-agnostic so Stripe
// (StripeProvider, stripe_provider.go) is ONE implementation among
// others a future change can add (e.g. Paddle, Braintree, a
// direct-debit processor) without touching internal/handler.
package billing

import "context"

// CreateSubscriptionInput is the processor-agnostic input for creating a
// new subscription. OrgID and PriceID are always required. ExistingCustomerID
// is optional — when non-empty the provider MUST reuse that processor-side
// customer record instead of creating a new one (avoids accumulating
// duplicate customer records for the same tenant across repeat
// subscription creates). IdempotencyKey, when non-empty, MUST be passed
// to the processor's own idempotency mechanism so a retried request
// (client timeout + retry, at-least-once delivery, etc.) never creates
// two processor-side subscriptions for the same logical request.
type CreateSubscriptionInput struct {
	OrgID              string
	PriceID            string
	ExistingCustomerID string
	IdempotencyKey     string
}

// UpdateSubscriptionInput is the processor-agnostic input for changing an
// existing subscription's price (plan change / upgrade / downgrade).
type UpdateSubscriptionInput struct {
	ExternalSubscriptionID string
	NewPriceID             string
	IdempotencyKey         string
}

// CancelSubscriptionInput is the processor-agnostic input for canceling an
// existing subscription.
type CancelSubscriptionInput struct {
	ExternalSubscriptionID string
	IdempotencyKey         string
}

// SubscriptionResult is the processor-agnostic result of a subscription
// lifecycle call. Status carries the REAL status string the processor
// returned (e.g. Stripe's "active" / "incomplete" / "trialing" /
// "past_due" / "canceled") — the caller MUST persist this value
// verbatim rather than assuming success implies "active"; a processor
// can accept a create call and still return a non-active status (for
// example "incomplete" when the initial invoice requires payment
// action) and reporting that honestly is the entire point of this
// package.
type SubscriptionResult struct {
	ExternalSubscriptionID string
	ExternalCustomerID     string
	Status                 string
}

// WebhookEvent is the processor-agnostic result of successfully verifying
// an inbound webhook payload's signature. Raw carries the original,
// already-verified full payload bytes; ObjectRaw carries just the
// verified event's nested "object" (e.g. the subscription object for a
// customer.subscription.* event) so a caller can extract the fields it
// needs (via a processor-specific helper such as
// ParseSubscriptionObject) without re-verifying or re-parsing the
// envelope.
type WebhookEvent struct {
	ID        string
	Type      string
	ObjectRaw []byte
	Raw       []byte
}

// PaymentProvider abstracts a real payment/subscription processor.
// Implementations MUST perform real network calls against the
// processor's API — a PaymentProvider that fabricates a result without
// contacting the processor reintroduces exactly the bluff this package
// exists to close. Test doubles satisfying this interface are
// permitted ONLY in unit-test source files (Constitution §11.4.27(A));
// every other test type (integration/e2e/stress/chaos) MUST exercise a
// real implementation against real (or real-test-mode) processor
// infrastructure.
type PaymentProvider interface {
	// Name returns a short, stable, lowercase identifier for the
	// concrete processor (e.g. "stripe"), persisted alongside every
	// subscription row so a mixed-processor history stays honestly
	// attributable.
	Name() string

	// CreateSubscription creates a new subscription against the real
	// processor and returns its assigned identifiers and real status.
	CreateSubscription(ctx context.Context, in CreateSubscriptionInput) (*SubscriptionResult, error)

	// UpdateSubscription changes an existing subscription's price
	// against the real processor and returns its resulting real status.
	UpdateSubscription(ctx context.Context, in UpdateSubscriptionInput) (*SubscriptionResult, error)

	// CancelSubscription cancels an existing subscription against the
	// real processor and returns its resulting real status.
	CancelSubscription(ctx context.Context, in CancelSubscriptionInput) (*SubscriptionResult, error)

	// VerifyWebhook cryptographically verifies an inbound webhook
	// request's signature against the processor's shared webhook
	// secret and returns the verified event. It MUST reject (non-nil
	// error, nil event) any payload whose signature does not verify —
	// an unverified payload MUST NEVER be treated as a trusted event.
	VerifyWebhook(payload []byte, signatureHeader string) (*WebhookEvent, error)
}
