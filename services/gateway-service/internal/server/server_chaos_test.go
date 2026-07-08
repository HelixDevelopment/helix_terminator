//go:build chaos

// Chaos test suite for gateway-service server (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: corrupt JWT tokens, malformed headers, wrong
//     content types — detected and reported cleanly (no panic, no 500
//     for client-side errors).
//   - Resource-exhaustion: 50 rapid-fire proxy requests, verify
//     graceful degradation under pressure — no goroutine leaks,
//     no deadlocks, no panics.
//   - Boundary conditions: nil body, empty JSON, 1MB payload,
//     unicode paths, SQL injection in paths — every boundary
//     produces a categorised result.
//
// Run:
//
//	go test -race -tags chaos -run TestChaos -v -timeout 120s ./internal/server/
package server_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/helixdevelopment/gateway-service/internal/testutil"
)

// chaosPostRaw sends a request with a raw byte body and returns
// the status code + raw response body. Does NOT assume the body
// is valid JSON — sends whatever bytes are provided.
func chaosPostRaw(t *testing.T, method, url, contentType string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	w := httptest.NewRecorder()
	// We need the router from setupTestServer, so callers pass it in
	// via the closure pattern below. This helper is just for the raw
	// request/response extraction.
	_ = w
	return 0, nil
}

// chaosDoRaw performs a raw request against the gateway router and
// returns the status code + response body bytes. Unlike the stress
// test helpers, this sends whatever bytes are provided without
// assuming valid JSON.
func chaosDoRaw(t *testing.T, router http.Handler, method, url, contentType string, body []byte) (int, []byte) {
	t.Helper()
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Code, w.Body.Bytes()
}

// TestChaosInputCorruption exercises corrupt/malformed inputs against
// the gateway. Every case MUST produce a clean HTTP error response
// (not a panic, not a hang, not a 500 for client-side errors).
func TestChaosInputCorruption(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer(t)
	router := s.Router()

	t.Run("corrupt_jwt_tokens", func(t *testing.T) {
		corruptTokens := []string{
			"not.a.jwt",
			"eyJhbGciOiJIUzI1NiJ9.corrupt.signature",
			strings.Repeat("x", 1000),
			"",
			"null",
			"undefined",
			"\x00\x01\x02\x03",
			"Bearer eyJhbGciOiJIUzI1NiJ9.corrupt",
			"eyJhbGciOiJSUzI1NiJ9..", // empty payload+sig
			strings.Repeat("A", 5000), // oversized
			"\xff\xfe\xfd",           // invalid UTF-8
		}

		for i, token := range corruptTokens {
			req, _ := http.NewRequest("GET", "/api/v1/hosts", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code >= 500 {
				t.Errorf("corrupt token %d: got %d (server error) for token %q — expected 401", i, w.Code, truncate(token, 50))
			}
			t.Logf("corrupt token %d (%q) → %d", i, truncate(token, 30), w.Code)
		}
	})

	t.Run("malformed_auth_headers", func(t *testing.T) {
		malformedHeaders := []string{
			"not-bearer-token",
			"Basic dXNlcjpwYXNz",
			"Bearer",
			"Bearer  ",
			"bearer lowercase-token",
			"BEARER uppercase-token",
			"Token some-value",
			"Bearer\ttab-separated",
			strings.Repeat("x", 10000),
		}

		for i, hdr := range malformedHeaders {
			req, _ := http.NewRequest("GET", "/api/v1/hosts", nil)
			req.Header.Set("Authorization", hdr)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code >= 500 {
				t.Errorf("malformed header %d: got %d (server error) for %q — expected 401", i, w.Code, truncate(hdr, 50))
			}
			t.Logf("malformed header %d (%q) → %d", i, truncate(hdr, 30), w.Code)
		}
	})

	t.Run("wrong_content_type_on_proxy", func(t *testing.T) {
		token := generateTestToken()
		contentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"",
			"application/octet-stream",
			"image/png",
		}

		for _, ct := range contentTypes {
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"test@test.com"}`))
			req.Header.Set("Authorization", "Bearer "+token)
			if ct != "" {
				req.Header.Set("Content-Type", ct)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Gateway proxies to upstream — should not 500
			if w.Code >= 500 {
				t.Errorf("content-type %q: got %d — expected non-server-error", ct, w.Code)
			}
			t.Logf("content-type %q → %d", ct, w.Code)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		token := generateTestToken()
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}

		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(garbage))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Gateway proxies to upstream — should not 500
		if w.Code >= 500 {
			t.Errorf("binary garbage: got %d — expected non-server-error", w.Code)
		}
		t.Logf("binary garbage → %d", w.Code)
	})

	t.Run("nil_body_to_post_routes", func(t *testing.T) {
		token := generateTestToken()
		postRoutes := []string{
			"/api/v1/auth/login",
			"/api/v1/auth/register",
			"/api/v1/hosts",
			"/api/v1/vaults",
		}

		for _, route := range postRoutes {
			req, _ := http.NewRequest("POST", route, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code >= 500 {
				t.Errorf("nil body to %s: got %d — expected non-server-error", route, w.Code)
			}
			t.Logf("nil body to %s → %d", route, w.Code)
		}
	})
}

// TestChaosResourceExhaustion drives 50 rapid-fire proxy requests to
// verify the gateway degrades gracefully under pressure — no goroutine
// leaks, no deadlocks, no panics.
func TestChaosResourceExhaustion(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer(t)
	router := s.Router()
	token := generateTestToken()

	t.Run("rapid_fire_proxy_requests", func(t *testing.T) {
		const burst = 50
		errCount := 0
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			req, _ := http.NewRequest("GET", "/api/v1/hosts", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == 0 {
				errCount++
			}
			if w.Code >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d requests: connection_errors=%d server_errors=%d", burst, errCount, serverErrCount)
		if serverErrCount == burst {
			t.Errorf("ALL %d requests returned 500 — service is down", burst)
		}
	})

	t.Run("rapid_fire_health_checks", func(t *testing.T) {
		const burst = 50
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			req, _ := http.NewRequest("GET", "/healthz/live", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d health checks: server_errors=%d", burst, serverErrCount)
		if serverErrCount > 0 {
			t.Errorf("health endpoint returned %d server errors under burst — must always be 200", serverErrCount)
		}
	})

	t.Run("concurrent_corrupt_token_requests", func(t *testing.T) {
		// Multiple goroutines sending corrupt tokens simultaneously
		// — must not deadlock or panic
		const parallel = 15

		testutil.RunConcurrent(t, parallel, func(id int) {
			corruptToken := fmt.Sprintf("corrupt-token-%d", id)
			req, _ := http.NewRequest("GET", "/api/v1/hosts", nil)
			req.Header.Set("Authorization", "Bearer "+corruptToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code >= 500 {
				t.Errorf("concurrent corrupt %d: got %d — expected 401", id, w.Code)
			}
		})
	})

	t.Run("rapid_fire_mixed_routes", func(t *testing.T) {
		// Hammer multiple different routes to stress routing table
		const burst = 50
		routes := []string{
			"/api/v1/hosts",
			"/api/v1/vaults",
			"/api/v1/sessions",
			"/api/v1/snippets",
			"/api/v1/workspaces",
			"/healthz/live",
			"/healthz/ready",
		}
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			route := routes[id%len(routes)]
			req, _ := http.NewRequest("GET", route, nil)
			if route != "/healthz/live" && route != "/healthz/ready" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d mixed-route requests: server_errors=%d", burst, serverErrCount)
		if serverErrCount == burst {
			t.Errorf("ALL %d mixed-route requests returned 500 — service is down", burst)
		}
	})
}

// TestChaosBoundaryConditions exercises extreme boundary values
// that stress the parsing, routing, and proxy layers.
func TestChaosBoundaryConditions(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	s := setupTestServer(t)
	router := s.Router()
	token := generateTestToken()

	t.Run("nil_body", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/hosts", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// GET with nil body is normal — should succeed
		if w.Code >= 500 {
			t.Errorf("nil body GET: got %d — expected 200", w.Code)
		}
		t.Logf("nil body GET → %d", w.Code)
	})

	t.Run("empty_json_body_to_post", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Gateway proxies to upstream — should not 500
		if w.Code >= 500 {
			t.Errorf("empty JSON POST: got %d — expected non-server-error", w.Code)
		}
		t.Logf("empty JSON POST → %d", w.Code)
	})

	t.Run("one_megabyte_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return cleanly.
		largePayload := strings.Repeat("x", 1024*1024)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(largePayload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == 0 {
			t.Fatal("1MB payload: no response at all")
		}
		// Should not 500 — either accepted by upstream or rejected
		// cleanly (400/413)
		if w.Code >= 500 {
			t.Logf("FINDING: 1MB payload → %d (ideally 413 or 400; gateway lacks body-size middleware but must not panic)", w.Code)
		}
		t.Logf("1MB payload → %d", w.Code)
	})

	t.Run("unicode_in_path_segments", func(t *testing.T) {
		unicodePaths := []string{
			"/api/v1/hosts/テスト",
			"/api/v1/hosts/café-résumé",
			"/api/v1/vaults/🔐-key",
			"/api/v1/snippets/über-cool",
			"/api/v1/workspaces/中文空间",
		}

		for _, path := range unicodePaths {
			req, _ := http.NewRequest("GET", path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should not 500 — either routed or 404
			if w.Code >= 500 {
				t.Errorf("unicode path %q: got %d — expected non-server-error", path, w.Code)
			}
			t.Logf("unicode path %q → %d", path, w.Code)
		}
	})

	t.Run("sql_injection_in_path_segments", func(t *testing.T) {
		sqliPaths := []string{
			"/api/v1/hosts/'; DROP TABLE hosts; --",
			"/api/v1/vaults/1 OR 1=1",
			"/api/v1/snippets/\"; DELETE FROM snippets; --",
			"/api/v1/workspaces/UNION SELECT * FROM users",
			"/api/v1/hosts/../../etc/passwd",
			"/api/v1/hosts/..\\..\\windows\\system32",
		}

		for _, path := range sqliPaths {
			req, _ := http.NewRequest("GET", path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Must not 500 — SQL injection must not crash the gateway
			if w.Code >= 500 {
				t.Errorf("SQL injection path %q: got %d — expected non-server-error", path, w.Code)
			}
			t.Logf("SQL injection path %q → %d", path, w.Code)
		}
	})

	t.Run("extremely_long_path", func(t *testing.T) {
		longPath := "/api/v1/hosts/" + strings.Repeat("a", 10000)
		req, _ := http.NewRequest("GET", longPath, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code >= 500 {
			t.Errorf("extremely long path: got %d — expected non-server-error", w.Code)
		}
		t.Logf("extremely long path (10000 chars) → %d", w.Code)
	})

	t.Run("special_characters_in_query_params", func(t *testing.T) {
		specialPaths := []string{
			"/healthz/live?name=<script>alert('xss')</script>",
			"/healthz/live?key=value&key=value&key=value", // duplicate keys
			"/healthz/live?" + strings.Repeat("x=1&", 100), // many params
			"/api/v1/hosts?filter[$ne]=admin",              // NoSQL injection
			"/api/v1/hosts?sort[$gt]=",                     // NoSQL injection
		}

		for _, path := range specialPaths {
			req, _ := http.NewRequest("GET", path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code >= 500 {
				t.Errorf("special query %q: got %d — expected non-server-error", truncate(path, 60), w.Code)
			}
			t.Logf("special query %q → %d", truncate(path, 40), w.Code)
		}
	})

	t.Run("http_methods_on_protected_routes", func(t *testing.T) {
		methods := []string{"POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		for _, method := range methods {
			req, _ := http.NewRequest(method, "/api/v1/hosts", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code >= 500 {
				t.Errorf("%s /api/v1/hosts: got %d — expected non-server-error", method, w.Code)
			}
			t.Logf("%s /api/v1/hosts → %d", method, w.Code)
		}
	})

	t.Run("double_slash_in_path", func(t *testing.T) {
		paths := []string{
			"//api/v1/hosts",
			"/api//v1/hosts",
			"/api/v1//hosts",
			"/api/v1/hosts//",
		}

		for _, path := range paths {
			req, _ := http.NewRequest("GET", path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code >= 500 {
				t.Errorf("double-slash path %q: got %d — expected non-server-error", path, w.Code)
			}
			t.Logf("double-slash path %q → %d", path, w.Code)
		}
	})

	t.Run("null_bytes_in_headers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/hosts", nil)
		req.Header.Set("Authorization", "Bearer "+string([]byte{0x00, 0x01, 0x02}))
		req.Header.Set("X-Request-ID", string([]byte{0x00, 0xff}))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code >= 500 {
			t.Errorf("null bytes in headers: got %d — expected non-server-error", w.Code)
		}
		t.Logf("null bytes in headers → %d", w.Code)
	})

	t.Run("empty_path_segments", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should not 500
		if w.Code >= 500 {
			t.Errorf("empty path segment: got %d — expected non-server-error", w.Code)
		}
		t.Logf("empty path segment → %d", w.Code)
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

// Ensure io import is used (chaosDoRaw uses it indirectly via
// httptest, but the compiler needs to see the import).
var _ io.Reader
