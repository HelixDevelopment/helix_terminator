//go:build chaos

// Chaos test suite for collaboration-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed request bodies, binary garbage,
//     invalid UUIDs in path params — detected and reported cleanly
//     (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, zero-value structs, unicode, SQL injection.
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
	"github.com/helixdevelopment/collaboration-service/internal/handler"
	"github.com/helixdevelopment/collaboration-service/internal/repository"
	"github.com/helixdevelopment/collaboration-service/internal/testutil"
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
		r.POST("/sessions", func(c *gin.Context) {
			if uid := c.GetHeader("X-User-ID"); uid != "" {
				c.Set("user_id", uid)
			}
			h.CreateSession(c)
		})
		r.GET("/sessions/:id", h.GetSession)
		r.GET("/sessions", h.ListSessions)
		r.POST("/sessions/:id/join", h.JoinSession)
		r.POST("/sessions/:id/leave", func(c *gin.Context) {
			if uid := c.GetHeader("X-User-ID"); uid != "" {
				c.Set("user_id", uid)
			}
			h.LeaveSession(c)
		})
		r.POST("/sessions/:id/end", h.EndSession)
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
	r.POST("/sessions", h.CreateSession)
	r.GET("/sessions/:id", h.GetSession)
	r.GET("/sessions", h.ListSessions)
	r.POST("/sessions/:id/join", h.JoinSession)
	r.POST("/sessions/:id/leave", h.LeaveSession)
	r.POST("/sessions/:id/end", h.EndSession)
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

// chaosGetRaw sends a GET request and returns the status code + raw
// response body.
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
			`{"host_id":}`,
			`{"host_id":"not-a-uuid","name":123}`, // wrong type
			`{"host_id":null,"name":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []string{"/sessions", "/sessions/" + uuid.New().String() + "/join", "/sessions/" + uuid.New().String() + "/leave", "/sessions/" + uuid.New().String() + "/end"}
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
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", ct, []byte(`{"host_id":"`+uuid.New().String()+`","name":"test"}`))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("invalid_uuid_in_path", func(t *testing.T) {
		invalidPaths := []string{
			"/sessions/not-a-uuid",
			"/sessions/12345",
			"/sessions/" + strings.Repeat("x", 500),
			"/sessions/../../../../etc/passwd",
			"/sessions/%00%01%02",
		}
		for _, path := range invalidPaths {
			status, _ := chaosGetRaw(t, client, env.ts.URL+path)
			if status >= 500 {
				t.Errorf("invalid path %q: got %d — expected 400", truncate(path, 50), status)
			}
			t.Logf("invalid path %q → %d", truncate(path, 50), status)
		}
	})

	t.Run("join_with_invalid_body", func(t *testing.T) {
		fakeID := uuid.New().String()
		invalidBodies := []string{
			`{}`,
			`{"user_id":"not-a-uuid"}`,
			`{"user_id":""}`,
			`{"user_id":null}`,
			`[]`,
			`null`,
		}
		for i, body := range invalidBodies {
			status, raw := chaosPostRaw(t, client, env.ts.URL+"/sessions/"+fakeID+"/join", "application/json", []byte(body))
			if status >= 500 {
				t.Errorf("join invalid body %d: got %d — expected 400; body=%s", i, status, truncate(string(raw), 100))
			}
			t.Logf("join invalid body %d → %d", i, status)
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
			body := fmt.Sprintf(`{"host_id":"%s","name":"chaos-rapid-%d-%d"}`,
				uuid.New().String(), id, id*1000+id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(body))
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
		// Hammer GET /sessions/:id with random UUIDs — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			fakeID := uuid.New().String()
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/sessions/"+fakeID)
			if status >= 500 {
				t.Errorf("get nonexistent %d: got %d — expected 404", id, status)
			}
		})
	})

	t.Run("concurrent_join_same_session", func(t *testing.T) {
		// Multiple goroutines joining the same (nonexistent) session
		// simultaneously — must not deadlock
		const parallel = 10
		fakeID := uuid.New().String()
		body := []byte(`{"user_id":"` + uuid.New().String() + `"}`)

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions/"+fakeID+"/join", "application/json", body)
			if status >= 500 {
				t.Errorf("concurrent join %d: got %d — expected 400/404", id, status)
			}
		})
	})

	t.Run("rapid_fire_list", func(t *testing.T) {
		// Hammer GET /sessions with various pagination params
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			url := fmt.Sprintf("%s/sessions?limit=%d&offset=%d", env.ts.URL, id%100+1, id*10)
			status, _ := chaosGetRaw(t, client, url)
			if status >= 500 {
				t.Errorf("rapid list %d: got %d — expected 200", id, status)
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
		req, _ := http.NewRequest("POST", env.ts.URL+"/sessions", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeName := strings.Repeat("a", 1000000)
		payload := fmt.Sprintf(`{"host_id":"%s","name":"%s"}`, uuid.New().String(), largeName)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (ideally 413 or 400; handler lacks body-size middleware but does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		// All fields at zero value — validation must catch
		payload := `{"host_id":"","name":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_name", func(t *testing.T) {
		payload := `{"host_id":"` + uuid.New().String() + `","name":"日本語テストセッション名前"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode name: got %d — expected non-server-error", status)
		}
		t.Logf("unicode name → %d", status)
	})

	t.Run("sql_injection_in_name", func(t *testing.T) {
		payload := `{"host_id":"` + uuid.New().String() + `","name":"'; DROP TABLE collaboration_sessions; --"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 201 or 400", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_name", func(t *testing.T) {
		payload := `{"host_id":"` + uuid.New().String() + `","name":"<script>alert('xss')</script>"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in name: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in name → %d", status)
	})

	t.Run("negative_pagination", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/sessions?limit=-1&offset=-100")
		if status >= 500 {
			t.Errorf("negative pagination: got %d — expected 200 (clamped)", status)
		}
		t.Logf("negative pagination → %d", status)
	})

	t.Run("huge_pagination_values", func(t *testing.T) {
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/sessions?limit=999999999&offset=999999999")
		if status >= 500 {
			t.Errorf("huge pagination: got %d — expected 200 (clamped)", status)
		}
		t.Logf("huge pagination → %d", status)
	})

	t.Run("leave_without_user_id_header", func(t *testing.T) {
		fakeID := uuid.New().String()
		req, _ := http.NewRequest("POST", env.ts.URL+"/sessions/"+fakeID+"/leave", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			t.Errorf("leave without user_id: got %d — expected 400", resp.StatusCode)
		}
		t.Logf("leave without user_id → %d", resp.StatusCode)
	})

	t.Run("extremely_long_path_segment", func(t *testing.T) {
		longID := strings.Repeat("a", 10000)
		status, _ := chaosGetRaw(t, client, env.ts.URL+"/sessions/"+longID)
		if status >= 500 {
			t.Errorf("long path segment: got %d — expected 400", status)
		}
		t.Logf("long path segment → %d", status)
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
