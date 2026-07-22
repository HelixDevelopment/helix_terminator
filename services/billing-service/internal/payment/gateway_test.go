package payment

// gateway_test.go — proves the Gateway is actually WIRED to Stripe: an enabled
// gateway's CreatePaymentIntent / CreateCustomer make a real client call
// (proven by pointing the gateway's client at an httptest server), while a
// disabled gateway returns a real ErrGatewayDisabled — NEVER a fabricated
// "charged"/"active" success (§11.4 anti-bluff, §11.4.6). White-box (package
// payment) so the test can point the unexported client at the test transport
// without adding production surface.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGateway_Disabled_ChargeOpsReturnRealError(t *testing.T) {
	g := newGateway(StripeConfig{}, false) // internal-only default, no key

	if g.Enabled() {
		t.Fatal("gateway with no key must be disabled")
	}
	if _, err := g.CreatePaymentIntent(context.Background(), PaymentIntentParams{AmountCents: 100, Currency: "usd"}); !errors.Is(err, ErrGatewayDisabled) {
		t.Errorf("disabled CreatePaymentIntent: got %v, want ErrGatewayDisabled", err)
	}
	if _, err := g.CreateCustomer(context.Background(), CustomerParams{Email: "a@b.c"}); !errors.Is(err, ErrGatewayDisabled) {
		t.Errorf("disabled CreateCustomer: got %v, want ErrGatewayDisabled", err)
	}
}

func TestGateway_Enabled_CreatePaymentIntent_CallsStripe(t *testing.T) {
	var hit bool
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"pi_gw","object":"payment_intent","amount":2500,"currency":"eur","status":"succeeded"}`))
	}))
	defer ts.Close()

	g := newGateway(StripeConfig{SecretKey: "sk_test_gw"}, true)
	// Point the wired client at the test transport (white-box injection).
	g.client = NewClient("sk_test_gw", WithBaseURL(ts.URL), WithHTTPClient(ts.Client()))

	pi, err := g.CreatePaymentIntent(context.Background(), PaymentIntentParams{AmountCents: 2500, Currency: "eur"})
	if err != nil {
		t.Fatalf("enabled CreatePaymentIntent: %v", err)
	}
	if !hit {
		t.Fatal("enabled gateway must actually call Stripe (server was never hit)")
	}
	if gotAuth != "Bearer sk_test_gw" {
		t.Errorf("Authorization = %q, want Bearer sk_test_gw", gotAuth)
	}
	if pi.ID != "pi_gw" || pi.Status != "succeeded" {
		t.Errorf("parsed pi = %+v, want id pi_gw status succeeded", pi)
	}
}

func TestGateway_Enabled_StripeErrorSurfaced(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"type":"card_error","code":"card_declined","message":"declined"}}`))
	}))
	defer ts.Close()

	g := newGateway(StripeConfig{SecretKey: "sk_test_gw"}, true)
	g.client = NewClient("sk_test_gw", WithBaseURL(ts.URL), WithHTTPClient(ts.Client()))

	pi, err := g.CreatePaymentIntent(context.Background(), PaymentIntentParams{AmountCents: 100, Currency: "usd"})
	if err == nil {
		t.Fatalf("a Stripe 402 must surface as an error, not a fake success (pi=%+v)", pi)
	}
	var se *Error
	if !errors.As(err, &se) || se.Code != "card_declined" {
		t.Fatalf("want *Error card_declined, got %T: %v", err, err)
	}
}

func TestGateway_VerifyWebhook_ReadyAndNotReady(t *testing.T) {
	payload := []byte(`{"id":"evt_gw"}`)
	now := time.Unix(1_700_000_000, 0)

	// Not ready (no webhook secret) → ErrWebhookSecretMissing.
	notReady := newGateway(StripeConfig{SecretKey: "sk_test_gw"}, true)
	if err := notReady.VerifyWebhook(payload, "t=1,v1=ab", now); !errors.Is(err, ErrWebhookSecretMissing) {
		t.Errorf("not-ready VerifyWebhook: got %v, want ErrWebhookSecretMissing", err)
	}

	// Ready → verifies a genuinely-signed fixture.
	ready := newGateway(StripeConfig{SecretKey: "sk_test_gw", WebhookSecret: testWebhookSecret}, true)
	if !ready.WebhookVerificationReady() {
		t.Fatal("expected WebhookVerificationReady() == true")
	}
	header := ComputeSignatureHeader(payload, testWebhookSecret, now)
	if err := ready.VerifyWebhook(payload, header, now); err != nil {
		t.Errorf("ready VerifyWebhook on valid fixture: %v", err)
	}
}
