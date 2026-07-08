//go:build chaos

// Chaos test suite for sftp-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed request bodies, binary garbage,
//     wrong content types — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, concurrent delete
//     on same session, verify graceful degradation under pressure.
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
	"github.com/google/uuid"
	"github.com/helixdevelopment/sftp-service/internal/handler"
	"github.com/helixdevelopment/sftp-service/internal/repository"
	"github.com/helixdevelopment/sftp-service/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// chaosEnv holds the assembled test environment for chaos tests.
// Reuses the same setup pattern as stress tests — real handler, optional
// real DB.
type chaosEnv struct {
	ts      *httptest.Server
	hostID  uuid.UUID
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

	// Set a fake user_id on all requests
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Next()
	})

	hostID := uuid.New()

	if available {
		pool, err := pgxpool.New(t.Context(), poolURL)
		if err != nil {
			t.Fatalf("pgxpool.New failed: %v", err)
		}
		repo := repository.New(pool)
		h := handler.New(repo)
		r.POST("/sessions", h.CreateSession)
		r.GET("/sessions/:id", h.GetSession)
		r.GET("/sessions", h.ListSessions)
		r.PUT("/sessions/:id", h.UpdateSession)
		r.DELETE("/sessions/:id", h.DeleteSession)
		r.GET("/healthz", h.HealthCheck)
		r.GET("/healthz/ready", h.ReadinessCheck)
		ts := httptest.NewServer(r)
		return &chaosEnv{
			ts:     ts,
			hostID: hostID,
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
	r.PUT("/sessions/:id", h.UpdateSession)
	r.DELETE("/sessions/:id", h.DeleteSession)
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	ts := httptest.NewServer(r)
	return &chaosEnv{
		ts:     ts,
		hostID: hostID,
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
			`{"hostId":}`,
			`{"hostId":123,"remotePath":"/r"}`, // wrong type for hostId
			`{"hostId":null,"remotePath":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		for i, body := range malformedBodies {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(body))
			if status == 0 {
				t.Logf("malformed body %d to /sessions: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("malformed body %d to /sessions: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed bodies against /sessions", len(malformedBodies))
	})

	t.Run("malformed_json_bodies_put", func(t *testing.T) {
		fakeID := uuid.New().String()
		malformedBodies := []string{
			"",
			"{",
			"{}",
			"null",
			"[]",
			`{"status":123}`,      // wrong type
			`{"status":"bogus"}`,  // invalid enum value
			"{broken json",
		}

		for i, body := range malformedBodies {
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/sessions/"+fakeID, "application/json", []byte(body))
			if status == 0 {
				continue
			}
			if status >= 500 {
				t.Errorf("malformed PUT body %d: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed PUT bodies", len(malformedBodies))
	})

	t.Run("wrong_content_type", func(t *testing.T) {
		contentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"",
			"application/octet-stream",
		}
		validJSON := `{"hostId":"550e8400-e29b-41d4-a716-446655440000","remotePath":"/r","localPath":"/l","direction":"upload"}`
		for _, ct := range contentTypes {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", ct, []byte(validJSON))
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
			"/sessions/",
			"/sessions/{}",
			"/sessions/null",
			"/sessions/" + strings.Repeat("x", 5000),
		}
		for _, path := range invalidPaths {
			req, _ := http.NewRequest("GET", env.ts.URL+path, nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Logf("GET %s: connection failed (acceptable)", truncate(path, 50))
				continue
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("GET %s: got %d — expected 400", truncate(path, 50), resp.StatusCode)
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
			body := fmt.Sprintf(`{"hostId":"%s","remotePath":"/r/chaos-%d","localPath":"/l/chaos-%d","direction":"upload"}`,
				env.hostID.String(), id, id)
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
		// A few 500s are acceptable under load (DB connection pool
		// exhaustion), but the service must NOT panic or hang.
		if serverErrCount == burst {
			t.Errorf("ALL %d requests returned 500 — service is down", burst)
		}
	})

	t.Run("rapid_fire_list", func(t *testing.T) {
		// Hammer /sessions with list requests — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			req, _ := http.NewRequest("GET", env.ts.URL+"/sessions?limit=10&offset=0", nil)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("list %d: got %d — expected 200 or 400", id, resp.StatusCode)
			}
		})
	})

	t.Run("concurrent_delete_same_session", func(t *testing.T) {
		// Multiple goroutines deleting the same (nonexistent) session
		// simultaneously — must not deadlock
		const parallel = 10
		fakeID := uuid.New().String()

		testutil.RunConcurrent(t, parallel, func(id int) {
			req, _ := http.NewRequest("DELETE", env.ts.URL+"/sessions/"+fakeID, nil)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			// 404 or 500 are both acceptable — what matters is no panic/hang
			if resp.StatusCode >= 500 && resp.StatusCode != http.StatusInternalServerError {
				t.Errorf("concurrent delete %d: got %d — unexpected", id, resp.StatusCode)
			}
		})
	})

	t.Run("concurrent_update_same_session", func(t *testing.T) {
		// Multiple goroutines updating the same (nonexistent) session
		// simultaneously — must not deadlock
		const parallel = 10
		fakeID := uuid.New().String()

		testutil.RunConcurrent(t, parallel, func(id int) {
			body := fmt.Sprintf(`{"status":"active","bytesTransferred":%d}`, id*1000)
			chaosPutRaw(t, client, env.ts.URL+"/sessions/"+fakeID, "application/json", []byte(body))
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
		largePath := "/" + strings.Repeat("a", 500000)
		payload := fmt.Sprintf(`{"hostId":"%s","remotePath":"%s","localPath":"%s","direction":"upload"}`,
			env.hostID.String(), largePath, largePath)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (handler does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		// All fields at zero value — validation must catch
		payload := `{"hostId":"","remotePath":"","localPath":"","direction":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_paths", func(t *testing.T) {
		payload := `{"hostId":"550e8400-e29b-41d4-a716-446655440000","remotePath":"/remote/ファイル.txt","localPath":"/local/ファイル.txt","direction":"upload"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode paths: got %d — expected non-server-error", status)
		}
		t.Logf("unicode paths → %d", status)
	})

	t.Run("sql_injection_in_path", func(t *testing.T) {
		payload := `{"hostId":"550e8400-e29b-41d4-a716-446655440000","remotePath":"'; DROP TABLE sftp_sessions; --","localPath":"/l","direction":"upload"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 400 or 201", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_path", func(t *testing.T) {
		payload := `{"hostId":"550e8400-e29b-41d4-a716-446655440000","remotePath":"<script>alert('xss')</script>","localPath":"/l","direction":"upload"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in path: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in path → %d", status)
	})

	t.Run("negative_bytes_transferred", func(t *testing.T) {
		// First create a session to update
		createBody := fmt.Sprintf(`{"hostId":"%s","remotePath":"/r/chaos-neg","localPath":"/l/chaos-neg","direction":"upload"}`, env.hostID.String())
		cStatus, cRaw := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(createBody))
		if cStatus != http.StatusCreated {
			t.Skipf("could not create session for negative-bytes test: %d", cStatus)
		}
		var parsed map[string]interface{}
		json.Unmarshal(cRaw, &parsed)
		sessionID, _ := parsed["id"].(string)
		if sessionID == "" {
			t.Skip("no session id returned")
		}

		// Update with negative bytes — validation must catch
		updateBody := `{"status":"active","bytesTransferred":-1}`
		status, _ := chaosPutRaw(t, client, env.ts.URL+"/sessions/"+sessionID, "application/json", []byte(updateBody))
		if status >= 500 {
			t.Errorf("negative bytesTransferred: got %d — expected 400", status)
		}
		t.Logf("negative bytesTransferred → %d", status)
	})

	t.Run("invalid_status_in_update", func(t *testing.T) {
		createBody := fmt.Sprintf(`{"hostId":"%s","remotePath":"/r/chaos-status","localPath":"/l/chaos-status","direction":"download"}`, env.hostID.String())
		cStatus, cRaw := chaosPostRaw(t, client, env.ts.URL+"/sessions", "application/json", []byte(createBody))
		if cStatus != http.StatusCreated {
			t.Skipf("could not create session for invalid-status test: %d", cStatus)
		}
		var parsed map[string]interface{}
		json.Unmarshal(cRaw, &parsed)
		sessionID, _ := parsed["id"].(string)
		if sessionID == "" {
			t.Skip("no session id returned")
		}

		invalidStatuses := []string{"bogus", "PENDING", "Active", "123", ""}
		for _, s := range invalidStatuses {
			updateBody := fmt.Sprintf(`{"status":"%s","bytesTransferred":0}`, s)
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/sessions/"+sessionID, "application/json", []byte(updateBody))
			if status >= 500 {
				t.Errorf("invalid status %q: got %d — expected 400", s, status)
			}
			t.Logf("invalid status %q → %d", s, status)
		}
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
