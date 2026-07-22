//go:build integration

// Package server_test — REAL cross-tenant isolation proof against a real
// PostgreSQL instance and the REAL billing-service HTTP server (T12,
// §11.4.27 / §11.4.107 / §11.4.115). Excluded from the default
// `go test ./...` run (build tag `integration`). Requires:
//
//	export DATABASE_URL="postgres://postgres:postgres@127.0.0.1:15432/billing_service_test?sslmode=disable"
//	go test -tags integration ./internal/server/...
//
// Forensic anchor (T12): ListSubscriptions (and GetSubscription,
// ListInvoices, GetInvoice) derived their tenant filter from a
// client-supplied "orgId" query parameter (or no filter at all when
// omitted), never from the caller's authenticated identity — any caller
// could read another org's subscriptions/invoices, or all orgs' data at
// once, by omitting/spoofing the orgId parameter. This file seeds TWO
// distinct, real tenants directly into Postgres via the real repository,
// then drives the real HTTP server (server.Router()) as each tenant and
// asserts tenant A can NEVER observe tenant B's rows. Run against the
// pre-fix handler this test FAILS (RED) — see docs/qa evidence captured
// alongside the T12 fix commit. Run against the fixed handler it PASSES
// (GREEN): tenant identity now comes exclusively from a validated JWT
// (Ed25519, same mechanism as gateway-service) via the gin context, never
// from client input, and requests carrying no valid identity are
// rejected with 401 rather than served unscoped data.
package server_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/billing-service/internal/model"
	"github.com/helixdevelopment/billing-service/internal/repository"
	"github.com/helixdevelopment/billing-service/internal/server"
	"github.com/helixdevelopment/billing-service/migrations"
)

// testClaims mirrors gateway-service's Claims struct (services/
// gateway-service/internal/server/server.go) — the gateway forwards the
// original signed Authorization bearer token to upstream services
// untouched (proxyTo clones request headers verbatim, never strips
// Authorization), so billing-service independently validates the SAME
// token shape with the SAME claim names.
type testClaims struct {
	UserID string `json:"userId"`
	OrgID  string `json:"orgId,omitempty"`
	jwt.RegisteredClaims
}

// integrationTestLogger adapts *testing.T to the migrations.Logger
// interface (mirrors internal/testutil/postgres_helper.go's
// pgTestLogger).
type integrationTestLogger struct{ t *testing.T }

func (l *integrationTestLogger) Printf(format string, v ...interface{}) {
	l.t.Logf("[migrations] "+format, v...)
}

// mustConnectAndMigrate connects to the real Postgres pointed at by
// DATABASE_URL and applies billing-service's REAL embedded migrations
// (migrations.Run — the exact mechanism cmd/billing-service/main.go
// runs at process startup, see migrations/migrate.go) idempotently.
// Skips (does not fail) when DATABASE_URL is unset — the correct
// §11.4.3 topology-appropriate behaviour for an integration test with
// no real target.
//
// Constitution §11.4.108 (fixed a pre-existing, unrelated defect
// discovered while validating this stream's own changes against this
// file, per §11.4.124 investigate-before-touching): this helper
// previously hand-read a file at "../../migrations/001_init.sql" (note:
// no ".up" suffix) that never existed on disk (the real files are
// 001_init.up.sql / 001_init.down.sql / 002_payment_provider.up.sql /
// 002_payment_provider.down.sql), and applied it to the default
// "public" schema — diverging from the real service's schema-per-
// service migration path (migrations.Run, search_path=migrations.Schema)
// cmd/billing-service/main.go actually uses. Every integration test in
// this file (and server_write_isolation_integration_test.go, which
// shares this helper) was therefore unable to run against a real
// Postgres instance at all, pre-existing and unrelated to Stripe/
// PaymentProvider work — captured evidence: `go test -tags integration`
// against a real container failed with "open
// ../../migrations/001_init.sql: no such file or directory" BEFORE this
// fix. Now reuses the real migrations.Run/ConnectionURL path so the
// pool returned actually matches production schema targeting.
func mustConnectAndMigrate(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping real-Postgres integration test (§11.4.3)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// §11.4.98: this test MUST be re-runnable endlessly against the same
	// disposable Postgres without manual intervention — reset the
	// service's dedicated schema to a clean slate first so every run
	// starts from the same real, freshly-migrated state.
	resetPool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to open pgxpool against DATABASE_URL")
	require.NoError(t, resetPool.Ping(ctx), "real Postgres at DATABASE_URL is not reachable")
	_, err = resetPool.Exec(ctx, "DROP SCHEMA IF EXISTS "+migrations.Schema+" CASCADE")
	require.NoError(t, err, "failed to reset %q schema before migrating", migrations.Schema)
	resetPool.Close()

	_, err = migrations.Run(dbURL, &integrationTestLogger{t: t})
	require.NoError(t, err, "failed to apply real embedded migrations to real Postgres")

	poolURL, err := migrations.ConnectionURL(dbURL)
	require.NoError(t, err, "failed to build schema-scoped connection URL")

	pool, err := pgxpool.New(ctx, poolURL)
	require.NoError(t, err, "failed to open schema-scoped pgxpool")
	require.NoError(t, pool.Ping(ctx), "schema-scoped pgxpool is not reachable")

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// mustNewServerWithRealJWTKey generates a real Ed25519 keypair, points
// billing-service's JWT_PUBLIC_KEY env var at the public half (mirroring
// how gateway-service is provisioned), and returns the built server
// alongside a signer bound to the private half so the test can mint
// tokens exactly as auth-service would.
func mustNewServerWithRealJWTKey(t *testing.T, repo *repository.Repository) (*server.Server, func(orgID, userID string) string) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	prevKey, hadPrevKey := os.LookupEnv("JWT_PUBLIC_KEY")
	require.NoError(t, os.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(pub)))
	t.Cleanup(func() {
		if hadPrevKey {
			os.Setenv("JWT_PUBLIC_KEY", prevKey)
		} else {
			os.Unsetenv("JWT_PUBLIC_KEY")
		}
	})

	srv := server.New(repo)

	sign := func(orgID, userID string) string {
		claims := testClaims{
			UserID: userID,
			OrgID:  orgID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		signed, err := tok.SignedString(priv)
		require.NoError(t, err)
		return signed
	}

	return srv, sign
}

func doRequest(t *testing.T, srv *server.Server, method, path, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	return w
}

// TestBillingCrossTenantIsolation_RealPostgres is the T12 anti-bluff proof.
// It seeds two REAL, distinct tenants' subscriptions and invoices directly
// through the real repository against a real Postgres instance, then
// drives the real HTTP server as each tenant and proves tenant A can never
// observe tenant B's rows — for ListSubscriptions, GetSubscription,
// ListInvoices, and GetInvoice — and that a request carrying no valid
// caller identity is rejected outright rather than served unscoped data.
func TestBillingCrossTenantIsolation_RealPostgres(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pool := mustConnectAndMigrate(t)
	repo := repository.New(pool)
	srv, sign := mustNewServerWithRealJWTKey(t, repo)

	ctx := context.Background()

	orgA := uuid.New()
	orgB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()

	subA := &model.Subscription{ID: uuid.New(), OrgID: orgA, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC()}
	subB := &model.Subscription{ID: uuid.New(), OrgID: orgB, PlanID: uuid.New(), Status: "active", StartedAt: time.Now().UTC()}
	require.NoError(t, repo.CreateSubscription(ctx, subA))
	require.NoError(t, repo.CreateSubscription(ctx, subB))

	invA := &model.Invoice{ID: uuid.New(), OrgID: orgA, SubscriptionID: subA.ID, AmountCents: 1000, Currency: "USD", Status: "pending", DueDate: time.Now().Add(24 * time.Hour)}
	invB := &model.Invoice{ID: uuid.New(), OrgID: orgB, SubscriptionID: subB.ID, AmountCents: 2000, Currency: "USD", Status: "pending", DueDate: time.Now().Add(24 * time.Hour)}
	require.NoError(t, repo.CreateInvoice(ctx, invA))
	require.NoError(t, repo.CreateInvoice(ctx, invB))

	tokenA := sign(orgA.String(), userA.String())
	_ = userB

	t.Run("ListSubscriptions scoped to caller org only", func(t *testing.T) {
		w := doRequest(t, srv, http.MethodGet, "/api/v1/subscriptions?limit=100", tokenA)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var resp model.ListSubscriptionsResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		var sawA, sawB bool
		for _, item := range resp.Items {
			if item.ID == subA.ID {
				sawA = true
			}
			if item.ID == subB.ID {
				sawB = true
			}
		}
		require.True(t, sawA, "tenant A's own subscription must be present")
		require.False(t, sawB, "CROSS-TENANT LEAK: tenant B's subscription was returned to tenant A")
	})

	t.Run("ListSubscriptions rejects requests with no caller identity", func(t *testing.T) {
		w := doRequest(t, srv, http.MethodGet, "/api/v1/subscriptions?limit=100", "")
		require.Equal(t, http.StatusUnauthorized, w.Code, "unauthenticated list must be rejected, not served unscoped data; body: %s", w.Body.String())
	})

	t.Run("GetSubscription blocks cross-tenant access by ID", func(t *testing.T) {
		w := doRequest(t, srv, http.MethodGet, "/api/v1/subscriptions/"+subB.ID.String(), tokenA)
		require.Equal(t, http.StatusNotFound, w.Code, "CROSS-TENANT LEAK: tenant A fetched tenant B's subscription by ID; body: %s", w.Body.String())
	})

	t.Run("GetSubscription serves the caller's own subscription", func(t *testing.T) {
		w := doRequest(t, srv, http.MethodGet, "/api/v1/subscriptions/"+subA.ID.String(), tokenA)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})

	t.Run("ListInvoices scoped to caller org only", func(t *testing.T) {
		w := doRequest(t, srv, http.MethodGet, "/api/v1/invoices?limit=100", tokenA)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var resp struct {
			Invoices []model.InvoiceResponse `json:"invoices"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		var sawA, sawB bool
		for _, item := range resp.Invoices {
			if item.ID == invA.ID {
				sawA = true
			}
			if item.ID == invB.ID {
				sawB = true
			}
		}
		require.True(t, sawA, "tenant A's own invoice must be present")
		require.False(t, sawB, "CROSS-TENANT LEAK: tenant B's invoice was returned to tenant A")
	})

	t.Run("GetInvoice blocks cross-tenant access by ID", func(t *testing.T) {
		w := doRequest(t, srv, http.MethodGet, "/api/v1/invoices/"+invB.ID.String(), tokenA)
		require.Equal(t, http.StatusNotFound, w.Code, "CROSS-TENANT LEAK: tenant A fetched tenant B's invoice by ID; body: %s", w.Body.String())
	})
}
