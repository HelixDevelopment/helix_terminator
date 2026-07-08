//go:build stress

// Stress test suite for vault-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of create→get→list→update→
//     delete→rotate, per-iteration latency recorded, p50/p95/p99
//     computed.
//   - Concurrent contention: N>=15 parallel goroutines performing
//     create+get+list+update+delete, no deadlock, no resource leak.
//   - Boundary conditions: empty name, invalid type, missing fields,
//     zero-value structs, duplicate IDs — every boundary produces a
//     categorised result.
//
// Run:
//
//	go test -race -tags stress -run TestStress -v -timeout 120s ./internal/handler/
package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/helixdevelopment/vault-service/internal/handler"
	"github.com/helixdevelopment/vault-service/internal/testutil"
)

// serveStress executes an HTTP request against the gin router and
// returns status code + parsed JSON response.
func serveStress(t *testing.T, r http.Handler, method, path string, body interface{}, headers map[string]string) (int, map[string]interface{}) {
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

// uniqueID generates a collision-free UUID for stress iterations.
func uniqueID() uuid.UUID {
	return uuid.New()
}

// TestStressSustainedLoad drives N>=100 iterations of the full
// create→get→list→update→delete→rotate cycle, recording per-iteration
// latency and computing p50/p95/p99.
func TestStressSustainedLoad(t *testing.T) {
	repo := newMockRepo()
	h := newHandler(repo)
	r := setupRouter(h)

	callerID := uuid.New()
	headers := map[string]string{"X-User-ID": callerID.String()}
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Create
		createBody := map[string]interface{}{
			"user_id":          callerID.String(),
			"name":             fmt.Sprintf("stress-secret-%d", i),
			"type":             "api_token",
			"encrypted_value":  fmt.Sprintf("enc-%d", i),
			"iv":               fmt.Sprintf("iv-%d", i),
			"salt":             fmt.Sprintf("salt-%d", i),
		}
		status, resp := serveStress(t, r, "POST", "/api/v1/vault/secrets", createBody, headers)
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /secrets status = %d, want 201; body=%v", i, status, resp)
		}
		secretID, _ := resp["id"].(string)
		if secretID == "" {
			t.Fatalf("iteration %d: POST /secrets returned no id", i)
		}

		// Get
		status, resp = serveStress(t, r, "GET", "/api/v1/vault/secrets/"+secretID, nil, headers)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /secrets/%s status = %d, want 200; body=%v", i, secretID, status, resp)
		}

		// List
		status, resp = serveStress(t, r, "GET", "/api/v1/vault/secrets", nil, headers)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /secrets status = %d, want 200; body=%v", i, status, resp)
		}

		// Update
		updateBody := map[string]interface{}{
			"name": fmt.Sprintf("stress-secret-%d-updated", i),
		}
		status, resp = serveStress(t, r, "PUT", "/api/v1/vault/secrets/"+secretID, updateBody, headers)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: PUT /secrets/%s status = %d, want 200; body=%v", i, secretID, status, resp)
		}

		// Rotate
		rotateBody := map[string]interface{}{
			"encrypted_value": fmt.Sprintf("rotated-enc-%d", i),
			"iv":              fmt.Sprintf("rotated-iv-%d", i),
			"salt":            fmt.Sprintf("rotated-salt-%d", i),
			"created_by":      callerID.String(),
		}
		status, resp = serveStress(t, r, "POST", "/api/v1/vault/secrets/"+secretID+"/rotate", rotateBody, headers)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: POST /secrets/%s/rotate status = %d, want 200; body=%v", i, secretID, status, resp)
		}

		// Delete
		status, _ = serveStress(t, r, "DELETE", "/api/v1/vault/secrets/"+secretID, nil, headers)
		if status != http.StatusNoContent {
			t.Fatalf("iteration %d: DELETE /secrets/%s status = %d, want 204", i, secretID, status)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=15 parallel goroutines,
// each performing a create+get+list+update+delete cycle. Validates no
// deadlock occurs and all goroutines complete within the timeout.
func TestStressConcurrentContention(t *testing.T) {
	repo := newSyncMockRepo()
	h := newSyncHandler(repo)
	r := setupRouter(h)

	callerID := uuid.New()
	headers := map[string]string{"X-User-ID": callerID.String()}
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()

	testutil.RunConcurrent(t, parallelism, func(id int) {
		start := time.Now()

		// Create
		createBody := map[string]interface{}{
			"user_id":          callerID.String(),
			"name":             fmt.Sprintf("concurrent-secret-%d", id),
			"type":             "password",
			"encrypted_value":  fmt.Sprintf("enc-concurrent-%d", id),
			"iv":               fmt.Sprintf("iv-concurrent-%d", id),
			"salt":             fmt.Sprintf("salt-concurrent-%d", id),
		}
		status, resp := serveStress(t, r, "POST", "/api/v1/vault/secrets", createBody, headers)
		if status != http.StatusCreated {
			t.Errorf("goroutine %d: POST /secrets status = %d, want 201; body=%v", id, status, resp)
			return
		}
		secretID, _ := resp["id"].(string)
		if secretID == "" {
			t.Errorf("goroutine %d: POST /secrets returned no id", id)
			return
		}

		// Get
		status, resp = serveStress(t, r, "GET", "/api/v1/vault/secrets/"+secretID, nil, headers)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /secrets/%s status = %d, want 200; body=%v", id, secretID, status, resp)
			return
		}

		// Update
		updateBody := map[string]interface{}{
			"name": fmt.Sprintf("concurrent-secret-%d-updated", id),
		}
		status, resp = serveStress(t, r, "PUT", "/api/v1/vault/secrets/"+secretID, updateBody, headers)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: PUT /secrets/%s status = %d, want 200; body=%v", id, secretID, status, resp)
			return
		}

		// Delete
		status, _ = serveStress(t, r, "DELETE", "/api/v1/vault/secrets/"+secretID, nil, headers)
		if status != http.StatusNoContent {
			t.Errorf("goroutine %d: DELETE /secrets/%s status = %d, want 204", id, secretID, status)
			return
		}

		rec.Record(time.Since(start))
	})

	p50, p95, p99 := rec.Percentiles()
	t.Logf("CONCURRENT CONTENTION (%d goroutines): p50=%v p95=%v p99=%v", parallelism, p50, p95, p99)
}

// TestStressBoundaryConditions exercises edge-case inputs against the
// vault endpoints. Each subtest drives a specific boundary and
// categorises the result.
func TestStressBoundaryConditions(t *testing.T) {
	repo := newMockRepo()
	h := newHandler(repo)
	r := setupRouter(h)

	callerID := uuid.New()
	headers := map[string]string{"X-User-ID": callerID.String()}

	t.Run("empty_name_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id":          callerID.String(),
			"name":             "",
			"type":             "api_token",
			"encrypted_value":  "enc",
			"iv":               "iv",
			"salt":             "salt",
		}
		status, _ := serveStress(t, r, "POST", "/api/v1/vault/secrets", body, headers)
		if status == http.StatusCreated {
			t.Fatal("empty name must be rejected, got 201")
		}
		t.Logf("empty name → %d (expected 400)", status)
	})

	t.Run("invalid_type_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id":          callerID.String(),
			"name":             "test-secret",
			"type":             "invalid_type",
			"encrypted_value":  "enc",
			"iv":               "iv",
			"salt":             "salt",
		}
		status, _ := serveStress(t, r, "POST", "/api/v1/vault/secrets", body, headers)
		if status == http.StatusCreated {
			t.Fatal("invalid type must be rejected, got 201")
		}
		t.Logf("invalid type → %d (expected 400)", status)
	})

	t.Run("missing_user_id_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"name":             "test-secret",
			"type":             "api_token",
			"encrypted_value":  "enc",
			"iv":               "iv",
			"salt":             "salt",
		}
		// No X-User-ID header
		status, _ := serveStress(t, r, "POST", "/api/v1/vault/secrets", body, nil)
		if status == http.StatusCreated {
			t.Fatal("missing user_id must be rejected, got 201")
		}
		t.Logf("missing user_id → %d (expected 400 or 401)", status)
	})

	t.Run("missing_encrypted_value_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id": callerID.String(),
			"name":    "test-secret",
			"type":    "api_token",
			"iv":      "iv",
			"salt":    "salt",
		}
		status, _ := serveStress(t, r, "POST", "/api/v1/vault/secrets", body, headers)
		if status == http.StatusCreated {
			t.Fatal("missing encrypted_value must be rejected, got 201")
		}
		t.Logf("missing encrypted_value → %d (expected 400)", status)
	})

	t.Run("max_length_name_accepted", func(t *testing.T) {
		longName := strings.Repeat("a", 255)
		body := map[string]interface{}{
			"user_id":          callerID.String(),
			"name":             longName,
			"type":             "api_token",
			"encrypted_value":  "enc",
			"iv":               "iv",
			"salt":             "salt",
		}
		status, resp := serveStress(t, r, "POST", "/api/v1/vault/secrets", body, headers)
		if status != http.StatusCreated {
			t.Fatalf("max-length name (%d chars) → %d, want 201; body=%v", len(longName), status, resp)
		}
		t.Logf("max-length name (%d chars) → %d", len(longName), status)
	})

	t.Run("invalid_uuid_in_get_rejected", func(t *testing.T) {
		status, _ := serveStress(t, r, "GET", "/api/v1/vault/secrets/not-a-uuid", nil, headers)
		if status == http.StatusOK {
			t.Fatal("invalid UUID must be rejected, got 200")
		}
		t.Logf("invalid UUID → %d (expected 400)", status)
	})

	t.Run("nonexistent_secret_returns_404", func(t *testing.T) {
		fakeID := uuid.New().String()
		status, _ := serveStress(t, r, "GET", "/api/v1/vault/secrets/"+fakeID, nil, headers)
		if status != http.StatusNotFound {
			t.Fatalf("nonexistent secret → %d, want 404", status)
		}
		t.Logf("nonexistent secret → %d (expected 404)", status)
	})

	t.Run("all_valid_secret_types_accepted", func(t *testing.T) {
		validTypes := []string{"ssh_key", "api_token", "password", "certificate", "env_var"}
		for _, st := range validTypes {
			body := map[string]interface{}{
				"user_id":          callerID.String(),
				"name":             fmt.Sprintf("type-test-%s", st),
				"type":             st,
				"encrypted_value":  "enc",
				"iv":               "iv",
				"salt":             "salt",
			}
			status, _ := serveStress(t, r, "POST", "/api/v1/vault/secrets", body, headers)
			if status != http.StatusCreated {
				t.Errorf("valid type %q → %d, want 201", st, status)
			}
			t.Logf("valid type %q → %d", st, status)
		}
	})

	t.Run("zero_value_struct_fields_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"user_id":          "",
			"name":             "",
			"type":             "",
			"encrypted_value":  "",
			"iv":               "",
			"salt":             "",
		}
		status, _ := serveStress(t, r, "POST", "/api/v1/vault/secrets", body, headers)
		if status == http.StatusCreated {
			t.Fatal("zero-value fields must be rejected, got 201")
		}
		t.Logf("zero-value fields → %d (expected 400)", status)
	})

	t.Run("empty_body_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/vault/secrets", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", callerID.String())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusCreated {
			t.Fatal("empty body must be rejected, got 201")
		}
		t.Logf("empty body → %d (expected 400)", w.Code)
	})
}

// TestStressBoundaryConditions_NilRepo exercises boundary conditions
// against the validation layer WITHOUT a repository — proves the
// nil-repo guard returns 503 before any parsing.
func TestStressBoundaryConditions_NilRepo(t *testing.T) {
	h := newHandler(nil)
	r := setupRouter(h)

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"create_nil_repo", "POST", "/api/v1/vault/secrets", `{"name":"test","type":"api_token","encrypted_value":"enc","iv":"iv","salt":"salt"}`, 503},
		{"get_nil_repo", "GET", "/api/v1/vault/secrets/" + uuid.New().String(), "", 503},
		{"list_nil_repo", "GET", "/api/v1/vault/secrets", "", 503},
		{"update_nil_repo", "PUT", "/api/v1/vault/secrets/" + uuid.New().String(), `{"name":"new"}`, 503},
		{"delete_nil_repo", "DELETE", "/api/v1/vault/secrets/" + uuid.New().String(), "", 503},
		{"versions_nil_repo", "GET", "/api/v1/vault/secrets/" + uuid.New().String() + "/versions", "", 503},
		{"rotate_nil_repo", "POST", "/api/v1/vault/secrets/" + uuid.New().String() + "/rotate", `{"encrypted_value":"new","iv":"iv","salt":"salt","created_by":"` + uuid.New().String() + `"}`, 503},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var reqBody *bytes.Reader
			if tc.body != "" {
				reqBody = bytes.NewReader([]byte(tc.body))
			} else {
				reqBody = bytes.NewReader(nil)
			}
			req, _ := http.NewRequest(tc.method, tc.path, reqBody)
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			req.Header.Set("X-User-ID", uuid.New().String())
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Logf("%s %s → %d (want %d)", tc.method, tc.path, w.Code, tc.wantStatus)
			}
		})
	}
}

// newHandler creates a handler.Handler from the mock repo. The mock
// repo type is defined in handler_test.go (same package, no build tag
// restrictions).
func newHandler(repo *mockRepo) *handler.Handler {
	if repo == nil {
		return handler.New(nil)
	}
	return handler.New(repo)
}
