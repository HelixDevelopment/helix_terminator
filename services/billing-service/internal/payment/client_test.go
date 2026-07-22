package payment

// client_test.go — proves the REAL Stripe client's request/response contract
// against a mock transport (httptest), with NO live Stripe traffic
// (Constitution §11.4.27: the transport is the real net/http path; only
// Stripe's far end is mocked). It asserts the exact request URL / method /
// auth / form body, that a 2xx is parsed, and — critically — that a Stripe
// 402/400 is surfaced as a REAL error and NEVER a fabricated success
// (§11.4 / §11.4.6).

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a Client pointed at ts with a matching transport, so
// every call hits the httptest server instead of api.stripe.com.
func newTestClient(secretKey string, ts *httptest.Server) *Client {
	return NewClient(secretKey, WithBaseURL(ts.URL), WithHTTPClient(ts.Client()))
}

func TestClient_CreatePaymentIntent_RequestShapeAndParse(t *testing.T) {
	const secret = "sk_test_shape_key"

	var gotMethod, gotPath, gotAuth, gotContentType, gotIdempotency, gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotIdempotency = r.Header.Get("Idempotency-Key")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "pi_1MockCharge",
			"object": "payment_intent",
			"amount": 4200,
			"currency": "usd",
			"status": "requires_payment_method",
			"customer": "cus_Mock123",
			"client_secret": "pi_1MockCharge_secret_abc"
		}`))
	}))
	defer ts.Close()

	c := newTestClient(secret, ts)
	pi, err := c.CreatePaymentIntent(context.Background(), PaymentIntentParams{
		AmountCents:    4200,
		Currency:       "USD",
		Customer:       "cus_Mock123",
		Description:    "invoice INV-1",
		Metadata:       map[string]string{"invoiceId": "abc-123"},
		IdempotencyKey: "idem-key-xyz",
	})
	if err != nil {
		t.Fatalf("CreatePaymentIntent returned error: %v", err)
	}

	// --- request contract ---
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/payment_intents" {
		t.Errorf("path = %q, want /v1/payment_intents", gotPath)
	}
	if gotAuth != "Bearer "+secret {
		t.Errorf("Authorization = %q, want Bearer <secret>", gotAuth)
	}
	if !strings.HasPrefix(gotContentType, "application/x-www-form-urlencoded") {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", gotContentType)
	}
	if gotIdempotency != "idem-key-xyz" {
		t.Errorf("Idempotency-Key = %q, want idem-key-xyz", gotIdempotency)
	}
	// Body: amount in cents, currency lowercased, customer, description,
	// metadata bracket-encoded.
	for _, want := range []string{
		"amount=4200",
		"currency=usd",
		"customer=cus_Mock123",
		"description=invoice+INV-1",
		"metadata%5BinvoiceId%5D=abc-123",
	} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("request body missing %q; body=%q", want, gotBody)
		}
	}

	// --- response parse ---
	if pi.ID != "pi_1MockCharge" {
		t.Errorf("pi.ID = %q, want pi_1MockCharge", pi.ID)
	}
	if pi.Amount != 4200 || pi.Currency != "usd" {
		t.Errorf("pi amount/currency = %d/%q, want 4200/usd", pi.Amount, pi.Currency)
	}
	if pi.Status != "requires_payment_method" {
		t.Errorf("pi.Status = %q, want requires_payment_method (Stripe's real status, not coerced)", pi.Status)
	}
}

func TestClient_CreateCustomer_RequestShapeAndParse(t *testing.T) {
	const secret = "sk_test_customer_key"

	var gotPath, gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"cus_Mock999","object":"customer","email":"ops@example.com"}`))
	}))
	defer ts.Close()

	c := newTestClient(secret, ts)
	cust, err := c.CreateCustomer(context.Background(), CustomerParams{
		Email:       "ops@example.com",
		Description: "org acme",
		Metadata:    map[string]string{"orgId": "org-7"},
	})
	if err != nil {
		t.Fatalf("CreateCustomer returned error: %v", err)
	}
	if gotPath != "/v1/customers" {
		t.Errorf("path = %q, want /v1/customers", gotPath)
	}
	for _, want := range []string{"email=ops%40example.com", "description=org+acme", "metadata%5BorgId%5D=org-7"} {
		if !strings.Contains(gotBody, want) {
			t.Errorf("request body missing %q; body=%q", want, gotBody)
		}
	}
	if cust.ID != "cus_Mock999" || cust.Email != "ops@example.com" {
		t.Errorf("parsed customer = %+v, want id cus_Mock999 email ops@example.com", cust)
	}
}

func TestClient_CardDeclined402_SurfacesRealError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired) // 402
		_, _ = w.Write([]byte(`{"error":{"type":"card_error","code":"card_declined","message":"Your card was declined.","param":"card"}}`))
	}))
	defer ts.Close()

	c := newTestClient("sk_test_declined", ts)
	pi, err := c.CreatePaymentIntent(context.Background(), PaymentIntentParams{AmountCents: 1000, Currency: "usd"})

	// Anti-bluff (§11.4): a declined charge MUST be a real error, NOT a
	// fabricated "charged" success.
	if err == nil {
		t.Fatalf("expected a real error on 402, got pi=%+v, err=nil (would be a fabricated success)", pi)
	}
	if pi != nil {
		t.Errorf("expected nil PaymentIntent on error, got %+v", pi)
	}
	var se *Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *payment.Error, got %T: %v", err, err)
	}
	if se.HTTPStatus != http.StatusPaymentRequired {
		t.Errorf("HTTPStatus = %d, want 402", se.HTTPStatus)
	}
	if se.Code != "card_declined" || se.Type != "card_error" {
		t.Errorf("error type/code = %q/%q, want card_error/card_declined", se.Type, se.Code)
	}
	if strings.Contains(se.Error(), "sk_test_declined") {
		t.Errorf("error string leaked the secret key: %q", se.Error())
	}
}

func TestClient_InvalidRequest400_SurfacesRealError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // 400
		_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"Amount must be at least 50 cents"}}`))
	}))
	defer ts.Close()

	c := newTestClient("sk_test_badreq", ts)
	_, err := c.CreatePaymentIntent(context.Background(), PaymentIntentParams{AmountCents: 1, Currency: "usd"})
	var se *Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *payment.Error, got %T: %v", err, err)
	}
	if se.HTTPStatus != http.StatusBadRequest || se.Type != "invalid_request_error" {
		t.Errorf("got status=%d type=%q, want 400/invalid_request_error", se.HTTPStatus, se.Type)
	}
}

// TestClient_LocalValidation_NoNetworkCall proves nonsensical charges are
// rejected locally BEFORE any Stripe request is made (a zero/negative amount
// or empty currency never reaches the wire).
func TestClient_LocalValidation_NoNetworkCall(t *testing.T) {
	hits := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()
	c := newTestClient("sk_test_local", ts)

	if _, err := c.CreatePaymentIntent(context.Background(), PaymentIntentParams{AmountCents: 0, Currency: "usd"}); err == nil {
		t.Error("expected error for zero amount")
	}
	if _, err := c.CreatePaymentIntent(context.Background(), PaymentIntentParams{AmountCents: 100, Currency: ""}); err == nil {
		t.Error("expected error for empty currency")
	}
	if hits != 0 {
		t.Errorf("local validation should make no network call, but server was hit %d time(s)", hits)
	}
}
