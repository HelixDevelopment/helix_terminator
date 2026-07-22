package server_test

// server_webhook_test.go — proves the Stripe webhook route is actually MOUNTED
// on the real billing-service router (the "armed but never mounted" gap PR #7
// left, closed here), reachable WITHOUT a JWT, and that it enforces signature
// verification end-to-end through the real gin engine: a genuinely-signed
// delivery gets 200, a tampered one 400, and — when no signing secret is
// configured — the route fails closed with 503 rather than silently accepting.
// No live Stripe traffic: fixtures are signed with the same HMAC scheme Stripe
// uses (payment.ComputeSignatureHeader).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/helixdevelopment/billing-service/internal/payment"
	"github.com/helixdevelopment/billing-service/internal/server"
)

func TestWebhookRoute_MountedAndVerifiesSignature(t *testing.T) {
	const secret = "whsec_route_test_secret"
	// server.New reads STRIPE_* at construction, so set env FIRST.
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_route")
	t.Setenv("STRIPE_WEBHOOK_SECRET", secret)

	s := server.New(nil) // repo unused by the webhook route
	router := s.Router()

	payload := `{"id":"evt_route","type":"payment_intent.succeeded"}`
	sig := payment.ComputeSignatureHeader([]byte(payload), secret, time.Now())

	// Valid signature, NO Authorization header → 200 (route is outside the JWT
	// group and accepts on a genuine signature).
	{
		req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(payload))
		req.Header.Set("Stripe-Signature", sig)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("valid signed webhook via router: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
		}
	}

	// Tampered body with the same signature → 400.
	{
		req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{"id":"evt_route","amount":999999}`))
		req.Header.Set("Stripe-Signature", sig)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("tampered webhook via router: status = %d, want 400", rec.Code)
		}
	}
}

func TestWebhookRoute_UnconfiguredFailsClosed(t *testing.T) {
	// No STRIPE_* env → gateway disabled → route present but fails closed 503,
	// never a silent 200 that would admit unverified events.
	t.Setenv("STRIPE_SECRET_KEY", "")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")

	s := server.New(nil)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{}`))
	req.Header.Set("Stripe-Signature", "t=1,v1=abcd")
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured webhook route: status = %d, want 503", rec.Code)
	}
}
