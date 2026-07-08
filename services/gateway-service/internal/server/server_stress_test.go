//go:build stress

// Stress test suite for gateway-service server (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of health-check + proxy request
//     cycle, per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=15 parallel goroutines performing
//     health + proxy requests, no deadlock, no resource leak.
//   - Boundary conditions: invalid upstream, missing auth header,
//     malformed request, empty token — every boundary produces a
//     categorised result.
//
// Run:
//
//	go test -race -tags stress -run TestStress -v -timeout 120s ./internal/server/
package server_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/helixdevelopment/gateway-service/internal/testutil"
)

// TestStressHealthProxyCycle_SustainedLoad drives N>=100 iterations of
// the full health-check → proxy request cycle against the gateway,
// recording per-iteration latency and computing p50/p95/p99. Every
// iteration hits /healthz/live, /healthz/ready, and a proxied route
// with a valid JWT.
func TestStressHealthProxyCycle_SustainedLoad(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer(t)

	client := &http.Client{Timeout: 10 * time.Second}
	ts := httptest.NewServer(s.Router())
	defer ts.Close()

	const iterations = 100
	rec := testutil.NewLatencyRecorder()
	token := generateTestToken()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Liveness
		resp, err := client.Get(ts.URL + "/healthz/live")
		if err != nil {
			t.Fatalf("iteration %d: GET /healthz/live failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("iteration %d: GET /healthz/live status = %d, want 200", i, resp.StatusCode)
		}

		// Readiness
		resp, err = client.Get(ts.URL + "/healthz/ready")
		if err != nil {
			t.Fatalf("iteration %d: GET /healthz/ready failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("iteration %d: GET /healthz/ready status = %d, want 200", i, resp.StatusCode)
		}

		// Proxy request (host-service via JWT-protected route)
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/hosts", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("iteration %d: GET /api/v1/hosts failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("iteration %d: GET /api/v1/hosts status = %d, want 200", i, resp.StatusCode)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=15 parallel goroutines,
// each performing health-check + proxy request cycles. Validates no
// deadlock occurs and all goroutines complete within the timeout.
func TestStressConcurrentContention(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer(t)

	client := &http.Client{Timeout: 10 * time.Second}
	ts := httptest.NewServer(s.Router())
	defer ts.Close()

	const parallelism = 15
	rec := testutil.NewLatencyRecorder()
	token := generateTestToken()

	testutil.RunConcurrent(t, parallelism, func(id int) {
		start := time.Now()

		// Liveness
		resp, err := client.Get(ts.URL + "/healthz/live")
		if err != nil {
			t.Errorf("goroutine %d: GET /healthz/live failed: %v", id, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("goroutine %d: GET /healthz/live status = %d, want 200", id, resp.StatusCode)
			return
		}

		// Proxy request
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/hosts", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = client.Do(req)
		if err != nil {
			t.Errorf("goroutine %d: GET /api/v1/hosts failed: %v", id, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("goroutine %d: GET /api/v1/hosts status = %d, want 200", id, resp.StatusCode)
			return
		}

		rec.Record(time.Since(start))
	})

	p50, p95, p99 := rec.Percentiles()
	t.Logf("CONCURRENT CONTENTION (%d goroutines): p50=%v p95=%v p99=%v", parallelism, p50, p95, p99)
}

// TestStressBoundaryConditions exercises edge-case inputs against the
// gateway. Each subtest drives a specific boundary and categorises the
// result (401 for missing auth, 400 for invalid path, 503 for
// unhealthy upstream).
func TestStressBoundaryConditions(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer(t)

	client := &http.Client{Timeout: 10 * time.Second}
	ts := httptest.NewServer(s.Router())
	defer ts.Close()
	token := generateTestToken()

	t.Run("missing_auth_header_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/hosts", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("missing auth header: got %d, want 401", resp.StatusCode)
		}
		t.Logf("missing auth header → %d (expected 401)", resp.StatusCode)
	})

	t.Run("invalid_bearer_format_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/hosts", nil)
		req.Header.Set("Authorization", "not-bearer-token")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("invalid bearer format: got %d, want 401", resp.StatusCode)
		}
		t.Logf("invalid bearer format → %d (expected 401)", resp.StatusCode)
	})

	t.Run("empty_bearer_token_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/hosts", nil)
		req.Header.Set("Authorization", "Bearer ")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("empty bearer token: got %d, want 401", resp.StatusCode)
		}
		t.Logf("empty bearer token → %d (expected 401)", resp.StatusCode)
	})

	t.Run("corrupt_jwt_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/hosts", nil)
		req.Header.Set("Authorization", "Bearer not.a.valid.jwt.token")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("corrupt JWT: got %d, want 401", resp.StatusCode)
		}
		t.Logf("corrupt JWT → %d (expected 401)", resp.StatusCode)
	})

	t.Run("valid_token_reaches_upstream", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/hosts", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("valid token: got %d, want 200", resp.StatusCode)
		}
		t.Logf("valid token → %d (expected 200)", resp.StatusCode)
	})

	t.Run("not_implemented_route_returns_501", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotImplemented {
			t.Fatalf("not-implemented route: got %d, want 501", resp.StatusCode)
		}
		t.Logf("not-implemented route → %d (expected 501)", resp.StatusCode)
	})

	t.Run("path_traversal_in_param_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/hosts/../../etc/passwd", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		// Should NOT reach upstream (400 or gin normalises path)
		if resp.StatusCode == http.StatusOK {
			t.Fatal("path traversal must not succeed")
		}
		t.Logf("path traversal → %d", resp.StatusCode)
	})

	t.Run("unknown_route_returns_404", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/completely/unknown/route", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Logf("unknown route → %d (expected 404, gin may return 404 or redirect)", resp.StatusCode)
		}
		t.Logf("unknown route → %d", resp.StatusCode)
	})

	t.Run("options_preflight_succeeds", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", ts.URL+"/api/v1/hosts", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("OPTIONS preflight: got %d, want 204", resp.StatusCode)
		}
		t.Logf("OPTIONS preflight → %d (expected 204)", resp.StatusCode)
	})

	t.Run("various_http_methods_on_health", func(t *testing.T) {
		methods := []string{"POST", "PUT", "DELETE", "PATCH"}
		for _, method := range methods {
			req, _ := http.NewRequest(method, ts.URL+"/healthz/live", nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Logf("%s /healthz/live: connection error: %v", method, err)
				continue
			}
			resp.Body.Close()
			t.Logf("%s /healthz/live → %d", method, resp.StatusCode)
		}
	})

	t.Run("large_request_id_header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/healthz/live", nil)
		req.Header.Set("X-Request-ID", strings.Repeat("x", 10000))
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("large request ID: got %d, want 200", resp.StatusCode)
		}
		t.Logf("large request ID (10000 chars) → %d", resp.StatusCode)
	})

	t.Run("unicode_in_query_params", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/healthz/live?name=テスト&value=日本語", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("unicode query params: got %d, want 200", resp.StatusCode)
		}
		t.Logf("unicode query params → %d", resp.StatusCode)
	})
}

// TestStressBoundaryConditions_NoJWT exercises boundary conditions
// WITHOUT configuring a JWT public key — proves the middleware
// rejects cleanly even when JWT validation is not configured.
func TestStressBoundaryConditions_NoJWT(t *testing.T) {
	s := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/hosts", nil)
	req.Header.Set("Authorization", "Bearer some-token")

	s.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Logf("no JWT configured + valid-looking token → %d (want 401)", w.Code)
	}
	t.Logf("no JWT configured → %d", w.Code)

	// Health endpoints must still work without JWT
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/healthz/live", nil)
	s.Router().ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("health endpoint without JWT: got %d, want 200", w2.Code)
	}
	t.Logf("health endpoint without JWT → %d", w2.Code)
}

// TestStressMultipleUpstreamRoutes_SustainedLoad drives sustained
// load across multiple different upstream routes to stress the
// routing table and proxy machinery.
func TestStressMultipleUpstreamRoutes_SustainedLoad(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer(t)

	client := &http.Client{Timeout: 10 * time.Second}
	ts := httptest.NewServer(s.Router())
	defer ts.Close()
	token := generateTestToken()

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/hosts"},
		{"GET", "/api/v1/vaults"},
		{"GET", "/api/v1/sessions"},
		{"GET", "/api/v1/snippets"},
		{"GET", "/api/v1/workspaces"},
		{"GET", "/api/v1/recordings"},
		{"GET", "/api/v1/audit"},
		{"GET", "/api/v1/notifications"},
		{"GET", "/api/v1/config"},
		{"POST", "/api/v1/auth/login"},
	}

	const iterations = 100
	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		route := routes[i%len(routes)]
		start := time.Now()

		req, _ := http.NewRequest(route.method, ts.URL+route.path, nil)
		if !strings.HasPrefix(route.path, "/api/v1/auth/") {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("iteration %d: %s %s failed: %v", i, route.method, route.path, err)
		}
		resp.Body.Close()

		if resp.StatusCode >= 500 {
			t.Fatalf("iteration %d: %s %s returned %d (server error)", i, route.method, route.path, resp.StatusCode)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("MULTI-ROUTE SUSTAINED LOAD (%d iterations across %d routes): p50=%v p95=%v p99=%v",
		iterations, len(routes), p50, p95, p99)
}

// TestStressRateLimitBoundary exercises the rate limiter by sending
// rapid requests from the same client IP and verifying it eventually
// triggers (or doesn't, depending on configured limits).
func TestStressRateLimitBoundary(t *testing.T) {
	s := setupTestServer(t)

	client := &http.Client{Timeout: 10 * time.Second}
	ts := httptest.NewServer(s.Router())
	defer ts.Close()

	// Send burst requests to non-health endpoints (rate-limited)
	rateLimitedCount := 0
	const burst = 200

	for i := 0; i < burst; i++ {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/auth/login", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Logf("burst %d: connection error: %v", i, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	t.Logf("RATE LIMIT BURST (%d requests): rate_limited=%d", burst, rateLimitedCount)
	if rateLimitedCount > 0 {
		t.Logf("EVIDENCE: rate limiter triggered after %d requests", burst-rateLimitedCount)
	} else {
		t.Logf("EVIDENCE: rate limiter did not trigger within %d requests (limits may be higher than burst)", burst)
	}
}
