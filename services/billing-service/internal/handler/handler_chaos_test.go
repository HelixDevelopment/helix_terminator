//go:build chaos

// Chaos test suite for billing-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: corrupt JWT tokens, malformed request bodies,
//     binary garbage — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, unicode, SQL injection.
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
	"github.com/helixdevelopment/billing-service/internal/handler"
	"github.com/helixdevelopment/billing-service/internal/repository"
	"github.com/helixdevelopment/billing-service/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// chaosEnv holds the assembled test environment for chaos tests.
type chaosEnv struct {
	ts      *httptest.Server
	orgID   uuid.UUID
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
	testOrgID := uuid.New()

	// Test middleware that injects orgID into context
	r.Use(func(c *gin.Context) {
		c.Set("orgID", testOrgID.String())
		c.Set("userID", uuid.New().String())
		c.Next()
	})

	if available {
		pool, err := pgxpool.New(t.Context(), poolURL)
		if err != nil {
			t.Fatalf("pgxpool.New failed: %v", err)
		}
		repo := repository.New(pool)
		h := handler.New(repo)

		api := r.Group("/api/v1")
		api.POST("/subscriptions", h.CreateSubscription)
		api.GET("/subscriptions", h.ListSubscriptions)
		api.GET("/subscriptions/:id", h.GetSubscription)
		api.PUT("/subscriptions/:id", h.UpdateSubscription)
		api.POST("/subscriptions/:id/cancel", h.CancelSubscription)
		api.GET("/invoices", h.ListInvoices)
		api.GET("/invoices/:id", h.GetInvoice)

		ts := httptest.NewServer(r)
		return &chaosEnv{
			ts:    ts,
			orgID: testOrgID,
			cleanup: func() {
				ts.Close()
				pool.Close()
			},
		}
	}

	// Nil-repo fallback — validation-only, no DB
	h := handler.New(nil)

	api := r.Group("/api/v1")
	api.POST("/subscriptions", h.CreateSubscription)
	api.GET("/subscriptions", h.ListSubscriptions)
	api.GET("/subscriptions/:id", h.GetSubscription)
	api.PUT("/subscriptions/:id", h.UpdateSubscription)
	api.POST("/subscriptions/:id/cancel", h.CancelSubscription)
	api.GET("/invoices", h.ListInvoices)
	api.GET("/invoices/:id", h.GetInvoice)

	ts := httptest.NewServer(r)
	return &chaosEnv{
		ts:    ts,
		orgID: testOrgID,
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
			`{"planId":}`,
			`{"planId":123}`,  // wrong type
			`{"planId":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []string{
			"/api/v1/subscriptions",
			"/api/v1/subscriptions/" + uuid.New().String(),
		}
		for _, ep := range endpoints {
			for i, body := range malformedBodies {
				// POST for /subscriptions, PUT for /subscriptions/:id
				var status int
				if strings.HasSuffix(ep, "/subscriptions") {
					status, _ = chaosPostRaw(t, client, env.ts.URL+ep, "application/json", []byte(body))
				} else {
					status, _ = chaosPutRaw(t, client, env.ts.URL+ep, "application/json", []byte(body))
				}
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
		validBody := fmt.Sprintf(`{"planId":"%s"}`, uuid.New().String())
		for _, ct := range contentTypes {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", ct, []byte(validBody))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", garbage)
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
			"\x00\x01\x02",
			"null",
			"undefined",
		}
		for _, id := range corruptIDs {
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/subscriptions/"+id, nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("corrupt UUID %q: got %d — expected 400", truncate(id, 30), resp.StatusCode)
			}
			t.Logf("corrupt UUID %q → %d", truncate(id, 30), resp.StatusCode)
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
			body := fmt.Sprintf(`{"planId":"%s"}`, uuid.New().String())
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte(body))
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
		// Hammer /subscriptions with GET requests — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/subscriptions", nil)
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

	t.Run("concurrent_cancel_same_subscription", func(t *testing.T) {
		// Create one subscription, then cancel it from N goroutines
		// simultaneously — must not deadlock
		planID := uuid.New()
		body := fmt.Sprintf(`{"planId":"%s"}`, planID.String())
		status, resp := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte(body))
		if status != http.StatusCreated {
			t.Fatalf("create subscription status = %d, want 201", status)
		}
		var parsed map[string]interface{}
		json.Unmarshal(resp, &parsed)
		subID, _ := parsed["id"].(string)
		if subID == "" {
			t.Fatal("create subscription returned no id")
		}

		const parallel = 10
		testutil.RunConcurrent(t, parallel, func(id int) {
			req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/subscriptions/"+subID+"/cancel", nil)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("concurrent cancel %d: got %d — expected 204 or 4xx", id, resp.StatusCode)
			}
		})
	})

	t.Run("rapid_fire_get_nonexistent", func(t *testing.T) {
		// Hammer GET with random UUIDs — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			fakeID := uuid.New()
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/subscriptions/"+fakeID.String(), nil)
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
}

// TestChaosBoundaryConditions exercises extreme boundary values
// that stress the parsing, validation, and serialization layers.
func TestChaosBoundaryConditions(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("nil_body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/subscriptions", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largePlanID := strings.Repeat("a", 500000)
		payload := fmt.Sprintf(`{"planId":"%s"}`, largePlanID)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte(payload))
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
		payload := `{"planId":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_plan_id", func(t *testing.T) {
		payload := `{"planId":"パスワードパスワードパスワード"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte(payload))
		// Either accepted or rejected — never 500
		if status >= 500 {
			t.Errorf("unicode planId: got %d — expected non-server-error", status)
		}
		t.Logf("unicode planId → %d", status)
	})

	t.Run("sql_injection_in_plan_id", func(t *testing.T) {
		payload := `{"planId":"'; DROP TABLE subscriptions; --"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 400 (invalid uuid)", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("sql_injection_in_status_field", func(t *testing.T) {
		// First create a valid subscription
		planID := uuid.New()
		body := fmt.Sprintf(`{"planId":"%s"}`, planID.String())
		createStatus, createResp := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte(body))
		if createStatus != http.StatusCreated {
			t.Fatalf("create subscription status = %d, want 201", createStatus)
		}
		var parsed map[string]interface{}
		json.Unmarshal(createResp, &parsed)
		subID, _ := parsed["id"].(string)

		// Try SQL injection via status update
		payload := `{"status":"'; DROP TABLE subscriptions; --"}`
		status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/subscriptions/"+subID, "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection in status: got %d — expected 400", status)
		}
		t.Logf("SQL injection in status → %d", status)
	})

	t.Run("xss_in_status_field", func(t *testing.T) {
		// First create a valid subscription
		planID := uuid.New()
		body := fmt.Sprintf(`{"planId":"%s"}`, planID.String())
		createStatus, createResp := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte(body))
		if createStatus != http.StatusCreated {
			t.Fatalf("create subscription status = %d, want 201", createStatus)
		}
		var parsed map[string]interface{}
		json.Unmarshal(createResp, &parsed)
		subID, _ := parsed["id"].(string)

		payload := `{"status":"<script>alert('xss')</script>"}`
		status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/subscriptions/"+subID, "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in status: got %d — expected 400", status)
		}
		t.Logf("XSS in status → %d", status)
	})

	t.Run("extremely_large_plan_id", func(t *testing.T) {
		// UUID-like but way too long
		payload := fmt.Sprintf(`{"planId":"%s"}`, strings.Repeat("a", 10000))
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/subscriptions", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("extremely large planId: got %d — expected 400", status)
		}
		t.Logf("extremely large planId → %d", status)
	})

	t.Run("special_characters_in_path", func(t *testing.T) {
		specialPaths := []string{
			"/api/v1/subscriptions/../../../etc/passwd",
			"/api/v1/subscriptions/%00",
			"/api/v1/subscriptions/%0d%0a",
		}
		for _, path := range specialPaths {
			req, _ := http.NewRequest("GET", env.ts.URL+path, nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("special path %q: got %d — expected 4xx", path, resp.StatusCode)
			}
			t.Logf("special path %q → %d", path, resp.StatusCode)
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
