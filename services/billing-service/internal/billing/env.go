package billing

import (
	"os"

	"github.com/stripe/stripe-go/v86"
)

// EnvStripeSecretKey and EnvStripeWebhookSecret are the environment
// variable names NewProviderFromEnv reads. Documented in full in
// docs/guides/BILLING.md.
const (
	EnvStripeSecretKey     = "STRIPE_SECRET_KEY"
	EnvStripeWebhookSecret = "STRIPE_WEBHOOK_SECRET"
)

// NewProviderFromEnv is the honest feature-flag (Constitution §11.4 /
// §11.4.99): it reads STRIPE_SECRET_KEY (and, optionally,
// STRIPE_WEBHOOK_SECRET) from the process environment and returns:
//
//   - (nil, nil) when STRIPE_SECRET_KEY is unset/empty — "no payment
//     provider is configured" is NOT an error, it is a legitimate,
//     honestly-reported operating mode. Callers (internal/server,
//     internal/handler) MUST treat a nil PaymentProvider as "respond
//     501 Not Implemented to any subscription-lifecycle-mutating
//     request" — NEVER as licence to fabricate a success.
//   - (*StripeProvider, nil) when STRIPE_SECRET_KEY is set — every
//     subsequent PaymentProvider call this process makes is a REAL
//     call against the real Stripe API (test-mode or live-mode,
//     entirely determined by which kind of key was supplied — this
//     package never inspects the key's "sk_test_"/"sk_live_" prefix
//     itself; that judgment belongs to whoever provisions the
//     environment, per docs/guides/BILLING.md's key-provisioning
//     section).
//
// STRIPE_WEBHOOK_SECRET may be left empty even when STRIPE_SECRET_KEY
// is set — subscription create/update/cancel then work normally, but
// StripeProvider.VerifyWebhook always fails closed (see
// stripe_provider.go) until it is also configured.
func NewProviderFromEnv() (PaymentProvider, error) {
	secretKey := os.Getenv(EnvStripeSecretKey)
	if secretKey == "" {
		return nil, nil
	}
	webhookSecret := os.Getenv(EnvStripeWebhookSecret)

	sc := stripe.NewClient(secretKey)
	return NewStripeProvider(sc, webhookSecret), nil
}
