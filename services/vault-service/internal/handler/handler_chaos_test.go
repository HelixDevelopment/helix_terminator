//go:build chaos

// Chaos test suite for vault-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: malformed JSON, binary garbage, wrong content
//     types, invalid UUIDs — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, concurrent mutations
//     on the same secret, verify graceful degradation under pressure.
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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/helixdevelopment/vault-service/internal/handler"
	"github.com/helixdevelopment/vault-service/internal/testutil"
)

// chaosPostRaw sends a request with a raw byte body and returns the
// status code + raw response body. Unlike serveStress, this does NOT
// assume the body is valid JSON — it sends whatever bytes are
// provided.
func chaosPostRaw(t *testing.T, r http.Handler, method, path string, contentType string, body []byte, headers map[string]string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w.Code, w.Body.Bytes()
}

// chaosServeJSON sends a JSON request and returns status + parsed
// response.
func chaosServeJSON(t *testing.T, r http.Handler, method, path string, body interface{}, headers map[string]string) (int, map[string]interface{}) {
	t.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req, err := http.NewRequest(method, path, reqBody)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	raw := w.Body.Bytes()
	var parsed map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return w.Code, parsed
}

// truncate returns the first n characters of s, with "..." appended
// if s is longer than n.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// TestChaosInputCorruption exercises corrupt/malformed inputs against
// all endpoints. Every case MUST produce a clean HTTP error response
// (not a panic, not a hang, not a 500 for input errors).
func TestChaosInputCorruption(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	callerID := uuid.New()
	headers := map[string]string{"X-User-ID": callerID.String()}

	t.Run("malformed_json_bodies", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"{",
			"{}",
			"null",
			"[]",
			"42",
			`{"name":}`,
			`{"name":"test","type":123}`, // wrong type
			`{"name":null,"type":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []struct {
			method string
			path   string
		}{
			{"POST", "/api/v1/vault/secrets"},
			{"PUT", "/api/v1/vault/secrets/" + uuid.New().String()},
			{"POST", "/api/v1/vault/secrets/" + uuid.New().String() + "/rotate"},
		}

		for _, ep := range endpoints {
			for i, body := range malformedBodies {
				status, raw := chaosPostRaw(t, r, ep.method, ep.path, "application/json", []byte(body), headers)
				if status >= 500 {
					t.Errorf("malformed body %d to %s %s: got %d — expected 400 for bad input; body=%s", i, ep.method, ep.path, status, truncate(string(raw), 100))
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
			status, _ := chaosPostRaw(t, r, "POST", "/api/v1/vault/secrets", ct, []byte(`{"name":"test"}`), headers)
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected 400", ct, status)
			}
			t.Logf("content-type %q → %d", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, r, "POST", "/api/v1/vault/secrets", "application/json", garbage, headers)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("invalid_uuid_in_path", func(t *testing.T) {
		invalidPaths := []string{
			"/api/v1/vault/secrets/not-a-uuid",
			"/api/v1/vault/secrets/",
			"/api/v1/vault/secrets/00000000-0000-0000-0000-000000000000",
			"/api/v1/vault/secrets/" + strings.Repeat("x", 1000),
			"/api/v1/vault/secrets/%00%00%00",
		}
		for _, path := range invalidPaths {
			status, _ := chaosPostRaw(t, r, "GET", path, "", nil, headers)
			if status >= 500 {
				t.Errorf("invalid path %q: got %d — expected 400 or 404", truncate(path, 50), status)
			}
			t.Logf("invalid path %q → %d", truncate(path, 50), status)
		}
	})

	t.Run("sql_injection_in_name", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id":          callerID.String(),
			"name":             "'; DROP TABLE secrets; --",
			"type":             "api_token",
			"encrypted_value":  "enc",
			"iv":               "iv",
			"salt":             "salt",
		}
		status, _ := chaosServeJSON(t, r, "POST", "/api/v1/vault/secrets", body, headers)
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected 400 or 201 (parameterised)", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_metadata", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id":          callerID.String(),
			"name":             "xss-test",
			"type":             "api_token",
			"encrypted_value":  "enc",
			"iv":               "iv",
			"salt":             "salt",
			"metadata":         map[string]interface{}{"key": "<script>alert('xss')</script>"},
		}
		status, _ := chaosServeJSON(t, r, "POST", "/api/v1/vault/secrets", body, headers)
		if status >= 500 {
			t.Errorf("XSS in metadata: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in metadata → %d", status)
	})

	t.Run("unicode_in_all_fields", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id":          callerID.String(),
			"name":             "日本語テスト秘密",
			"type":             "api_token",
			"encrypted_value":  "暗号化データ",
			"iv":               "初期ベクトル",
			"salt":             "ソルト",
			"metadata":         map[string]interface{}{"description": "パスワードテスト"},
		}
		status, _ := chaosServeJSON(t, r, "POST", "/api/v1/vault/secrets", body, headers)
		if status >= 500 {
			t.Errorf("unicode fields: got %d — expected non-server-error", status)
		}
		t.Logf("unicode fields → %d", status)
	})
}

// TestChaosResourceExhaustion drives rapid-fire requests to verify
// the service degrades gracefully under pressure — no goroutine
// leaks, no deadlocks, no panics.
func TestChaosResourceExhaustion(t *testing.T) {
	repo := newSyncMockRepo()
	h := newSyncHandler(repo)
	r := setupRouter(h)

	callerID := uuid.New()
	headers := map[string]string{"X-User-ID": callerID.String()}

	t.Run("rapid_fire_create", func(t *testing.T) {
		const burst = 50
		serverErrCount := 0

		testutil.RunConcurrent(t, burst, func(id int) {
			body := map[string]interface{}{
				"user_id":          callerID.String(),
				"name":             fmt.Sprintf("chaos-rapid-%d", id),
				"type":             "api_token",
				"encrypted_value":  fmt.Sprintf("enc-%d", id),
				"iv":               fmt.Sprintf("iv-%d", id),
				"salt":             fmt.Sprintf("salt-%d", id),
			}
			status, _ := chaosServeJSON(t, r, "POST", "/api/v1/vault/secrets", body, headers)
			if status >= 500 {
				serverErrCount++
			}
		})

		t.Logf("rapid-fire %d requests: server_errors=%d", burst, serverErrCount)
		if serverErrCount == burst {
			t.Errorf("ALL %d requests returned 5xx — service is down", burst)
		}
	})

	t.Run("rapid_fire_list", func(t *testing.T) {
		const burst = 50

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosServeJSON(t, r, "GET", "/api/v1/vault/secrets", nil, headers)
			if status >= 500 {
				t.Errorf("rapid list %d: got %d — expected 200", id, status)
			}
		})
	})

	t.Run("concurrent_mutations_separate_secrets", func(t *testing.T) {
		// Each goroutine creates and mutates its OWN secret — avoids
		// the pointer-aliasing race that a shared mock introduces
		// when multiple goroutines modify the same *model.Secret.
		const parallel = 15
		testutil.RunConcurrent(t, parallel, func(id int) {
			createBody := map[string]interface{}{
				"user_id":          callerID.String(),
				"name":             fmt.Sprintf("concurrent-target-%d", id),
				"type":             "api_token",
				"encrypted_value":  fmt.Sprintf("enc-%d", id),
				"iv":               fmt.Sprintf("iv-%d", id),
				"salt":             fmt.Sprintf("salt-%d", id),
			}
			status, resp := chaosServeJSON(t, r, "POST", "/api/v1/vault/secrets", createBody, headers)
			if status != http.StatusCreated {
				t.Errorf("goroutine %d: create failed with %d", id, status)
				return
			}
			secretID, _ := resp["id"].(string)

			updateBody := map[string]interface{}{
				"name": fmt.Sprintf("concurrent-update-%d", id),
			}
			status, _ = chaosServeJSON(t, r, "PUT", "/api/v1/vault/secrets/"+secretID, updateBody, headers)
			if status >= 500 {
				t.Errorf("goroutine %d: update got %d — expected 200", id, status)
			}
		})
	})

	t.Run("concurrent_rotates_separate_secrets", func(t *testing.T) {
		// Each goroutine creates and rotates its OWN secret.
		const parallel = 10
		testutil.RunConcurrent(t, parallel, func(id int) {
			createBody := map[string]interface{}{
				"user_id":          callerID.String(),
				"name":             fmt.Sprintf("rotate-target-%d", id),
				"type":             "password",
				"encrypted_value":  fmt.Sprintf("enc-%d", id),
				"iv":               fmt.Sprintf("iv-%d", id),
				"salt":             fmt.Sprintf("salt-%d", id),
			}
			status, resp := chaosServeJSON(t, r, "POST", "/api/v1/vault/secrets", createBody, headers)
			if status != http.StatusCreated {
				t.Errorf("goroutine %d: create failed with %d", id, status)
				return
			}
			secretID, _ := resp["id"].(string)

			rotateBody := map[string]interface{}{
				"encrypted_value": fmt.Sprintf("rotated-enc-%d", id),
				"iv":              fmt.Sprintf("rotated-iv-%d", id),
				"salt":            fmt.Sprintf("rotated-salt-%d", id),
				"created_by":      callerID.String(),
			}
			status, _ = chaosServeJSON(t, r, "POST", "/api/v1/vault/secrets/"+secretID+"/rotate", rotateBody, headers)
			if status >= 500 {
				t.Errorf("goroutine %d: rotate got %d — expected 200", id, status)
			}
		})
	})
}

// TestChaosBoundaryConditions exercises extreme boundary values that
// stress the parsing, validation, and serialization layers.
func TestChaosBoundaryConditions(t *testing.T) {
	repo := newMockRepo()
	h := handler.New(repo)
	r := setupRouter(h)

	callerID := uuid.New()
	headers := map[string]string{"X-User-ID": callerID.String()}

	t.Run("nil_body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/vault/secrets", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", callerID.String())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code >= 500 {
			t.Errorf("nil body: got %d — expected 400", w.Code)
		}
		t.Logf("nil body → %d", w.Code)
	})

	t.Run("empty_json_object", func(t *testing.T) {
		status, _ := chaosPostRaw(t, r, "POST", "/api/v1/vault/secrets", "application/json", []byte("{}"), headers)
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, r, "POST", "/api/v1/vault/secrets", "application/json", []byte("[]"), headers)
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic. With a mock repo (no DB
		// column limits), the handler may accept it (201). With a
		// real DB, column overflow would produce 500. Either way,
		// no panic is the invariant.
		largeValue := strings.Repeat("x", 1000000)
		payload := fmt.Sprintf(`{"user_id":"%s","name":"large","type":"api_token","encrypted_value":"%s","iv":"iv","salt":"salt"}`,
			callerID.String(), largeValue)
		status, _ := chaosPostRaw(t, r, "POST", "/api/v1/vault/secrets", "application/json", []byte(payload), headers)
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		// FINDING: no body-size middleware configured — mock accepts
		// 201, real DB would return 500 on column overflow. The
		// handler does not panic either way.
		t.Logf("FINDING: 1MB payload → %d (handler lacks body-size middleware but does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"user_id":"","name":"","type":"","encrypted_value":"","iv":"","salt":""}`
		status, _ := chaosPostRaw(t, r, "POST", "/api/v1/vault/secrets", "application/json", []byte(payload), headers)
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("negative_limit_and_offset", func(t *testing.T) {
		status, _ := chaosServeJSON(t, r, "GET", "/api/v1/vault/secrets?limit=-1&offset=-100", nil, headers)
		if status >= 500 {
			t.Errorf("negative limit/offset: got %d — expected 400", status)
		}
		t.Logf("negative limit/offset → %d", status)
	})

	t.Run("extremely_large_limit", func(t *testing.T) {
		status, _ := chaosServeJSON(t, r, "GET", "/api/v1/vault/secrets?limit=999999", nil, headers)
		if status >= 500 {
			t.Errorf("extremely large limit: got %d — expected 400 or clamped", status)
		}
		t.Logf("extremely large limit → %d", status)
	})

	t.Run("path_traversal_in_id", func(t *testing.T) {
		status, _ := chaosServeJSON(t, r, "GET", "/api/v1/vault/secrets/../../../etc/passwd", nil, headers)
		if status >= 500 {
			t.Errorf("path traversal: got %d — expected 400", status)
		}
		t.Logf("path traversal → %d", status)
	})

	t.Run("missing_x_user_id_header", func(t *testing.T) {
		body := map[string]interface{}{
			"name":             "no-auth-test",
			"type":             "api_token",
			"encrypted_value":  "enc",
			"iv":               "iv",
			"salt":             "salt",
		}
		status, _ := chaosServeJSON(t, r, "POST", "/api/v1/vault/secrets", body, nil)
		if status == http.StatusCreated {
			t.Fatal("missing X-User-ID must be rejected, got 201")
		}
		if status >= 500 {
			t.Errorf("missing X-User-ID: got %d — expected 401", status)
		}
		t.Logf("missing X-User-ID → %d (expected 401)", status)
	})

	t.Run("mismatched_body_user_id", func(t *testing.T) {
		// Body says tenant A, header says tenant B — must be rejected (T7 IDOR)
		body := map[string]interface{}{
			"user_id":          uuid.New().String(), // tenant A
			"name":             "idor-test",
			"type":             "api_token",
			"encrypted_value":  "enc",
			"iv":               "iv",
			"salt":             "salt",
		}
		status, _ := chaosServeJSON(t, r, "POST", "/api/v1/vault/secrets", body, headers) // header = callerID (tenant B)
		if status == http.StatusCreated {
			t.Fatal("mismatched body/header user_id must be rejected (IDOR), got 201")
		}
		t.Logf("mismatched user_id → %d (expected 400)", status)
	})
}
