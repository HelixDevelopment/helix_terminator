//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/user-service/internal/handler"
	"github.com/helixdevelopment/user-service/internal/repository"
	"github.com/helixdevelopment/user-service/internal/testinfra"
)

// TestReadinessCheck_RealDatabase_ReflectsGenuineConnectivity is the
// T8-6 anti-bluff proof (Constitution §11.4 / §11.4.69 / §11.4.107):
// ReadinessCheck MUST report ready (200) only when the database is
// genuinely reachable, and not-ready (503) the moment it is not -
// never a fabricated "status":"ready" regardless of DB state.
//
// Against the pre-fix handler (unconditional `{"status":"ready"}`, no
// DB check whatsoever) this test FAILS on the closed-pool assertion
// below: the handler keeps returning 200/ready even after the pool is
// closed, reproducing the live PASS-bluff a crashed-DB user-service
// would otherwise hide from orchestrator/k8s health gating. Post-fix
// it must PASS both assertions.
func TestReadinessCheck_RealDatabase_ReflectsGenuineConnectivity(t *testing.T) {
	dbURL := testinfra.StartPostgres(t)

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New failed: %v", err)
	}

	repo := repository.New(pool)
	h := handler.New(repo)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/healthz", h.HealthCheck)

	doRequest := func(path string) (*httptest.ResponseRecorder, map[string]interface{}) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		r.ServeHTTP(w, req)
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response for %s: %v (body=%s)", path, err, w.Body.String())
		}
		return w, resp
	}

	// Real, reachable DB -> genuinely ready.
	w, resp := doRequest("/healthz/ready")
	if w.Code != http.StatusOK {
		t.Fatalf("reachable DB: expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	if resp["status"] != "ready" {
		t.Fatalf("reachable DB: expected status:ready, got %v (body=%s)", resp["status"], w.Body.String())
	}

	// Genuinely sever DB connectivity - the exact bug scenario (a
	// crashed/unreachable database). Pre-fix, the handler fabricates
	// status:ready here regardless (the T8-6 bluff); post-fix it MUST
	// report 503 + status:not_ready.
	pool.Close()

	w, resp = doRequest("/healthz/ready")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed DB: expected 503 Service Unavailable, got %d (body=%s) - readiness endpoint fabricated ready despite unreachable database (T8-6 bluff)", w.Code, w.Body.String())
	}
	if resp["status"] == "ready" {
		t.Fatalf("closed DB: expected non-ready status, got %v (body=%s)", resp["status"], w.Body.String())
	}

	// Liveness is NOT readiness: /healthz MUST stay unconditional 200
	// even with the DB pool closed - a live-but-not-ready process must
	// still report itself alive so the orchestrator does not
	// kill+restart-loop it while it waits for the DB to come back.
	w, _ = doRequest("/healthz")
	if w.Code != http.StatusOK {
		t.Fatalf("liveness (/healthz) must stay 200 regardless of DB state, got %d (body=%s)", w.Code, w.Body.String())
	}
}
