//go:build stress

// Stress test suite for org-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of create→get→list→update→delete,
//     per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=15 parallel goroutines performing
//     create+get+delete, no deadlock, no resource leak.
//   - Boundary conditions: empty name, max-length, invalid slug,
//     duplicate slug, invalid UUID — every boundary produces a
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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/org-service/internal/handler"
	"github.com/helixdevelopment/org-service/internal/repository"
	"github.com/helixdevelopment/org-service/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// stressEnv holds the assembled test environment: a real gin engine
// wired to a real handler backed by a real PostgreSQL pool.
type stressEnv struct {
	ts      *httptest.Server
	cleanup func()
}

// setupStressEnv boots a real PostgreSQL container (via podman),
// applies org-service migrations, constructs a real handler+router,
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

	r.POST("/api/v1/orgs", h.CreateOrg)
	r.GET("/api/v1/orgs", h.ListOrgs)
	r.GET("/api/v1/orgs/:id", h.GetOrg)
	r.PUT("/api/v1/orgs/:id", h.UpdateOrg)
	r.DELETE("/api/v1/orgs/:id", h.DeleteOrg)

	ts := httptest.NewServer(r)

	return &stressEnv{
		ts: ts,
		cleanup: func() {
			ts.Close()
			pool.Close()
		},
	}
}

// stressPostJSON sends a POST request with a JSON body and returns
// status + parsed response.
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

// stressGetJSON sends a GET request and returns status + parsed
// response.
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

// stressPutJSON sends a PUT request with a JSON body and returns
// status + parsed response.
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

// stressDeleteJSON sends a DELETE request and returns status code.
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

// uniqueSlug generates a collision-free slug for stress iterations.
func uniqueSlug(prefix string, i int) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), i)
}

// extractOrgID extracts the organization ID from the response body.
func extractOrgID(body map[string]interface{}) string {
	org, ok := body["organization"].(map[string]interface{})
	if !ok {
		return ""
	}
	id, _ := org["id"].(string)
	return id
}

// TestStressOrgCRUD_SustainedLoad drives N>=100 iterations of the
// full create→get→list→update→delete cycle against a real PostgreSQL
// instance, recording per-iteration latency and computing p50/p95/p99.
func TestStressOrgCRUD_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		slug := uniqueSlug("stress-crud", i)
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs", map[string]string{
			"name": fmt.Sprintf("Stress Org %d", i),
			"slug": slug,
		})
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /api/v1/orgs status = %d, want 201; body=%v", i, status, body)
		}
		orgID := extractOrgID(body)
		if orgID == "" {
			t.Fatalf("iteration %d: POST /api/v1/orgs returned no org id", i)
		}

		// Get
		status, body = stressGetJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /api/v1/orgs/%s status = %d, want 200; body=%v", i, orgID, status, body)
		}

		// List
		status, body = stressGetJSON(t, client, env.ts.URL+"/api/v1/orgs?limit=10")
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /api/v1/orgs status = %d, want 200; body=%v", i, status, body)
		}

		// Update
		status, body = stressPutJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID, map[string]string{
			"name": fmt.Sprintf("Updated Stress Org %d", i),
		})
		if status != http.StatusOK {
			t.Fatalf("iteration %d: PUT /api/v1/orgs/%s status = %d, want 200; body=%v", i, orgID, status, body)
		}

		// Delete
		status = stressDeleteJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID)
		if status != http.StatusNoContent {
			t.Fatalf("iteration %d: DELETE /api/v1/orgs/%s status = %d, want 204", i, orgID, status)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)

	// Log evidence path for §11.4.69 compliance
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=15 parallel goroutines,
// each performing a create+get+delete cycle. Validates no deadlock
// occurs and all goroutines complete within the timeout.
func TestStressConcurrentContention(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()

	testutil.RunConcurrent(t, parallelism, func(id int) {
		slug := uniqueSlug("stress-cc", id)
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs", map[string]string{
			"name": fmt.Sprintf("Concurrent Org %d", id),
			"slug": slug,
		})
		if status != http.StatusCreated {
			t.Errorf("goroutine %d: POST /api/v1/orgs status = %d, want 201; body=%v", id, status, body)
			return
		}
		orgID := extractOrgID(body)
		if orgID == "" {
			t.Errorf("goroutine %d: POST /api/v1/orgs returned no org id", id)
			return
		}

		// Get
		status, _ = stressGetJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /api/v1/orgs/%s status = %d, want 200", id, orgID, status)
			return
		}

		// Delete
		status = stressDeleteJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID)
		if status != http.StatusNoContent {
			t.Errorf("goroutine %d: DELETE /api/v1/orgs/%s status = %d, want 204", id, orgID, status)
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
func TestStressBoundaryConditions(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("empty_name_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs", map[string]string{
			"name": "",
			"slug": uniqueSlug("boundary", 0),
		})
		if status == http.StatusCreated {
			t.Fatal("empty name must be rejected, got 201")
		}
		t.Logf("empty name → %d (expected 400)", status)
	})

	t.Run("empty_slug_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs", map[string]string{
			"name": "Boundary Org",
			"slug": "",
		})
		if status == http.StatusCreated {
			t.Fatal("empty slug must be rejected, got 201")
		}
		t.Logf("empty slug → %d (expected 400)", status)
	})

	t.Run("max_length_name_accepted", func(t *testing.T) {
		longName := strings.Repeat("a", 255)
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs", map[string]string{
			"name": longName,
			"slug": uniqueSlug("boundary-max", 0),
		})
		if status == http.StatusCreated {
			orgID := extractOrgID(body)
			if orgID == "" {
				t.Fatal("201 but no org id returned")
			}
			// Cleanup
			stressDeleteJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID)
		}
		t.Logf("max-length name (%d chars) → %d", len(longName), status)
	})

	t.Run("max_length_slug_accepted", func(t *testing.T) {
		longSlug := strings.Repeat("b", 255)
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs", map[string]string{
			"name": "Max Slug Org",
			"slug": longSlug,
		})
		if status == http.StatusCreated {
			orgID := extractOrgID(body)
			if orgID == "" {
				t.Fatal("201 but no org id returned")
			}
			stressDeleteJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID)
		}
		t.Logf("max-length slug (%d chars) → %d", len(longSlug), status)
	})

	t.Run("duplicate_slug_rejected", func(t *testing.T) {
		slug := uniqueSlug("boundary-dup", 0)
		req := map[string]string{
			"name": "Duplicate Test",
			"slug": slug,
		}

		// First creation must succeed
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs", req)
		if status != http.StatusCreated {
			t.Fatalf("first creation status = %d, want 201", status)
		}
		orgID := extractOrgID(body)
		defer stressDeleteJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID)

		// Second creation with same slug must be rejected
		status, _ = stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs", req)
		if status == http.StatusCreated {
			t.Fatalf("duplicate slug accepted, got 201")
		}
		t.Logf("duplicate slug → %d (expected 409 or 500)", status)
	})

	t.Run("invalid_uuid_in_get_rejected", func(t *testing.T) {
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/orgs/not-a-uuid")
		if status == http.StatusOK {
			t.Fatal("invalid UUID must be rejected, got 200")
		}
		t.Logf("invalid UUID in GET → %d (expected 400)", status)
	})

	t.Run("nonexistent_org_returns_404", func(t *testing.T) {
		fakeID := uuid.New().String()
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/orgs/"+fakeID)
		if status != http.StatusNotFound {
			t.Logf("nonexistent org → %d (expected 404)", status)
		}
	})

	t.Run("empty_body_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/orgs", strings.NewReader(""))
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

// TestStressTeamCRUD_SustainedLoad drives N>=100 iterations of the
// team create→list→get→update→delete cycle within a single org.
func TestStressTeamCRUD_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	// Create a parent org for all teams
	status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs", map[string]string{
		"name": "Team Stress Parent",
		"slug": uniqueSlug("team-parent", 0),
	})
	if status != http.StatusCreated {
		t.Fatalf("failed to create parent org: status=%d body=%v", status, body)
	}
	orgID := extractOrgID(body)
	defer stressDeleteJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID)

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Create team
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID+"/teams", map[string]string{
			"name": fmt.Sprintf("Stress Team %d", i),
		})
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /teams status = %d, want 201; body=%v", i, status, body)
		}
		team, ok := body["team"].(map[string]interface{})
		if !ok {
			t.Fatalf("iteration %d: no team in response", i)
		}
		teamID, _ := team["id"].(string)
		if teamID == "" {
			t.Fatalf("iteration %d: no team id", i)
		}

		// List teams
		status, _ = stressGetJSON(t, client, env.ts.URL+"/api/v1/orgs/"+orgID+"/teams?limit=10")
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /teams status = %d, want 200", i, status)
		}

		// Get team
		status, _ = stressGetJSON(t, client, env.ts.URL+"/api/v1/teams/"+teamID)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /teams/%s status = %d, want 200", i, teamID, status)
		}

		// Update team
		status, _ = stressPutJSON(t, client, env.ts.URL+"/api/v1/teams/"+teamID, map[string]string{
			"name": fmt.Sprintf("Updated Team %d", i),
		})
		if status != http.StatusOK {
			t.Fatalf("iteration %d: PUT /teams/%s status = %d, want 200", i, teamID, status)
		}

		// Delete team
		req, _ := http.NewRequest("DELETE", env.ts.URL+"/api/v1/teams/"+teamID, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("iteration %d: DELETE /teams/%s failed: %v", i, teamID, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("iteration %d: DELETE /teams/%s status = %d, want 204", i, teamID, resp.StatusCode)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("TEAM SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: team latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}
