package payment

// webhook_test.go — proves Stripe webhook signature verification ACCEPTS a
// correctly-signed fixture and REJECTS a tampered body, a stale timestamp, a
// wrong secret, and a missing/malformed header. Fixtures are signed with the
// same documented HMAC-SHA256 scheme Stripe uses (ComputeSignatureHeader), so
// an accepted fixture is genuinely valid and a rejected one is genuinely
// invalid — no live Stripe delivery involved (§11.4 honest boundary).

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testWebhookSecret = "whsec_test_signing_secret_123"

func TestVerifyWebhookSignature_AcceptsCorrectlySigned(t *testing.T) {
	payload := []byte(`{"id":"evt_1","type":"payment_intent.succeeded"}`)
	now := time.Unix(1_700_000_000, 0)
	header := ComputeSignatureHeader(payload, testWebhookSecret, now)

	if err := VerifyWebhookSignature(payload, header, testWebhookSecret, DefaultWebhookTolerance, now); err != nil {
		t.Fatalf("correctly-signed payload should verify, got: %v", err)
	}
}

func TestVerifyWebhookSignature_RejectsTamperedBody(t *testing.T) {
	original := []byte(`{"id":"evt_1","amount":100}`)
	now := time.Unix(1_700_000_000, 0)
	header := ComputeSignatureHeader(original, testWebhookSecret, now)

	// Attacker flips the amount but keeps the signature computed over the
	// original body.
	tampered := []byte(`{"id":"evt_1","amount":999999}`)
	err := VerifyWebhookSignature(tampered, header, testWebhookSecret, DefaultWebhookTolerance, now)
	if !errors.Is(err, ErrWebhookNoValidSignature) {
		t.Fatalf("tampered body must be rejected as no-valid-signature, got: %v", err)
	}
}

func TestVerifyWebhookSignature_RejectsWrongSecret(t *testing.T) {
	payload := []byte(`{"id":"evt_1"}`)
	now := time.Unix(1_700_000_000, 0)
	header := ComputeSignatureHeader(payload, testWebhookSecret, now)

	err := VerifyWebhookSignature(payload, header, "whsec_a_different_secret", DefaultWebhookTolerance, now)
	if !errors.Is(err, ErrWebhookNoValidSignature) {
		t.Fatalf("a signature under a different secret must be rejected, got: %v", err)
	}
}

func TestVerifyWebhookSignature_RejectsStaleTimestamp(t *testing.T) {
	payload := []byte(`{"id":"evt_1"}`)
	signedAt := time.Unix(1_700_000_000, 0)
	header := ComputeSignatureHeader(payload, testWebhookSecret, signedAt)

	// The signature is genuine, but "now" is 10 minutes after signing —
	// outside the 5-minute tolerance. It must be rejected as too-old, NOT
	// mistaken for a forgery.
	now := signedAt.Add(10 * time.Minute)
	err := VerifyWebhookSignature(payload, header, testWebhookSecret, DefaultWebhookTolerance, now)
	if !errors.Is(err, ErrWebhookTooOld) {
		t.Fatalf("stale-but-correctly-signed payload must be rejected as too-old, got: %v", err)
	}

	// And within tolerance (4 minutes later) the SAME signature still verifies,
	// proving the staleness check is the discriminator, not the signature.
	if err := VerifyWebhookSignature(payload, header, testWebhookSecret, DefaultWebhookTolerance, signedAt.Add(4*time.Minute)); err != nil {
		t.Fatalf("within-tolerance delivery should verify, got: %v", err)
	}
}

func TestVerifyWebhookSignature_RejectsMissingAndMalformedHeader(t *testing.T) {
	payload := []byte(`{"id":"evt_1"}`)
	now := time.Unix(1_700_000_000, 0)

	if err := VerifyWebhookSignature(payload, "", testWebhookSecret, DefaultWebhookTolerance, now); !errors.Is(err, ErrWebhookNoSignatureHeader) {
		t.Errorf("empty header: got %v, want ErrWebhookNoSignatureHeader", err)
	}
	if err := VerifyWebhookSignature(payload, "garbage-no-fields", testWebhookSecret, DefaultWebhookTolerance, now); !errors.Is(err, ErrWebhookMalformedHeader) {
		t.Errorf("garbage header: got %v, want ErrWebhookMalformedHeader", err)
	}
	if err := VerifyWebhookSignature(payload, "t=notanumber,v1=abcd", testWebhookSecret, DefaultWebhookTolerance, now); !errors.Is(err, ErrWebhookMalformedHeader) {
		t.Errorf("non-numeric timestamp: got %v, want ErrWebhookMalformedHeader", err)
	}
	if err := VerifyWebhookSignature(payload, "v1=abcd", testWebhookSecret, DefaultWebhookTolerance, now); !errors.Is(err, ErrWebhookMalformedHeader) {
		t.Errorf("missing timestamp: got %v, want ErrWebhookMalformedHeader", err)
	}
}

func TestVerifyWebhookSignature_MissingSecret(t *testing.T) {
	if err := VerifyWebhookSignature([]byte(`{}`), "t=1,v1=ab", "", DefaultWebhookTolerance, time.Now()); !errors.Is(err, ErrWebhookSecretMissing) {
		t.Fatalf("no secret configured must fail closed, got: %v", err)
	}
}

// TestVerifyWebhookSignature_AcceptsMultipleV1 proves a header carrying more
// than one v1 (as happens during a signing-secret roll) verifies when ANY of
// them matches the configured secret.
func TestVerifyWebhookSignature_AcceptsMultipleV1(t *testing.T) {
	payload := []byte(`{"id":"evt_roll"}`)
	now := time.Unix(1_700_000_000, 0)
	valid := ComputeSignatureHeader(payload, testWebhookSecret, now) // "t=..,v1=<good>"
	// Prepend a bogus v1 for a to-be-retired secret: "t=..,v1=<bad>,v1=<good>".
	parts := strings.SplitN(valid, ",", 2) // ["t=..", "v1=<good>"]
	multi := parts[0] + ",v1=deadbeef," + parts[1]

	if err := VerifyWebhookSignature(payload, multi, testWebhookSecret, DefaultWebhookTolerance, now); err != nil {
		t.Fatalf("header with one bad + one good v1 should verify, got: %v", err)
	}
}

// --- WebhookHandler HTTP entry (the "handler entry that rejects an invalid
// signature") ---

// enabledWebhookGateway builds a gateway whose config has both a secret key and
// a webhook secret, so WebhookVerificationReady() is true — a white-box
// construction that needs no environment variables.
func enabledWebhookGateway() *Gateway {
	return newGateway(StripeConfig{SecretKey: "sk_test_wh", WebhookSecret: testWebhookSecret}, true)
}

func TestWebhookHandler_AcceptsValidRejectsTampered(t *testing.T) {
	g := enabledWebhookGateway()
	h := g.WebhookHandler()

	payload := []byte(`{"id":"evt_http","type":"payment_intent.succeeded"}`)
	header := ComputeSignatureHeader(payload, testWebhookSecret, time.Now())

	// Valid delivery → 200.
	{
		req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(string(payload)))
		req.Header.Set("Stripe-Signature", header)
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("valid signature: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
		}
	}

	// Tampered body (same signature) → 400.
	{
		req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{"id":"evt_http","amount":999999}`))
		req.Header.Set("Stripe-Signature", header)
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("tampered body: status = %d, want 400", rec.Code)
		}
	}

	// Missing signature header → 400.
	{
		req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(string(payload)))
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("missing header: status = %d, want 400", rec.Code)
		}
	}
}

// TestWebhookHandler_UnconfiguredFailsClosed proves a gateway without a webhook
// secret returns 503 (honest not-ready) rather than a silent 200 that would let
// unverified events through.
func TestWebhookHandler_UnconfiguredFailsClosed(t *testing.T) {
	g := newGateway(StripeConfig{SecretKey: "sk_test_wh"}, true) // no WebhookSecret
	h := g.WebhookHandler()

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{}`))
	req.Header.Set("Stripe-Signature", "t=1,v1=abcd")
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured webhook: status = %d, want 503", rec.Code)
	}
}
