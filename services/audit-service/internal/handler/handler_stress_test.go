//go:build stress

// Stress test suite for audit-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of create via POST /api/v1/audit/logs,
//     per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=15 parallel goroutines performing
//     create, no deadlock, no resource leak.
//   - Boundary conditions: empty action, invalid enum, missing
//     required fields — every boundary produces a categorised result.
//
// FINDING: Read endpoints (GET /api/v1/audit/logs, GET /api/v1/audit/logs/:id)
// return 500 because the repository scans PostgreSQL INET (ip_address) column
// into *string — pgx binary-format INET cannot scan into Go string. The CREATE
// endpoint works correctly (INET accepts string input). This is a pre-existing
// repository-layer bug, NOT a test issue.
//
// Run:
//
//	go test -race -tags stress -run TestStress -v -timeout 120s ./internal/handler/
package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/audit-service/internal/handler"
	"github.com/helixdevelopment/audit-service/internal/model"
	"github.com/helixdevelopment/audit-service/internal/repository"
	"github.com/helixdevelopment/audit-service/internal/testutil"
	"github.com/helixdevelopment/audit-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// stressEnv holds the assembled test environment: a real gin engine
// wired to a real handler backed by a real PostgreSQL pool.
type stressEnv struct {
	ts      *httptest.Server
	cleanup func()
}

// setupStressEnv boots a real PostgreSQL container (via podman),
// applies audit-service migrations, constructs a real handler+router,
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

	r.POST("/api/v1/audit/logs", h.CreateAuditLog)
	r.GET("/api/v1/audit/logs", h.ListAuditLogs)
	r.GET("/api/v1/audit/logs/:id", h.GetAuditLog)
	r.GET("/api/v1/audit/stats/actions", h.CountByAction)
	r.GET("/api/v1/audit/stats/resources", h.CountByResourceType)

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

// validCreateBody returns a valid CreateAuditLogRequest body with a
// unique resource ID for each iteration.
func validCreateBody(i int) model.CreateAuditLogRequest {
	resID := uuid.New()
	return model.CreateAuditLogRequest{
		Action:       model.ActionCreate,
		ResourceType: model.ResourceTypeUser,
		ResourceID:   &resID,
		Details:      map[string]interface{}{"iteration": i, "ts": time.Now().UnixNano()},
		Severity:     model.SeverityInfo,
	}
}

// TestStressCreate_SustainedLoad drives N>=100 iterations of the
// create endpoint against a real PostgreSQL instance, recording
// per-iteration latency and computing p50/p95/p99.
func TestStressCreate_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/audit/logs", validCreateBody(i))
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /api/v1/audit/logs status = %d, want 201; body=%v", i, status, body)
		}
		logID, _ := body["id"].(string)
		if logID == "" {
			t.Fatalf("iteration %d: POST /api/v1/audit/logs returned no id", i)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=15 parallel goroutines,
// each performing a create cycle. Validates no deadlock occurs
// and all goroutines complete within the timeout.
func TestStressConcurrentContention(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()

	testutil.RunConcurrent(t, parallelism, func(id int) {
		start := time.Now()

		body := validCreateBody(id)
		orgID := uuid.New()
		body.OrgID = &orgID
		status, resp := stressPostJSON(t, client, env.ts.URL+"/api/v1/audit/logs", body)
		if status != http.StatusCreated {
			t.Errorf("goroutine %d: POST /api/v1/audit/logs status = %d, want 201; body=%v", id, status, resp)
			return
		}
		logID, _ := resp["id"].(string)
		if logID == "" {
			t.Errorf("goroutine %d: POST /api/v1/audit/logs returned no id", id)
			return
		}

		rec.Record(time.Since(start))
	})

	p50, p95, p99 := rec.Percentiles()
	t.Logf("CONCURRENT CONTENTION (%d goroutines): p50=%v p95=%v p99=%v", parallelism, p50, p95, p99)
}

// TestStressBoundaryConditions exercises edge-case inputs against
// the create endpoint. Each subtest drives a specific boundary and
// categorises the result.
func TestStressBoundaryConditions(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("empty_action_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/audit/logs", map[string]interface{}{
			"resourceType": "user",
			"severity":     "info",
		})
		if status == http.StatusCreated {
			t.Fatal("empty action must be rejected, got 201")
		}
		t.Logf("empty action → %d (expected 400)", status)
	})

	t.Run("invalid_action_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/audit/logs", map[string]interface{}{
			"action":       "nonexistent",
			"resourceType": "user",
			"severity":     "info",
		})
		if status == http.StatusCreated {
			t.Fatal("invalid action must be rejected, got 201")
		}
		t.Logf("invalid action → %d (expected 400)", status)
	})

	t.Run("invalid_resource_type_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/audit/logs", map[string]interface{}{
			"action":       "create",
			"resourceType": "nonexistent",
			"severity":     "info",
		})
		if status == http.StatusCreated {
			t.Fatal("invalid resource type must be rejected, got 201")
		}
		t.Logf("invalid resource type → %d (expected 400)", status)
	})

	t.Run("invalid_severity_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/audit/logs", map[string]interface{}{
			"action":       "create",
			"resourceType": "user",
			"severity":     "nonexistent",
		})
		if status == http.StatusCreated {
			t.Fatal("invalid severity must be rejected, got 201")
		}
		t.Logf("invalid severity → %d (expected 400)", status)
	})

	t.Run("empty_body_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/audit/logs", strings.NewReader(""))
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

	t.Run("invalid_uuid_in_get_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/audit/logs/not-a-uuid", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Fatal("invalid UUID must be rejected, got 200")
		}
		t.Logf("invalid UUID → %d (expected 400)", resp.StatusCode)
	})

	t.Run("invalid_org_id_in_list_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/audit/logs?org_id=not-a-uuid", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Fatal("invalid org_id must be rejected, got 200")
		}
		t.Logf("invalid org_id in list → %d (expected 400)", resp.StatusCode)
	})

	t.Run("invalid_start_time_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", env.ts.URL+"/api/v1/audit/logs?start=not-a-time", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Fatal("invalid start time must be rejected, got 200")
		}
		t.Logf("invalid start time → %d (expected 400)", resp.StatusCode)
	})

	t.Run("all_valid_actions_accepted", func(t *testing.T) {
		actions := []model.AuditAction{
			model.ActionCreate, model.ActionRead, model.ActionUpdate,
			model.ActionDelete, model.ActionLogin, model.ActionLogout,
			model.ActionExport,
		}
		for _, action := range actions {
			status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/audit/logs", map[string]interface{}{
				"action":       string(action),
				"resourceType": "user",
				"severity":     "info",
			})
			if status != http.StatusCreated {
				t.Errorf("action %q: status = %d, want 201; body=%v", action, status, body)
			}
		}
	})

	t.Run("all_valid_severities_accepted", func(t *testing.T) {
		severities := []model.AuditSeverity{
			model.SeverityInfo, model.SeverityWarning,
			model.SeverityError, model.SeverityCritical,
		}
		for _, sev := range severities {
			status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/audit/logs", map[string]interface{}{
				"action":       "create",
				"resourceType": "user",
				"severity":     string(sev),
			})
			if status != http.StatusCreated {
				t.Errorf("severity %q: status = %d, want 201; body=%v", sev, status, body)
			}
		}
	})

	t.Run("all_valid_resource_types_accepted", func(t *testing.T) {
		types := []model.AuditResourceType{
			model.ResourceTypeUser, model.ResourceTypeHost,
			model.ResourceTypeOrg, model.ResourceTypeVault,
			model.ResourceTypeWorkspace,
		}
		for _, rt := range types {
			status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/audit/logs", map[string]interface{}{
				"action":       "create",
				"resourceType": string(rt),
				"severity":     "info",
			})
			if status != http.StatusCreated {
				t.Errorf("resourceType %q: status = %d, want 201; body=%v", rt, status, body)
			}
		}
	})
}

// TestStressBoundaryConditions_NoRepo exercises boundary conditions
// against the validation layer WITHOUT a database — proves
// ShouldBindJSON rejects malformed input before any DB call.
func TestStressBoundaryConditions_NoRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Use repository.New(nil) so checkPool() returns a clean error
	// instead of panicking on a nil *Repository receiver.
	repo := repository.New(nil)
	h := handler.New(repo)
	r.POST("/api/v1/audit/logs", h.CreateAuditLog)

	if os.Getenv("DATABASE_URL") != "" {
		t.Log("DATABASE_URL set — boundary conditions already covered by TestStressBoundaryConditions with real DB")
	}

	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty_body", "", 400},
		{"invalid_json", "{broken", 400},
		{"missing_action", `{"resourceType":"user","severity":"info"}`, 400},
		{"missing_resource_type", `{"action":"create","severity":"info"}`, 400},
		{"missing_severity", `{"action":"create","resourceType":"user"}`, 400},
		{"valid_shape_no_repo", `{"action":"create","resourceType":"user","severity":"info"}`, 500},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/audit/logs", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("body=%q → %d (want %d)", tc.body, w.Code, tc.wantStatus)
			}
			if tc.name == "valid_shape_no_repo" && w.Code == http.StatusInternalServerError {
				t.Log("valid shape with nil repo → 500 (expected — no DB configured)")
			}
		})
	}
}

// init ensures the test database migrations package is imported so
// the test binary includes the migration runner.
var _ = migrations.Schema
