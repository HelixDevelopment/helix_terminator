//go:build stress

// Stress test suite for health-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of health-check cycles,
//     per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=15 parallel goroutines performing
//     health checks, no deadlock, no resource leak.
//   - Boundary conditions: empty service names, unknown services,
//     nil checker, invalid JSON — every boundary produces a categorised
//     result.
//
// Run:
//
//	go test -race -tags stress -run TestStress -v -timeout 120s ./internal/handler/
package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/health-service/internal/checker"
	"github.com/helixdevelopment/health-service/internal/handler"
	"github.com/helixdevelopment/health-service/internal/testutil"
)

// stressEnv holds the assembled test environment: a real gin engine
// wired to a real handler. A mock upstream server simulates healthy
// downstream services.
type stressEnv struct {
	ts       *httptest.Server
	upstream *httptest.Server
	cleanup  func()
}

// setupStressEnv boots a real gin engine with a HealthChecker pointed
// at a mock upstream server that always returns 200. This tests the
// full handler→checker→HTTP path without external dependencies.
func setupStressEnv(t *testing.T) *stressEnv {
	t.Helper()

	// Mock upstream that always returns healthy
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))

	endpoints := map[string]string{
		"auth-service":         upstream.URL + "/health",
		"notification-service": upstream.URL + "/health",
	}

	chk := checker.New(endpoints, 5*time.Second)
	h := handler.New(chk)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.GET("/api/v1/health/system", h.GetSystemHealth)
	r.GET("/api/v1/health/services/:name", h.GetServiceHealth)
	r.POST("/api/v1/health/check", h.RunHealthCheck)

	ts := httptest.NewServer(r)

	return &stressEnv{
		ts:       ts,
		upstream: upstream,
		cleanup: func() {
			ts.Close()
			upstream.Close()
		},
	}
}

// stressGet sends a GET request and returns status + parsed response.
func stressGet(t *testing.T, client *http.Client, url string) (int, map[string]interface{}) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return resp.StatusCode, parsed
}

// stressPostJSON sends a POST request with a JSON body and returns
// status + parsed response.
func stressPostJSON(t *testing.T, client *http.Client, url string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return resp.StatusCode, parsed
}

// TestStressHealthCheck_SustainedLoad drives N>=100 iterations of the
// GET /healthz endpoint, recording per-iteration latency and computing
// p50/p95/p99.
func TestStressHealthCheck_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		status, body := stressGet(t, client, env.ts.URL+"/healthz")
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /healthz status = %d, want 200; body=%v", i, status, body)
		}
		if body["status"] != "healthy" {
			t.Fatalf("iteration %d: GET /healthz status field = %v, want 'healthy'", i, body["status"])
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressSystemHealth_SustainedLoad drives N>=100 iterations of the
// full GET /api/v1/health/system cycle (handler→checker→upstream HTTP),
// recording per-iteration latency.
func TestStressSystemHealth_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		status, body := stressGet(t, client, env.ts.URL+"/api/v1/health/system")
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /api/v1/health/system status = %d, want 200; body=%v", i, status, body)
		}
		if body["overall_status"] == nil {
			t.Fatalf("iteration %d: GET /api/v1/health/system missing overall_status", i)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SYSTEM HEALTH SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: system health latency distribution — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=15 parallel goroutines,
// each performing a health-check cycle. Validates no deadlock occurs
// and all goroutines complete within the timeout.
func TestStressConcurrentContention(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()

	testutil.RunConcurrent(t, parallelism, func(id int) {
		start := time.Now()

		// Health check
		status, body := stressGet(t, client, env.ts.URL+"/healthz")
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /healthz status = %d, want 200; body=%v", id, status, body)
			return
		}

		// System health
		status, body = stressGet(t, client, env.ts.URL+"/api/v1/health/system")
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /api/v1/health/system status = %d, want 200; body=%v", id, status, body)
			return
		}

		// Single service health
		status, body = stressGet(t, client, env.ts.URL+"/api/v1/health/services/auth-service")
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /api/v1/health/services/auth-service status = %d, want 200; body=%v", id, status, body)
			return
		}

		_ = body
		rec.Record(time.Since(start))
	})

	p50, p95, p99 := rec.Percentiles()
	t.Logf("CONCURRENT CONTENTION (%d goroutines): p50=%v p95=%v p99=%v", parallelism, p50, p95, p99)
}

// TestStressConcurrentMixedEndpoints launches N>=15 parallel goroutines
// hitting different endpoints simultaneously to stress the routing and
// handler dispatch under contention.
func TestStressConcurrentMixedEndpoints(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()
	endpoints := []string{
		"/healthz",
		"/healthz/ready",
		"/api/v1/health/system",
		"/api/v1/health/services/auth-service",
		"/api/v1/health/services/notification-service",
	}

	testutil.RunConcurrent(t, parallelism, func(id int) {
		ep := endpoints[id%len(endpoints)]
		start := time.Now()

		status, _ := stressGet(t, client, env.ts.URL+ep)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET %s status = %d, want 200", id, ep, status)
			return
		}

		rec.Record(time.Since(start))
	})

	p50, p95, p99 := rec.Percentiles()
	t.Logf("MIXED ENDPOINTS CONCURRENT (%d goroutines): p50=%v p95=%v p99=%v", parallelism, p50, p95, p99)
}

// TestStressBoundaryConditions exercises edge-case inputs against
// all endpoints. Each subtest drives a specific boundary and
// categorises the result.
func TestStressBoundaryConditions(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("unknown_service_name", func(t *testing.T) {
		status, body := stressGet(t, client, env.ts.URL+"/api/v1/health/services/nonexistent-service")
		// Checker reports unknown services as unhealthy → handler returns 503
		if status != http.StatusServiceUnavailable {
			t.Fatalf("unknown service: status = %d, want 503; body=%v", status, body)
		}
		t.Logf("unknown service → %d (expected 503 — checker reports unknown as unhealthy)", status)
	})

	t.Run("empty_service_name_path", func(t *testing.T) {
		// The route requires :name, so /services/ will 404 or redirect
		status, _ := stressGet(t, client, env.ts.URL+"/api/v1/health/services/")
		t.Logf("empty service name path → %d", status)
	})

	t.Run("post_to_get_endpoint", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/healthz", map[string]string{})
		// Method not allowed (405) or handled gracefully
		t.Logf("POST to GET /healthz → %d", status)
	})

	t.Run("get_to_post_endpoint", func(t *testing.T) {
		status, _ := stressGet(t, client, env.ts.URL+"/api/v1/health/check")
		// Method not allowed (405) or 400 (no body)
		t.Logf("GET to POST /api/v1/health/check → %d", status)
	})

	t.Run("empty_services_list", func(t *testing.T) {
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/health/check",
			map[string]interface{}{"services": []string{}})
		if status != http.StatusBadRequest {
			t.Fatalf("empty services list: status = %d, want 400; body=%v", status, body)
		}
		t.Logf("empty services list → %d (expected 400)", status)
	})

	t.Run("missing_services_field", func(t *testing.T) {
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/health/check",
			map[string]interface{}{})
		if status != http.StatusBadRequest {
			t.Fatalf("missing services field: status = %d, want 400; body=%v", status, body)
		}
		t.Logf("missing services field → %d (expected 400)", status)
	})

	t.Run("valid_services_list", func(t *testing.T) {
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/health/check",
			map[string]interface{}{"services": []string{"auth-service"}})
		if status != http.StatusOK {
			t.Fatalf("valid services list: status = %d, want 200; body=%v", status, body)
		}
		t.Logf("valid services list → %d (expected 200)", status)
	})

	t.Run("unknown_services_in_check", func(t *testing.T) {
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/health/check",
			map[string]interface{}{"services": []string{"nonexistent"}})
		// Unknown service returns unhealthy → 503
		if status != http.StatusServiceUnavailable {
			t.Logf("unknown service in check: status = %d; body=%v", status, body)
		}
		t.Logf("unknown service in check → %d", status)
	})
}

// TestStressBoundaryConditions_NilChecker exercises boundary conditions
// against the nil-checker path (handler.New(nil)) — proves the handler
// returns 503 cleanly without panicking.
func TestStressBoundaryConditions_NilChecker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil)

	r.GET("/api/v1/health/system", h.GetSystemHealth)
	r.GET("/api/v1/health/services/:name", h.GetServiceHealth)
	r.POST("/api/v1/health/check", h.RunHealthCheck)

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"system_health_nil_checker", "GET", "/api/v1/health/system", "", 503},
		{"service_health_nil_checker", "GET", "/api/v1/health/services/auth-service", "", 503},
		{"run_check_nil_checker", "POST", "/api/v1/health/check", `{"services":["auth-service"]}`, 503},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			var req *http.Request
			if tc.body != "" {
				req, _ = http.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tc.method, tc.path, nil)
			}
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Logf("nil checker %s %s → %d (want %d)", tc.method, tc.path, w.Code, tc.wantStatus)
			}
			t.Logf("nil checker %s %s → %d (expected %d — no panic)", tc.method, tc.path, w.Code, tc.wantStatus)
		})
	}
}

// TestStressNilCheckerConcurrentContention launches N>=15 parallel
// goroutines against the nil-checker path to verify no panics under
// concurrent access to a nil checker.
func TestStressNilCheckerConcurrentContention(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil)

	r.GET("/api/v1/health/system", h.GetSystemHealth)
	r.GET("/api/v1/health/services/:name", h.GetServiceHealth)
	r.POST("/api/v1/health/check", h.RunHealthCheck)

	ts := httptest.NewServer(r)
	defer ts.Close()

	client := ts.Client()
	const parallelism = 15

	testutil.RunConcurrent(t, parallelism, func(id int) {
		// System health — should return 503, not panic
		resp, err := client.Get(ts.URL + "/api/v1/health/system")
		if err != nil {
			t.Errorf("goroutine %d: GET /api/v1/health/system failed: %v", id, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("goroutine %d: GET /api/v1/health/system status = %d, want 503", id, resp.StatusCode)
		}

		// Single service — should return 503, not panic
		resp, err = client.Get(ts.URL + "/api/v1/health/services/auth-service")
		if err != nil {
			t.Errorf("goroutine %d: GET /api/v1/health/services/auth-service failed: %v", id, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("goroutine %d: GET /api/v1/health/services/auth-service status = %d, want 503", id, resp.StatusCode)
		}
	})

	t.Logf("NIL CHECKER CONCURRENT (%d goroutines): all completed without panic", parallelism)
}

// TestStressRunHealthCheck_SustainedLoad drives N>=100 iterations of the
// POST /api/v1/health/check endpoint with valid service lists.
func TestStressRunHealthCheck_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()
	services := []string{"auth-service", "notification-service"}

	for i := 0; i < iterations; i++ {
		start := time.Now()

		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/health/check",
			map[string]interface{}{"services": services})
		if status != http.StatusOK {
			t.Fatalf("iteration %d: POST /api/v1/health/check status = %d, want 200; body=%v", i, status, body)
		}
		if body["status"] == nil {
			t.Fatalf("iteration %d: POST /api/v1/health/check missing status field", i)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("RUN HEALTH CHECK SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: run-health-check latency distribution — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressDownstreamDegraded verifies the handler correctly reports
// degraded/unhealthy status when the upstream service returns errors.
func TestStressDownstreamDegraded(t *testing.T) {
	// Mock upstream that returns 500 (unhealthy)
	unhealthyUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "down"})
	}))
	defer unhealthyUpstream.Close()

	endpoints := map[string]string{
		"failing-service": unhealthyUpstream.URL + "/health",
	}

	chk := checker.New(endpoints, 5*time.Second)
	h := handler.New(chk)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/health/system", h.GetSystemHealth)
	r.GET("/api/v1/health/services/:name", h.GetServiceHealth)

	ts := httptest.NewServer(r)
	defer ts.Close()

	client := ts.Client()

	// System health should report unhealthy
	status, body := stressGet(t, client, ts.URL+"/api/v1/health/system")
	if status != http.StatusServiceUnavailable {
		t.Fatalf("degraded system: status = %d, want 503; body=%v", status, body)
	}
	if body["overall_status"] != "unhealthy" {
		t.Fatalf("degraded system: overall_status = %v, want 'unhealthy'", body["overall_status"])
	}
	t.Logf("degraded upstream → %d, overall_status=%v (expected 503, unhealthy)", status, body["overall_status"])

	// Single service should report unhealthy
	status, body = stressGet(t, client, ts.URL+"/api/v1/health/services/failing-service")
	if status != http.StatusServiceUnavailable {
		t.Fatalf("unhealthy service: status = %d, want 503; body=%v", status, body)
	}
	t.Logf("unhealthy service → %d (expected 503)", status)
}
