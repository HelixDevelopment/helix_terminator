package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/billing-service/internal/billing"
	"github.com/helixdevelopment/billing-service/internal/handler"
	"github.com/helixdevelopment/billing-service/internal/model"
	"github.com/helixdevelopment/billing-service/internal/repository"
	"github.com/helixdevelopment/billing-service/internal/testutil"
)

// mockProvider is a hand-written test double for billing.PaymentProvider
// — permitted here per Constitution §11.4.27(A) because this is a
// UNIT-test source file (package handler_test). It exists to let
// handler-layer unit tests exercise the "a provider IS configured"
// branch of every handler without making a real network call; the
// provider's OWN real-processor behaviour is proven separately by
// internal/billing's unit tests (real cryptographic webhook proof,
// real parameter-construction proofs against a fake stripeClient) and
// by stripe_provider_integration_test.go (build tag "integration")
// against the real Stripe API. Every other test type here that needs a
// genuinely-configured provider (stress/chaos/integration) MUST use a
// real billing.NewProviderFromEnv() result, never this mock — see
// handler_stress_test.go / handler_chaos_test.go's STRIPE_SECRET_KEY
// gating.
type mockProvider struct {
	createFn func(ctx context.Context, in billing.CreateSubscriptionInput) (*billing.SubscriptionResult, error)
	updateFn func(ctx context.Context, in billing.UpdateSubscriptionInput) (*billing.SubscriptionResult, error)
	cancelFn func(ctx context.Context, in billing.CancelSubscriptionInput) (*billing.SubscriptionResult, error)
}

func (m *mockProvider) Name() string { return "mock" }

func (m *mockProvider) CreateSubscription(ctx context.Context, in billing.CreateSubscriptionInput) (*billing.SubscriptionResult, error) {
	if m.createFn != nil {
		return m.createFn(ctx, in)
	}
	return &billing.SubscriptionResult{ExternalSubscriptionID: "sub_mock", ExternalCustomerID: "cus_mock", Status: "active"}, nil
}

func (m *mockProvider) UpdateSubscription(ctx context.Context, in billing.UpdateSubscriptionInput) (*billing.SubscriptionResult, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, in)
	}
	return &billing.SubscriptionResult{ExternalSubscriptionID: in.ExternalSubscriptionID, Status: "active"}, nil
}

func (m *mockProvider) CancelSubscription(ctx context.Context, in billing.CancelSubscriptionInput) (*billing.SubscriptionResult, error) {
	if m.cancelFn != nil {
		return m.cancelFn(ctx, in)
	}
	return &billing.SubscriptionResult{ExternalSubscriptionID: in.ExternalSubscriptionID, Status: "canceled"}, nil
}

func (m *mockProvider) VerifyWebhook(payload []byte, signatureHeader string) (*billing.WebhookEvent, error) {
	return nil, fmt.Errorf("mockProvider.VerifyWebhook not implemented by this test double")
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

// setupAuthedTestHandler mounts the billing API routes with a lightweight
// test-only middleware that injects a caller org identity into the gin
// context under the SAME key ("orgID") the real authMiddleware
// (internal/server/server.go) populates from a validated JWT claim.
// T14: every write handler now derives the subscription's org
// EXCLUSIVELY from this context value — never from client-supplied
// input — so unit-level tests that need to reach past the 401
// "missing or invalid caller identity" guard use this helper. opts are
// forwarded to handler.New (e.g. handler.WithProvider(&mockProvider{})).
func setupAuthedTestHandler(t *testing.T, orgID string, opts ...handler.Option) (*handler.Handler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := repository.New(nil)
	h := handler.New(repo, opts...)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("orgID", orgID)
		c.Next()
	})
	r.POST("/api/v1/subscriptions", h.CreateSubscription)
	r.GET("/api/v1/subscriptions/:id", h.GetSubscription)
	r.PUT("/api/v1/subscriptions/:id", h.UpdateSubscription)
	r.DELETE("/api/v1/subscriptions/:id", h.CancelSubscription)
	r.GET("/api/v1/subscriptions", h.ListSubscriptions)

	return h, r
}

func TestHealthCheck(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.GET("/healthz", h.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["status"] != "healthy" {
		t.Fatalf("expected status healthy, got %v", resp["status"])
	}
}

func TestReadinessCheck(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.GET("/healthz/ready", h.ReadinessCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}
}

func TestCreateSubscriptionValidation(t *testing.T) {
	r := setupTestRouter()
	h := handler.New(nil)
	r.POST("/api/v1/subscriptions", h.CreateSubscription)

	body := model.CreateSubscriptionRequest{
		PlanID: "not-a-uuid",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusCreated {
		t.Fatal("expected non-2xx for invalid input")
	}
}

// ---------------------------------------------------------------------------
// T14: write-side IDOR closure tests
// ---------------------------------------------------------------------------

// TestCreateSubscription_RequiresCallerIdentity is the T14 RED→GREEN proof:
// a well-formed create request with NO caller identity in the gin context
// (the pre-fix code path derived the subscription's org from a
// client-supplied body field) MUST be rejected 401.
func TestCreateSubscription_RequiresCallerIdentity(t *testing.T) {
	r := setupTestRouter()
	repo := repository.New(nil)
	h := handler.New(repo)
	r.POST("/api/v1/subscriptions", h.CreateSubscription)

	body := model.CreateSubscriptionRequest{
		PlanID: uuid.New().String(),
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d — body: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); !containsStr(got, "missing or invalid caller identity") {
		t.Fatalf("expected 'missing or invalid caller identity' in body, got: %s", got)
	}
}

// TestCreateSubscription_UsesContextOrgID proves the created subscription's
// org comes exclusively from the caller's JWT context, not from any
// client-supplied field. Pre-fix, a legacy client could send an "orgId"
// body field to attribute the subscription to any org.
//
// A mock provider is wired (Constitution §11.4.27(A): permitted in this
// unit-test file) purely to reach PAST the honest-501 "no provider
// configured" gate this endpoint now has — see
// TestCreateSubscription_NoProvider_Returns501 for the dedicated proof
// of THAT gate. This test's own concern is unchanged: with no DB wired
// (repository.New(nil)), the request must still fail at PERSISTENCE
// (503), proving it reached business logic carrying the CALLER's org
// identity — not a client-supplied one — all the way through.
func TestCreateSubscription_UsesContextOrgID(t *testing.T) {
	callerOrg := uuid.New().String()
	_, r := setupAuthedTestHandler(t, callerOrg, handler.WithProvider(&mockProvider{}))

	// The request body has NO orgId field (it was removed from the
	// request struct in T14). The subscription must be owned by callerOrg.
	priceID := "price_test123"
	body := model.CreateSubscriptionRequest{
		PlanID:        uuid.New().String(),
		StripePriceID: &priceID,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// No DB wired (repository.New(nil)), so the request fails at
	// persistence (503) — but it MUST fail there having reached
	// business logic with the CALLER's org identity. A pre-fix build
	// would have attempted to use a client-supplied orgId and reached
	// the same 503, but via a DIFFERENT (insecure) code path.
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (no DB), got %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------
// Constitution §11.4 anti-bluff: honest-501 payments-provider gate.
//
// THE BLUFF THIS SUITE PROVES CLOSED: CreateSubscription used to
// persist a new subscription row with Status:"active" unconditionally
// — no payment processor was ever contacted. It now REQUIRES a
// configured billing.PaymentProvider and, with none configured,
// responds 501 "payments provider not configured" — NEVER a fabricated
// "active". This is the real test the task's honest-501 requirement
// asks for: handler.New(repo) below is called with ZERO options, so
// h.provider is nil — exactly the state a freshly-deployed process
// with STRIPE_SECRET_KEY unset is in (see
// internal/billing.NewProviderFromEnv, internal/server.New).
// ---------------------------------------------------------------------

// TestCreateSubscription_NoProvider_Returns501 is the honest-501
// RED→GREEN proof at the handler layer: a fully well-formed, correctly
// authenticated create request, with NO PaymentProvider configured,
// MUST be rejected 501 — never persisted, never reported as a success
// of any kind.
func TestCreateSubscription_NoProvider_Returns501(t *testing.T) {
	callerOrg := uuid.New().String()
	_, r := setupAuthedTestHandler(t, callerOrg) // NO handler.WithProvider(...) — the honest default.

	priceID := "price_test123"
	body := model.CreateSubscriptionRequest{
		PlanID:        uuid.New().String(),
		StripePriceID: &priceID,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 (no payments provider configured), got %d — body: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); !containsStr(got, "payments provider not configured") {
		t.Fatalf("expected 'payments provider not configured' in body, got: %s", got)
	}
}

// TestCreateSubscription_NoProvider_NeverFabricatesActive is the direct
// negative proof of the ORIGINAL bluff: even with a syntactically
// perfect create request (valid caller identity, valid planId), the
// response body MUST NEVER contain a fabricated "active" status when no
// provider is configured — it must be the 501 error body, full stop.
func TestCreateSubscription_NoProvider_NeverFabricatesActive(t *testing.T) {
	callerOrg := uuid.New().String()
	_, r := setupAuthedTestHandler(t, callerOrg)

	priceID := "price_test123"
	body := model.CreateSubscriptionRequest{PlanID: uuid.New().String(), StripePriceID: &priceID}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if status, ok := resp["status"]; ok && status == "active" {
		t.Fatalf("BLUFF DETECTED: response fabricated status=active with no payment provider configured — body: %s", w.Body.String())
	}
	if _, hasID := resp["id"]; hasID {
		t.Fatalf("BLUFF DETECTED: response carries a subscription id with no payment provider configured (implies a row was persisted) — body: %s", w.Body.String())
	}
}

// TestCreateSubscription_MissingStripePriceID_Returns400 proves the
// provider-configured path still refuses a request that omits the
// required stripePriceId, rather than silently proceeding with an
// empty/zero-value price.
func TestCreateSubscription_MissingStripePriceID_Returns400(t *testing.T) {
	callerOrg := uuid.New().String()
	_, r := setupAuthedTestHandler(t, callerOrg, handler.WithProvider(&mockProvider{}))

	body := model.CreateSubscriptionRequest{PlanID: uuid.New().String()} // no StripePriceID
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (missing stripePriceId), got %d — body: %s", w.Code, w.Body.String())
	}
}

// dbBackedTestHandler boots a real, disposable PostgreSQL container
// (rootless podman, §11.4.161) with migrations applied and returns a
// Handler wired to it, with the given provider, PLUS the underlying
// *repository.Repository so tests can seed rows directly against the
// SAME pool (handler.Handler intentionally does not export its repo
// field in production code — Constitution §11.4.28 decoupling — so
// this indirection lives only here, in a unit-test file). Constitution
// §11.4.27(A): the DATABASE side of these tests is real, never mocked
// — only the PaymentProvider (the seam this file's mockProvider exists
// to isolate) is a test double, and only because this is a unit-test
// file. SKIPs honestly when podman is unavailable.
func dbBackedTestHandler(t *testing.T, provider billing.PaymentProvider) (*handler.Handler, *gin.Engine, *repository.Repository) {
	t.Helper()
	poolURL, available := testutil.StartTestPostgres(t)
	if !available {
		t.Skip("SKIP: podman not available — cannot run this test against a real database (topology_unsupported)")
	}
	pool, err := pgxpool.New(t.Context(), poolURL)
	if err != nil {
		t.Fatalf("pgxpool.New failed: %v", err)
	}
	t.Cleanup(pool.Close)

	repo := repository.New(pool)
	h := handler.New(repo, handler.WithProvider(provider))
	r := gin.New()
	return h, r, repo
}

// TestCreateSubscription_HonestStatusPassthrough proves the handler
// persists+returns the REAL status the provider returned (here,
// deliberately "incomplete", never "active") rather than assuming
// success means active. Runs against a REAL database (see
// dbBackedTestHandler) so the mock provider's result is genuinely
// reached and genuinely persisted — not short-circuited by an earlier
// "database not connected" response.
func TestCreateSubscription_HonestStatusPassthrough(t *testing.T) {
	callerOrg := uuid.New()
	mp := &mockProvider{
		createFn: func(ctx context.Context, in billing.CreateSubscriptionInput) (*billing.SubscriptionResult, error) {
			return &billing.SubscriptionResult{ExternalSubscriptionID: "sub_incomplete", ExternalCustomerID: "cus_x", Status: "incomplete"}, nil
		},
	}
	h, r, _ := dbBackedTestHandler(t, mp)
	r.Use(func(c *gin.Context) { c.Set("orgID", callerOrg.String()); c.Next() })
	r.POST("/api/v1/subscriptions", h.CreateSubscription)

	priceID := "price_test123"
	body := model.CreateSubscriptionRequest{PlanID: uuid.New().String(), StripePriceID: &priceID}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp model.SubscriptionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != "incomplete" {
		t.Fatalf("BLUFF DETECTED: expected honest status passthrough 'incomplete', got %q — a bluff would report 'active'", resp.Status)
	}
	if resp.ExternalSubscriptionID != "sub_incomplete" {
		t.Fatalf("expected ExternalSubscriptionID sub_incomplete, got %q", resp.ExternalSubscriptionID)
	}
	if resp.Provider != "mock" {
		t.Fatalf("expected Provider mock, got %q", resp.Provider)
	}
}

// TestCreateSubscription_ProviderErrorPropagates proves a real
// processor-side rejection (e.g. an invalid price) surfaces as a 502
// with the processor's own detail — never swallowed into any kind of
// success, and never persisted.
func TestCreateSubscription_ProviderErrorPropagates(t *testing.T) {
	callerOrg := uuid.New()
	mp := &mockProvider{
		createFn: func(ctx context.Context, in billing.CreateSubscriptionInput) (*billing.SubscriptionResult, error) {
			return nil, fmt.Errorf("resource_missing: no such price: %q", in.PriceID)
		},
	}
	h, r, _ := dbBackedTestHandler(t, mp)
	r.Use(func(c *gin.Context) { c.Set("orgID", callerOrg.String()); c.Next() })
	r.POST("/api/v1/subscriptions", h.CreateSubscription)
	r.GET("/api/v1/subscriptions", h.ListSubscriptions)

	priceID := "price_bad"
	body := model.CreateSubscriptionRequest{PlanID: uuid.New().String(), StripePriceID: &priceID}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/subscriptions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 (provider rejected), got %d — body: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); !containsStr(got, "no such price") {
		t.Fatalf("expected processor error detail in body, got: %s", got)
	}

	// Nothing must have been persisted.
	w2 := httptest.NewRecorder()
	listReq, _ := http.NewRequest("GET", "/api/v1/subscriptions", nil)
	r.ServeHTTP(w2, listReq)
	var listResp model.ListSubscriptionsResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to unmarshal list response: %v", err)
	}
	if listResp.Total != 0 {
		t.Fatalf("BLUFF DETECTED: a rejected create left %d row(s) persisted, want 0", listResp.Total)
	}
}

// TestUpdateSubscription_RequiresCallerIdentity is the T14 RED→GREEN proof:
// a well-formed update request with NO caller identity in the gin context
// MUST be rejected 401.
func TestUpdateSubscription_RequiresCallerIdentity(t *testing.T) {
	r := setupTestRouter()
	repo := repository.New(nil)
	h := handler.New(repo)
	r.PUT("/api/v1/subscriptions/:id", h.UpdateSubscription)

	subID := uuid.New().String()
	newStatus := "canceled"
	body := model.UpdateSubscriptionRequest{
		Status: &newStatus,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/subscriptions/%s", subID), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d — body: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); !containsStr(got, "missing or invalid caller identity") {
		t.Fatalf("expected 'missing or invalid caller identity' in body, got: %s", got)
	}
}

// TestUpdateSubscription_InvalidIDBeforeIdentity proves request-shape
// validation (bad UUID in path) runs BEFORE the identity check — a
// regression guard that the T14 fix didn't change the validation order.
func TestUpdateSubscription_InvalidIDBeforeIdentity(t *testing.T) {
	r := setupTestRouter()
	repo := repository.New(nil)
	h := handler.New(repo)
	r.PUT("/api/v1/subscriptions/:id", h.UpdateSubscription)

	newStatus := "canceled"
	body := model.UpdateSubscriptionRequest{
		Status: &newStatus,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/subscriptions/not-a-uuid", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestCancelSubscription_RequiresCallerIdentity is the T14 RED→GREEN proof:
// a cancel request with NO caller identity in the gin context MUST be
// rejected 401.
func TestCancelSubscription_RequiresCallerIdentity(t *testing.T) {
	r := setupTestRouter()
	repo := repository.New(nil)
	h := handler.New(repo)
	r.DELETE("/api/v1/subscriptions/:id", h.CancelSubscription)

	subID := uuid.New().String()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/subscriptions/%s", subID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d — body: %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); !containsStr(got, "missing or invalid caller identity") {
		t.Fatalf("expected 'missing or invalid caller identity' in body, got: %s", got)
	}
}

// TestCancelSubscription_InvalidIDBeforeIdentity proves request-shape
// validation (bad UUID in path) runs BEFORE the identity check.
func TestCancelSubscription_InvalidIDBeforeIdentity(t *testing.T) {
	r := setupTestRouter()
	repo := repository.New(nil)
	h := handler.New(repo)
	r.DELETE("/api/v1/subscriptions/:id", h.CancelSubscription)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/subscriptions/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------
// Constitution §11.4 anti-bluff: processor-backed Update/Cancel MUST
// go through the configured provider, never mutate local state alone.
// ---------------------------------------------------------------------

// TestCancelSubscription_ProcessorBacked_NoProvider_Returns501 proves a
// subscription that WAS created through a real processor
// (ExternalSubscriptionID != "") can never be canceled while no
// provider is configured — doing so locally-only would silently
// diverge this service's record from the processor's real state.
func TestCancelSubscription_ProcessorBacked_NoProvider_Returns501(t *testing.T) {
	h, r, repo := dbBackedTestHandler(t, nil) // explicit nil provider
	orgID := uuid.New()
	r.Use(func(c *gin.Context) { c.Set("orgID", orgID.String()); c.Next() })
	r.DELETE("/api/v1/subscriptions/:id", h.CancelSubscription)

	sub := &model.Subscription{
		ID: uuid.New(), OrgID: orgID, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC(),
		Provider: "stripe", ExternalSubscriptionID: "sub_real123", ExternalCustomerID: "cus_real123",
	}
	if err := repo.CreateSubscription(context.Background(), sub); err != nil {
		t.Fatalf("seed CreateSubscription failed: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/subscriptions/"+sub.ID.String(), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 (processor-backed row, no provider configured), got %d — body: %s", w.Code, w.Body.String())
	}

	got, err := repo.GetSubscriptionByID(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriptionByID failed: %v", err)
	}
	if got.Status != "active" {
		t.Fatalf("BLUFF DETECTED: subscription status changed to %q with no provider ever contacted, want unchanged %q", got.Status, "active")
	}
}

// TestCancelSubscription_ProcessorBacked_CallsProvider proves canceling
// a processor-backed subscription with a configured provider actually
// calls it (not just mutates the DB) and persists the REAL returned
// status.
func TestCancelSubscription_ProcessorBacked_CallsProvider(t *testing.T) {
	var calledWith string
	mp := &mockProvider{
		cancelFn: func(ctx context.Context, in billing.CancelSubscriptionInput) (*billing.SubscriptionResult, error) {
			calledWith = in.ExternalSubscriptionID
			return &billing.SubscriptionResult{ExternalSubscriptionID: in.ExternalSubscriptionID, Status: "canceled"}, nil
		},
	}
	h, r, repo := dbBackedTestHandler(t, mp)
	orgID := uuid.New()
	r.Use(func(c *gin.Context) { c.Set("orgID", orgID.String()); c.Next() })
	r.DELETE("/api/v1/subscriptions/:id", h.CancelSubscription)

	sub := &model.Subscription{
		ID: uuid.New(), OrgID: orgID, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC(),
		Provider: "mock", ExternalSubscriptionID: "sub_to_cancel", ExternalCustomerID: "cus_1",
	}
	if err := repo.CreateSubscription(context.Background(), sub); err != nil {
		t.Fatalf("seed CreateSubscription failed: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/subscriptions/"+sub.ID.String(), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}
	if calledWith != "sub_to_cancel" {
		t.Fatalf("BLUFF DETECTED: provider.CancelSubscription was not called with the row's external subscription id (got %q) — cancel may have been applied locally-only", calledWith)
	}

	got, err := repo.GetSubscriptionByID(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriptionByID failed: %v", err)
	}
	if got.Status != "canceled" {
		t.Fatalf("Status = %q, want %q (the provider's real returned status)", got.Status, "canceled")
	}
}

// TestUpdateSubscription_PlanChange_ProcessorBacked_RequiresStripePriceID
// proves a plan change on a processor-backed subscription is rejected
// 400 when stripePriceId is omitted, rather than silently proceeding.
func TestUpdateSubscription_PlanChange_ProcessorBacked_RequiresStripePriceID(t *testing.T) {
	h, r, repo := dbBackedTestHandler(t, &mockProvider{})
	orgID := uuid.New()
	r.Use(func(c *gin.Context) { c.Set("orgID", orgID.String()); c.Next() })
	r.PUT("/api/v1/subscriptions/:id", h.UpdateSubscription)

	sub := &model.Subscription{
		ID: uuid.New(), OrgID: orgID, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC(),
		Provider: "mock", ExternalSubscriptionID: "sub_x", ExternalCustomerID: "cus_x",
	}
	if err := repo.CreateSubscription(context.Background(), sub); err != nil {
		t.Fatalf("seed CreateSubscription failed: %v", err)
	}

	newPlanID := uuid.New().String()
	body := model.UpdateSubscriptionRequest{PlanID: &newPlanID} // no StripePriceID
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/subscriptions/"+sub.ID.String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (missing stripePriceId for plan change), got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestUpdateSubscription_PlanAndStatusTogether_Rejected proves the
// mutually-exclusive validation: a single PUT cannot both change the
// plan (a real processor operation) and set a local status literal in
// the same request — this closes the ordering ambiguity of which value
// would "win".
func TestUpdateSubscription_PlanAndStatusTogether_Rejected(t *testing.T) {
	callerOrg := uuid.New().String()
	_, r := setupAuthedTestHandler(t, callerOrg, handler.WithProvider(&mockProvider{}))

	planID := uuid.New().String()
	status := "canceled"
	body := model.UpdateSubscriptionRequest{PlanID: &planID, Status: &status}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/subscriptions/"+uuid.New().String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (plan+status together), got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestUpdateSubscription_StatusActive_Rejected proves the closed
// allowed-status-values fix: "active" is no longer a value a client can
// assert via PUT — it removes exactly the local-only-bluff path this
// endpoint previously permitted (a client PUTting status="active" with
// zero processor involvement).
func TestUpdateSubscription_StatusActive_Rejected(t *testing.T) {
	callerOrg := uuid.New().String()
	_, r := setupAuthedTestHandler(t, callerOrg, handler.WithProvider(&mockProvider{}))

	status := "active"
	body := model.UpdateSubscriptionRequest{Status: &status}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/subscriptions/"+uuid.New().String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("BLUFF-CAPABLE PATH DETECTED: expected 400 rejecting status=active via PUT, got %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------
// StripeWebhook
// ---------------------------------------------------------------------

// TestStripeWebhook_NoProvider_Returns501 proves the webhook endpoint
// honestly refuses when no provider is configured — it cannot verify a
// signature it has no secret for, so it must never pretend to.
func TestStripeWebhook_NoProvider_Returns501(t *testing.T) {
	h := handler.New(repository.New(nil))
	r := gin.New()
	r.POST("/api/v1/webhooks/stripe", h.StripeWebhook)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/stripe", bytes.NewBufferString(`{}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d — body: %s", w.Code, w.Body.String())
	}
}

// TestStripeWebhook_InvalidSignature_Rejected proves a payload whose
// signature does not verify is rejected 400 — the webhook endpoint
// never trusts an unverified payload, however well-formed its JSON.
func TestStripeWebhook_InvalidSignature_Rejected(t *testing.T) {
	mp := &mockProvider{}
	h := handler.New(repository.New(nil), handler.WithProvider(mp))
	r := gin.New()
	r.POST("/api/v1/webhooks/stripe", h.StripeWebhook)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/stripe", bytes.NewBufferString(`{"id":"evt_1","type":"customer.subscription.updated"}`))
	req.Header.Set("Stripe-Signature", "t=1,v1=deadbeef")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (signature verification failed — mockProvider.VerifyWebhook always errors), got %d — body: %s", w.Code, w.Body.String())
	}
}

// containsStr is a test helper to avoid importing strings in the test file.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
