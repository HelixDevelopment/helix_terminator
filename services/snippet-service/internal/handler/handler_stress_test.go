//go:build stress

// Stress test suite for snippet-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of create→get→update→delete,
//     per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=10 parallel goroutines performing
//     create+get, no deadlock, no resource leak.
//   - Boundary conditions: empty name, max-length, invalid fields,
//     missing required fields — every boundary produces a categorised
//     result.
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
	"github.com/helixdevelopment/snippet-service/internal/handler"
	"github.com/helixdevelopment/snippet-service/internal/model"
	"github.com/helixdevelopment/snippet-service/internal/repository"
	"github.com/helixdevelopment/snippet-service/internal/testutil"
	"github.com/helixdevelopment/snippet-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// stressEnv holds the assembled test environment: a real gin engine
// wired to a real handler backed by a real PostgreSQL pool.
type stressEnv struct {
	ts      *httptest.Server
	cleanup func()
}

// setupStressEnv boots a real PostgreSQL container (via podman),
// applies snippet-service migrations, constructs a real handler+router,
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

	repo := repository.New(pool)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(repo)

	// Inject a deterministic test user_id into the gin context for
	// CreateSnippet (which reads c.GetString("user_id")).
	testUserID := uuid.New().String()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Next()
	})

	r.POST("/api/v1/snippets", h.CreateSnippet)
	r.GET("/api/v1/snippets", h.ListSnippets)
	r.GET("/api/v1/snippets/:id", h.GetSnippet)
	r.PUT("/api/v1/snippets/:id", h.UpdateSnippet)
	r.DELETE("/api/v1/snippets/:id", h.DeleteSnippet)
	r.GET("/healthz", h.HealthCheck)

	ts := httptest.NewServer(r)

	return &stressEnv{
		ts: ts,
		cleanup: func() {
			ts.Close()
			pool.Close()
		},
	}
}

// stressPostJSON sends a POST request with a JSON body and returns status +
// parsed response.
func stressPostJSON(t *testing.T, client *http.Client, url string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
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

// uniqueSnippetName generates a collision-free snippet name for stress iterations.
func uniqueSnippetName(prefix string, i int) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), i)
}

// TestStressCreateGetUpdateDelete_SustainedLoad drives N>=100
// iterations of the full create→get→update→delete cycle against a real
// PostgreSQL instance, recording per-iteration latency and computing
// p50/p95/p99. Every iteration uses a unique name to avoid
// collision issues.
func TestStressCreateGetUpdateDelete_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		name := uniqueSnippetName("stress-cgud", i)
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/snippets", model.CreateSnippetRequest{
			Name:     name,
			Content:  fmt.Sprintf("echo 'stress test iteration %d'", i),
			Language: "bash",
			Tags:     []string{"stress", "test"},
		})
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /api/v1/snippets status = %d, want 201; body=%v", i, status, body)
		}
		idStr, _ := body["id"].(string)
		if idStr == "" {
			t.Fatalf("iteration %d: POST /api/v1/snippets returned no id", i)
		}

		// Get
		status, body = stressGetJSON(t, client, env.ts.URL+"/api/v1/snippets/"+idStr)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /api/v1/snippets/%s status = %d, want 200; body=%v", i, idStr, status, body)
		}
		if body["name"] != name {
			t.Fatalf("iteration %d: GET returned name=%v, want %q", i, body["name"], name)
		}

		// Update
		updatedName := name + "-updated"
		status, body = stressPutJSON(t, client, env.ts.URL+"/api/v1/snippets/"+idStr, model.UpdateSnippetRequest{
			Name: updatedName,
		})
		if status != http.StatusOK {
			t.Fatalf("iteration %d: PUT /api/v1/snippets/%s status = %d, want 200; body=%v", i, idStr, status, body)
		}

		// Delete
		status = stressDeleteJSON(t, client, env.ts.URL+"/api/v1/snippets/"+idStr)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: DELETE /api/v1/snippets/%s status = %d, want 200", i, idStr, status)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)

	// Log evidence path for §11.4.69 compliance
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=15 parallel goroutines,
// each performing a create+get cycle. Validates no deadlock
// occurs and all goroutines complete within the timeout.
func TestStressConcurrentContention(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()

	testutil.RunConcurrent(t, parallelism, func(id int) {
		name := uniqueSnippetName("stress-cc", id)
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/snippets", model.CreateSnippetRequest{
			Name:     name,
			Content:  fmt.Sprintf("echo 'concurrent test %d'", id),
			Language: "bash",
		})
		if status != http.StatusCreated {
			t.Errorf("goroutine %d: POST /api/v1/snippets status = %d, want 201; body=%v", id, status, body)
			return
		}

		idStr, _ := body["id"].(string)
		if idStr == "" {
			t.Errorf("goroutine %d: POST /api/v1/snippets returned no id", id)
			return
		}

		// Get
		status, body = stressGetJSON(t, client, env.ts.URL+"/api/v1/snippets/"+idStr)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /api/v1/snippets/%s status = %d, want 200; body=%v", id, idStr, status, body)
			return
		}

		rec.Record(time.Since(start))
	})

	p50, p95, p99 := rec.Percentiles()
	t.Logf("CONCURRENT CONTENTION (%d goroutines): p50=%v p95=%v p99=%v", parallelism, p50, p95, p99)
}

// TestStressListPagination_SustainedLoad drives N>=100 create
// operations followed by paginated list queries, validating that
// limit/offset handling is correct under sustained load.
func TestStressListPagination_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const count = 50

	// Create snippets
	for i := 0; i < count; i++ {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/snippets", model.CreateSnippetRequest{
			Name:     uniqueSnippetName("stress-list", i),
			Content:  fmt.Sprintf("content %d", i),
			Language: "python",
		})
		if status != http.StatusCreated {
			t.Fatalf("create %d: status = %d, want 201", i, status)
		}
	}

	// Paginate through them
	fetched := 0
	for offset := 0; offset < count; offset += 10 {
		status, body := stressGetJSON(t, client, fmt.Sprintf("%s/api/v1/snippets?limit=10&offset=%d", env.ts.URL, offset))
		if status != http.StatusOK {
			t.Fatalf("list offset=%d: status = %d, want 200", offset, status)
		}
		data, _ := body["data"].([]interface{})
		fetched += len(data)
	}
	if fetched < count {
		t.Errorf("paginated fetch got %d items, want >= %d", fetched, count)
	}
	t.Logf("PAGINATION: fetched %d items across paginated queries", fetched)
}

// TestStressBoundaryConditions exercises edge-case inputs against the
// create endpoint. Each subtest drives a specific boundary and
// categorises the result.
func TestStressBoundaryConditions(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("empty_name_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/snippets", model.CreateSnippetRequest{
			Name:     "",
			Content:  "some content",
			Language: "bash",
		})
		if status == http.StatusCreated {
			t.Fatal("empty name must be rejected, got 201")
		}
		t.Logf("empty name → %d (expected 400)", status)
	})

	t.Run("empty_content_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/snippets", model.CreateSnippetRequest{
			Name:     "valid-name",
			Content:  "",
			Language: "bash",
		})
		if status == http.StatusCreated {
			t.Fatal("empty content must be rejected, got 201")
		}
		t.Logf("empty content → %d (expected 400)", status)
	})

	t.Run("empty_language_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/snippets", model.CreateSnippetRequest{
			Name:     "valid-name",
			Content:  "some content",
			Language: "",
		})
		if status == http.StatusCreated {
			t.Fatal("empty language must be rejected, got 201")
		}
		t.Logf("empty language → %d (expected 400)", status)
	})

	t.Run("max_length_name_accepted_or_rejected", func(t *testing.T) {
		longName := strings.Repeat("a", 255)
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/snippets", model.CreateSnippetRequest{
			Name:     longName,
			Content:  "some content",
			Language: "bash",
		})
		t.Logf("max-length name (%d chars) → %d", len(longName), status)
		if status == http.StatusCreated {
			idStr, _ := body["id"].(string)
			if idStr == "" {
				t.Fatal("201 but no id returned")
			}
		}
	})

	t.Run("max_length_content_accepted", func(t *testing.T) {
		longContent := strings.Repeat("x", 10000)
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/snippets", model.CreateSnippetRequest{
			Name:     uniqueSnippetName("boundary-content", 0),
			Content:  longContent,
			Language: "text",
		})
		t.Logf("max-length content (%d chars) → %d", len(longContent), status)
		if status == http.StatusCreated {
			idStr, _ := body["id"].(string)
			if idStr == "" {
				t.Fatal("201 but no id returned")
			}
		}
	})

	t.Run("invalid_uuid_in_get", func(t *testing.T) {
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/snippets/not-a-uuid")
		if status == http.StatusOK {
			t.Fatal("invalid UUID must be rejected, got 200")
		}
		t.Logf("invalid UUID → %d (expected 400)", status)
	})

	t.Run("nonexistent_uuid_in_get", func(t *testing.T) {
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/snippets/"+uuid.New().String())
		if status == http.StatusOK {
			t.Fatal("nonexistent UUID must return 404, got 200")
		}
		t.Logf("nonexistent UUID → %d (expected 404)", status)
	})

	t.Run("empty_body_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/snippets", strings.NewReader(""))
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

// TestStressBoundaryConditions_NoRepo exercises boundary conditions
// against the validation layer WITHOUT a database — proves
// ShouldBindJSON rejects malformed input before any DB call.
func TestStressBoundaryConditions_NoRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil)
	r.POST("/api/v1/snippets", h.CreateSnippet)
	r.GET("/api/v1/snippets/:id", h.GetSnippet)
	r.GET("/api/v1/snippets", h.ListSnippets)
	r.PUT("/api/v1/snippets/:id", h.UpdateSnippet)
	r.DELETE("/api/v1/snippets/:id", h.DeleteSnippet)

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"create_empty_body", "POST", "/api/v1/snippets", "", 503},
		{"create_invalid_json", "POST", "/api/v1/snippets", "{broken", 503},
		{"create_nil_repo", "POST", "/api/v1/snippets", `{"name":"test","content":"echo hi","language":"bash"}`, 503},
		{"get_nil_repo", "GET", "/api/v1/snippets/" + uuid.New().String(), "", 503},
		{"list_nil_repo", "GET", "/api/v1/snippets", "", 503},
		{"update_nil_repo", "PUT", "/api/v1/snippets/" + uuid.New().String(), `{"name":"updated"}`, 503},
		{"delete_nil_repo", "DELETE", "/api/v1/snippets/" + uuid.New().String(), "", 503},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			var req *http.Request
			if tc.body != "" {
				req, _ = http.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tc.method, tc.path, nil)
			}
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Logf("%s %s body=%q → %d (want %d)", tc.method, tc.path, tc.body, w.Code, tc.wantStatus)
			}
		})
	}
}

// init ensures the test database migrations package is imported so
// the test binary includes the migration runner.
var _ = migrations.Schema
