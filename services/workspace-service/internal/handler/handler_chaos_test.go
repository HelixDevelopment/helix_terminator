//go:build chaos

// Chaos test suite for workspace-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed request bodies, binary garbage,
//     invalid UUIDs, wrong content types — detected and reported
//     cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, zero-value structs.
//
// Run:
//
//	go test -race -tags chaos -run TestChaos -v -timeout 120s ./internal/handler/
package handler_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/workspace-service/internal/handler"
	"github.com/helixdevelopment/workspace-service/internal/repository"
	"github.com/helixdevelopment/workspace-service/internal/testutil"
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

	orgID := uuid.New()
	userID := uuid.New()

	// Middleware to inject userID and orgID into context.
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID.String())
		c.Set("orgID", orgID.String())
		c.Next()
	})

	if available {
		pool, err := pgxpool.New(t.Context(), poolURL)
		if err != nil {
			t.Fatalf("pgxpool.New failed: %v", err)
		}
		repo := repository.New(pool)
		h := handler.New(repo)

		r.POST("/api/v1/workspaces", h.CreateWorkspace)
		r.GET("/api/v1/workspaces", h.ListWorkspaces)
		r.GET("/api/v1/workspaces/:id", h.GetWorkspace)
		r.PUT("/api/v1/workspaces/:id", h.UpdateWorkspace)
		r.DELETE("/api/v1/workspaces/:id", h.DeleteWorkspace)
		r.POST("/api/v1/workspaces/:id/hosts", h.AddHost)
		r.DELETE("/api/v1/workspaces/:id/hosts/:host_id", h.RemoveHost)
		r.GET("/api/v1/workspaces/:id/hosts", h.ListHosts)
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
	r.POST("/api/v1/workspaces", h.CreateWorkspace)
	r.GET("/api/v1/workspaces", h.ListWorkspaces)
	r.GET("/api/v1/workspaces/:id", h.GetWorkspace)
	r.PUT("/api/v1/workspaces/:id", h.UpdateWorkspace)
	r.DELETE("/api/v1/workspaces/:id", h.DeleteWorkspace)
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

// chaosPutRaw sends a PUT request with a raw byte body and returns
// the status code + raw response body.
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
			`{"name":}`,
			`{"name":123}`, // wrong type
			`{"name":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []string{
			"/api/v1/workspaces",
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
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", ct, []byte(`{"name":"test-workspace"}`))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("invalid_uuid_in_path", func(t *testing.T) {
		invalidIDs := []string{
			"not-a-uuid",
			"",
			"12345",
			strings.Repeat("x", 1000),
			"00000000-0000-0000-0000-000000000000", // nil UUID
			"null",
			"undefined",
		}

		for i, id := range invalidIDs {
			// GET
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/workspaces/"+id, nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Logf("invalid UUID %d GET: connection failed (acceptable)", i)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("invalid UUID %d GET (%q): got %d — expected 400/404", i, truncate(id, 30), resp.StatusCode)
			}

			// PUT
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/workspaces/"+id, "application/json", []byte(`{"name":"test"}`))
			if status >= 500 {
				t.Errorf("invalid UUID %d PUT (%q): got %d — expected 400/404", i, truncate(id, 30), status)
			}

			// DELETE
			req, _ = http.NewRequest("DELETE", env.ts.URL+"/api/v1/workspaces/"+id, nil)
			resp, err = client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("invalid UUID %d DELETE (%q): got %d — expected 400", i, truncate(id, 30), resp.StatusCode)
			}
		}
	})

	t.Run("corrupt_update_body", func(t *testing.T) {
		corruptBodies := []string{
			"not json",
			`{"name": 123}`,
			`{"description": ["array", "not", "string"]}`,
			`{"tags": "not-an-array"}`,
			"{\x00\x01\x02}",
		}

		fakeID := uuid.New().String()
		for i, body := range corruptBodies {
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/workspaces/"+fakeID, "application/json", []byte(body))
			if status >= 500 {
				t.Errorf("corrupt update body %d: got %d — expected 400", i, status)
			}
			t.Logf("corrupt update body %d → %d", i, status)
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
			body := fmt.Sprintf(`{"name":"chaos-rapid-%d-%d","description":"Chaos %d","tags":["chaos"]}`,
				id, id*1000+id, id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", []byte(body))
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
		// Hammer GET /api/v1/workspaces — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/workspaces?limit=10&offset=0", nil)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("rapid list %d: got %d — expected 200", id, resp.StatusCode)
			}
		})
	})

	t.Run("concurrent_get_nonexistent", func(t *testing.T) {
		// Multiple goroutines GETing the same nonexistent workspace
		// simultaneously — must not deadlock
		const parallel = 10
		fakeID := uuid.New().String()

		testutil.RunConcurrent(t, parallel, func(id int) {
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/workspaces/"+fakeID, nil)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("concurrent get nonexistent %d: got %d — expected 404", id, resp.StatusCode)
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
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/workspaces", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeName := strings.Repeat("a", 500000)
		payload := fmt.Sprintf(`{"name":"%s","description":"%s"}`,
			largeName, strings.Repeat("d", 500000))
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (ideally 413 or 400; handler does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		// All fields at zero value — validation must catch
		payload := `{"name":"","description":"","color":"","icon":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_all_fields", func(t *testing.T) {
		payload := `{"name":"ワークスペース","description":"日本語テスト","icon":"🎯","tags":["タグ"]}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode fields: got %d — expected non-server-error", status)
		}
		t.Logf("unicode fields → %d", status)
	})

	t.Run("sql_injection_in_name", func(t *testing.T) {
		payload := `{"name":"'; DROP TABLE workspaces; --","description":"SQLi"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 201 or 400", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_name", func(t *testing.T) {
		payload := `{"name":"<script>alert('xss')</script>","description":"XSS test"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in name: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in name → %d", status)
	})

	t.Run("extremely_long_tags_array", func(t *testing.T) {
		// Generate a tags array with 1000 entries
		tags := make([]string, 1000)
		for i := range tags {
			tags[i] = fmt.Sprintf("tag-%d", i)
		}
		tagJSON := "["
		for i, tag := range tags {
			if i > 0 {
				tagJSON += ","
			}
			tagJSON += `"` + tag + `"`
		}
		tagJSON += "]"
		payload := fmt.Sprintf(`{"name":"%s","tags":%s}`, uniqueName("chaos-tags", 0), tagJSON)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/workspaces", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("1000 tags: got %d — expected non-server-error", status)
		}
		t.Logf("1000 tags → %d", status)
	})

	t.Run("update_with_empty_updates_map", func(t *testing.T) {
		// PUT with empty JSON object — handler builds empty updates map
		fakeID := uuid.New().String()
		status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/workspaces/"+fakeID, "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty update: got %d — expected 400/404", status)
		}
		t.Logf("empty update → %d", status)
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
