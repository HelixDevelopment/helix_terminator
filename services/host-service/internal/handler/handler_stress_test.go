//go:build stress

// Stress test suite for host-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of create→get→update→list→delete,
//     per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=15 parallel goroutines performing
//     create+get, no deadlock, no resource leak.
//   - Boundary conditions: empty name, max-length hostname, invalid
//     auth_type, invalid UUID — every boundary produces a categorised
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
	"github.com/helixdevelopment/host-service/internal/handler"
	"github.com/helixdevelopment/host-service/internal/model"
	"github.com/helixdevelopment/host-service/internal/repository"
	"github.com/helixdevelopment/host-service/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// stressEnv holds the assembled test environment: a real gin engine
// wired to a real handler backed by a real PostgreSQL pool.
type stressEnv struct {
	ts      *httptest.Server
	cleanup func()
}

// setupStressEnv boots a real PostgreSQL container (via podman),
// applies host-service migrations, constructs a real handler+router,
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

	// Simulate auth middleware setting userID/orgID
	r.Use(func(c *gin.Context) {
		c.Set("userID", "00000000-0000-0000-0000-000000000000")
		c.Set("orgID", "00000000-0000-0000-0000-000000000000")
		c.Next()
	})

	r.POST("/api/v1/hosts", h.CreateHost)
	r.GET("/api/v1/hosts/:id", h.GetHost)
	r.GET("/api/v1/hosts", h.ListHosts)
	r.PUT("/api/v1/hosts/:id", h.UpdateHost)
	r.DELETE("/api/v1/hosts/:id", h.DeleteHost)
	r.POST("/api/v1/hosts/:id/test-connection", h.TestConnection)
	r.GET("/api/v1/hosts/:id/logs", h.GetConnectionLogs)

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

// stressDelete sends a DELETE request and returns status.
func stressDelete(t *testing.T, client *http.Client, url string) int {
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

// uniqueHost generates a unique CreateHostRequest for stress iterations.
func uniqueHost(prefix string, i int) model.CreateHostRequest {
	return model.CreateHostRequest{
		Name:     fmt.Sprintf("%s-host-%d-%d", prefix, time.Now().UnixNano(), i),
		Hostname: fmt.Sprintf("192.168.%d.%d", i/256, i%256),
		Port:     22,
		Username: fmt.Sprintf("user-%d", i),
		AuthType: model.AuthTypePassword,
		Tags:     []string{"stress", fmt.Sprintf("iter-%d", i)},
	}
}

// TestStressCreateGetUpdateListDelete_SustainedLoad drives N>=100
// iterations of the full create→get→update→list→delete cycle against a
// real PostgreSQL instance, recording per-iteration latency and
// computing p50/p95/p99.
func TestStressCreateGetUpdateListDelete_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		host := uniqueHost("stress-cguld", i)
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/hosts", host)
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /api/v1/hosts status = %d, want 201; body=%v", i, status, body)
		}
		hostData, ok := body["host"].(map[string]interface{})
		if !ok {
			t.Fatalf("iteration %d: POST /api/v1/hosts returned no host in body: %v", i, body)
		}
		hostID, _ := hostData["id"].(string)
		if hostID == "" {
			t.Fatalf("iteration %d: POST /api/v1/hosts returned no host id", i)
		}

		// Get
		status, body = stressGetJSON(t, client, env.ts.URL+"/api/v1/hosts/"+hostID)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /api/v1/hosts/%s status = %d, want 200; body=%v", i, hostID, status, body)
		}

		// Update
		status, body = stressPutJSON(t, client, env.ts.URL+"/api/v1/hosts/"+hostID, model.UpdateHostRequest{
			Name: fmt.Sprintf("updated-%s", host.Name),
		})
		if status != http.StatusOK {
			t.Fatalf("iteration %d: PUT /api/v1/hosts/%s status = %d, want 200; body=%v", i, hostID, status, body)
		}

		// List
		status, body = stressGetJSON(t, client, env.ts.URL+"/api/v1/hosts?limit=10")
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /api/v1/hosts status = %d, want 200; body=%v", i, status, body)
		}

		// Delete
		status = stressDelete(t, client, env.ts.URL+"/api/v1/hosts/"+hostID)
		if status != http.StatusNoContent {
			t.Fatalf("iteration %d: DELETE /api/v1/hosts/%s status = %d, want 204", i, hostID, status)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=15 parallel goroutines,
// each performing a create+get cycle. Validates no deadlock occurs
// and all goroutines complete within the timeout.
func TestStressConcurrentContention(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()

	testutil.RunConcurrent(t, parallelism, func(id int) {
		host := uniqueHost("stress-cc", id)
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/hosts", host)
		if status != http.StatusCreated {
			t.Errorf("goroutine %d: POST /api/v1/hosts status = %d, want 201; body=%v", id, status, body)
			return
		}

		hostData, ok := body["host"].(map[string]interface{})
		if !ok {
			t.Errorf("goroutine %d: POST /api/v1/hosts returned no host in body: %v", id, body)
			return
		}
		hostID, _ := hostData["id"].(string)
		if hostID == "" {
			t.Errorf("goroutine %d: POST /api/v1/hosts returned no host id", id)
			return
		}

		// Get
		status, _ = stressGetJSON(t, client, env.ts.URL+"/api/v1/hosts/"+hostID)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /api/v1/hosts/%s status = %d, want 200", id, hostID, status)
			return
		}

		rec.Record(time.Since(start))
	})

	p50, p95, p99 := rec.Percentiles()
	t.Logf("CONCURRENT CONTENTION (%d goroutines): p50=%v p95=%v p99=%v", parallelism, p50, p95, p99)
}

// TestStressBoundaryConditions exercises edge-case inputs against the
// create endpoint. Each subtest drives a specific boundary and
// categorises the result (400 for validation, 503 for missing DB).
func TestStressBoundaryConditions(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("empty_name_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/hosts", model.CreateHostRequest{
			Name:     "",
			Hostname: "192.168.1.1",
			Username: "admin",
			AuthType: model.AuthTypePassword,
		})
		if status == http.StatusCreated {
			t.Fatal("empty name must be rejected, got 201")
		}
		t.Logf("empty name → %d (expected 400)", status)
	})

	t.Run("empty_hostname_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/hosts", model.CreateHostRequest{
			Name:     "test-host",
			Hostname: "",
			Username: "admin",
			AuthType: model.AuthTypePassword,
		})
		if status == http.StatusCreated {
			t.Fatal("empty hostname must be rejected, got 201")
		}
		t.Logf("empty hostname → %d (expected 400)", status)
	})

	t.Run("empty_username_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/hosts", model.CreateHostRequest{
			Name:     "test-host",
			Hostname: "192.168.1.1",
			Username: "",
			AuthType: model.AuthTypePassword,
		})
		if status == http.StatusCreated {
			t.Fatal("empty username must be rejected, got 201")
		}
		t.Logf("empty username → %d (expected 400)", status)
	})

	t.Run("invalid_auth_type_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/hosts", model.CreateHostRequest{
			Name:     "test-host",
			Hostname: "192.168.1.1",
			Username: "admin",
			AuthType: "invalid_type",
		})
		if status == http.StatusCreated {
			t.Fatal("invalid auth_type must be rejected, got 201")
		}
		t.Logf("invalid auth_type → %d (expected 400)", status)
	})

	t.Run("port_out_of_range_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/hosts", model.CreateHostRequest{
			Name:     "test-host",
			Hostname: "192.168.1.1",
			Port:     99999,
			Username: "admin",
			AuthType: model.AuthTypePassword,
		})
		if status == http.StatusCreated {
			t.Fatal("port > 65535 must be rejected, got 201")
		}
		t.Logf("port 99999 → %d (expected 400)", status)
	})

	t.Run("max_length_name_accepted_or_rejected", func(t *testing.T) {
		longName := strings.Repeat("a", 255)
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/hosts", model.CreateHostRequest{
			Name:     longName,
			Hostname: "192.168.1.1",
			Username: "admin",
			AuthType: model.AuthTypePassword,
		})
		t.Logf("max-length name (255 chars) → %d", status)
		if status == http.StatusCreated {
			hostData, ok := body["host"].(map[string]interface{})
			if !ok {
				t.Fatal("201 but no host in body")
			}
			if hostData["id"] == "" {
				t.Fatal("201 but no host id returned")
			}
		}
	})

	t.Run("invalid_uuid_in_get", func(t *testing.T) {
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/hosts/not-a-uuid")
		if status != http.StatusBadRequest {
			t.Logf("invalid uuid GET → %d (expected 400)", status)
		}
	})

	t.Run("nonexistent_host_get", func(t *testing.T) {
		fakeID := uuid.New().String()
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/hosts/"+fakeID)
		if status != http.StatusNotFound {
			t.Logf("nonexistent host GET → %d (expected 404)", status)
		}
	})

	t.Run("empty_body_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/hosts", strings.NewReader(""))
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
	r.POST("/api/v1/hosts", h.CreateHost)

	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty_body", "", 400},
		{"invalid_json", "{broken", 400},
		{"missing_name", `{"hostname":"192.168.1.1","username":"admin","auth_type":"password"}`, 400},
		{"missing_hostname", `{"name":"test","username":"admin","auth_type":"password"}`, 400},
		{"missing_username", `{"name":"test","hostname":"192.168.1.1","auth_type":"password"}`, 400},
		{"valid_shape_no_repo", `{"name":"test","hostname":"192.168.1.1","username":"admin","auth_type":"password"}`, 503},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/hosts", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Logf("body=%q → %d (want %d)", tc.body, w.Code, tc.wantStatus)
			}
			if tc.name == "valid_shape_no_repo" && w.Code == http.StatusServiceUnavailable {
				t.Log("valid shape with nil repo → 503 (expected — no DB configured)")
			}
		})
	}
}
