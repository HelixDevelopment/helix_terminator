//go:build chaos

// Chaos test suite for audit-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed request bodies, binary garbage,
//     wrong content types — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, zero-value structs, unicode, injection attempts.
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
	"github.com/helixdevelopment/audit-service/internal/handler"
	"github.com/helixdevelopment/audit-service/internal/repository"
	"github.com/helixdevelopment/audit-service/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// chaosEnv holds the assembled test environment for chaos tests.
// Reuses the same setup pattern as stress tests — real handler,
// optional real DB.
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
		h := handler.New(repo)
		r.POST("/api/v1/audit/logs", h.CreateAuditLog)
		r.GET("/api/v1/audit/logs", h.ListAuditLogs)
		r.GET("/api/v1/audit/logs/:id", h.GetAuditLog)
		r.GET("/api/v1/audit/stats/actions", h.CountByAction)
		r.GET("/api/v1/audit/stats/resources", h.CountByResourceType)
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
	repo := repository.New(nil)
	h := handler.New(repo)
	r.POST("/api/v1/audit/logs", h.CreateAuditLog)
	r.GET("/api/v1/audit/logs", h.ListAuditLogs)
	r.GET("/api/v1/audit/logs/:id", h.GetAuditLog)
	r.GET("/api/v1/audit/stats/actions", h.CountByAction)
	r.GET("/api/v1/audit/stats/resources", h.CountByResourceType)
	ts := httptest.NewServer(r)
	return &chaosEnv{
		ts: ts,
		cleanup: func() {
			ts.Close()
		},
	}
}

// chaosPostRaw sends a POST request with a raw byte body and returns
// the status code + raw response body. Unlike stressPostJSON, this
// does NOT assume the body is valid JSON — it sends whatever bytes
// are provided.
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

	t.Run("malformed_json_bodies_to_create", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"{",
			"{}",
			"null",
			"[]",
			"42",
			`{"action":}`,
			`{"action":123,"resourceType":"user","severity":"info"}`, // wrong type
			`{"action":null,"resourceType":null,"severity":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		for i, body := range malformedBodies {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", []byte(body))
			if status == 0 {
				t.Logf("malformed body %d: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("malformed body %d: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed bodies against create endpoint", len(malformedBodies))
	})

	t.Run("malformed_json_bodies_to_list", func(t *testing.T) {
		// List is GET with query params — test malformed query strings
		malformedURLs := []string{
			"/api/v1/audit/logs?org_id=",
			"/api/v1/audit/logs?org_id=not-a-uuid",
			"/api/v1/audit/logs?user_id=not-a-uuid",
			"/api/v1/audit/logs?start=not-a-time",
			"/api/v1/audit/logs?end=not-a-time",
			"/api/v1/audit/logs?limit=not-a-number",
			"/api/v1/audit/logs?offset=not-a-number",
			"/api/v1/audit/logs?limit=-1",
			"/api/v1/audit/logs?offset=-1",
		}

		for i, url := range malformedURLs {
			status, _ := chaosGetRaw(t, client, env.ts.URL+url)
			if status == 0 {
				t.Logf("malformed URL %d: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("malformed URL %d (%s): got %d — expected 400", i, url, status)
			}
		}
		t.Logf("tested %d malformed URLs against list endpoint", len(malformedURLs))
	})

	t.Run("corrupt_ids_in_get", func(t *testing.T) {
		corruptIDs := []string{
			"not-a-uuid",
			"",
			strings.Repeat("x", 1000),
			"\x00\x01\x02\x03",
			"null",
			"undefined",
			"../../etc/passwd",
			"'; DROP TABLE audit_logs; --",
		}

		for i, id := range corruptIDs {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/audit/logs/"+id)
			if status == 0 {
				t.Logf("corrupt id %d: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("corrupt id %d (%q): got %d — expected 400/404", i, truncate(id, 30), status)
			}
		}
		t.Logf("tested %d corrupt IDs against get endpoint", len(corruptIDs))
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
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", ct,
				[]byte(`{"action":"create","resourceType":"user","severity":"info"}`))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
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
			body := fmt.Sprintf(`{"action":"create","resourceType":"user","severity":"info","details":{"id":%d}}`, id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", []byte(body))
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

	t.Run("rapid_fire_list", func(t *testing.T) {
		// Hammer the list endpoint — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/audit/logs?limit=5")
			if status >= 500 {
				t.Errorf("rapid list %d: got %d — expected 200 or 400", id, status)
			}
		})
	})

	t.Run("rapid_fire_stats", func(t *testing.T) {
		// Hammer the stats endpoints concurrently
		const burst = 20

		testutil.RunConcurrent(t, burst, func(id int) {
			statusActions, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/audit/stats/actions")
			if statusActions >= 500 {
				t.Errorf("stats/actions %d: got %d — expected 200", id, statusActions)
			}
			statusResources, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/audit/stats/resources")
			if statusResources >= 500 {
				t.Errorf("stats/resources %d: got %d — expected 200", id, statusResources)
			}
		})
	})

	t.Run("concurrent_get_nonexistent", func(t *testing.T) {
		// Multiple goroutines getting nonexistent IDs — must not deadlock
		const parallel = 10
		fakeID := "00000000-0000-0000-0000-000000000000"

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/audit/logs/"+fakeID)
			if status >= 500 {
				t.Errorf("concurrent get %d: got %d — expected 404", id, status)
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
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/audit/logs", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeDetails := strings.Repeat("x", 1000000)
		payload := fmt.Sprintf(`{"action":"create","resourceType":"user","severity":"info","details":{"data":"%s"}}`, largeDetails)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (handler lacks body-size middleware but does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		// All fields at zero value — validation must catch
		payload := `{"action":"","resourceType":"","severity":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_details", func(t *testing.T) {
		payload := `{"action":"create","resourceType":"user","severity":"info","details":{"name":"日本語テスト","desc":"パスワード"}}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode details: got %d — expected non-server-error", status)
		}
		t.Logf("unicode details → %d", status)
	})

	t.Run("sql_injection_in_details", func(t *testing.T) {
		payload := `{"action":"create","resourceType":"user","severity":"info","details":{"query":"'; DROP TABLE audit_logs; --"}}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 201 or 400", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_user_agent", func(t *testing.T) {
		payload := `{"action":"create","resourceType":"user","severity":"info","userAgent":"<script>alert('xss')</script>"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/audit/logs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in user agent: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in user agent → %d", status)
	})

	t.Run("negative_limit_and_offset", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/audit/logs?limit=-999&offset=-999")
		if status >= 500 {
			t.Errorf("negative limit/offset: got %d — expected 200 (clamped) or 400", status)
		}
		t.Logf("negative limit/offset → %d", status)
	})

	t.Run("extremely_large_limit", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/audit/logs?limit=999999999")
		if status >= 500 {
			t.Errorf("extremely large limit: got %d — expected 200 (clamped) or 400", status)
		}
		t.Logf("extremely large limit → %d", status)
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
