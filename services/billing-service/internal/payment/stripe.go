// Package payment holds billing-service's payment-gateway integration seam.
// The concrete gateway is Stripe, configured EXCLUSIVELY from environment
// variables (Constitution §11.4.10 credentials-handling mandate — secrets are
// NEVER hardcoded in source and NEVER emitted in logs). It mirrors the shape of
// notification-service's delivery.SMTPConfigFromEnv: a config struct plus a
// *FromEnv constructor whose ok result is false when the tier is unconfigured,
// an HONEST "not configured" default rather than a fabricated-active state.
//
// What connects to Stripe (this change): PR #7 armed the gateway (it detected
// the key) but made no Stripe call. The gateway now holds a real *Client
// (client.go) and exposes real operations — CreateCustomer, CreatePaymentIntent
// (the charge primitive matching this service's invoice amount model), and
// VerifyWebhook (webhook.go) — that perform genuine Stripe API round-trips when
// Enabled().
//
// Honest boundary (Constitution §11.4 anti-bluff covenant): when NO
// STRIPE_SECRET_KEY is set the gateway is DISABLED and every charge operation
// returns a real ErrGatewayDisabled error — the internal-only dev default is
// unchanged, and NOTHING is ever fabricated as "charged"/"active". When a key
// IS present the operations make real HTTPS calls to Stripe; a Stripe rejection
// is surfaced as a real error, never a fake success. These paths are proven in
// tests ONLY against a mock transport (httptest). A live charge, and
// verification of a webhook Stripe REALLY delivered, require the operator's
// actual Stripe test keys and a real Stripe-signed event — OPERATOR-GATED
// (§11.4.10); this package does NOT claim live charges verified.
package payment

import (
	"context"
	"errors"
	"os"
	"time"
)

// ErrGatewayDisabled is returned by the gateway's charge operations when no
// STRIPE_SECRET_KEY is configured. It is a real error, deliberately distinct
// from a fabricated success: an internal-only deployment MUST see a hard
// failure if it tries to charge, never a silent fake "charged".
var ErrGatewayDisabled = errors.New("stripe gateway disabled: no STRIPE_SECRET_KEY configured")

// StripeConfig holds Stripe API credentials. Values are sourced from
// environment variables only (Constitution §11.4.10) and are NEVER written to
// logs — see Gateway.Mode, which reports only a coarse state word.
type StripeConfig struct {
	// SecretKey is the Stripe secret API key (sk_live_… / sk_test_…). Its mere
	// presence arms the gateway; its value is never logged.
	SecretKey string
	// WebhookSecret is the signing secret (whsec_…) used to verify inbound
	// Stripe webhook signatures. Optional — absent means webhook verification
	// is not yet ready even when the secret key is present.
	WebhookSecret string
}

// StripeConfigFromEnv builds a StripeConfig from the STRIPE_* environment
// variables. ok is false when STRIPE_SECRET_KEY is unset, meaning Stripe
// billing is not configured for this deployment — an honest "not configured"
// state, not an error (mirrors delivery.SMTPConfigFromEnv). Callers MUST NOT
// fabricate a charge or a verified webhook in that case.
//
// Recognised variables: STRIPE_SECRET_KEY (required to arm the gateway),
// STRIPE_WEBHOOK_SECRET (optional; enables webhook-signature verification
// readiness).
func StripeConfigFromEnv() (StripeConfig, bool) {
	secretKey := os.Getenv("STRIPE_SECRET_KEY")
	if secretKey == "" {
		return StripeConfig{}, false
	}
	return StripeConfig{
		SecretKey:     secretKey,
		WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}, true
}

// Gateway is billing-service's payment-gateway seam. It is constructed from
// environment configuration, records whether Stripe credentials are present,
// and — when enabled — holds a real Stripe *Client through which its
// operations perform genuine Stripe API calls (see the package doc's honest
// boundary).
type Gateway struct {
	cfg     StripeConfig
	enabled bool
	client  *Client
}

// newGateway assembles a Gateway from a resolved config. When enabled it
// constructs the real Stripe client bound to the secret key. It is the shared
// core of NewGateway (env-driven, production) and the white-box test path
// (which points client at an httptest transport).
func newGateway(cfg StripeConfig, enabled bool) *Gateway {
	g := &Gateway{cfg: cfg, enabled: enabled}
	if enabled {
		g.client = NewClient(cfg.SecretKey)
	}
	return g
}

// NewGateway constructs a Gateway from the STRIPE_* environment variables. When
// no STRIPE_SECRET_KEY is set the gateway is DISABLED (the internal-only dev
// default: billing runs without a live payment provider) and its charge
// operations return ErrGatewayDisabled. When a secret key is present the
// gateway is ENABLED and its operations make real Stripe API calls.
func NewGateway() *Gateway {
	cfg, ok := StripeConfigFromEnv()
	return newGateway(cfg, ok)
}

// Enabled reports whether a STRIPE_SECRET_KEY is present (the gateway is armed).
// It never implies a successful Stripe connection — only that a credential was
// supplied.
func (g *Gateway) Enabled() bool {
	return g != nil && g.enabled
}

// WebhookVerificationReady reports whether inbound Stripe webhook signatures
// could be verified — true only when BOTH the gateway is enabled AND a
// STRIPE_WEBHOOK_SECRET is present. It does not itself verify any webhook.
func (g *Gateway) WebhookVerificationReady() bool {
	return g.Enabled() && g.cfg.WebhookSecret != ""
}

// Mode returns a coarse, log-safe description of the gateway's state for
// startup logging. It NEVER contains any secret material (Constitution
// §11.4.10) — only one of a small closed set of state words.
func (g *Gateway) Mode() string {
	if !g.Enabled() {
		return "disabled (no STRIPE_SECRET_KEY — internal dev default)"
	}
	if g.WebhookVerificationReady() {
		return "enabled (stripe key present; live charges + webhook verification connected)"
	}
	return "enabled (stripe key present; live charges connected; webhook verification NOT ready — set STRIPE_WEBHOOK_SECRET)"
}

// CreateCustomer creates a Stripe Customer through the real client. It returns
// ErrGatewayDisabled when no Stripe key is configured (never a fabricated
// customer), and surfaces any Stripe API error verbatim.
func (g *Gateway) CreateCustomer(ctx context.Context, p CustomerParams) (*Customer, error) {
	if !g.Enabled() || g.client == nil {
		return nil, ErrGatewayDisabled
	}
	return g.client.CreateCustomer(ctx, p)
}

// CreatePaymentIntent creates a Stripe PaymentIntent (the real charge) through
// the real client. It returns ErrGatewayDisabled when no Stripe key is
// configured — an internal-only deployment sees a hard error rather than a fake
// "charged" — and surfaces any Stripe API error (e.g. HTTP 402 card declined)
// verbatim. The returned PaymentIntent.Status is Stripe's real status, never
// coerced.
func (g *Gateway) CreatePaymentIntent(ctx context.Context, p PaymentIntentParams) (*PaymentIntent, error) {
	if !g.Enabled() || g.client == nil {
		return nil, ErrGatewayDisabled
	}
	return g.client.CreatePaymentIntent(ctx, p)
}

// VerifyWebhook verifies a Stripe webhook signature against the configured
// STRIPE_WEBHOOK_SECRET using Stripe's documented HMAC-SHA256 scheme (see
// webhook.go). It returns ErrWebhookSecretMissing when webhook verification is
// not configured, and a specific rejection error otherwise. now is injected for
// deterministic testing of the timestamp-tolerance check.
func (g *Gateway) VerifyWebhook(payload []byte, sigHeader string, now time.Time) error {
	if !g.WebhookVerificationReady() {
		return ErrWebhookSecretMissing
	}
	return VerifyWebhookSignature(payload, sigHeader, g.cfg.WebhookSecret, DefaultWebhookTolerance, now)
}
