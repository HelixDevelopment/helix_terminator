package payment

// client.go — the REAL Stripe REST client that PR #7's scaffolding armed but
// never connected. It talks to Stripe's HTTP API directly (net/http + the
// stdlib) rather than pulling in github.com/stripe/stripe-go, because that
// SDK is NOT a dependency of this module (see go.mod) and the two calls this
// service needs — create a Customer and create a PaymentIntent — are a thin
// form-encoded POST each. The http.Client and base URL are injectable so the
// client is exercised end-to-end against an httptest server with ZERO live
// Stripe traffic (Constitution §11.4.27 real-system-under-test for the
// transport, §11.4.6 no-guessing about request shape).
//
// Honest boundary (Constitution §11.4 anti-bluff covenant): the client makes a
// REAL HTTPS request when pointed at api.stripe.com — it does NOT fabricate a
// charge. A non-2xx Stripe response is surfaced as a real *Error, NEVER
// swallowed into a fake "charged/active" success. Live charges require the
// operator's actual Stripe secret key (§11.4.10) and are OPERATOR-GATED; this
// package's tests prove the request/response contract only, against a mock
// transport.
//
// Credential discipline (Constitution §11.4.10): the secret key is held only
// to set the Authorization: Bearer header and is NEVER written to logs nor
// embedded in any error string (see Error.Error, which reports only Stripe's
// own error type/code/message and the HTTP status).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// defaultBaseURL is Stripe's production API root. Overridable via WithBaseURL
// so tests point the client at an httptest server.
const defaultBaseURL = "https://api.stripe.com"

// maxRespBytes bounds how much of a Stripe response body we read, so a
// pathological/huge body can never exhaust memory (defensive; real Stripe
// responses are small JSON documents).
const maxRespBytes = 1 << 20 // 1 MiB

// Client is a minimal Stripe REST client. Construct it with NewClient. It is
// safe for concurrent use by multiple goroutines (net/http.Client is).
type Client struct {
	httpClient *http.Client
	baseURL    string
	secretKey  string
}

// ClientOption customises a Client at construction time (dependency injection
// for testability — Constitution §11.4.27).
type ClientOption func(*Client)

// WithHTTPClient injects the underlying *http.Client. Tests pass an
// httptest-server-backed client so no live Stripe request is made; production
// omits it and gets a sane defaulted client with a timeout.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithBaseURL overrides the Stripe API root (default https://api.stripe.com).
// Tests set this to their httptest server URL.
func WithBaseURL(base string) ClientOption {
	return func(c *Client) {
		if base != "" {
			c.baseURL = strings.TrimRight(base, "/")
		}
	}
}

// NewClient builds a Stripe client bound to secretKey. Options may inject a
// custom transport / base URL for testing. The default transport carries a
// 30-second timeout so a hung Stripe request can never block a caller forever.
func NewClient(secretKey string, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
		secretKey:  secretKey,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Customer is the subset of a Stripe Customer object this service reads back
// after creation. Additional Stripe fields are ignored by encoding/json.
type Customer struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Created     int64  `json:"created"`
	Livemode    bool   `json:"livemode"`
}

// CustomerParams are the inputs to CreateCustomer. All fields are optional per
// Stripe's API; billing-service typically sets Email + Description (the org's
// billing contact) and a Metadata[orgId] back-reference.
type CustomerParams struct {
	Email          string
	Name           string
	Description    string
	Metadata       map[string]string
	IdempotencyKey string
}

// PaymentIntent is the subset of a Stripe PaymentIntent this service reads
// back. Status is Stripe's real lifecycle value (e.g. "requires_payment_method",
// "requires_confirmation", "succeeded") — it is reported verbatim, never
// coerced into a fabricated "active"/"paid" (Constitution §11.4 / §11.4.6).
type PaymentIntent struct {
	ID           string `json:"id"`
	Object       string `json:"object"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
	Status       string `json:"status"`
	Customer     string `json:"customer"`
	ClientSecret string `json:"client_secret"`
	Description  string `json:"description"`
	Created      int64  `json:"created"`
	Livemode     bool   `json:"livemode"`
}

// PaymentIntentParams are the inputs to CreatePaymentIntent. AmountCents maps
// 1:1 to a billing-service invoice's AmountCents, and Currency to its Currency
// — a PaymentIntent is the Stripe primitive that matches this service's
// existing invoice/amount model (the service records amounts on invoices, not
// Stripe price IDs, so there is no Stripe Subscription/price to map to).
type PaymentIntentParams struct {
	AmountCents    int64
	Currency       string
	Customer       string // optional Stripe customer id (cus_…)
	Description    string
	ReceiptEmail   string
	Metadata       map[string]string
	IdempotencyKey string
}

// CreateCustomer creates a Stripe Customer (POST /v1/customers). It returns a
// real *Error when Stripe rejects the request.
func (c *Client) CreateCustomer(ctx context.Context, p CustomerParams) (*Customer, error) {
	form := url.Values{}
	if p.Email != "" {
		form.Set("email", p.Email)
	}
	if p.Name != "" {
		form.Set("name", p.Name)
	}
	if p.Description != "" {
		form.Set("description", p.Description)
	}
	for k, v := range p.Metadata {
		form.Set("metadata["+k+"]", v)
	}

	var cust Customer
	if err := c.postForm(ctx, "/v1/customers", form, p.IdempotencyKey, &cust); err != nil {
		return nil, err
	}
	return &cust, nil
}

// CreatePaymentIntent creates a Stripe PaymentIntent (POST /v1/payment_intents)
// — the real charge primitive. AmountCents must be positive and Currency must
// be set; both are validated locally before any network call so a nonsensical
// charge never reaches Stripe. On a Stripe rejection (e.g. HTTP 402 card
// declined, HTTP 400 invalid params) the real *Error is returned, never a
// fabricated success.
func (c *Client) CreatePaymentIntent(ctx context.Context, p PaymentIntentParams) (*PaymentIntent, error) {
	if p.AmountCents <= 0 {
		return nil, fmt.Errorf("stripe: payment intent amount must be positive, got %d", p.AmountCents)
	}
	if p.Currency == "" {
		return nil, fmt.Errorf("stripe: payment intent currency is required")
	}

	form := url.Values{}
	form.Set("amount", strconv.FormatInt(p.AmountCents, 10))
	form.Set("currency", strings.ToLower(p.Currency))
	if p.Customer != "" {
		form.Set("customer", p.Customer)
	}
	if p.Description != "" {
		form.Set("description", p.Description)
	}
	if p.ReceiptEmail != "" {
		form.Set("receipt_email", p.ReceiptEmail)
	}
	for k, v := range p.Metadata {
		form.Set("metadata["+k+"]", v)
	}

	var pi PaymentIntent
	if err := c.postForm(ctx, "/v1/payment_intents", form, p.IdempotencyKey, &pi); err != nil {
		return nil, err
	}
	return &pi, nil
}

// postForm performs a form-encoded POST to path, authenticating with the
// secret key as an HTTP Bearer credential exactly as Stripe requires. On a
// 2xx it decodes the body into out; on any non-2xx it returns the parsed
// Stripe *Error. It NEVER converts a non-2xx into a success.
func (c *Client) postForm(ctx context.Context, path string, form url.Values, idempotencyKey string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("stripe: build request for %s: %w", path, err)
	}
	// Bearer auth with the secret key (Stripe's documented scheme). The key
	// is never logged and never placed in an error message (§11.4.10).
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("stripe: request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
	if err != nil {
		return fmt.Errorf("stripe: reading response from %s: %w", path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseError(resp.StatusCode, body)
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("stripe: decoding response from %s: %w", path, err)
		}
	}
	return nil
}

// Error is a real Stripe API error surfaced to the caller. It carries the HTTP
// status plus Stripe's own error type/code/message so the caller can react
// (retry, decline, alert) — and it deliberately carries NO credential material
// (§11.4.10).
type Error struct {
	HTTPStatus int
	Type       string
	Code       string
	Message    string
	Param      string
}

// Error implements the error interface. It reports only Stripe-supplied,
// non-secret fields.
func (e *Error) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "stripe returned a non-2xx response with no error message"
	}
	if e.Code != "" {
		return fmt.Sprintf("stripe api error (http %d, type=%q, code=%q): %s", e.HTTPStatus, e.Type, e.Code, msg)
	}
	return fmt.Sprintf("stripe api error (http %d, type=%q): %s", e.HTTPStatus, e.Type, msg)
}

// stripeErrorEnvelope models Stripe's standard {"error": {...}} response body.
type stripeErrorEnvelope struct {
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   string `json:"param"`
	} `json:"error"`
}

// parseError turns a non-2xx Stripe response into a real *Error. When the body
// is not the expected envelope it still returns a real *Error carrying the HTTP
// status, so a caller is NEVER handed a nil error on a failed call.
func parseError(status int, body []byte) *Error {
	e := &Error{HTTPStatus: status}
	var env stripeErrorEnvelope
	if err := json.Unmarshal(body, &env); err == nil {
		e.Type = env.Error.Type
		e.Code = env.Error.Code
		e.Message = env.Error.Message
		e.Param = env.Error.Param
	}
	return e
}
