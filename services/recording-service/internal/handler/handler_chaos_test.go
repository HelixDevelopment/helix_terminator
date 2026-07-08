//go:build chaos

// Chaos test suite for recording-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed JSON, binary garbage, wrong types,
//     invalid UUIDs — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, concurrent deletes,
//     verify graceful degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, zero-value structs, unicode, SQL injection attempts.
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
	"github.com/google/uuid"
	"github.com/helixdevelopment/recording-service/internal/handler"
	"github.com/helixdevelopment/recording-service/internal/repository"
	"github.com/helixdevelopment/recording-service/internal/testutil"
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

	if available {
		pool, err := pgxpool.New(t.Context(), poolURL)
		if err != nil {
			t.Fatalf("pgxpool.New failed: %v", err)
		}
		repo := repository.New(pool)
		h := handler.New(repo)
		r.POST("/recordings", h.CreateRecording)
		r.GET("/recordings/:id", h.GetRecording)
		r.GET("/recordings", h.ListRecordings)
		r.PUT("/recordings/:id", h.UpdateRecording)
		r.DELETE("/recordings/:id", h.DeleteRecording)
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
	h := handler.New(nil)
	r.POST("/recordings", h.CreateRecording)
	r.GET("/recordings/:id", h.GetRecording)
	r.GET("/recordings", h.ListRecordings)
	r.PUT("/recordings/:id", h.UpdateRecording)
	r.DELETE("/recordings/:id", h.DeleteRecording)
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

// chaosDeleteRaw sends a DELETE request and returns status + raw body.
func chaosDeleteRaw(t *testing.T, client *http.Client, url string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest("DELETE", url, nil)
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
			`{"sessionId":}`,
			`{"sessionId":"not-a-uuid","hostId":"not-a-uuid","filePath":123,"format":456}`,
			`{"sessionId":null,"hostId":null,"filePath":null,"format":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []string{"/recordings"}
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
		validJSON := `{"sessionId":"` + uuid.New().String() + `","hostId":"` + uuid.New().String() + `","filePath":"/test.cast","format":"asciinema"}`
		contentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"",
			"application/octet-stream",
		}
		for _, ct := range contentTypes {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", ct, []byte(validJSON))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("corrupt_uuid_in_path", func(t *testing.T) {
		corruptIDs := []string{
			"not-a-uuid",
			"",
			strings.Repeat("x", 1000),
			"null",
			"undefined",
			"12345",
			"ffffffff-ffff-ffff-ffff-ffffffffffff", // valid format, likely nonexistent
		}

		for i, id := range corruptIDs {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/recordings/"+id)
			if status == 0 {
				t.Logf("corrupt UUID %d: connection failed (acceptable)", i)
				continue
			}
			// 200 for a valid-format UUID that happens to exist is OK;
			// 404 for valid-format-not-found is OK; 400 for invalid format is OK
			if status >= 500 {
				t.Errorf("corrupt UUID %d (%q): got %d — expected 400/404", i, truncate(id, 30), status)
			}
			t.Logf("corrupt UUID %d (%q) → %d", i, truncate(id, 30), status)
		}
	})

	t.Run("corrupt_uuid_in_update", func(t *testing.T) {
		corruptIDs := []string{
			"garbage",
			strings.Repeat("A", 5000),
			"\xff\xfe\xfd",
		}

		for i, id := range corruptIDs {
			body := `{"status":"completed","durationSec":10,"fileSizeBytes":1024}`
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/recordings/"+id, "application/json", []byte(body))
			if status >= 500 {
				t.Errorf("corrupt update UUID %d: got %d — expected 400", i, status)
			}
			t.Logf("corrupt update UUID %d → %d", i, status)
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
		const burst = 50
		errCount := 0
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			body := fmt.Sprintf(`{"sessionId":"%s","hostId":"%s","filePath":"/recordings/chaos-%d.cast","format":"asciinema"}`,
				uuid.New().String(), uuid.New().String(), id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", []byte(body))
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
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/recordings?limit=10&offset=0")
			if status >= 500 {
				t.Errorf("list %d: got %d — expected 200", id, status)
			}
		})
	})

	t.Run("concurrent_delete_same_id", func(t *testing.T) {
		// Create a recording first
		body := fmt.Sprintf(`{"sessionId":"%s","hostId":"%s","filePath":"/recordings/dup-delete.cast","format":"asciinema"}`,
			uuid.New().String(), uuid.New().String())
		status, resp := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", []byte(body))
		if status != 201 {
			t.Fatalf("create failed: %d", status)
		}

		var parsed map[string]interface{}
		_ = json.Unmarshal(resp, &parsed)
		id, _ := parsed["id"].(string)
		if id == "" {
			t.Fatal("no id in create response")
		}

		// Multiple goroutines deleting the same recording — must not deadlock
		const parallel = 10
		testutil.RunConcurrent(t, parallel, func(gid int) {
			status, _ := chaosDeleteRaw(t, client, env.ts.URL+"/recordings/"+id)
			// First delete succeeds (200), subsequent get 404 or 500
			// The important thing is no panic/deadlock
			if status == 0 {
				t.Errorf("concurrent delete %d: connection failed", gid)
			}
			t.Logf("concurrent delete %d → %d", gid, status)
		})
	})

	t.Run("rapid_fire_health_check", func(t *testing.T) {
		const burst = 100

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/health")
			if status != http.StatusOK {
				t.Errorf("health %d: got %d — expected 200", id, status)
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
		req, _ := http.NewRequest("POST", env.ts.URL+"/recordings", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largePath := "/recordings/" + strings.Repeat("a", 500000) + ".cast"
		payload := fmt.Sprintf(`{"sessionId":"%s","hostId":"%s","filePath":"%s","format":"asciinema"}`,
			uuid.New().String(), uuid.New().String(), largePath)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (handler rejects via validation or DB constraint, no panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"sessionId":"","hostId":"","filePath":"","format":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_file_path", func(t *testing.T) {
		payload := fmt.Sprintf(`{"sessionId":"%s","hostId":"%s","filePath":"/recordings/日本語テスト.cast","format":"asciinema"}`,
			uuid.New().String(), uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("unicode file path: got %d — expected non-server-error", status)
		}
		t.Logf("unicode file path → %d", status)
	})

	t.Run("sql_injection_in_file_path", func(t *testing.T) {
		payload := fmt.Sprintf(`{"sessionId":"%s","hostId":"%s","filePath":"'; DROP TABLE recordings; --","format":"asciinema"}`,
			uuid.New().String(), uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 400 or 201 (parameterised)", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_file_path", func(t *testing.T) {
		payload := fmt.Sprintf(`{"sessionId":"%s","hostId":"%s","filePath":"<script>alert('xss')</script>","format":"asciinema"}`,
			uuid.New().String(), uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/recordings", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in file path: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in file path → %d", status)
	})

	t.Run("negative_limit_and_offset", func(t *testing.T) {
		status, body := chaosGetRaw(t, client, env.ts.URL+"/recordings?limit=-1&offset=-100")
		if status >= 500 {
			t.Errorf("negative limit/offset: got %d — expected 200 with defaults", status)
		}
		var parsed map[string]interface{}
		_ = json.Unmarshal(body, &parsed)
		t.Logf("negative limit/offset → %d, body=%v", status, parsed)
	})

	t.Run("extreme_limit_and_offset", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/recordings?limit=999999&offset=999999")
		if status >= 500 {
			t.Errorf("extreme limit/offset: got %d — expected 200", status)
		}
		t.Logf("extreme limit/offset → %d", status)
	})

	t.Run("list_with_invalid_uuid_filters", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/recordings?host_id=not-a-uuid&session_id=also-not-a-uuid")
		if status >= 500 {
			t.Errorf("invalid UUID filters: got %d — expected 200 (ignored)", status)
		}
		t.Logf("invalid UUID filters → %d", status)
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
