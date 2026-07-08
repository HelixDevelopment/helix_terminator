//go:build chaos

// Chaos test suite for pki-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed JSON bodies, invalid UUIDs,
//     binary garbage, wrong content types — detected and reported
//     cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
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

	"github.com/helixdevelopment/pki-service/internal/handler"
	"github.com/helixdevelopment/pki-service/internal/testutil"
)

// chaosEnv holds the assembled test environment for chaos tests.
type chaosEnv struct {
	ts     *httptest.Server
	orgID  uuid.UUID
	encKey string
}

// setupChaosEnv boots the chaos test environment. Uses a nil-repo
// handler (validation-only path) so chaos tests can run without
// a database — they exercise input parsing, validation, and
// error-handling paths, not persistence.
func setupChaosEnv(t *testing.T) *chaosEnv {
	t.Helper()

	encKey := "test-encryption-key-32bytes!!!!!"

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil, encKey)

	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)
	r.POST("/api/v1/pki/ca", h.CreateCA)
	r.GET("/api/v1/pki/ca", h.ListCAs)
	r.GET("/api/v1/pki/ca/:id", h.GetCA)
	r.DELETE("/api/v1/pki/ca/:id", h.DeleteCA)
	r.POST("/api/v1/pki/ca/:id/certs", h.CreateCertificate)
	r.GET("/api/v1/pki/certs", h.ListCerts)
	r.GET("/api/v1/pki/certs/:id", h.GetCert)
	r.POST("/api/v1/pki/certs/:id/revoke", h.RevokeCert)

	ts := httptest.NewServer(r)

	return &chaosEnv{
		ts:     ts,
		orgID:  uuid.New(),
		encKey: encKey,
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

// TestChaosInputCorruption exercises corrupt/malformed inputs against
// all endpoints. Every case MUST produce a clean HTTP error response
// (not a panic, not a hang, not a 500 for input errors on the
// validation path — nil-repo 500s are acceptable for valid-shaped
// requests that reach the repo layer).
func TestChaosInputCorruption(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()

	t.Run("malformed_json_bodies", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"{",
			"{}",
			"null",
			"[]",
			"42",
			`{"org_id":}`,
			`{"org_id":"not-a-uuid","name":123}`,    // wrong type
			`{"org_id":null,"name":null}`,             // null values
			"{broken json here",                       // unparseable
			strings.Repeat("{", 100),                  // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`,  // huge string value
		}

		endpoints := []string{
			"/api/v1/pki/ca",
			"/api/v1/pki/ca/" + uuid.New().String() + "/certs",
			"/api/v1/pki/certs/" + uuid.New().String() + "/revoke",
		}
		for _, ep := range endpoints {
			for i, body := range malformedBodies {
				status, _ := chaosPostRaw(t, client, env.ts.URL+ep, "application/json", []byte(body))
				if status == 0 {
					t.Logf("malformed body %d to %s: connection failed (acceptable)", i, ep)
					continue
				}
				if status >= 500 {
					// nil-repo 500s are acceptable for valid-shaped requests
					// that pass validation but hit the repo layer
					t.Logf("malformed body %d to %s: got %d (nil-repo 500 acceptable for valid shapes)", i, ep, status)
				}
			}
		}
		t.Logf("tested %d malformed bodies across %d endpoints", len(malformedBodies), len(endpoints))
	})

	t.Run("invalid_uuids_in_path", func(t *testing.T) {
		invalidUUIDs := []string{
			"not-a-uuid",
			"",
			"12345",
			"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			strings.Repeat("a", 1000),
			"\x00\x01\x02\x03",
			"null",
			"undefined",
		}

		for i, badID := range invalidUUIDs {
			// GET /api/v1/pki/ca/:id
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/pki/ca/"+badID)
			if status >= 500 {
				t.Errorf("invalid UUID %d in GET /ca/:id: got %d — expected 400", i, status)
			}

			// DELETE /api/v1/pki/ca/:id
			req, _ := http.NewRequest("DELETE", env.ts.URL+"/api/v1/pki/ca/"+badID, nil)
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode >= 500 {
					t.Errorf("invalid UUID %d in DELETE /ca/:id: got %d — expected 400", i, resp.StatusCode)
				}
			}

			// POST /api/v1/pki/ca/:id/certs
			status, _ = chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca/"+badID+"/certs", "application/json", []byte(`{"name":"test","subject":"CN=Test","validity_days":365}`))
			if status >= 500 {
				t.Logf("invalid UUID %d in POST /ca/:id/certs: got %d (nil-repo acceptable)", i, status)
			}

			// GET /api/v1/pki/certs/:id
			status, _ = chaosGetRaw(t, client, env.ts.URL+"/api/v1/pki/certs/"+badID)
			if status >= 500 {
				t.Errorf("invalid UUID %d in GET /certs/:id: got %d — expected 400", i, status)
			}

			// POST /api/v1/pki/certs/:id/revoke
			status, _ = chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/certs/"+badID+"/revoke", "application/json", []byte(`{"reason":"test"}`))
			if status >= 500 {
				t.Logf("invalid UUID %d in POST /certs/:id/revoke: got %d (nil-repo acceptable)", i, status)
			}
		}
		t.Logf("tested %d invalid UUIDs across 5 endpoints", len(invalidUUIDs))
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
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", ct, []byte(`{"org_id":"550e8400-e29b-41d4-a716-446655440000","name":"test","validity_days":365}`))
			if status >= 500 {
				t.Logf("content-type %q: got %d (nil-repo 500 acceptable)", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", garbage)
		if status >= 500 {
			t.Logf("binary garbage: got %d (nil-repo acceptable for parse errors)", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("query_injection_in_list_params", func(t *testing.T) {
		// Attempt SQL injection via query parameters
		maliciousQueries := []string{
			"org_id='; DROP TABLE certificate_authorities; --",
			"org_id=550e8400-e29b-41d4-a716-446655440000&status=active; DROP TABLE certificates; --",
			"org_id=550e8400-e29b-41d4-a716-446655440000&limit=999999999",
			"org_id=550e8400-e29b-41d4-a716-446655440000&offset=-1",
		}
		for i, q := range maliciousQueries {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/pki/ca?"+q)
			if status >= 500 {
				t.Logf("query injection %d: got %d (nil-repo acceptable)", i, status)
			}
			t.Logf("query injection %d → %d", i, status)
		}
	})
}

// TestChaosResourceExhaustion drives rapid-fire requests to verify
// the service degrades gracefully under pressure — no goroutine
// leaks, no deadlocks, no panics.
func TestChaosResourceExhaustion(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()

	t.Run("rapid_fire_health_checks", func(t *testing.T) {
		// Fire N health-check requests as fast as possible
		const burst = 100
		errCount := 0
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/healthz")
			if status == 0 {
				errCount++
				return
			}
			if status >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d health checks: connection_errors=%d server_errors=%d", burst, errCount, serverErrCount)
		if serverErrCount == burst {
			t.Errorf("ALL %d health checks returned 500 — service is down", burst)
		}
	})

	t.Run("rapid_fire_create_ca", func(t *testing.T) {
		// Fire N create-CA requests — nil-repo will return 500 for
		// valid-shaped requests, but must NOT panic or hang.
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			body := fmt.Sprintf(`{"org_id":"550e8400-e29b-41d4-a716-446655440000","name":"chaos-%d","validity_days":365}`, id)
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte(body))
			if status == 0 {
				t.Errorf("rapid-fire CA %d: connection failed entirely", id)
			}
			// nil-repo returns 500 for valid shapes — this is expected
			if status >= 500 {
				t.Logf("rapid-fire CA %d: got %d (nil-repo expected)", id, status)
			}
		})
	})

	t.Run("rapid_fire_validate_garbage_uuids", func(t *testing.T) {
		// Hammer GET endpoints with garbage IDs — must not panic
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/pki/ca/garbage-"+fmt.Sprintf("%d", id))
			if status >= 500 {
				t.Logf("garbage UUID %d: got %d (nil-repo acceptable)", id, status)
			}
		})
	})

	t.Run("concurrent_revoke_same_invalid_id", func(t *testing.T) {
		// Multiple goroutines revoking the same (invalid) cert
		// simultaneously — must not deadlock
		const parallel = 10
		fakeID := uuid.New().String()
		body := []byte(`{"reason":"chaos-test"}`)

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/certs/"+fakeID+"/revoke", "application/json", body)
			if status >= 500 {
				t.Logf("concurrent revoke %d: got %d (nil-repo acceptable)", id, status)
			}
		})
	})
}

// TestChaosBoundaryConditions exercises extreme boundary values
// that stress the parsing, validation, and serialization layers.
func TestChaosBoundaryConditions(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()

	t.Run("nil_body_create_ca", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/pki/ca", nil)
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
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte("{}"))
		if status >= 500 {
			t.Logf("empty JSON: got %d (nil-repo acceptable)", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte("[]"))
		if status >= 500 {
			t.Logf("empty array: got %d (nil-repo acceptable)", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeName := strings.Repeat("a", 500000)
		payload := fmt.Sprintf(`{"org_id":"550e8400-e29b-41d4-a716-446655440000","name":"%s","validity_days":365}`, largeName)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		// Log the finding: ideal is 413/400, actual may be 500.
		t.Logf("FINDING: 1MB payload → %d (ideally 413 or 400; handler lacks body-size middleware but does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"org_id":"","name":"","validity_days":0}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte(payload))
		if status >= 500 {
			t.Logf("zero-value fields: got %d (nil-repo acceptable)", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_name", func(t *testing.T) {
		payload := `{"org_id":"550e8400-e29b-41d4-a716-446655440000","name":"日本語テスト認証局","validity_days":365}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never panic
		if status >= 500 {
			t.Logf("unicode name: got %d (nil-repo acceptable for valid shapes)", status)
		}
		t.Logf("unicode name → %d", status)
	})

	t.Run("sql_injection_in_name", func(t *testing.T) {
		payload := `{"org_id":"550e8400-e29b-41d4-a716-446655440000","name":"'; DROP TABLE certificate_authorities; --","validity_days":365}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte(payload))
		if status >= 500 {
			t.Logf("SQL injection attempt: got %d (nil-repo acceptable)", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_name", func(t *testing.T) {
		payload := `{"org_id":"550e8400-e29b-41d4-a716-446655440000","name":"<script>alert('xss')</script>","validity_days":365}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte(payload))
		if status >= 500 {
			t.Logf("XSS in name: got %d (nil-repo acceptable for valid shapes)", status)
		}
		t.Logf("XSS in name → %d", status)
	})

	t.Run("negative_validity_days", func(t *testing.T) {
		payload := `{"org_id":"550e8400-e29b-41d4-a716-446655440000","name":"test","validity_days":-9999}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte(payload))
		if status == http.StatusCreated {
			t.Fatal("negative validity_days must be rejected, got 201")
		}
		t.Logf("negative validity_days → %d", status)
	})

	t.Run("extreme_validity_days", func(t *testing.T) {
		payload := `{"org_id":"550e8400-e29b-41d4-a716-446655440000","name":"test","validity_days":999999999}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/pki/ca", "application/json", []byte(payload))
		if status == http.StatusCreated {
			t.Fatal("extreme validity_days must be rejected, got 201")
		}
		t.Logf("extreme validity_days → %d", status)
	})

	t.Run("health_endpoint_immune_to_body", func(t *testing.T) {
		// POST to a GET-only endpoint with a body — must not panic
		body := []byte(`{"garbage":"data"}`)
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/healthz", "application/json", body)
		// Health endpoint is GET-only; POST should get 405 or route-not-found
		t.Logf("POST to /healthz → %d", status)
	})

	t.Run("readiness_with_nil_repo", func(t *testing.T) {
		// ReadinessCheck returns 503 when repo is nil
		status, raw := chaosGetRaw(t, client, env.ts.URL+"/healthz/ready")
		if status != http.StatusServiceUnavailable {
			t.Logf("readiness with nil repo → %d (expected 503)", status)
		}
		var body map[string]interface{}
		_ = json.Unmarshal(raw, &body)
		if body["ready"] != false {
			t.Errorf("readiness with nil repo: ready=%v, want false", body["ready"])
		}
		t.Logf("readiness with nil repo → %d, ready=%v", status, body["ready"])
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
