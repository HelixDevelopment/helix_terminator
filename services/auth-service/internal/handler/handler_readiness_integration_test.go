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

	"github.com/helixdevelopment/auth-service/internal/handler"
	"github.com/helixdevelopment/auth-service/internal/repository"
	"github.com/helixdevelopment/auth-service/internal/testinfra"
)

// TestReadinessCheck_RealDatabase_ReflectsGenuineConnectivity is the
// T8-6 anti-bluff proof (Constitution §11.4 / §11.4.69 / §11.4.107):
// ReadinessCheck MUST report ready:true (200) only when the database
// is genuinely reachable, and ready:false (503) the moment it is not
// - never a fabricated ready:true regardless of DB state.
//
// Against the pre-fix handler (unconditional `{"ready": true}`, no
// DB check whatsoever) this test FAILS on the closed-pool assertion
// below: the handler keeps returning 200/ready:true even after the
// pool is closed, reproducing the live PASS-bluff a crashed-DB
// auth-service would otherwise hide from orchestrator/k8s health
// gating. Post-fix it must PASS both assertions.
func TestReadinessCheck_RealDatabase_ReflectsGenuineConnectivity(t *testing.T) {
	dbURL := testinfra.StartPostgres(t)

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New failed: %v", err)
	}

	repo := repository.New(pool)
	h := handler.New(repo, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/healthz/live", h.HealthCheck)

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
	if resp["ready"] != true {
		t.Fatalf("reachable DB: expected ready:true, got %v (body=%s)", resp["ready"], w.Body.String())
	}

	// Genuinely sever DB connectivity - the exact bug scenario (a
	// crashed/unreachable database). Pre-fix, the handler fabricates
	// ready:true here regardless (the T8-6 bluff); post-fix it MUST
	// report 503 + ready:false.
	pool.Close()

	w, resp = doRequest("/healthz/ready")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed DB: expected 503 Service Unavailable, got %d (body=%s) - readiness endpoint fabricated ready despite unreachable database (T8-6 bluff)", w.Code, w.Body.String())
	}
	if resp["ready"] != false {
		t.Fatalf("closed DB: expected ready:false, got %v (body=%s)", resp["ready"], w.Body.String())
	}

	// Liveness is NOT readiness: /healthz/live MUST stay unconditional
	// 200 even with the DB pool closed - a live-but-not-ready process
	// must still report itself alive so the orchestrator does not
	// kill+restart-loop it while it waits for the DB to come back.
	w, _ = doRequest("/healthz/live")
	if w.Code != http.StatusOK {
		t.Fatalf("liveness must stay 200 regardless of DB state, got %d (body=%s)", w.Code, w.Body.String())
	}
}
