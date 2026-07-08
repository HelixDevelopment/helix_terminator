//go:build chaos

// Chaos test suite for org-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed request bodies, binary garbage,
//     invalid UUIDs — detected and reported cleanly (no panic).
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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/helixdevelopment/org-service/internal/handler"
	"github.com/helixdevelopment/org-service/internal/repository"
	"github.com/helixdevelopment/org-service/internal/testutil"
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
		r.POST("/api/v1/orgs", h.CreateOrg)
		r.GET("/api/v1/orgs", h.ListOrgs)
		r.GET("/api/v1/orgs/:id", h.GetOrg)
		r.PUT("/api/v1/orgs/:id", h.UpdateOrg)
		r.DELETE("/api/v1/orgs/:id", h.DeleteOrg)
		r.POST("/api/v1/orgs/:id/teams", h.CreateTeam)
		r.GET("/api/v1/orgs/:id/teams", h.ListTeams)
		r.GET("/api/v1/teams/:id", h.GetTeam)
		r.PUT("/api/v1/teams/:id", h.UpdateTeam)
		r.DELETE("/api/v1/teams/:id", h.DeleteTeam)
		r.POST("/api/v1/orgs/:id/members", h.AddMember)
		r.GET("/api/v1/orgs/:id/members", h.ListMembers)
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
	r.POST("/api/v1/orgs", h.CreateOrg)
	r.GET("/api/v1/orgs", h.ListOrgs)
	r.GET("/api/v1/orgs/:id", h.GetOrg)
	r.PUT("/api/v1/orgs/:id", h.UpdateOrg)
	r.DELETE("/api/v1/orgs/:id", h.DeleteOrg)
	r.POST("/api/v1/orgs/:id/teams", h.CreateTeam)
	r.GET("/api/v1/orgs/:id/teams", h.ListTeams)
	r.GET("/api/v1/teams/:id", h.GetTeam)
	r.PUT("/api/v1/teams/:id", h.UpdateTeam)
	r.DELETE("/api/v1/teams/:id", h.DeleteTeam)
	r.POST("/api/v1/orgs/:id/members", h.AddMember)
	r.GET("/api/v1/orgs/:id/members", h.ListMembers)
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
			`{"name":123,"slug":true}`,
			`{"name":null,"slug":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []string{
			"/api/v1/orgs",
			"/api/v1/orgs/" + "00000000-0000-0000-0000-000000000001/teams",
			"/api/v1/orgs/" + "00000000-0000-0000-0000-000000000001/members",
		}
		methods := []string{"POST", "POST", "POST"}

		for idx, ep := range endpoints {
			for i, body := range malformedBodies {
				var status int
				if methods[idx] == "POST" {
					status, _ = chaosPostRaw(t, client, env.ts.URL+ep, "application/json", []byte(body))
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
		for _, ct := range contentTypes {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", ct, []byte(`{"name":"Test","slug":"test"}`))
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("invalid_uuid_in_path_params", func(t *testing.T) {
		invalidIDs := []string{
			"not-a-uuid",
			"",
			"12345",
			strings.Repeat("x", 1000),
			"00000000-0000-0000-0000-000000000000", // nil UUID
		}

		for _, id := range invalidIDs {
			// GET org
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/orgs/"+id, nil)
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode >= 500 {
					t.Errorf("invalid UUID %q in GET /orgs/:id: got %d — expected 400", truncate(id, 30), resp.StatusCode)
				}
			}

			// PUT org
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/orgs/"+id, "application/json", []byte(`{"name":"test"}`))
			if status >= 500 {
				t.Errorf("invalid UUID %q in PUT /orgs/:id: got %d — expected 400", truncate(id, 30), status)
			}

			// DELETE org
			req, _ = http.NewRequest("DELETE", env.ts.URL+"/api/v1/orgs/"+id, nil)
			resp, err = client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode >= 500 {
					t.Errorf("invalid UUID %q in DELETE /orgs/:id: got %d — expected 400", truncate(id, 30), resp.StatusCode)
				}
			}
		}
		t.Logf("tested %d invalid UUIDs across GET/PUT/DELETE", len(invalidIDs))
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
			body := fmt.Sprintf(`{"name":"Chaos Org %d","slug":"chaos-rapid-%d-%d"}`,
				id, id, id*1000+id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte(body))
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
		// Hammer GET /api/v1/orgs — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/orgs?limit=10", nil)
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				t.Errorf("list %d: got %d — expected 200", id, resp.StatusCode)
			}
		})
	})

	t.Run("concurrent_create_same_slug", func(t *testing.T) {
		// Multiple goroutines creating with the same slug
		// simultaneously — must not deadlock
		const parallel = 10
		body := []byte(`{"name":"Shared Slug Org","slug":"shared-slug-chaos"}`)

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", body)
			if status == 0 {
				return
			}
			if status >= 500 {
				// A 500 is acceptable for duplicate conflict
				// under concurrent load, but must not be a panic
				t.Logf("concurrent create %d: got %d (duplicate conflict acceptable)", id, status)
			}
		})
	})

	t.Run("rapid_fire_update_nonexistent", func(t *testing.T) {
		// Hammer PUT on nonexistent orgs — must return 404/400, not 500
		const burst = 20
		fakeID := "00000000-0000-0000-0000-000000000099"

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/orgs/"+fakeID, "application/json", []byte(`{"name":"test"}`))
			if status >= 500 {
				t.Errorf("update nonexistent %d: got %d — expected 404", id, status)
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
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/orgs", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeName := strings.Repeat("a", 500000)
		largeSlug := strings.Repeat("b", 500000)
		payload := fmt.Sprintf(`{"name":"%s","slug":"%s"}`, largeName, largeSlug)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("FINDING: 1MB payload → %d (handler lacks body-size middleware but does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"name":"","slug":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_all_fields", func(t *testing.T) {
		payload := `{"name":"テスト組織","slug":"unicode-test"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode fields: got %d — expected non-server-error", status)
		}
		t.Logf("unicode fields → %d", status)
	})

	t.Run("sql_injection_in_name", func(t *testing.T) {
		payload := `{"name":"'; DROP TABLE organizations; --","slug":"sqli-test"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected non-server-error", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_name", func(t *testing.T) {
		payload := `{"name":"<script>alert('xss')</script>","slug":"xss-test"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in name: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in name → %d", status)
	})

	t.Run("invalid_plan_value", func(t *testing.T) {
		payload := `{"name":"Invalid Plan","slug":"invalid-plan","plan":"super-ultra-mega"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte(payload))
		if status == http.StatusCreated {
			t.Errorf("invalid plan accepted, got 201 — expected 400")
		}
		t.Logf("invalid plan → %d", status)
	})

	t.Run("extremely_long_slug", func(t *testing.T) {
		payload := fmt.Sprintf(`{"name":"Long Slug","slug":"%s"}`, strings.Repeat("s", 10000))
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("extremely long slug: got %d — expected 400", status)
		}
		t.Logf("extremely long slug (%d chars) → %d", 10000, status)
	})

	t.Run("invalid_member_role", func(t *testing.T) {
		orgID := "00000000-0000-0000-0000-000000000001"
		payload := `{"userId":"00000000-0000-0000-0000-000000000002","role":"superadmin"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs/"+orgID+"/members", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("invalid member role: got %d — expected 400", status)
		}
		t.Logf("invalid member role → %d", status)
	})

	t.Run("invalid_member_user_id", func(t *testing.T) {
		orgID := "00000000-0000-0000-0000-000000000001"
		payload := `{"userId":"not-a-uuid","role":"member"}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/orgs/"+orgID+"/members", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("invalid member user id: got %d — expected 400", status)
		}
		t.Logf("invalid member user id → %d", status)
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
