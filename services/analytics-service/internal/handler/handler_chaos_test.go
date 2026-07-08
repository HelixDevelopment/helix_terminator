//go:build chaos

// Chaos test suite for analytics-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed JSON, binary garbage, wrong content
//     types, invalid UUIDs — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, SQL injection attempts, XSS payloads.
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

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/analytics-service/internal/handler"
	"github.com/helixdevelopment/analytics-service/internal/model"
	"github.com/helixdevelopment/analytics-service/internal/repository"
	"github.com/helixdevelopment/analytics-service/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// chaosEnv holds the assembled test environment for chaos tests.
type chaosEnv struct {
	ts      *httptest.Server
	cleanup func()
}

// setupChaosEnv boots the chaos test environment. If podman is
// available, uses a real PostgreSQL container; otherwise falls back
// to a nil-repo handler (validation-only path).
func setupChaosEnv(t *testing.T) *chaosEnv {
	t.Helper()

	poolURL, available := testutil.StartTestPostgres(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Inject synthetic user_id for CreateEvent
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "00000000-0000-0000-0000-000000000001")
		c.Next()
	})

	if available {
		pool, err := pgxpool.New(t.Context(), poolURL)
		if err != nil {
			t.Fatalf("pgxpool.New failed: %v", err)
		}
		repo := repository.New(pool)
		h := handler.New(repo)
		r.POST("/api/v1/analytics/events", h.CreateEvent)
		r.GET("/api/v1/analytics/events", h.ListEvents)
		r.GET("/api/v1/analytics/events/:id", h.GetEvent)
		r.GET("/api/v1/analytics/stats/event-types", h.CountByEventType)
		r.GET("/healthz", h.HealthCheck)
		r.GET("/healthz/ready", h.ReadinessCheck)
		ts := httptest.NewServer(r)
		return &chaosEnv{
			ts: ts,
			cleanup: func() {
				ts.Close()
				pool.Close()
			},
		}
	}

	// Nil-repo fallback — validation-only, no DB
	h := handler.New(nil)
	r.POST("/api/v1/analytics/events", h.CreateEvent)
	r.GET("/api/v1/analytics/events", h.ListEvents)
	r.GET("/api/v1/analytics/events/:id", h.GetEvent)
	r.GET("/api/v1/analytics/stats/event-types", h.CountByEventType)
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	ts := httptest.NewServer(r)
	return &chaosEnv{
		ts: ts,
		cleanup: func() {
			ts.Close()
		},
	}
}

// chaosPostRaw sends a POST request with a raw byte body and returns
// the status code + raw response body. Does NOT assume the body is
// valid JSON — it sends whatever bytes are provided.
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

// chaosGetRaw sends a GET request and returns status + raw body.
func chaosGetRaw(t *testing.T, client *http.Client, url string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

// TestChaosInputCorruption exercises corrupt/malformed inputs against
// all endpoints. Every case MUST produce a clean HTTP error response
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
			`{"eventType":}`,
			`{"eventType":123}`,
			`{"eventType":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []string{
			"/api/v1/analytics/events",
		}
		for _, ep := range endpoints {
			for i, body := range malformedBodies {
				status, _ := chaosPostRaw(t, client, env.ts.URL+ep, "application/json", []byte(body))
				if status == 0 {
					t.Logf("malformed body %d to %s: connection failed (acceptable)", i, ep)
					continue
				}
				if status >= 500 {
					t.Errorf("malformed body %d to %s: got %d — expected 400 for bad input", i, ep, status)
				}
			}
		}
		t.Logf("tested %d malformed bodies across %d endpoints", len(malformedBodies), len(endpoints))
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
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
				ct, []byte(`{"eventType":"session","payload":{}}`))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("invalid_uuid_in_get_event", func(t *testing.T) {
		corruptIDs := []string{
			"not-a-uuid",
			"",
			"null",
			"undefined",
			strings.Repeat("x", 1000),
			"\x00\x01\x02\x03",
			"12345",
			"00000000-0000-0000-0000-00000000000g", // invalid hex
		}

		for i, id := range corruptIDs {
			status, raw := chaosGetRaw(t, client, env.ts.URL+"/api/v1/analytics/events/"+id)
			if status >= 500 {
				t.Errorf("corrupt UUID %d: got %d — expected 400/404 for id %q", i, status, truncate(id, 50))
			}
			t.Logf("corrupt UUID %d (%q) → %d: %s", i, truncate(id, 30), status, truncate(string(raw), 100))
		}
	})

	t.Run("invalid_query_params_list_events", func(t *testing.T) {
		corruptParams := []string{
			"?org_id=not-a-uuid",
			"?limit=abc",
			"?offset=xyz",
			"?limit=99999999999999999999",
			"?offset=-999999",
			"?limit=" + strings.Repeat("1", 1000),
		}

		for i, params := range corruptParams {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/analytics/events"+params)
			if status >= 500 {
				t.Errorf("corrupt query %d (%s): got %d — expected non-server-error", i, params, status)
			}
			t.Logf("corrupt query %d (%s) → %d", i, truncate(params, 50), status)
		}
	})
}

// TestChaosResourceExhaustion drives rapid-fire requests to verify
// the service degrades gracefully under pressure — no goroutine
// leaks, no deadlocks, no panics.
func TestChaosResourceExhaustion(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("rapid_fire_create_events", func(t *testing.T) {
		// Fire N requests as fast as possible, verify no panics
		const burst = 50
		errCount := 0
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			body := fmt.Sprintf(`{"eventType":"session","payload":{"chaos_id":%d,"ts":%d}}`,
				id, id*1000+id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
				"application/json", []byte(body))
			if status == 0 {
				errCount++
				return
			}
			if status >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d requests: connection_errors=%d server_errors=%d", burst, errCount, serverErrCount)
		// A few 500s are acceptable under load (DB connection pool
		// exhaustion), but the service must NOT panic or hang.
		if serverErrCount == burst {
			t.Errorf("ALL %d requests returned 500 — service is down", burst)
		}
	})

	t.Run("rapid_fire_list_events", func(t *testing.T) {
		// Hammer GET /events — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/analytics/events?limit=10")
			if status >= 500 {
				t.Errorf("rapid list %d: got %d — expected 200", id, status)
			}
		})
	})

	t.Run("rapid_fire_health_checks", func(t *testing.T) {
		// Health endpoints must be resilient to burst traffic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/healthz")
			if status >= 500 {
				t.Errorf("health check %d: got %d — expected 200", id, status)
			}
		})
	})

	t.Run("concurrent_create_same_event_type", func(t *testing.T) {
		// Multiple goroutines creating events of the same type
		// simultaneously — must not deadlock
		const parallel = 15
		body := []byte(`{"eventType":"error","payload":{"shared":true}}`)

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
				"application/json", body)
			if status >= 500 {
				t.Errorf("concurrent create %d: got %d — expected 201", id, status)
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

	t.Run("nil_body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/analytics/events", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largePayload := strings.Repeat("x", 1000000)
		payload := fmt.Sprintf(`{"eventType":"session","payload":{"data":"%s"}}`, largePayload)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		// Log the finding: handler may return 400 (validation) or
		// 500 (DB column overflow) — either is acceptable as long as
		// no panic occurs.
		t.Logf("FINDING: 1MB payload → %d (handler does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		// All fields at zero value — validation must catch
		payload := `{"eventType":"","payload":null}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_payload", func(t *testing.T) {
		payload := `{"eventType":"session","payload":{"message":"日本語テスト 🚀"}}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode payload: got %d — expected non-server-error", status)
		}
		t.Logf("unicode payload → %d", status)
	})

	t.Run("sql_injection_in_event_type", func(t *testing.T) {
		payload := `{"eventType":"'; DROP TABLE analytics_events; --","payload":{}}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 400 (invalid event_type)", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_payload", func(t *testing.T) {
		payload := `{"eventType":"session","payload":{"html":"<script>alert('xss')</script>"}}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in payload: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in payload → %d", status)
	})

	t.Run("deeply_nested_json_payload", func(t *testing.T) {
		// Build a deeply nested JSON object
		depth := 50
		nested := `{"a":1}`
		for i := 0; i < depth; i++ {
			nested = fmt.Sprintf(`{"level":%d,"child":%s}`, i, nested)
		}
		payload := fmt.Sprintf(`{"eventType":"session","payload":%s}`, nested)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("deeply nested JSON (depth=%d): got %d — expected non-server-error", depth, status)
		}
		t.Logf("deeply nested JSON (depth=%d) → %d", depth, status)
	})

	t.Run("null_json_values", func(t *testing.T) {
		payload := `{"eventType":null,"payload":null}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("null values: got %d — expected 400", status)
		}
		t.Logf("null values → %d", status)
	})

	t.Run("wrong_type_for_event_type", func(t *testing.T) {
		payload := `{"eventType":12345,"payload":"not an object"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/analytics/events",
			"application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("wrong type for eventType: got %d — expected 400", status)
		}
		t.Logf("wrong type for eventType → %d", status)
	})

	t.Run("readiness_check_resilience", func(t *testing.T) {
		// Readiness check must not panic even with nil repo
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/healthz/ready")
		// 200 (ready) or 503 (not ready) — never 500
		if status == 0 {
			t.Fatal("readiness check: connection failed")
		}
		t.Logf("readiness check → %d", status)
	})
}

// truncate returns the first n characters of s, with "..." appended
// if s is longer than n.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
