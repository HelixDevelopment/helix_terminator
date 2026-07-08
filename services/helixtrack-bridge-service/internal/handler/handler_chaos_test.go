//go:build chaos

// Chaos test suite for helixtrack-bridge-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed JSON, invalid UUIDs, binary garbage,
//     wrong content types — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, SQL injection, XSS payloads.
//
// Run:
//
//	go test -race -tags chaos -run TestChaos -v -timeout 120s ./internal/handler/
package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/handler"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/repository"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// chaosSpyAuth always succeeds — satisfies handler.Authenticator for
// chaos tests that need CreateBridge to proceed past the auth gate.
type chaosSpyAuth struct{}

func (s *chaosSpyAuth) EnsureAuthenticated(_ context.Context) error {
	return nil
}

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

	if available {
		pool, err := pgxpool.New(t.Context(), poolURL)
		if err != nil {
			t.Fatalf("pgxpool.New failed: %v", err)
		}
		repo := repository.New(pool)
		core := &chaosSpyAuth{}
		h := handler.New(repo, core)
		r.POST("/bridges", h.CreateBridge)
		r.GET("/bridges/:id", h.GetBridge)
		r.GET("/bridges", h.ListBridges)
		r.PUT("/bridges/:id", h.UpdateBridge)
		r.DELETE("/bridges/:id", h.DeleteBridge)
		r.GET("/health", h.HealthCheck)
		r.GET("/ready", h.ReadinessCheck)
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
	h := handler.New(nil, nil)
	r.POST("/bridges", h.CreateBridge)
	r.GET("/bridges/:id", h.GetBridge)
	r.GET("/bridges", h.ListBridges)
	r.PUT("/bridges/:id", h.UpdateBridge)
	r.DELETE("/bridges/:id", h.DeleteBridge)
	r.GET("/health", h.HealthCheck)
	r.GET("/ready", h.ReadinessCheck)
	ts := httptest.NewServer(r)
	return &chaosEnv{
		ts: ts,
		cleanup: func() {
			ts.Close()
		},
	}
}

// chaosPostRaw sends a POST request with a raw byte body and returns
// the status code + raw response body. Unlike stressJSON, this does
// NOT assume the body is valid JSON — it sends whatever bytes are
// provided.
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

// chaosPutRaw sends a PUT request with a raw byte body.
func chaosPutRaw(t *testing.T, client *http.Client, url string, contentType string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
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
			`{"integrationId":}`,
			`{"integrationId":123,"orgId":456}`, // wrong types
			`{"integrationId":null,"orgId":null,"name":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []struct {
			method string
			path   string
		}{
			{"POST", "/bridges"},
			{"PUT", "/bridges/" + uuid.New().String()},
		}
		for _, ep := range endpoints {
			for i, body := range malformedBodies {
				var status int
				if ep.method == "POST" {
					status, _ = chaosPostRaw(t, client, env.ts.URL+ep.path, "application/json", []byte(body))
				} else {
					status, _ = chaosPutRaw(t, client, env.ts.URL+ep.path, "application/json", []byte(body))
				}
				if status == 0 {
					t.Logf("malformed body %d to %s %s: connection failed (acceptable)", i, ep.method, ep.path)
					continue
				}
				if status >= 500 {
					t.Errorf("malformed body %d to %s %s: got %d — expected 400 for bad input", i, ep.method, ep.path, status)
				}
			}
		}
		t.Logf("tested %d malformed bodies across %d endpoints", len(malformedBodies), len(endpoints))
	})

	t.Run("invalid_uuid_in_path", func(t *testing.T) {
		invalidUUIDs := []string{
			"not-a-uuid",
			"",
			"12345",
			strings.Repeat("x", 1000),
			"550e8400-e29b-41d4-a716", // truncated
			"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		}

		for i, badID := range invalidUUIDs {
			req, _ := http.NewRequest("GET", env.ts.URL+"/bridges/"+badID, nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Logf("invalid uuid %d: request failed (acceptable)", i)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("invalid uuid %d (%q): got %d — expected 400", i, truncate(badID, 30), resp.StatusCode)
			}
		}
	})

	t.Run("wrong_content_type", func(t *testing.T) {
		contentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"",
			"application/octet-stream",
		}
		validBody := `{"integrationId":"test","orgId":"550e8400-e29b-41d4-a716-446655440000","name":"Test"}`
		for _, ct := range contentTypes {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", ct, []byte(validBody))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("invalid_status_values_in_update", func(t *testing.T) {
		invalidStatuses := []string{
			"INVALID",
			"active; DROP TABLE bridges; --",
			"<script>alert('xss')</script>",
			strings.Repeat("a", 10000),
			"",
			"null",
		}

		fakeID := uuid.New().String()
		for i, statusVal := range invalidStatuses {
			body := fmt.Sprintf(`{"status":"%s"}`, statusVal)
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/bridges/"+fakeID, "application/json", []byte(body))
			if status >= 500 {
				t.Errorf("invalid status %d (%q): got %d — expected 400", i, truncate(statusVal, 30), status)
			}
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

	t.Run("rapid_fire_create", func(t *testing.T) {
		// Fire N requests as fast as possible, verify no panics
		const burst = 50
		errCount := 0
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			body := fmt.Sprintf(`{"integrationId":"chaos-%d","orgId":"%s","name":"Chaos %d"}`,
				id, uuid.New().String(), id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(body))
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

	t.Run("rapid_fire_get_nonexistent", func(t *testing.T) {
		// Hammer GET with nonexistent IDs — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			fakeID := uuid.New().String()
			req, _ := http.NewRequest("GET", env.ts.URL+"/bridges/"+fakeID, nil)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("get nonexistent %d: got %d — expected 404", id, resp.StatusCode)
			}
		})
	})

	t.Run("rapid_fire_health_check", func(t *testing.T) {
		// Hammer health check — must always return 200
		const burst = 50

		testutil.RunConcurrent(t, burst, func(id int) {
			req, _ := http.NewRequest("GET", env.ts.URL+"/health", nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("health check %d: request failed: %v", id, err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("health check %d: got %d — expected 200", id, resp.StatusCode)
			}
		})
	})

	t.Run("concurrent_update_same_bridge", func(t *testing.T) {
		// Multiple goroutines updating the same bridge simultaneously
		// — must not deadlock
		const parallel = 10

		// Create a bridge first (if DB available)
		body := fmt.Sprintf(`{"integrationId":"concurrent-update","orgId":"%s","name":"Shared"}`, uuid.New().String())
		status, respBody := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(body))
		if status != http.StatusCreated {
			t.Skip("cannot create bridge for concurrent update test (no DB or auth failure)")
		}

		// Extract ID from response
		var parsed map[string]interface{}
		_ = json.Unmarshal(respBody, &parsed)
		bridgeID, _ := parsed["id"].(string)
		if bridgeID == "" {
			t.Skip("bridge created but no ID returned")
		}

		testutil.RunConcurrent(t, parallel, func(id int) {
			updateBody := fmt.Sprintf(`{"name":"Updated by %d"}`, id)
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/bridges/"+bridgeID, "application/json", []byte(updateBody))
			if status >= 500 {
				t.Errorf("concurrent update %d: got %d — expected 200 or 400", id, status)
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
		req, _ := http.NewRequest("POST", env.ts.URL+"/bridges", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeName := strings.Repeat("a", 500000)
		largeConfig := strings.Repeat("b", 500000)
		payload := fmt.Sprintf(`{"integrationId":"large-test","orgId":"%s","name":"%s","config":"%s"}`,
			uuid.New().String(), largeName, largeConfig)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (ideally 413 or 400; handler may lack body-size middleware but must not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"integrationId":"","orgId":"","name":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_all_fields", func(t *testing.T) {
		payload := fmt.Sprintf(`{"integrationId":"intégration-日本語","orgId":"%s","name":"ブリッジ名前"}`, uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode fields: got %d — expected non-server-error", status)
		}
		t.Logf("unicode fields → %d", status)
	})

	t.Run("sql_injection_in_name", func(t *testing.T) {
		payload := fmt.Sprintf(`{"integrationId":"sqli-test","orgId":"%s","name":"'; DROP TABLE helixtrack_bridges; --"}`, uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 201 or 400", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("sql_injection_in_org_id", func(t *testing.T) {
		payload := `{"integrationId":"sqli-org","orgId":"'; DROP TABLE helixtrack_bridges; --","name":"SQLi Org"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection in org_id: got %d — expected 400 (invalid UUID)", status)
		}
		t.Logf("SQL injection in org_id → %d", status)
	})

	t.Run("xss_in_name", func(t *testing.T) {
		payload := fmt.Sprintf(`{"integrationId":"xss-test","orgId":"%s","name":"<script>alert('xss')</script>"}`, uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in name: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in name → %d", status)
	})

	t.Run("xss_in_integration_id", func(t *testing.T) {
		payload := fmt.Sprintf(`{"integrationId":"<script>alert('xss')</script>","orgId":"%s","name":"XSS Integration"}`, uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in integration_id: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in integration_id → %d", status)
	})

	t.Run("null_json_values", func(t *testing.T) {
		payload := `{"integrationId":null,"orgId":null,"name":null,"config":null}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("null values: got %d — expected 400", status)
		}
		t.Logf("null values → %d", status)
	})

	t.Run("nested_json_objects", func(t *testing.T) {
		payload := `{"integrationId":{"nested":"object"},"orgId":{"deep":{"nested":true}},"name":12345}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("nested objects: got %d — expected 400", status)
		}
		t.Logf("nested objects → %d", status)
	})
}
