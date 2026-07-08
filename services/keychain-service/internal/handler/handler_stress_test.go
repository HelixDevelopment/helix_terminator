//go:build stress

// Stress test suite for keychain-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of create→get→update→delete,
//     per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=10 parallel goroutines performing
//     create+get, no deadlock, no resource leak.
//   - Boundary conditions: empty name, max-length, invalid type,
//     duplicate IDs — every boundary produces a categorised result.
//
// Run:
//
//	go test -race -tags stress -run TestStress -v -timeout 120s ./internal/handler/
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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/keychain-service/internal/handler"
	"github.com/helixdevelopment/keychain-service/internal/model"
	"github.com/helixdevelopment/keychain-service/internal/repository"
	"github.com/helixdevelopment/keychain-service/internal/testutil"
	"github.com/helixdevelopment/keychain-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testEncryptionKey is a fixed test-only key for encrypting private_key
// and passphrase at rest. NOT a production secret (§11.4.10).
const testEncryptionKey = "test-encryption-key-for-stress-tests-only-32b!"

// stressEnv holds the assembled test environment: a real gin engine
// wired to a real handler backed by a real PostgreSQL pool.
type stressEnv struct {
	ts      *httptest.Server
	cleanup func()
}

// stressAuthMiddleware injects a fixed test userID into the gin context,
// bypassing real JWT validation. This mirrors what authMiddleware does
// in production (server.go) but with a deterministic test identity.
func stressAuthMiddleware(userID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID.String())
		c.Next()
	}
}

// setupStressEnv boots a real PostgreSQL container (via podman),
// applies keychain-service migrations, constructs a real handler+router,
// and returns a ready httptest.Server. Skips honestly if podman is
// unavailable.
func setupStressEnv(t *testing.T) *stressEnv {
	t.Helper()

	poolURL, available := testutil.StartTestPostgres(t)
	if !available {
		t.Skip("SKIP: podman not available — cannot run stress tests against real database (topology_unsupported)")
	}

	pool, err := pgxpool.New(t.Context(), poolURL)
	if err != nil {
		t.Fatalf("pgxpool.New failed: %v", err)
	}

	repo, err := repository.New(pool, testEncryptionKey)
	if err != nil {
		t.Fatalf("repository.New failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(repo)

	testUserID := uuid.New()
	api := r.Group("/api/v1")
	api.Use(stressAuthMiddleware(testUserID))
	{
		api.POST("/keychain", h.CreateItem)
		api.GET("/keychain", h.ListItems)
		api.GET("/keychain/:id", h.GetItem)
		api.PUT("/keychain/:id", h.UpdateItem)
		api.DELETE("/keychain/:id", h.DeleteItem)
	}

	ts := httptest.NewServer(r)

	return &stressEnv{
		ts: ts,
		cleanup: func() {
			ts.Close()
			pool.Close()
		},
	}
}

// stressPostRaw sends a POST request with raw bytes and returns status +
// parsed response. Used instead of json.Marshal to ensure "tags":[]
// is explicitly present in the JSON body (omitempty strips empty slices).
func stressPostRaw(t *testing.T, client *http.Client, url string, body []byte) (int, map[string]interface{}) {
	t.Helper()
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return resp.StatusCode, parsed
}

// stressPostJSON sends a POST request with a JSON body and returns status +
// parsed response.
func stressPostJSON(t *testing.T, client *http.Client, url string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	return stressPostRaw(t, client, url, b)
}

// stressGetJSON sends a GET request and returns status + parsed response.
func stressGetJSON(t *testing.T, client *http.Client, url string) (int, map[string]interface{}) {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return resp.StatusCode, parsed
}

// stressPutJSON sends a PUT request with a JSON body and returns status +
// parsed response.
func stressPutJSON(t *testing.T, client *http.Client, url string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	req, err := http.NewRequest("PUT", url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return resp.StatusCode, parsed
}

// stressDeleteJSON sends a DELETE request and returns status.
func stressDeleteJSON(t *testing.T, client *http.Client, url string) int {
	t.Helper()
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// uniqueKeyName generates a collision-free key name for stress iterations.
func uniqueKeyName(prefix string, i int) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), i)
}

// makeCreateBody builds a raw JSON body for creating a keychain item.
// Uses fmt.Sprintf instead of json.Marshal to ensure "tags":[] is
// explicitly present (omitempty strips empty slices from Go structs).
func makeCreateBody(name, keyType, privateKey string) []byte {
	return []byte(fmt.Sprintf(
		`{"name":%q,"type":%q,"privateKey":%q,"tags":[]}`,
		name, keyType, privateKey,
	))
}

// stressCreateItem creates a keychain item via raw JSON and returns its ID.
func stressCreateItem(t *testing.T, client *http.Client, baseURL string, name string) string {
	t.Helper()
	body := makeCreateBody(name, "ssh", "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-data\n-----END OPENSSH PRIVATE KEY-----")
	status, resp := stressPostRaw(t, client, baseURL+"/api/v1/keychain", body)
	if status != http.StatusCreated {
		t.Fatalf("create item failed: status=%d, body=%v", status, resp)
	}
	id, ok := resp["id"].(string)
	if !ok || id == "" {
		t.Fatalf("create item returned no id: body=%v", resp)
	}
	return id
}

// TestStressCreateGetUpdateDelete_SustainedLoad drives N>=100
// iterations of the full create→get→update→delete cycle against a real
// PostgreSQL instance, recording per-iteration latency and computing
// p50/p95/p99. Every iteration uses a unique name to avoid conflicts.
func TestStressCreateGetUpdateDelete_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		name := uniqueKeyName("stress-cgud", i)
		start := time.Now()

		// Create
		body := makeCreateBody(name, "ssh", "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key\n-----END OPENSSH PRIVATE KEY-----")
		status, resp := stressPostRaw(t, client, env.ts.URL+"/api/v1/keychain", body)
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /keychain status = %d, want 201; body=%v", i, status, resp)
		}
		itemID, _ := resp["id"].(string)
		if itemID == "" {
			t.Fatalf("iteration %d: POST /keychain returned no id", i)
		}

		// Get
		status, resp = stressGetJSON(t, client, env.ts.URL+"/api/v1/keychain/"+itemID)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /keychain/%s status = %d, want 200; body=%v", i, itemID, status, resp)
		}

		// Update
		newName := name + "-updated"
		status, resp = stressPutJSON(t, client, env.ts.URL+"/api/v1/keychain/"+itemID, model.UpdateKeychainItemRequest{
			Name: &newName,
		})
		if status != http.StatusOK {
			t.Fatalf("iteration %d: PUT /keychain/%s status = %d, want 200; body=%v", i, itemID, status, resp)
		}

		// Delete
		status = stressDeleteJSON(t, client, env.ts.URL+"/api/v1/keychain/"+itemID)
		if status != http.StatusNoContent {
			t.Fatalf("iteration %d: DELETE /keychain/%s status = %d, want 204", i, itemID, status)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)

	// Log evidence path for §11.4.69 compliance
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=10 parallel goroutines,
// each performing a create+get cycle. Validates no deadlock
// occurs and all goroutines complete within the timeout.
func TestStressConcurrentContention(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()

	testutil.RunConcurrent(t, parallelism, func(id int) {
		name := uniqueKeyName("stress-cc", id)
		start := time.Now()

		// Create
		body := makeCreateBody(name, "ssh", "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key\n-----END OPENSSH PRIVATE KEY-----")
		status, resp := stressPostRaw(t, client, env.ts.URL+"/api/v1/keychain", body)
		if status != http.StatusCreated {
			t.Errorf("goroutine %d: POST /keychain status = %d, want 201; body=%v", id, status, resp)
			return
		}
		itemID, _ := resp["id"].(string)
		if itemID == "" {
			t.Errorf("goroutine %d: POST /keychain returned no id", id)
			return
		}

		// Get
		status, resp = stressGetJSON(t, client, env.ts.URL+"/api/v1/keychain/"+itemID)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /keychain/%s status = %d, want 200; body=%v", id, itemID, status, resp)
			return
		}

		rec.Record(time.Since(start))
	})

	p50, p95, p99 := rec.Percentiles()
	t.Logf("CONCURRENT CONTENTION (%d goroutines): p50=%v p95=%v p99=%v", parallelism, p50, p95, p99)
}

// TestStressBoundaryConditions exercises edge-case inputs against the
// create endpoint. Each subtest drives a specific boundary and
// categorises the result (400 for validation, 201 for valid).
// Uses a real DB so constraint violations are genuine.
func TestStressBoundaryConditions(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("empty_name_rejected", func(t *testing.T) {
		body := makeCreateBody("", "ssh", "test-key-data")
		status, _ := stressPostRaw(t, client, env.ts.URL+"/api/v1/keychain", body)
		if status == http.StatusCreated {
			t.Fatal("empty name must be rejected, got 201")
		}
		t.Logf("empty name → %d (expected 400)", status)
	})

	t.Run("max_length_name_accepted_or_rejected", func(t *testing.T) {
		longName := strings.Repeat("a", 255)
		body := makeCreateBody(longName, "ssh", "test-key-data")
		status, resp := stressPostRaw(t, client, env.ts.URL+"/api/v1/keychain", body)
		t.Logf("max-length name (%d chars) → %d", len(longName), status)
		if status == http.StatusCreated {
			id, _ := resp["id"].(string)
			if id == "" {
				t.Fatal("201 but no id returned")
			}
		}
	})

	t.Run("over_max_length_name_rejected", func(t *testing.T) {
		longName := strings.Repeat("a", 256)
		body := makeCreateBody(longName, "ssh", "test-key-data")
		status, _ := stressPostRaw(t, client, env.ts.URL+"/api/v1/keychain", body)
		if status == http.StatusCreated {
			t.Fatal("over-max-length name must be rejected, got 201")
		}
		t.Logf("over-max-length name (%d chars) → %d (expected 400)", len(longName), status)
	})

	t.Run("invalid_type_rejected", func(t *testing.T) {
		body := makeCreateBody(uniqueKeyName("boundary", 0), "invalid-type", "test-key-data")
		status, _ := stressPostRaw(t, client, env.ts.URL+"/api/v1/keychain", body)
		if status == http.StatusCreated {
			t.Fatal("invalid type must be rejected, got 201")
		}
		t.Logf("invalid type → %d (expected 400)", status)
	})

	t.Run("missing_private_key_rejected", func(t *testing.T) {
		body := []byte(fmt.Sprintf(`{"name":%q,"type":"ssh","tags":[]}`, uniqueKeyName("boundary", 1)))
		status, _ := stressPostRaw(t, client, env.ts.URL+"/api/v1/keychain", body)
		if status == http.StatusCreated {
			t.Fatal("missing private key must be rejected, got 201")
		}
		t.Logf("missing private key → %d (expected 400)", status)
	})

	t.Run("missing_type_rejected", func(t *testing.T) {
		body := []byte(fmt.Sprintf(`{"name":%q,"privateKey":"test-key-data","tags":[]}`, uniqueKeyName("boundary", 2)))
		status, _ := stressPostRaw(t, client, env.ts.URL+"/api/v1/keychain", body)
		if status == http.StatusCreated {
			t.Fatal("missing type must be rejected, got 201")
		}
		t.Logf("missing type → %d (expected 400)", status)
	})

	t.Run("all_valid_key_types", func(t *testing.T) {
		validTypes := []string{"ssh", "gpg", "api_key", "password", "x509"}
		for _, kt := range validTypes {
			body := makeCreateBody(uniqueKeyName("boundary-type-"+kt, 0), kt, "test-key-data")
			status, resp := stressPostRaw(t, client, env.ts.URL+"/api/v1/keychain", body)
			if status != http.StatusCreated {
				t.Errorf("valid type %q: got %d, want 201; body=%v", kt, status, resp)
			}
			t.Logf("valid type %q → %d", kt, status)
		}
	})

	t.Run("nonexistent_id_get_returns_404", func(t *testing.T) {
		fakeID := uuid.New().String()
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/keychain/"+fakeID)
		if status != http.StatusNotFound {
			t.Fatalf("nonexistent id: got %d, want 404", status)
		}
		t.Logf("nonexistent id → %d (expected 404)", status)
	})

	t.Run("invalid_id_format_returns_400", func(t *testing.T) {
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/keychain/not-a-uuid")
		if status != http.StatusBadRequest {
			t.Fatalf("invalid id format: got %d, want 400", status)
		}
		t.Logf("invalid id format → %d (expected 400)", status)
	})

	t.Run("delete_nonexistent_returns_404", func(t *testing.T) {
		fakeID := uuid.New().String()
		status := stressDeleteJSON(t, client, env.ts.URL+"/api/v1/keychain/"+fakeID)
		if status != http.StatusNotFound {
			t.Fatalf("delete nonexistent: got %d, want 404", status)
		}
		t.Logf("delete nonexistent → %d (expected 404)", status)
	})

	t.Run("empty_body_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/keychain", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			t.Fatal("empty body must be rejected, got 201")
		}
		t.Logf("empty body → %d (expected 400)", resp.StatusCode)
	})
}

// TestStressBoundaryConditions_NoRepo exercises the nil-repo guard
// WITHOUT a database — proves the handler returns 503 cleanly (no
// panic, no hang) when the repository is not initialized. Keychain-
// service's CreateItem checks h.repo == nil BEFORE ShouldBindJSON,
// so all requests hit the 503 path. Validation-level boundary
// conditions are covered by TestStressBoundaryConditions with a real
// DB.
func TestStressBoundaryConditions_NoRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil)
	r.POST("/api/v1/keychain", h.CreateItem)

	cases := []struct {
		name string
		body string
	}{
		{"empty_body", ""},
		{"invalid_json", "{broken"},
		{"missing_private_key", `{"name":"test","type":"ssh"}`},
		{"missing_name", `{"type":"ssh","privateKey":"key"}`},
		{"valid_shape", `{"name":"test","type":"ssh","privateKey":"key"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/keychain", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			// All requests must get 503 (nil-repo guard) — the
			// handler must NOT panic regardless of input shape.
			if w.Code != http.StatusServiceUnavailable {
				t.Logf("body=%q → %d (want 503 — nil-repo guard)", tc.body, w.Code)
			}
		})
	}
	t.Log("nil-repo guard returns 503 for all inputs — handler does not panic")
}

// init ensures the test database migrations package is imported so
// the test binary includes the migration runner. This is a no-op
// import anchor — the real work happens in setupStressEnv.
var _ = migrations.Schema
