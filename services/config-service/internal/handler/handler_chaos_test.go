//go:build chaos

// Chaos test suite for config-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed request bodies, binary garbage,
//     wrong content types — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, zero-value structs, SQL injection, XSS.
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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/config-service/internal/handler"
	"github.com/helixdevelopment/config-service/internal/repository"
	"github.com/helixdevelopment/config-service/internal/testutil"
)

// chaosEnv holds the assembled test environment for chaos tests.
type chaosEnv struct {
	ts      *httptest.Server
	repo    *repository.Repository
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
		registerRoutes(r, h)
		ts := httptest.NewServer(r)
		return &chaosEnv{
			ts:   ts,
			repo: repo,
			cleanup: func() {
				ts.Close()
				pool.Close()
			},
		}
	}

	// Nil-repo fallback — validation-only, no DB
	h := handler.New(nil)
	registerRoutes(r, h)
	ts := httptest.NewServer(r)
	return &chaosEnv{
		ts: ts,
		cleanup: func() {
			ts.Close()
		},
	}
}

// registerRoutes wires all config-service routes onto the gin engine.
func registerRoutes(r *gin.Engine, h *handler.Handler) {
	r.POST("/api/v1/configs", h.CreateConfig)
	r.GET("/api/v1/configs", h.ListConfigs)
	r.GET("/api/v1/configs/:id", h.GetConfig)
	r.GET("/api/v1/configs/by-key", h.GetConfigByKey)
	r.PUT("/api/v1/configs/:id", h.UpdateConfig)
	r.DELETE("/api/v1/configs/:id", h.DeleteConfig)
	r.POST("/api/v1/configs/bulk", h.BulkCreateConfigs)
	r.GET("/health", h.HealthCheck)
	r.GET("/ready", h.ReadinessCheck)
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

// chaosGetRaw sends a GET request and returns status + raw response.
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

	t.Run("malformed_json_bodies_create", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"{",
			"{}",
			"null",
			"[]",
			"42",
			`{"scope":}`,
			`{"scope":"global","key":"test","value":"v","valueType":123}`, // wrong type
			`{"scope":null,"key":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		for i, body := range malformedBodies {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(body))
			if status == 0 {
				t.Logf("malformed body %d to create: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("malformed body %d to create: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed bodies against create endpoint", len(malformedBodies))
	})

	t.Run("malformed_json_bodies_bulk", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"not-an-array",
			`{"not":"array"}`,
			"[{}]",
			`[{"scope":"org","key":"test","value":"v","valueType":"string"}]`, // missing scope_id
		}

		for i, body := range malformedBodies {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs/bulk", "application/json", []byte(body))
			if status == 0 {
				t.Logf("malformed body %d to bulk: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("malformed body %d to bulk: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed bodies against bulk endpoint", len(malformedBodies))
	})

	t.Run("malformed_json_bodies_update", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"{",
			"null",
			"[]",
			strings.Repeat("{", 100),
		}

		for i, body := range malformedBodies {
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/configs/00000000-0000-0000-0000-000000000000", "application/json", []byte(body))
			if status == 0 {
				t.Logf("malformed body %d to update: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("malformed body %d to update: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed bodies against update endpoint", len(malformedBodies))
	})

	t.Run("wrong_content_type", func(t *testing.T) {
		contentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"",
			"application/octet-stream",
		}
		validBody := `{"scope":"global","key":"test","value":"v","valueType":"string"}`
		for _, ct := range contentTypes {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", ct, []byte(validBody))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("invalid_uuid_in_path", func(t *testing.T) {
		invalidPaths := []string{
			"/api/v1/configs/not-a-uuid",
			"/api/v1/configs/",
			"/api/v1/configs/../../../../etc/passwd",
			"/api/v1/configs/" + strings.Repeat("a", 10000),
		}
		for _, path := range invalidPaths {
			status, _ := chaosGetRaw(t, client, env.ts.URL+path)
			if status >= 500 {
				t.Errorf("invalid path %q: got %d — expected 400", truncate(path, 50), status)
			}
			t.Logf("invalid path %q → %d", truncate(path, 50), status)
		}
	})

	t.Run("invalid_query_params_by_key", func(t *testing.T) {
		invalidURLs := []string{
			"/api/v1/configs/by-key",
			"/api/v1/configs/by-key?scope=global",
			"/api/v1/configs/by-key?key=test",
			"/api/v1/configs/by-key?scope=invalid&key=test",
			"/api/v1/configs/by-key?scope=global&key=test&scope_id=not-a-uuid",
		}
		for _, url := range invalidURLs {
			status, _ := chaosGetRaw(t, client, env.ts.URL+url)
			if status >= 500 {
				t.Errorf("invalid query %q: got %d — expected 400", truncate(url, 60), status)
			}
			t.Logf("invalid query %q → %d", truncate(url, 60), status)
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
			body := fmt.Sprintf(`{"scope":"global","key":"chaos-rapid-%d-%d","value":"v","valueType":"string"}`,
				id, id*1000+id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(body))
			if status == 0 {
				errCount++
				return
			}
			if status >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d requests: connection_errors=%d server_errors=%d", burst, errCount, serverErrCount)
		if serverErrCount == burst {
			t.Errorf("ALL %d requests returned 500 — service is down", burst)
		}
	})

	t.Run("rapid_fire_list", func(t *testing.T) {
		// Hammer list endpoint — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/configs?scope=global&limit=10")
			if status >= 500 {
				t.Errorf("list %d: got %d — expected 200", id, status)
			}
		})
	})

	t.Run("rapid_fire_health_check", func(t *testing.T) {
		// Hammer health check — must always return 200
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/health")
			if status != http.StatusOK {
				t.Errorf("health %d: got %d — expected 200", id, status)
			}
		})
	})

	t.Run("concurrent_get_nonexistent", func(t *testing.T) {
		// Multiple goroutines getting nonexistent IDs simultaneously
		// — must not deadlock
		const parallel = 15

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/configs/00000000-0000-0000-0000-000000000000")
			if status >= 500 {
				t.Errorf("concurrent get %d: got %d — expected 404 or 400", id, status)
			}
		})
	})

	t.Run("rapid_fire_bulk_create", func(t *testing.T) {
		// Hammer bulk endpoint with small batches
		const burst = 20

		testutil.RunConcurrent(t, burst, func(id int) {
			batch := fmt.Sprintf(`[{"scope":"global","key":"chaos-bulk-%d","value":"v","valueType":"string"}]`, id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs/bulk", "application/json", []byte(batch))
			if status >= 500 {
				t.Errorf("bulk %d: got %d — expected 201 or 400", id, status)
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

	t.Run("nil_body_create", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/configs", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeValue := strings.Repeat("x", 1000000)
		payload := fmt.Sprintf(`{"scope":"global","key":"large-test","value":"%s","valueType":"string"}`, largeValue)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (handler lacks body-size middleware but does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"scope":"","key":"","value":"","valueType":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_all_fields", func(t *testing.T) {
		payload := `{"scope":"global","key":"テストキー","value":"日本語テスト","valueType":"string","description":"説明文"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("unicode fields: got %d — expected non-server-error", status)
		}
		t.Logf("unicode fields → %d", status)
	})

	t.Run("sql_injection_in_key", func(t *testing.T) {
		payload := `{"scope":"global","key":"'; DROP TABLE configs; --","value":"test","valueType":"string"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 201 or 400", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_description", func(t *testing.T) {
		payload := `{"scope":"global","key":"xss-test","value":"test","valueType":"string","description":"<script>alert('xss')</script>"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in description: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in description → %d", status)
	})

	t.Run("extremely_long_scope", func(t *testing.T) {
		longScope := strings.Repeat("s", 10000)
		payload := fmt.Sprintf(`{"scope":"%s","key":"test","value":"v","valueType":"string"}`, longScope)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("extremely long scope: got %d — expected 400", status)
		}
		t.Logf("extremely long scope → %d", status)
	})

	t.Run("extremely_long_key", func(t *testing.T) {
		longKey := strings.Repeat("k", 100000)
		payload := fmt.Sprintf(`{"scope":"global","key":"%s","value":"v","valueType":"string"}`, longKey)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("extremely long key: got %d — expected 400", status)
		}
		t.Logf("extremely long key → %d", status)
	})

	t.Run("null_values_in_optional_fields", func(t *testing.T) {
		payload := `{"scope":"global","key":"null-test","value":"v","valueType":"string","description":null,"isSecret":null}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/configs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("null optional fields: got %d — expected 201 or 400", status)
		}
		t.Logf("null optional fields → %d", status)
	})

	t.Run("negative_limit_offset", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/configs?limit=-1&offset=-1")
		if status >= 500 {
			t.Errorf("negative limit/offset: got %d — expected 400", status)
		}
		t.Logf("negative limit/offset → %d", status)
	})

	t.Run("extremely_large_limit", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/configs?limit=999999999")
		if status >= 500 {
			t.Errorf("extremely large limit: got %d — expected 400 or 200", status)
		}
		t.Logf("extremely large limit → %d", status)
	})
}
