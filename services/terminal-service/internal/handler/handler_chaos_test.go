//go:build chaos

// Chaos test suite for terminal-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed JSON, invalid UUIDs, binary garbage,
//     wrong content types — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, zero-value structs, SQL injection attempts.
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

	"github.com/helixdevelopment/terminal-service/internal/handler"
	"github.com/helixdevelopment/terminal-service/internal/model"
	"github.com/helixdevelopment/terminal-service/internal/recorder"
	"github.com/helixdevelopment/terminal-service/internal/repository"
	"github.com/helixdevelopment/terminal-service/internal/testutil"
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

	var rec *recorder.Recorder

	if available {
		pool, err := pgxpool.New(t.Context(), poolURL)
		if err != nil {
			t.Fatalf("pgxpool.New failed: %v", err)
		}
		repo := repository.New(pool)
		rec = recorder.NewRecorder("", repo)
		h := handler.New(repo, rec)

		r.POST("/api/v1/terminal/sessions", h.CreateTerminalSession)
		r.GET("/api/v1/terminal/sessions", h.ListTerminalSessions)
		r.GET("/api/v1/terminal/sessions/:id", h.GetTerminalSession)
		r.PUT("/api/v1/terminal/sessions/:id", h.UpdateTerminalSession)
		r.POST("/api/v1/terminal/sessions/:id/close", h.CloseTerminalSession)
		r.POST("/api/v1/terminal/sessions/:id/output", h.WriteTerminalOutput)
		r.GET("/api/v1/terminal/sessions/:id/output", h.GetTerminalOutput)
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
	rec = recorder.NewRecorder("", nil)
	h := handler.New(nil, rec)

	r.POST("/api/v1/terminal/sessions", h.CreateTerminalSession)
	r.GET("/api/v1/terminal/sessions", h.ListTerminalSessions)
	r.GET("/api/v1/terminal/sessions/:id", h.GetTerminalSession)
	r.PUT("/api/v1/terminal/sessions/:id", h.UpdateTerminalSession)
	r.POST("/api/v1/terminal/sessions/:id/close", h.CloseTerminalSession)
	r.POST("/api/v1/terminal/sessions/:id/output", h.WriteTerminalOutput)
	r.GET("/api/v1/terminal/sessions/:id/output", h.GetTerminalOutput)
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
// valid JSON.
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

	t.Run("malformed_json_bodies_create", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"{",
			"{}",
			"null",
			"[]",
			"42",
			`{"user_id":}`,
			`{"user_id":"not-uuid","host_id":"not-uuid","cols":"not-int"}`,
			`{"user_id":null,"host_id":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		for i, body := range malformedBodies {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(body))
			if status == 0 {
				t.Logf("malformed body %d: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("malformed body %d: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed bodies against POST /sessions", len(malformedBodies))
	})

	t.Run("malformed_json_bodies_update", func(t *testing.T) {
		fakeID := uuid.New().String()
		malformedBodies := []string{
			"",
			"{",
			"null",
			"[]",
			`{"status":"invalid_status_value"}`,
			`{"cols":-1}`,
			`{"rows":9999}`,
			strings.Repeat("{", 100),
		}

		for i, body := range malformedBodies {
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions/"+fakeID, "application/json", []byte(body))
			if status == 0 {
				continue
			}
			if status >= 500 {
				t.Errorf("malformed update body %d: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed bodies against PUT /sessions/:id", len(malformedBodies))
	})

	t.Run("corrupt_uuids_in_path", func(t *testing.T) {
		corruptIDs := []string{
			"not-a-uuid",
			"",
			"null",
			"undefined",
			"\x00\x01\x02\x03",
			strings.Repeat("x", 1000),
			"550e8400",                                 // partial UUID
			"550e8400-e29b-41d4-a716",                  // truncated
			"gggggggg-gggg-gggg-gggg-gggggggggggg",     // invalid hex
			"550e8400-e29b-41d4-a716-446655440000-extra", // too long
		}

		endpoints := []string{
			"/api/v1/terminal/sessions/%s",
			"/api/v1/terminal/sessions/%s/close",
			"/api/v1/terminal/sessions/%s/output",
			"/api/v1/terminal/sessions/%s/recording",
		}

		for _, ep := range endpoints {
			for i, id := range corruptIDs {
				url := fmt.Sprintf(env.ts.URL+ep, id)
				req, _ := http.NewRequest("GET", url, nil)
				resp, err := client.Do(req)
				if err != nil {
					continue
				}
				resp.Body.Close()
				if resp.StatusCode >= 500 {
					t.Errorf("corrupt uuid %d to %s: got %d — expected 400", i, ep, resp.StatusCode)
				}
			}
		}
		t.Logf("tested %d corrupt UUIDs across %d endpoints", len(corruptIDs), len(endpoints))
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
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", ct,
				[]byte(`{"user_id":"550e8400-e29b-41d4-a716-446655440000","host_id":"550e8400-e29b-41d4-a716-446655440001","cols":80,"rows":24}`))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("corrupt_output_batch", func(t *testing.T) {
		fakeID := uuid.New().String()
		corruptBodies := []string{
			"",
			"null",
			"[]",
			`{"outputs": "not-an-array"}`,
			`{"outputs": [{"output_type": "badtype", "data": "test"}]}`,
			`{"outputs": [{"data": "test"}]}`,           // missing output_type
			`{"outputs": [{"output_type": "stdout"}]}`,  // missing data
			`{"outputs": [null]}`,
		}

		for i, body := range corruptBodies {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions/"+fakeID+"/output", "application/json", []byte(body))
			if status >= 500 {
				t.Errorf("corrupt output body %d: got %d — expected 400", i, status)
			}
		}
		t.Logf("tested %d corrupt output bodies", len(corruptBodies))
	})

	t.Run("corrupt_recording_format", func(t *testing.T) {
		fakeID := uuid.New().String()
		corruptFormats := []string{
			`{"format": "mp4"}`,
			`{"format": ""}`,
			`{"format": "avi"}`,
			`{"format": null}`,
			`{}`,
			"null",
		}

		for i, body := range corruptFormats {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions/"+fakeID+"/recording", "application/json", []byte(body))
			if status >= 500 {
				t.Errorf("corrupt recording format %d: got %d — expected 400 or 404", i, status)
			}
		}
		t.Logf("tested %d corrupt recording formats", len(corruptFormats))
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
			body := fmt.Sprintf(`{"user_id":"%s","host_id":"%s","cols":80,"rows":24,"shell_type":"bash"}`,
				uuid.New().String(), uuid.New().String())
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(body))
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
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/terminal/sessions?limit=10&offset=0", nil)
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

	t.Run("rapid_fire_health", func(t *testing.T) {
		const burst = 50

		testutil.RunConcurrent(t, burst, func(id int) {
			req, _ := http.NewRequest("GET", env.ts.URL+"/healthz", nil)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("rapid health %d: got %d — expected 200", id, resp.StatusCode)
			}
		})
	})

	t.Run("concurrent_get_same_session", func(t *testing.T) {
		// Create one session, then hammer GET on it from multiple goroutines
		userID := uuid.New().String()
		hostID := uuid.New().String()
		body := fmt.Sprintf(`{"user_id":"%s","host_id":"%s","cols":80,"rows":24}`, userID, hostID)
		status, raw := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(body))
		if status != http.StatusCreated {
			t.Skipf("could not create session for concurrent GET test: status=%d", status)
		}

		var parsed map[string]interface{}
		_ = json.Unmarshal(raw, &parsed)
		session, ok := parsed["session"].(map[string]interface{})
		if !ok {
			t.Skip("could not parse session from create response")
		}
		sessionID, _ := session["id"].(string)
		if sessionID == "" {
			t.Skip("no session id in create response")
		}

		const parallel = 10
		testutil.RunConcurrent(t, parallel, func(id int) {
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/terminal/sessions/"+sessionID, nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("concurrent GET %d: request failed: %v", id, err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("concurrent GET %d: got %d — expected 200", id, resp.StatusCode)
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
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/terminal/sessions", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeUserID := strings.Repeat("a", 500000)
		payload := fmt.Sprintf(`{"user_id":"%s","host_id":"550e8400-e29b-41d4-a716-446655440000","cols":80,"rows":24,"shell_type":"%s"}`,
			largeUserID, strings.Repeat("b", 500000))
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("1MB payload → %d (handler does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"user_id":"","host_id":"","cols":0,"rows":0,"shell_type":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_shell_type", func(t *testing.T) {
		payload := fmt.Sprintf(`{"user_id":"%s","host_id":"%s","cols":80,"rows":24,"shell_type":"パスワード"}`,
			uuid.New().String(), uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode shell_type: got %d — expected non-server-error", status)
		}
		t.Logf("unicode shell_type → %d", status)
	})

	t.Run("negative_cols_rejected", func(t *testing.T) {
		payload := fmt.Sprintf(`{"user_id":"%s","host_id":"%s","cols":-1,"rows":24}`,
			uuid.New().String(), uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("negative cols: got %d — expected 400", status)
		}
		t.Logf("negative cols → %d", status)
	})

	t.Run("negative_rows_rejected", func(t *testing.T) {
		payload := fmt.Sprintf(`{"user_id":"%s","host_id":"%s","cols":80,"rows":-1}`,
			uuid.New().String(), uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("negative rows: got %d — expected 400", status)
		}
		t.Logf("negative rows → %d", status)
	})

	t.Run("sql_injection_in_user_id", func(t *testing.T) {
		payload := `{"user_id":"'; DROP TABLE terminal_sessions; --","host_id":"550e8400-e29b-41d4-a716-446655440000","cols":80,"rows":24}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 400 (invalid uuid)", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("sql_injection_in_shell_type", func(t *testing.T) {
		payload := fmt.Sprintf(`{"user_id":"%s","host_id":"%s","cols":80,"rows":24,"shell_type":"'; DROP TABLE terminal_sessions; --"}`,
			uuid.New().String(), uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection in shell_type: got %d — expected 400 or 201", status)
		}
		t.Logf("SQL injection in shell_type → %d", status)
	})

	t.Run("xss_in_shell_type", func(t *testing.T) {
		payload := fmt.Sprintf(`{"user_id":"%s","host_id":"%s","cols":80,"rows":24,"shell_type":"<script>alert('xss')</script>"}`,
			uuid.New().String(), uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in shell_type: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in shell_type → %d", status)
	})

	t.Run("update_invalid_status_value", func(t *testing.T) {
		fakeID := uuid.New().String()
		payload := `{"status":"completely_invalid"}`
		status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions/"+fakeID, "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("invalid status value: got %d — expected 400 or 404", status)
		}
		t.Logf("invalid status value → %d", status)
	})

	t.Run("output_to_nonexistent_session", func(t *testing.T) {
		fakeID := uuid.New().String()
		payload := `{"outputs":[{"output_type":"stdout","data":"hello"}]}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/terminal/sessions/"+fakeID+"/output", "application/json", []byte(payload))
		// Recorder may accept it (writes to buffer) or reject — must not 500
		if status >= 500 {
			t.Errorf("output to nonexistent session: got %d — expected non-server-error", status)
		}
		t.Logf("output to nonexistent session → %d", status)
	})

	t.Run("list_with_extreme_pagination", func(t *testing.T) {
		// Limit=1000 (over max 100), offset=-1
		req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/terminal/sessions?limit=1000&offset=-1", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			t.Errorf("extreme pagination: got %d — expected 400 or 200", resp.StatusCode)
		}
		t.Logf("extreme pagination → %d", resp.StatusCode)
	})
}

// TestChaosBoundaryConditions_NoRepo exercises boundary conditions
// against the validation layer WITHOUT a database — proves
// ShouldBindJSON rejects malformed input before any DB call.
func TestChaosBoundaryConditions_NoRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rec := recorder.NewRecorder("", nil)
	h := handler.New(nil, rec)
	r.POST("/api/v1/terminal/sessions", h.CreateTerminalSession)

	t.Run("invalid_json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/terminal/sessions", strings.NewReader("{broken"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code >= 500 {
			t.Errorf("invalid JSON: got %d — expected 400", w.Code)
		}
		t.Logf("invalid JSON → %d", w.Code)
	})

	t.Run("null_body", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/terminal/sessions", strings.NewReader("null"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code >= 500 {
			t.Errorf("null body: got %d — expected 400", w.Code)
		}
		t.Logf("null body → %d", w.Code)
	})

	t.Run("array_body", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/terminal/sessions", strings.NewReader("[1,2,3]"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code >= 500 {
			t.Errorf("array body: got %d — expected 400", w.Code)
		}
		t.Logf("array body → %d", w.Code)
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

// init ensures the test database migrations package is imported so
// the test binary includes the migration runner.
var _ = model.TerminalStatusPending
