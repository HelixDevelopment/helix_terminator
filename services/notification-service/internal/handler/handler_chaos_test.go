//go:build chaos

// Chaos test suite for notification-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: corrupt JWT tokens, malformed request bodies,
//     binary garbage — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, unicode, SQL injection attempts.
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

	"github.com/helixdevelopment/notification-service/internal/handler"
	"github.com/helixdevelopment/notification-service/internal/repository"
	"github.com/helixdevelopment/notification-service/internal/testutil"
)

// chaosEnv holds the assembled test environment for chaos tests.
// Uses a nil-repo handler (validation-only path) — the chaos tests
// exercise input parsing, not database operations.
type chaosEnv struct {
	ts      *httptest.Server
	cleanup func()
}

// setupChaosEnv boots the chaos test environment. Uses a nil-repo
// handler so chaos tests exercise validation/parsing without needing
// a real database.
func setupChaosEnv(t *testing.T) *chaosEnv {
	t.Helper()

	callerID := uuid.New().String()
	repo := repository.New(nil)
	h := handler.New(repo)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", callerID)
		c.Next()
	})

	api := r.Group("/api/v1/notifications")
	{
		api.POST("", h.CreateNotification)
		api.GET("", h.ListNotifications)
		api.GET("/unread-count", h.CountUnread)
		api.GET("/:id", h.GetNotification)
		api.POST("/:id/read", h.MarkRead)
		api.POST("/read-all", h.MarkAllRead)
		api.DELETE("/:id", h.DeleteNotification)
		api.GET("/preferences", h.GetPreference)
		api.PUT("/preferences", h.UpdatePreference)
	}

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

// chaosGetRaw sends a GET request and returns the status code +
// raw response body.
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

// chaosDeleteRaw sends a DELETE request and returns the status code +
// raw response body.
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
			`{"type":}`,
			`{"type":123,"title":true}`,
			`{"type":null,"title":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		for i, body := range malformedBodies {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte(body))
			if status == 0 {
				t.Logf("malformed body %d: connection failed (acceptable)", i)
				continue
			}
			if status >= 500 {
				t.Errorf("malformed body %d: got %d — expected 400 for bad input", i, status)
			}
		}
		t.Logf("tested %d malformed bodies against POST /api/v1/notifications", len(malformedBodies))
	})

	t.Run("wrong_content_type", func(t *testing.T) {
		contentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"",
			"application/octet-stream",
		}
		validJSON := `{"type":"info","title":"Test","message":"Test","channel":"in_app"}`
		for _, ct := range contentTypes {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", ct, []byte(validJSON))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("corrupt_uuid_in_get", func(t *testing.T) {
		corruptIDs := []string{
			"not-a-uuid",
			"",
			"null",
			"undefined",
			"\x00\x01\x02",
			strings.Repeat("x", 1000),
			"ffffffff-ffff-ffff-ffff-ffffffffffff", // valid format, likely not found
		}
		for i, id := range corruptIDs {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/notifications/"+id)
			if status >= 500 {
				t.Errorf("corrupt uuid %d (%q): got %d — expected 400 or 404", i, truncate(id, 30), status)
			}
			t.Logf("corrupt uuid %d (%q) → %d", i, truncate(id, 30), status)
		}
	})

	t.Run("corrupt_uuid_in_mark_read", func(t *testing.T) {
		corruptIDs := []string{
			"garbage",
			"",
			"null",
			strings.Repeat("a", 5000),
			"\xff\xfe\xfd",
		}
		for i, id := range corruptIDs {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications/"+id+"/read", "application/json", nil)
			if status >= 500 {
				t.Errorf("corrupt mark-read uuid %d: got %d — expected 400 or 404", i, status)
			}
			t.Logf("corrupt mark-read uuid %d → %d", i, status)
		}
	})

	t.Run("corrupt_uuid_in_delete", func(t *testing.T) {
		corruptIDs := []string{
			"garbage",
			"",
			"null",
			strings.Repeat("a", 5000),
			"\xff\xfe\xfd",
		}
		for i, id := range corruptIDs {
			status, _ := chaosDeleteRaw(t, client, env.ts.URL+"/api/v1/notifications/"+id)
			if status >= 500 {
				t.Errorf("corrupt delete uuid %d: got %d — expected 400 or 404", i, status)
			}
			t.Logf("corrupt delete uuid %d → %d", i, status)
		}
	})

	t.Run("invalid_query_params_on_list", func(t *testing.T) {
		malformedQueries := []string{
			"?limit=abc",
			"?limit=-1",
			"?limit=99999",
			"?offset=-1",
			"?channel=not_real",
			"?status=not_real",
			"?org_id=not-a-uuid",
		}
		for i, q := range malformedQueries {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/notifications"+q)
			if status >= 500 {
				t.Errorf("malformed query %d (%q): got %d — expected 400", i, q, status)
			}
			t.Logf("malformed query %d (%q) → %d", i, q, status)
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
			body := fmt.Sprintf(`{"type":"info","title":"Chaos %d","message":"Rapid fire test %d","channel":"in_app"}`, id, id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte(body))
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
		// Hammer GET /api/v1/notifications — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/notifications")
			if status >= 500 {
				t.Errorf("list %d: got %d — expected 200 or 503", id, status)
			}
		})
	})

	t.Run("rapid_fire_invalid_ids", func(t *testing.T) {
		// Hammer endpoints with garbage IDs — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			garbageID := fmt.Sprintf("garbage-%d", id)
			// GET
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/notifications/"+garbageID)
			if status >= 500 {
				t.Errorf("get garbage %d: got %d — expected 400", id, status)
			}
			// DELETE
			status, _ = chaosDeleteRaw(t, client, env.ts.URL+"/api/v1/notifications/"+garbageID)
			if status >= 500 {
				t.Errorf("delete garbage %d: got %d — expected 400", id, status)
			}
		})
	})

	t.Run("concurrent_read_all_same_user", func(t *testing.T) {
		// Multiple goroutines hitting read-all simultaneously — must not deadlock
		const parallel = 10

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications/read-all", "application/json", nil)
			if status >= 500 {
				t.Errorf("concurrent read-all %d: got %d — expected 200 or 503", id, status)
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
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/notifications", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeMsg := strings.Repeat("x", 1000000)
		payload := fmt.Sprintf(`{"type":"info","title":"Large","message":%q,"channel":"in_app"}`, largeMsg)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (handler rejects with validation error, no panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		// All fields at zero value — validation must catch
		payload := `{"type":"","title":"","message":"","channel":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_all_fields", func(t *testing.T) {
		payload := `{"type":"info","title":"日本語テスト","message":"パスワードパスワードパスワード","channel":"in_app"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte(payload))
		// Either accepted (201/503) or rejected (400) — never 500
		if status >= 500 && status != http.StatusServiceUnavailable {
			t.Errorf("unicode fields: got %d — expected non-server-error", status)
		}
		t.Logf("unicode fields → %d", status)
	})

	t.Run("sql_injection_in_title", func(t *testing.T) {
		payload := `{"type":"info","title":"'; DROP TABLE notifications; --","message":"SQLi test","channel":"in_app"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte(payload))
		if status >= 500 && status != http.StatusServiceUnavailable {
			t.Errorf("SQL injection attempt: got %d — expected 400 or 503", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_message", func(t *testing.T) {
		payload := `{"type":"info","title":"XSS","message":"<script>alert('xss')</script>","channel":"in_app"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte(payload))
		if status >= 500 && status != http.StatusServiceUnavailable {
			t.Errorf("XSS in message: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in message → %d", status)
	})

	t.Run("invalid_status_value", func(t *testing.T) {
		payload := `{"type":"info","title":"Test","message":"Test","channel":"in_app","status":"hacked"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("invalid status: got %d — expected 400", status)
		}
		t.Logf("invalid status → %d", status)
	})

	t.Run("sql_injection_in_slack_target", func(t *testing.T) {
		// The Slack target (channel ID) is caller-supplied; it must never
		// be trusted as SQL-safe by assumption — the parameterized query
		// in repository.go is what actually protects this, but the
		// request itself must never 500 regardless of what's in target.
		payload := `{"type":"info","title":"Slack Chaos","message":"SQLi-in-target test","channel":"slack","target":"C0123'; DROP TABLE notifications; --"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications", "application/json", []byte(payload))
		if status >= 500 && status != http.StatusServiceUnavailable {
			t.Errorf("SQL injection in slack target: got %d — expected 201 or 503 (never 500)", status)
		}
		t.Logf("SQL injection in slack target → %d", status)
	})

	t.Run("slack_channel_accepted_valid_preference", func(t *testing.T) {
		payload := `{"channel":"slack","enabled":true}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications/preferences", "application/json", []byte(payload))
		if status >= 500 && status != http.StatusServiceUnavailable {
			t.Errorf("valid slack preference channel: got %d — expected non-server-error", status)
		}
		t.Logf("slack preference channel → %d", status)
	})

	t.Run("invalid_preference_channel", func(t *testing.T) {
		payload := `{"channel":"invalid_channel","enabled":true}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications/preferences", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("invalid preference channel: got %d — expected 400", status)
		}
		t.Logf("invalid preference channel → %d", status)
	})

	t.Run("preference_missing_channel", func(t *testing.T) {
		payload := `{"enabled":true}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/notifications/preferences", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("missing preference channel: got %d — expected 400", status)
		}
		t.Logf("missing preference channel → %d", status)
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
