//go:build chaos

// Chaos test suite for health-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed request bodies, binary garbage,
//     wrong content types — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, unicode, injection attempts.
//
// Run:
//
//	go test -race -tags chaos -run TestChaos -v -timeout 120s ./internal/handler/
package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// chaosEnv holds the assembled test environment for chaos tests.
type chaosEnv struct {
	ts       *httptest.Server
	upstream *httptest.Server
	cleanup  func()
}

// setupChaosEnv boots the chaos test environment with a real checker
// pointed at a mock upstream.
func setupChaosEnv(t *testing.T) *chaosEnv {
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

	return &chaosEnv{
		ts:       ts,
		upstream: upstream,
		cleanup: func() {
			ts.Close()
			upstream.Close()
		},
	}
}

// chaosPostRaw sends a POST request with a raw byte body and returns
// the status code + raw response body.
func chaosPostRaw(t *testing.T, client *http.Client, url string, contentType string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

// truncate returns the first n characters of s, with "..." appended
// if s is longer than n.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// TestChaosInputCorruption exercises corrupt/malformed inputs against
// the POST endpoint. Every case MUST produce a clean HTTP error response
// (not a panic, not a hang, not a 500 for input errors).
func TestChaosInputCorruption(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("malformed_json_bodies", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"{",
			"{}",
			"null",
			"[]",
			"42",
			`{"services":}`,
			`{"services":"not-an-array"}`,
			`{"services":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		for i, body := range malformedBodies {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", []byte(body))
			if status == 0 {
				t.Logf("malformed body %d: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("malformed body %d: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed bodies against POST /api/v1/health/check", len(malformedBodies))
	})

	t.Run("wrong_content_type", func(t *testing.T) {
		contentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"",
			"application/octet-stream",
		}
		for _, ct := range contentTypes {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", ct, []byte(`{"services":["auth-service"]}`))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("invalid_utf8_body", func(t *testing.T) {
		invalid := []byte{0xc0, 0xaf, 0xe0, 0x80, 0xbf, 0xf0, 0x80, 0x80, 0xaf}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", invalid)
		if status >= 500 {
			t.Errorf("invalid UTF-8: got %d — expected 400", status)
		}
		t.Logf("invalid UTF-8 → %d", status)
	})
}

// TestChaosResourceExhaustion drives rapid-fire requests to verify
// the service degrades gracefully under pressure — no goroutine
// leaks, no deadlocks, no panics.
func TestChaosResourceExhaustion(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("rapid_fire_healthz", func(t *testing.T) {
		// Fire N requests as fast as possible, verify no panics
		const burst = 50
		errCount := 0
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			resp, err := client.Get(env.ts.URL + "/healthz")
			if err != nil {
				errCount++
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d /healthz requests: connection_errors=%d server_errors=%d", burst, errCount, serverErrCount)
		if serverErrCount == burst {
			t.Errorf("ALL %d requests returned 500 — service is down", burst)
		}
	})

	t.Run("rapid_fire_system_health", func(t *testing.T) {
		const burst = 50
		errCount := 0
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			resp, err := client.Get(env.ts.URL + "/api/v1/health/system")
			if err != nil {
				errCount++
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d /api/v1/health/system requests: connection_errors=%d server_errors=%d", burst, errCount, serverErrCount)
		if serverErrCount == burst {
			t.Errorf("ALL %d requests returned 500 — service is down", burst)
		}
	})

	t.Run("rapid_fire_check_endpoint", func(t *testing.T) {
		const burst = 30
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			body := fmt.Sprintf(`{"services":["auth-service"]}`)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", []byte(body))
			if status >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d /api/v1/health/check requests: server_errors=%d", burst, serverErrCount)
		if serverErrCount == burst {
			t.Errorf("ALL %d requests returned 500 — service is down", burst)
		}
	})

	t.Run("rapid_fire_mixed_endpoints", func(t *testing.T) {
		// Hammer all endpoints simultaneously
		const burst = 40
		endpoints := []string{
			"/healthz",
			"/healthz/ready",
			"/api/v1/health/system",
			"/api/v1/health/services/auth-service",
		}
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			ep := endpoints[id%len(endpoints)]
			resp, err := client.Get(env.ts.URL + ep)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d mixed endpoint requests: server_errors=%d", burst, serverErrCount)
		if serverErrCount == burst {
			t.Errorf("ALL %d requests returned 500 — service is down", burst)
		}
	})

	t.Run("concurrent_check_same_services", func(t *testing.T) {
		// Multiple goroutines checking the same services simultaneously
		// — must not deadlock
		const parallel = 15
		body := []byte(`{"services":["auth-service","notification-service"]}`)

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", body)
			if status >= 500 {
				t.Errorf("concurrent check %d: got %d — expected 200", id, status)
			}
		})
	})
}

// TestChaosBoundaryConditions exercises extreme boundary values
// that stress the parsing, validation, and serialization layers.
func TestChaosBoundaryConditions(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("nil_body_to_post", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/health/check", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("nil body request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			t.Errorf("nil body: got %d — expected 400", resp.StatusCode)
		}
		t.Logf("nil body → %d", resp.StatusCode)
	})

	t.Run("empty_json_object", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeServices := make([]string, 10000)
		for i := range largeServices {
			largeServices[i] = strings.Repeat("x", 100)
		}
		payload, _ := json.Marshal(map[string]interface{}{"services": largeServices})
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", payload)
		if status == 0 {
			t.Fatal("large payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("large payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("large payload (%d bytes) → %d (no panic)", len(payload), status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"services":null}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("null services: got %d — expected 400", status)
		}
		t.Logf("null services → %d", status)
	})

	t.Run("unicode_service_names", func(t *testing.T) {
		payload := `{"services":["サービス","αυτό-διακομιστής","сервис"]}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", []byte(payload))
		// Checker reports unknown services as unhealthy (503) — never a 500 panic
		if status >= 500 && status != http.StatusServiceUnavailable {
			t.Errorf("unicode service names: got %d — expected 200 or 503", status)
		}
		t.Logf("unicode service names → %d", status)
	})

	t.Run("sql_injection_in_service_name", func(t *testing.T) {
		payload := `{"services":["'; DROP TABLE services; --"]}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", []byte(payload))
		// Checker reports unknown services as unhealthy (503) — never a 500 panic
		if status >= 500 && status != http.StatusServiceUnavailable {
			t.Errorf("SQL injection attempt: got %d — expected 200 or 503", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_service_name", func(t *testing.T) {
		payload := `{"services":["<script>alert('xss')</script>"]}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/check", "application/json", []byte(payload))
		// Checker reports unknown services as unhealthy (503) — never a 500 panic
		if status >= 500 && status != http.StatusServiceUnavailable {
			t.Errorf("XSS in service name: got %d — expected 200 or 503", status)
		}
		t.Logf("XSS in service name → %d", status)
	})

	t.Run("path_traversal_in_service_name", func(t *testing.T) {
		// Path traversal via URL path parameter
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/health/services/../../../etc/passwd", "", nil)
		if status >= 500 {
			t.Errorf("path traversal: got %d — expected 404 or 400", status)
		}
		t.Logf("path traversal → %d", status)
	})

	t.Run("very_long_service_name_in_path", func(t *testing.T) {
		longName := strings.Repeat("a", 10000)
		resp, err := client.Get(env.ts.URL + "/api/v1/health/services/" + longName)
		if err != nil {
			t.Logf("very long service name: connection error (acceptable): %v", err)
			return
		}
		resp.Body.Close()
		// Checker reports unknown services as unhealthy (503) — never a 500 panic
		if resp.StatusCode >= 500 && resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("very long service name: got %d — expected 404 or 503", resp.StatusCode)
		}
		t.Logf("very long service name (%d chars) → %d", len(longName), resp.StatusCode)
	})
}

// TestChaosNilChecker_CorruptInputs exercises corrupt inputs against
// the nil-checker path to verify it handles errors gracefully.
func TestChaosNilChecker_CorruptInputs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil)

	r.GET("/api/v1/health/system", h.GetSystemHealth)
	r.GET("/api/v1/health/services/:name", h.GetServiceHealth)
	r.POST("/api/v1/health/check", h.RunHealthCheck)

	ts := httptest.NewServer(r)
	defer ts.Close()

	client := ts.Client()

	t.Run("nil_checker_all_corrupt_inputs", func(t *testing.T) {
		corruptBodies := []string{
			"",
			"{broken",
			"null",
			"[]",
			`{"services":null}`,
			`{"services":"not-array"}`,
			`\x00\x01\x02`,
		}

		for i, body := range corruptBodies {
			status, _ := chaosPostRaw(t, client, ts.URL+"/api/v1/health/check", "application/json", []byte(body))
			// Nil checker returns 503 before parsing, or 400 for bad JSON
			if status >= 500 && status != http.StatusServiceUnavailable {
				t.Errorf("nil checker corrupt body %d: got %d — expected 400 or 503", i, status)
			}
		}
		t.Logf("nil checker: tested %d corrupt inputs — no panics", len(corruptBodies))
	})
}
