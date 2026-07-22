package payment

// webhook.go — REAL Stripe webhook signature verification (the second half of
// the "armed but not connected" gap PR #7 left). Stripe signs every webhook
// delivery with an HMAC-SHA256 over the timestamp-prefixed raw body, keyed by
// the endpoint's signing secret (whsec_…). This file implements Stripe's
// documented manual-verification scheme and exposes an http.HandlerFunc entry
// that REJECTS any delivery whose signature is absent, malformed, forged, or
// stale.
//
// Source verified 2026-07-22 against Stripe's official docs
// (https://docs.stripe.com/webhooks — "Verify signatures manually"):
//   - Stripe-Signature header: "t=<unix-ts>,v1=<hex-sig>[,v0=<hex-sig>]" (there
//     may be MULTIPLE v1 entries during a signing-secret roll — accept if ANY
//     matches).
//   - signed_payload = "<t>" + "." + <raw request body>.
//   - expected = hex( HMAC-SHA256(signed_payload, signing_secret) ).
//   - compare with a CONSTANT-TIME comparison (crypto/hmac.Equal).
//   - default tolerance 5 minutes; Stripe explicitly warns a tolerance of 0
//     disables the recency check, so 0 here means "no recency check" and any
//     positive value enforces it.
//
// Honest boundary (Constitution §11.4): this verifies a signature the caller
// supplies. Proving it against a webhook Stripe REALLY delivered requires the
// operator's real STRIPE_WEBHOOK_SECRET and a live Stripe event (OPERATOR-GATED,
// §11.4.10). The tests here sign fixtures with a known secret and prove accept
// / reject / stale — the verification algorithm itself, not live delivery.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultWebhookTolerance is Stripe's documented default clock-skew tolerance
// between the signed timestamp and the current time (5 minutes).
const DefaultWebhookTolerance = 5 * time.Minute

// maxWebhookBytes bounds the webhook body the handler reads — a defensive cap
// so a hostile caller cannot stream an unbounded body.
const maxWebhookBytes = 1 << 20 // 1 MiB

// Webhook verification errors. Each is a distinct sentinel so callers/tests can
// tell WHY a delivery was rejected without string-matching.
var (
	// ErrWebhookSecretMissing means no signing secret was configured, so no
	// signature can be verified (fail closed, never fabricate acceptance).
	ErrWebhookSecretMissing = errors.New("stripe webhook: signing secret not configured")
	// ErrWebhookNoSignatureHeader means the Stripe-Signature header was absent.
	ErrWebhookNoSignatureHeader = errors.New("stripe webhook: missing Stripe-Signature header")
	// ErrWebhookMalformedHeader means the header did not contain a parseable
	// timestamp and at least one v1 signature.
	ErrWebhookMalformedHeader = errors.New("stripe webhook: malformed Stripe-Signature header")
	// ErrWebhookNoValidSignature means no v1 signature matched the computed
	// HMAC — the body was forged or the wrong secret was used.
	ErrWebhookNoValidSignature = errors.New("stripe webhook: no matching v1 signature")
	// ErrWebhookTooOld means the signature matched but the timestamp is outside
	// the allowed tolerance (a replayed/stale delivery).
	ErrWebhookTooOld = errors.New("stripe webhook: timestamp outside tolerance")
)

// VerifyWebhookSignature verifies a Stripe webhook signature over payload using
// Stripe's documented scheme. secret is the endpoint signing secret (whsec_…);
// sigHeader is the raw Stripe-Signature header value; tolerance bounds the
// allowed clock skew (0 disables the recency check, per Stripe's docs); now is
// the reference time (injected so tests are deterministic — §11.4.6, no
// hidden dependence on the wall clock).
//
// It returns nil ONLY when a v1 signature matches AND the timestamp is within
// tolerance. Signature matching runs before the recency check so a correctly
// signed but stale delivery is reported as ErrWebhookTooOld (not confused with
// a forgery). The comparison is constant-time (crypto/hmac.Equal), so a
// timing side-channel cannot leak how close a forged signature was.
func VerifyWebhookSignature(payload []byte, sigHeader, secret string, tolerance time.Duration, now time.Time) error {
	if secret == "" {
		return ErrWebhookSecretMissing
	}
	if strings.TrimSpace(sigHeader) == "" {
		return ErrWebhookNoSignatureHeader
	}

	timestamp, sigs := parseSignatureHeader(sigHeader)
	if timestamp == "" || len(sigs) == 0 {
		return ErrWebhookMalformedHeader
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrWebhookMalformedHeader
	}

	// signed_payload = "<timestamp>.<raw body>" — the timestamp string is used
	// verbatim (never re-formatted) so it byte-matches what Stripe signed.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := mac.Sum(nil)

	matched := false
	for _, s := range sigs {
		got, decErr := hex.DecodeString(s)
		if decErr != nil {
			continue // a non-hex v1 can never match; skip it
		}
		if hmac.Equal(got, expected) {
			matched = true
			break
		}
	}
	if !matched {
		return ErrWebhookNoValidSignature
	}

	if tolerance > 0 {
		diff := now.Unix() - ts
		if diff < 0 {
			diff = -diff
		}
		if diff > int64(tolerance.Seconds()) {
			return ErrWebhookTooOld
		}
	}
	return nil
}

// parseSignatureHeader splits a Stripe-Signature header into its timestamp and
// the set of v1 signatures. Unknown schemes (v0, future v*) are ignored. A
// signing-secret roll can produce multiple v1 entries; all are returned so the
// caller accepts on any match.
func parseSignatureHeader(header string) (timestamp string, v1sigs []string) {
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]
		switch key {
		case "t":
			timestamp = val
		case "v1":
			if val != "" {
				v1sigs = append(v1sigs, val)
			}
		}
	}
	return timestamp, v1sigs
}

// ComputeSignatureHeader builds a Stripe-Signature header value that signs
// payload at time t with secret, exactly as Stripe's servers would. It exists
// so tests (and a future local replay tool) can produce genuine, verifiable
// fixtures without any live Stripe traffic — the same HMAC path Stripe uses, so
// a signature this produces is one VerifyWebhookSignature accepts and a
// tampered body is one it rejects.
func ComputeSignatureHeader(payload []byte, secret string, t time.Time) string {
	timestamp := strconv.FormatInt(t.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	return "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

// WebhookHandler returns an http.HandlerFunc that verifies inbound Stripe
// webhook signatures and REJECTS any delivery that fails. It is the "handler
// entry that rejects an invalid signature" this change adds. Responses:
//
//   - 503 Service Unavailable when the gateway is not configured for webhook
//     verification (no secret) — an HONEST "not ready" rather than a silent 200
//     that would let unverified events through.
//   - 400 Bad Request when the signature is absent, malformed, forged, or
//     stale.
//   - 200 OK only when the signature genuinely verifies.
//
// It reads the RAW body (Stripe signs the exact bytes) and never trusts a
// re-serialised form.
func (g *Gateway) WebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !g.WebhookVerificationReady() {
			http.Error(w, "webhook verification not configured", http.StatusServiceUnavailable)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBytes))
		if err != nil {
			http.Error(w, "cannot read request body", http.StatusBadRequest)
			return
		}
		if err := g.VerifyWebhook(body, r.Header.Get("Stripe-Signature"), time.Now()); err != nil {
			// The specific reason is intentionally not echoed to the caller (it
			// would help an attacker probe); a rejected delivery is a uniform
			// 400. The reason IS available to server-side logging via err.
			http.Error(w, "invalid stripe signature", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	}
}
