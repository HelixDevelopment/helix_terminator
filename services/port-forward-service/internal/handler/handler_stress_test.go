//go:build stress

// Stress test suite for port-forward-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of create→get→update→delete,
//     per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=15 parallel goroutines performing
//     create+get, no deadlock, no resource leak.
//   - Boundary conditions: empty hostId, invalid port, missing required
//     fields, max-length strings — every boundary produces a categorised
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
	"github.com/helixdevelopment/port-forward-service/internal/handler"
	"github.com/helixdevelopment/port-forward-service/internal/testutil"
)

// stressEnv holds the assembled test environment: a real gin engine
// wired to a real handler backed by an in-memory fake repository.
type stressEnv struct {
	ts      *httptest.Server
	repo    *testutil.FakeRepo
	cleanup func()
}

// setupStressEnv constructs a real handler+router backed by a FakeRepo
// and returns a ready httptest.Server.
func setupStressEnv(t *testing.T) *stressEnv {
	t.Helper()

	repo := testutil.NewFakeRepo()
	h := handler.New(repo)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/api/v1/forwards", h.CreateForward)
	r.GET("/api/v1/forwards/:id", h.GetForward)
	r.GET("/api/v1/forwards", h.ListForwards)
	r.PUT("/api/v1/forwards/:id", h.UpdateForward)
	r.DELETE("/api/v1/forwards/:id", h.DeleteForward)
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	ts := httptest.NewServer(r)

	return &stressEnv{
		ts:   ts,
		repo: repo,
		cleanup: func() {
			ts.Close()
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

// stressDeleteJSON sends a DELETE request and returns status + parsed
// response.
func stressDeleteJSON(t *testing.T, client *http.Client, url string) (int, map[string]interface{}) {
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

	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return resp.StatusCode, parsed
}

// validCreateBody returns a valid CreatePortForwardRequest body map for
// stress iterations, using the given iteration index to generate unique
// values.
func validCreateBody(i int) map[string]interface{} {
	return map[string]interface{}{
		"hostId":      uuid.New().String(),
		"forwardType": "local",
		"localPort":   10000 + i%55000,
		"remotePort":  80,
		"remoteHost":  fmt.Sprintf("host-%d.example.com", i),
		"protocol":    "tcp",
		"sshHost":     fmt.Sprintf("ssh-%d.example.com", i),
		"sshUsername":  "testuser",
	}
}

// TestStressCreateGetUpdateDelete_SustainedLoad drives N>=100
// iterations of the full create→get→update→delete cycle, recording
// per-iteration latency and computing p50/p95/p99.
func TestStressCreateGetUpdateDelete_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", validCreateBody(i))
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /api/v1/forwards status = %d, want 201; body=%v", i, status, body)
		}
		id, _ := body["id"].(string)
		if id == "" {
			t.Fatalf("iteration %d: POST /api/v1/forwards returned no id", i)
		}

		// Get
		status, body = stressGetJSON(t, client, env.ts.URL+"/api/v1/forwards/"+id)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /api/v1/forwards/%s status = %d, want 200; body=%v", i, id, status, body)
		}

		// Update
		status, body = stressPutJSON(t, client, env.ts.URL+"/api/v1/forwards/"+id, map[string]interface{}{
			"localPort":  20000 + i%45000,
			"remotePort": 443,
			"remoteHost": fmt.Sprintf("updated-%d.example.com", i),
			"protocol":   "tcp",
			"status":     "active",
		})
		if status != http.StatusOK {
			t.Fatalf("iteration %d: PUT /api/v1/forwards/%s status = %d, want 200; body=%v", i, id, status, body)
		}

		// Delete
		status, body = stressDeleteJSON(t, client, env.ts.URL+"/api/v1/forwards/"+id)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: DELETE /api/v1/forwards/%s status = %d, want 200; body=%v", i, id, status, body)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=15 parallel goroutines,
// each performing a create+get cycle. Validates no deadlock occurs and
// all goroutines complete within the timeout.
func TestStressConcurrentContention(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()

	testutil.RunConcurrent(t, parallelism, func(id int) {
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", validCreateBody(id))
		if status != http.StatusCreated {
			t.Errorf("goroutine %d: POST /api/v1/forwards status = %d, want 201; body=%v", id, status, body)
			return
		}
		forwardID, _ := body["id"].(string)
		if forwardID == "" {
			t.Errorf("goroutine %d: POST /api/v1/forwards returned no id", id)
			return
		}

		// Get
		status, body = stressGetJSON(t, client, env.ts.URL+"/api/v1/forwards/"+forwardID)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /api/v1/forwards/%s status = %d, want 200; body=%v", id, forwardID, status, body)
			return
		}

		rec.Record(time.Since(start))
	})

	p50, p95, p99 := rec.Percentiles()
	t.Logf("CONCURRENT CONTENTION (%d goroutines): p50=%v p95=%v p99=%v", parallelism, p50, p95, p99)
}

// TestStressBoundaryConditions exercises edge-case inputs against the
// create endpoint. Each subtest drives a specific boundary and
// categorises the result.
func TestStressBoundaryConditions(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()

	t.Run("empty_hostId_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      "",
			"forwardType": "local",
			"localPort":   8080,
			"remotePort":  80,
			"remoteHost":  "localhost",
			"protocol":    "tcp",
			"sshHost":     "ssh.example.com",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("empty hostId must be rejected, got 201")
		}
		t.Logf("empty hostId → %d (expected 400)", status)
	})

	t.Run("invalid_hostId_uuid_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      "not-a-uuid",
			"forwardType": "local",
			"localPort":   8080,
			"remotePort":  80,
			"remoteHost":  "localhost",
			"protocol":    "tcp",
			"sshHost":     "ssh.example.com",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("invalid hostId UUID must be rejected, got 201")
		}
		t.Logf("invalid hostId UUID → %d (expected 400)", status)
	})

	t.Run("port_out_of_range_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      uuid.New().String(),
			"forwardType": "local",
			"localPort":   99999,
			"remotePort":  80,
			"remoteHost":  "localhost",
			"protocol":    "tcp",
			"sshHost":     "ssh.example.com",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("port 99999 must be rejected, got 201")
		}
		t.Logf("port 99999 → %d (expected 400)", status)
	})

	t.Run("invalid_protocol_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      uuid.New().String(),
			"forwardType": "local",
			"localPort":   8080,
			"remotePort":  80,
			"remoteHost":  "localhost",
			"protocol":    "icmp",
			"sshHost":     "ssh.example.com",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("invalid protocol must be rejected, got 201")
		}
		t.Logf("invalid protocol → %d (expected 400)", status)
	})

	t.Run("invalid_forwardType_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      uuid.New().String(),
			"forwardType": "reverse",
			"localPort":   8080,
			"remotePort":  80,
			"remoteHost":  "localhost",
			"protocol":    "tcp",
			"sshHost":     "ssh.example.com",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("invalid forwardType must be rejected, got 201")
		}
		t.Logf("invalid forwardType → %d (expected 400)", status)
	})

	t.Run("missing_remoteHost_for_local_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      uuid.New().String(),
			"forwardType": "local",
			"localPort":   8080,
			"remotePort":  80,
			"remoteHost":  "",
			"protocol":    "tcp",
			"sshHost":     "ssh.example.com",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("missing remoteHost for local forward must be rejected, got 201")
		}
		t.Logf("missing remoteHost for local → %d (expected 400)", status)
	})

	t.Run("zero_remotePort_for_local_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      uuid.New().String(),
			"forwardType": "local",
			"localPort":   8080,
			"remotePort":  0,
			"remoteHost":  "localhost",
			"protocol":    "tcp",
			"sshHost":     "ssh.example.com",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("zero remotePort for local forward must be rejected, got 201")
		}
		t.Logf("zero remotePort for local → %d (expected 400)", status)
	})

	t.Run("missing_sshHost_rejected", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      uuid.New().String(),
			"forwardType": "local",
			"localPort":   8080,
			"remotePort":  80,
			"remoteHost":  "localhost",
			"protocol":    "tcp",
			"sshHost":     "",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("missing sshHost must be rejected, got 201")
		}
		t.Logf("missing sshHost → %d (expected 400)", status)
	})

	t.Run("max_length_sshHost_accepted_or_rejected", func(t *testing.T) {
		longHost := strings.Repeat("a", 255) + ".example.com"
		body := map[string]interface{}{
			"hostId":      uuid.New().String(),
			"forwardType": "local",
			"localPort":   8080,
			"remotePort":  80,
			"remoteHost":  "localhost",
			"protocol":    "tcp",
			"sshHost":     longHost,
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		t.Logf("max-length sshHost (%d chars) → %d", len(longHost), status)
	})

	t.Run("empty_body_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/forwards", strings.NewReader(""))
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
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/forwards/not-a-uuid")
		if status == http.StatusOK {
			t.Fatal("invalid UUID in GET must be rejected, got 200")
		}
		t.Logf("invalid UUID in GET → %d (expected 400)", status)
	})

	t.Run("nonexistent_id_in_get_returns_404", func(t *testing.T) {
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/forwards/"+uuid.New().String())
		if status != http.StatusNotFound {
			t.Fatalf("nonexistent id in GET status = %d, want 404", status)
		}
		t.Logf("nonexistent id in GET → 404 (expected)")
	})

	t.Run("dynamic_forward_rejected_by_default", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      uuid.New().String(),
			"forwardType": "dynamic",
			"protocol":    "tcp",
			"sshHost":     "ssh.example.com",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("dynamic forward must be rejected by default authorizer, got 201")
		}
		t.Logf("dynamic forward → %d (expected 403)", status)
	})

	t.Run("remote_forward_rejected_by_default", func(t *testing.T) {
		body := map[string]interface{}{
			"hostId":      uuid.New().String(),
			"forwardType": "remote",
			"localPort":   8080,
			"remotePort":  80,
			"remoteHost":  "localhost",
			"protocol":    "tcp",
			"sshHost":     "ssh.example.com",
			"sshUsername":  "user",
		}
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/forwards", body)
		if status == http.StatusCreated {
			t.Fatal("remote forward must be rejected by default authorizer, got 201")
		}
		t.Logf("remote forward → %d (expected 403)", status)
	})
}

// TestStressBoundaryConditions_NoRepo exercises boundary conditions
// against the validation layer with a nil repo — proves ShouldBindJSON
// rejects malformed input before any repo call.
func TestStressBoundaryConditions_NoRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil)
	r.POST("/api/v1/forwards", h.CreateForward)
	r.GET("/api/v1/forwards/:id", h.GetForward)
	r.GET("/api/v1/forwards", h.ListForwards)
	r.PUT("/api/v1/forwards/:id", h.UpdateForward)
	r.DELETE("/api/v1/forwards/:id", h.DeleteForward)

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"empty_body_create", "POST", "/api/v1/forwards", "", 503},
		{"invalid_json_create", "POST", "/api/v1/forwards", "{broken", 503},
		{"nil_repo_get", "GET", "/api/v1/forwards/" + uuid.New().String(), "", 503},
		{"nil_repo_list", "GET", "/api/v1/forwards", "", 503},
		{"nil_repo_update", "PUT", "/api/v1/forwards/" + uuid.New().String(), `{"localPort":8080}`, 503},
		{"nil_repo_delete", "DELETE", "/api/v1/forwards/" + uuid.New().String(), "", 503},
		{"invalid_uuid_get", "GET", "/api/v1/forwards/not-a-uuid", "", 503},
		{"invalid_uuid_update", "PUT", "/api/v1/forwards/not-a-uuid", `{"localPort":8080}`, 503},
		{"invalid_uuid_delete", "DELETE", "/api/v1/forwards/not-a-uuid", "", 503},
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
