// Package payment holds billing-service's payment-gateway integration seam.
// The concrete gateway is Stripe, configured EXCLUSIVELY from environment
// variables (Constitution §11.4.10 credentials-handling mandate — secrets are
// NEVER hardcoded in source and NEVER emitted in logs). It mirrors the shape of
// notification-service's delivery.SMTPConfigFromEnv: a config struct plus a
// *FromEnv constructor whose ok result is false when the tier is unconfigured,
// an HONEST "not configured" default rather than a fabricated-active state.
//
// Honest boundary (Constitution §11.4 anti-bluff covenant): the presence of a
// STRIPE_SECRET_KEY ARMS the gateway — it does NOT perform any Stripe API call,
// charge, refund, or webhook round-trip here. "Armed" means only "credentials
// are present, so a future real charge/verify path may run"; it NEVER asserts
// connectivity to Stripe. A real Stripe client (charges, webhook verification)
// is a separate future change and is deliberately not built in this seam.
package payment

import "os"

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
// environment configuration and records whether Stripe credentials are present.
// It performs NO Stripe network calls — see the package doc's honest boundary.
type Gateway struct {
	cfg     StripeConfig
	enabled bool
}

// NewGateway constructs a Gateway from the STRIPE_* environment variables. When
// no STRIPE_SECRET_KEY is set the gateway is DISABLED (the internal-only dev
// default: billing runs without a live payment provider). When a secret key is
// present the gateway is ARMED — credentials are staged for a future real
// charge path, with no Stripe API call performed here.
func NewGateway() *Gateway {
	cfg, ok := StripeConfigFromEnv()
	return &Gateway{cfg: cfg, enabled: ok}
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
		return "armed (stripe key present; webhook verification ready)"
	}
	return "armed (stripe key present; webhook verification NOT ready — set STRIPE_WEBHOOK_SECRET)"
}
