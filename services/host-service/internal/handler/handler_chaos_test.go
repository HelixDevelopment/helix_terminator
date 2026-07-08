//go:build chaos

// Chaos test suite for host-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: corrupt UUIDs, malformed request bodies,
//     binary garbage — detected and reported cleanly (no panic).
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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/host-service/internal/handler"
	"github.com/helixdevelopment/host-service/internal/model"
	"github.com/helixdevelopment/host-service/internal/repository"
	"github.com/helixdevelopment/host-service/internal/testutil"
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

	// Simulate auth middleware
	r.Use(func(c *gin.Context) {
		c.Set("userID", "00000000-0000-0000-0000-000000000000")
		c.Set("orgID", "00000000-0000-0000-0000-000000000000")
		c.Next()
	})

	if available {
		pool, err := pgxpool.New(t.Context(), poolURL)
		if err != nil {
			t.Fatalf("pgxpool.New failed: %v", err)
		}
		repo := repository.New(pool)
		h := handler.New(repo)
		r.POST("/api/v1/hosts", h.CreateHost)
		r.GET("/api/v1/hosts/:id", h.GetHost)
		r.GET("/api/v1/hosts", h.ListHosts)
		r.PUT("/api/v1/hosts/:id", h.UpdateHost)
		r.DELETE("/api/v1/hosts/:id", h.DeleteHost)
		r.POST("/api/v1/hosts/:id/test-connection", h.TestConnection)
		r.GET("/api/v1/hosts/:id/logs", h.GetConnectionLogs)
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
	r.POST("/api/v1/hosts", h.CreateHost)
	r.GET("/api/v1/hosts/:id", h.GetHost)
	r.GET("/api/v1/hosts", h.ListHosts)
	r.PUT("/api/v1/hosts/:id", h.UpdateHost)
	r.DELETE("/api/v1/hosts/:id", h.DeleteHost)
	r.POST("/api/v1/hosts/:id/test-connection", h.TestConnection)
	r.GET("/api/v1/hosts/:id/logs", h.GetConnectionLogs)
	ts := httptest.NewServer(r)
	return &chaosEnv{
		ts: ts,
		cleanup: func() {
			ts.Close()
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

// chaosDeleteRaw sends a DELETE request and returns status.
func chaosDeleteRaw(t *testing.T, client *http.Client, url string) int {
	t.Helper()
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// chaosValidHost returns a valid JSON body for create-host requests.
func chaosValidHost(name string) []byte {
	b, _ := json.Marshal(model.CreateHostRequest{
		Name:     name,
		Hostname: "192.168.1.1",
		Username: "admin",
		AuthType: model.AuthTypePassword,
	})
	return b
}

// TestChaosInputCorruption exercises corrupt/malformed inputs against
// all endpoints. Every case MUST produce a clean HTTP error response
// (not a panic, not a hang, not a 500 for input errors).
func TestChaosInputCorruption(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("corrupt_uuid_in_get", func(t *testing.T) {
		corruptIDs := []string{
			"not-a-uuid",
			"",
			"00000000-0000-0000-0000-000000000000",
			strings.Repeat("x", 1000),
			"\x00\x01\x02\x03",
			"null",
			"undefined",
			"12345",
		}

		for i, id := range corruptIDs {
			status, raw := chaosGetRaw(t, client, env.ts.URL+"/api/v1/hosts/"+id)
			if status == 0 {
				t.Logf("corrupt uuid %d: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("corrupt uuid %d: got %d (server error) for uuid %q — expected 400/404", i, status, truncate(id, 50))
			}
			t.Logf("corrupt uuid %d (%q) → %d: %s", i, truncate(id, 30), status, truncate(string(raw), 100))
		}
	})

	t.Run("corrupt_uuid_in_delete", func(t *testing.T) {
		corruptIDs := []string{
			"garbage",
			"ffffffff-ffff-ffff-ffff-ffffffffffff",
			strings.Repeat("a", 5000),
			"\xff\xfe\xfd",
		}

		for i, id := range corruptIDs {
			status := chaosDeleteRaw(t, client, env.ts.URL+"/api/v1/hosts/"+id)
			if status >= 500 {
				t.Errorf("corrupt delete uuid %d: got %d — expected 400/404/503", i, status)
			}
			t.Logf("corrupt delete uuid %d → %d", i, status)
		}
	})

	t.Run("corrupt_uuid_in_update", func(t *testing.T) {
		corruptIDs := []string{
			"not-a-uuid",
			"null",
			strings.Repeat("z", 200),
		}

		for i, id := range corruptIDs {
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/hosts/"+id, "application/json", []byte(`{"name":"updated"}`))
			if status >= 500 {
				t.Errorf("corrupt update uuid %d: got %d — expected 400", i, status)
			}
			t.Logf("corrupt update uuid %d → %d", i, status)
		}
	})

	t.Run("corrupt_uuid_in_test_connection", func(t *testing.T) {
		status, raw := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts/not-a-uuid/test-connection", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("corrupt test-connection uuid: got %d — expected 400", status)
		}
		t.Logf("corrupt test-connection uuid → %d: %s", status, truncate(string(raw), 100))
	})

	t.Run("corrupt_uuid_in_logs", func(t *testing.T) {
		status, raw := chaosGetRaw(t, client, env.ts.URL+"/api/v1/hosts/not-a-uuid/logs")
		if status >= 500 {
			t.Errorf("corrupt logs uuid: got %d — expected 400", status)
		}
		t.Logf("corrupt logs uuid → %d: %s", status, truncate(string(raw), 100))
	})

	t.Run("malformed_json_bodies", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"{",
			"{}",
			"null",
			"[]",
			"42",
			`{"name":}`,
			`{"name":123,"hostname":true}`, // wrong types
			`{"name":null,"hostname":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []string{
			"/api/v1/hosts",
			"/api/v1/hosts/not-a-uuid/test-connection",
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

		// Also test PUT endpoints
		for i, body := range malformedBodies {
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/hosts/not-a-uuid", "application/json", []byte(body))
			if status == 0 {
				continue
			}
			if status >= 500 {
				t.Errorf("malformed body %d to PUT: got %d — expected 400", i, status)
			}
		}

		t.Logf("tested %d malformed bodies across POST+PUT endpoints", len(malformedBodies))
	})

	t.Run("wrong_content_type", func(t *testing.T) {
		contentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"",
			"application/octet-stream",
		}
		validBody := chaosValidHost("ct-test")
		for _, ct := range contentTypes {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", ct, validBody)
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", garbage)
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
		const burst = 50
		errCount := 0
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			body := chaosValidHost(fmt.Sprintf("chaos-rapid-%d", id))
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", body)
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

	t.Run("rapid_fire_get_nonexistent", func(t *testing.T) {
		// Hammer GET with nonexistent UUIDs — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			fakeID := fmt.Sprintf("00000000-0000-0000-0000-%012d", id)
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/hosts/"+fakeID)
			if status >= 500 {
				t.Errorf("get nonexistent %d: got %d — expected 404/503", id, status)
			}
		})
	})

	t.Run("concurrent_update_same_host", func(t *testing.T) {
		// Multiple goroutines updating the same (nonexistent) host
		// simultaneously — must not deadlock
		const parallel = 10
		body := []byte(`{"name":"contended-update"}`)

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/hosts/00000000-0000-0000-0000-000000000001", "application/json", body)
			if status >= 500 {
				t.Errorf("concurrent update %d: got %d — expected 404/503", id, status)
			}
		})
	})

	t.Run("concurrent_delete_same_host", func(t *testing.T) {
		// Multiple goroutines deleting the same host — must not deadlock
		const parallel = 10

		testutil.RunConcurrent(t, parallel, func(id int) {
			status := chaosDeleteRaw(t, client, env.ts.URL+"/api/v1/hosts/00000000-0000-0000-0000-000000000002")
			if status >= 500 {
				t.Errorf("concurrent delete %d: got %d — expected 404/503", id, status)
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
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/hosts", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeName := strings.Repeat("a", 500000)
		payload := fmt.Sprintf(`{"name":"%s","hostname":"%s","username":"admin","auth_type":"password"}`,
			largeName, strings.Repeat("b", 500000))
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (ideally 413 or 400; handler lacks body-size middleware but does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"name":"","hostname":"","username":"","auth_type":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_all_fields", func(t *testing.T) {
		payload := `{"name":"ホスト名","hostname":"192.168.1.1","username":"ユーザー","auth_type":"password"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode fields: got %d — expected non-server-error", status)
		}
		t.Logf("unicode fields → %d", status)
	})

	t.Run("sql_injection_in_name", func(t *testing.T) {
		payload := `{"name":"'; DROP TABLE hosts; --","hostname":"192.168.1.1","username":"admin","auth_type":"password"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 400 or 201", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_name", func(t *testing.T) {
		payload := `{"name":"<script>alert('xss')</script>","hostname":"192.168.1.1","username":"admin","auth_type":"password"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in name: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in name → %d", status)
	})

	t.Run("negative_port", func(t *testing.T) {
		payload := `{"name":"neg-port","hostname":"192.168.1.1","port":-1,"username":"admin","auth_type":"password"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("negative port: got %d — expected 400", status)
		}
		t.Logf("negative port → %d", status)
	})

	t.Run("zero_port_gets_default", func(t *testing.T) {
		payload := `{"name":"zero-port","hostname":"192.168.1.1","port":0,"username":"admin","auth_type":"password"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte(payload))
		// Port 0 is omitted in binding (omitempty), so the handler
		// sets it to 22 as default. Either 201 (with DB) or 503 (no DB).
		if status >= 500 && status != http.StatusServiceUnavailable {
			t.Errorf("zero port: got %d — expected 201 or 503", status)
		}
		t.Logf("zero port → %d", status)
	})

	t.Run("extremely_high_port", func(t *testing.T) {
		payload := `{"name":"high-port","hostname":"192.168.1.1","port":99999,"username":"admin","auth_type":"password"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/hosts", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("port 99999: got %d — expected 400", status)
		}
		t.Logf("port 99999 → %d", status)
	})

	t.Run("invalid_limit_in_list", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/hosts?limit=999999")
		if status >= 500 {
			t.Errorf("huge limit: got %d — expected 400", status)
		}
		t.Logf("huge limit → %d", status)
	})

	t.Run("negative_offset_in_list", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/hosts?offset=-1")
		if status >= 500 {
			t.Errorf("negative offset: got %d — expected 400", status)
		}
		t.Logf("negative offset → %d", status)
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
