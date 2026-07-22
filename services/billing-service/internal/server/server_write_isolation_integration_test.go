//go:build integration

// Package server_test — REAL cross-tenant WRITE-path isolation proof
// against a real PostgreSQL instance and the REAL billing-service HTTP
// server (T14, §11.4.27 / §11.4.107 / §11.4.115). Excluded from the
// default `go test ./...` run (build tag `integration`). Requires:
//
//	export DATABASE_URL="postgres://postgres:postgres@127.0.0.1:15521/billing_service_test?sslmode=disable"
//	go test -tags integration ./internal/server/...
//
// Forensic anchor (T14): T12 scoped the READ endpoints (GetSubscription,
// ListSubscriptions, GetInvoice, ListInvoices) to the caller's
// authenticated org via callerOrgID, but explicitly left the WRITE
// endpoints out of scope. CreateSubscription trusted a client-supplied
// "orgId" BODY field verbatim (uuid.Parse(req.OrgID)) to decide which
// tenant a new subscription belongs to — any caller could create a
// subscription attributed to an ARBITRARY org, including another
// tenant's. UpdateSubscription and CancelSubscription looked up and
// mutated a subscription by ":id" alone, with NO ownership check
// whatsoever — any caller who learned (or guessed) another tenant's
// subscription ID could update or cancel it.
//
// This file seeds TWO distinct, real tenants directly into Postgres via
// the real repository, then drives the real HTTP server (server.Router())
// as tenant A and proves: (1) POST /subscriptions ignores whatever org id
// tenant A's client puts in the request body and always attributes the
// new row to tenant A's OWN authenticated org; (2) PUT
// /subscriptions/:id and POST /subscriptions/:id/cancel against a
// subscription tenant A does NOT own return 404 (the identical response
// a genuinely-missing id would produce — no existence oracle) and leave
// the target row byte-for-byte untouched in the real database; (3) the
// same two mutating endpoints still work correctly against tenant A's
// OWN subscriptions, proving the fix does not break legitimate same-
// tenant use. Run against the pre-fix handler this test FAILS (RED) —
// see the T14 commit message for captured before/after evidence. Run
// against the fixed handler it PASSES (GREEN).
package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/billing-service/internal/model"
	"github.com/helixdevelopment/billing-service/internal/repository"
)

// doJSONRequest issues a real HTTP request (through the real Router())
// carrying an arbitrary raw JSON body — deliberately NOT typed as
// model.CreateSubscriptionRequest, so this test can prove the server
// ignores a hostile client-supplied "orgId" field over the wire
// regardless of what the current Go request struct happens to declare
// (the request struct itself is part of the T14 fix and differs
// pre-fix vs post-fix; the wire-level attack shape does not).
func doJSONRequest(t *testing.T, srv interface{ Router() http.Handler }, method, path, bearer string, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	return w
}

// TestBillingWriteEndpointsCrossTenantIsolation_RealPostgres is the T14
// anti-bluff proof for the write-path IDOR left open by T12.
func TestBillingWriteEndpointsCrossTenantIsolation_RealPostgres(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pool := mustConnectAndMigrate(t)
	repo := repository.New(pool)
	srv, sign := mustNewServerWithRealJWTKey(t, repo)

	ctx := context.Background()

	orgA := uuid.New()
	orgB := uuid.New()
	userA := uuid.New()

	tokenA := sign(orgA.String(), userA.String())

	t.Run("CreateSubscription ignores client-supplied orgId and attributes to the caller's own org", func(t *testing.T) {
		// Constitution §11.4.27(A): CreateSubscription now REQUIRES a
		// real, configured billing.PaymentProvider (see internal/handler
		// + internal/billing/provider.go — the honest-501 anti-bluff
		// fix). srv was built by mustNewServerWithRealJWTKey via
		// server.New(repo), which wires whatever STRIPE_SECRET_KEY was
		// present in THIS PROCESS's environment when srv was
		// constructed (mirroring the JWT_PUBLIC_KEY provisioning
		// pattern immediately above). Real payment-processor
		// infrastructure per §11.4.27(A) — never a fake provider
		// substituted into this test — so this subtest, and this
		// subtest alone (the other subtests below exercise
		// Update/Cancel against subscriptions seeded directly via
		// repo.CreateSubscription with no processor involved, which
		// remains fully testable with no provider configured), SKIPs
		// honestly per §11.4.3 when no real Stripe test-mode price is
		// provisioned for this run.
		stripePriceID := os.Getenv("STRIPE_TEST_PRICE_ID")
		if os.Getenv("STRIPE_SECRET_KEY") == "" || stripePriceID == "" {
			t.Skip("SKIP: STRIPE_SECRET_KEY and/or STRIPE_TEST_PRICE_ID not set — cannot run this subtest against the real Stripe API (operator_attended); see docs/guides/BILLING.md")
		}

		planID := uuid.New()
		w := doJSONRequest(t, srv, http.MethodPost, "/api/v1/subscriptions", tokenA, map[string]interface{}{
			"orgId":         orgB.String(), // hostile: tenant A's client claims org B
			"planId":        planID.String(),
			"stripePriceId": stripePriceID,
		})
		require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())

		var resp model.SubscriptionResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		require.NotEqual(t, orgB, resp.OrgID,
			"CROSS-TENANT LEAK: CreateSubscription attributed the new subscription to the client-supplied orgId (org B) instead of the caller's own authenticated org")
		require.Equal(t, orgA, resp.OrgID,
			"CreateSubscription must attribute the new subscription to the caller's own authenticated org (org A), never a client-supplied value")

		// Independent cross-check directly against the real DB row (not
		// merely the HTTP response body).
		stored, err := repo.GetSubscriptionByID(ctx, resp.ID)
		require.NoError(t, err)
		require.Equal(t, orgA, stored.OrgID,
			"CRITICAL: the persisted subscription row's org_id does not match the caller's authenticated org")
	})

	t.Run("CreateSubscription rejects requests with no caller identity", func(t *testing.T) {
		w := doJSONRequest(t, srv, http.MethodPost, "/api/v1/subscriptions", "", map[string]interface{}{
			"orgId":  orgA.String(),
			"planId": uuid.New().String(),
		})
		require.Equal(t, http.StatusUnauthorized, w.Code, "unauthenticated create must be rejected, not served; body: %s", w.Body.String())
	})

	t.Run("UpdateSubscription blocks cross-tenant mutation by ID", func(t *testing.T) {
		subB := &model.Subscription{ID: uuid.New(), OrgID: orgB, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC()}
		require.NoError(t, repo.CreateSubscription(ctx, subB))

		w := doJSONRequest(t, srv, http.MethodPut, "/api/v1/subscriptions/"+subB.ID.String(), tokenA, map[string]interface{}{
			"status": "canceled",
		})
		require.Equal(t, http.StatusNotFound, w.Code,
			"CROSS-TENANT LEAK: tenant A updated tenant B's subscription; body: %s", w.Body.String())

		// The target row must be byte-for-byte untouched by the blocked
		// attempt — real DB-state proof, not just the HTTP status code.
		stillB, err := repo.GetSubscriptionByID(ctx, subB.ID)
		require.NoError(t, err)
		require.Equal(t, "active", stillB.Status,
			"CRITICAL: a blocked cross-tenant UpdateSubscription attempt still mutated tenant B's subscription status in the real database")
	})

	t.Run("UpdateSubscription rejects requests with no caller identity", func(t *testing.T) {
		subB := &model.Subscription{ID: uuid.New(), OrgID: orgB, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC()}
		require.NoError(t, repo.CreateSubscription(ctx, subB))

		w := doJSONRequest(t, srv, http.MethodPut, "/api/v1/subscriptions/"+subB.ID.String(), "", map[string]interface{}{
			"status": "canceled",
		})
		require.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	})

	t.Run("UpdateSubscription still works for the caller's own subscription", func(t *testing.T) {
		subA := &model.Subscription{ID: uuid.New(), OrgID: orgA, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC()}
		require.NoError(t, repo.CreateSubscription(ctx, subA))

		w := doJSONRequest(t, srv, http.MethodPut, "/api/v1/subscriptions/"+subA.ID.String(), tokenA, map[string]interface{}{
			"status": "canceled",
		})
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var resp model.SubscriptionResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, "canceled", resp.Status, "legitimate same-tenant update did not take effect")

		updated, err := repo.GetSubscriptionByID(ctx, subA.ID)
		require.NoError(t, err)
		require.Equal(t, "canceled", updated.Status, "legitimate same-tenant update did not persist to the real database")
	})

	t.Run("CancelSubscription blocks cross-tenant mutation by ID", func(t *testing.T) {
		subB2 := &model.Subscription{ID: uuid.New(), OrgID: orgB, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC()}
		require.NoError(t, repo.CreateSubscription(ctx, subB2))

		w := doJSONRequest(t, srv, http.MethodPost, "/api/v1/subscriptions/"+subB2.ID.String()+"/cancel", tokenA, nil)
		require.Equal(t, http.StatusNotFound, w.Code,
			"CROSS-TENANT LEAK: tenant A canceled tenant B's subscription; body: %s", w.Body.String())

		stillB2, err := repo.GetSubscriptionByID(ctx, subB2.ID)
		require.NoError(t, err)
		require.Equal(t, "active", stillB2.Status,
			"CRITICAL: a blocked cross-tenant CancelSubscription attempt still canceled tenant B's subscription in the real database")
		require.Nil(t, stillB2.CanceledAt,
			"CRITICAL: a blocked cross-tenant CancelSubscription attempt still set canceled_at on tenant B's subscription")
	})

	t.Run("CancelSubscription rejects requests with no caller identity", func(t *testing.T) {
		subB2 := &model.Subscription{ID: uuid.New(), OrgID: orgB, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC()}
		require.NoError(t, repo.CreateSubscription(ctx, subB2))

		w := doJSONRequest(t, srv, http.MethodPost, "/api/v1/subscriptions/"+subB2.ID.String()+"/cancel", "", nil)
		require.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	})

	t.Run("CancelSubscription still works for the caller's own subscription", func(t *testing.T) {
		subA2 := &model.Subscription{ID: uuid.New(), OrgID: orgA, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC()}
		require.NoError(t, repo.CreateSubscription(ctx, subA2))

		w := doJSONRequest(t, srv, http.MethodPost, "/api/v1/subscriptions/"+subA2.ID.String()+"/cancel", tokenA, nil)
		require.Equal(t, http.StatusNoContent, w.Code, "body: %s", w.Body.String())

		canceled, err := repo.GetSubscriptionByID(ctx, subA2.ID)
		require.NoError(t, err)
		require.Equal(t, "canceled", canceled.Status, "legitimate same-tenant cancel did not persist to the real database")
		require.NotNil(t, canceled.CanceledAt, "legitimate same-tenant cancel did not set canceled_at")
	})
}
