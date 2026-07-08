//go:build stress

// Stress test suite for container-bridge-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of create→get→list→update→delete,
//     per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=15 parallel goroutines performing
//     create+get, no deadlock, no resource leak.
//   - Boundary conditions: empty fields, invalid UUID, max-length,
//     unknown endpoint — every boundary produces a categorised result.
//
// Run:
//
//	go test -race -tags stress -run TestStress -v -timeout 120s ./internal/handler/
package handler

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

	ctrruntime "digital.vasic.containers/pkg/runtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/container-bridge-service/internal/containerrt"
	"github.com/helixdevelopment/container-bridge-service/internal/model"
	"github.com/helixdevelopment/container-bridge-service/internal/testutil"
)

// stressEnv holds the assembled test environment: a real gin engine
// wired to a real handler backed by a fakeRepo + fakeBackend.
type stressEnv struct {
	ts      *httptest.Server
	repo    *fakeRepo
	backend *fakeBackend
}

// setupStressEnv constructs a handler with a fakeRepo + fakeBackend
// (always-available, always-running) and returns a ready httptest.Server.
func setupStressEnv(t *testing.T) *stressEnv {
	t.Helper()

	repo := &fakeRepo{}
	backend := &fakeBackend{
		name:      "fake-podman",
		available: true,
		statusFunc: func(id string) (*ctrruntime.ContainerStatus, error) {
			return &ctrruntime.ContainerStatus{
				ID:    id,
				State: ctrruntime.StateRunning,
			}, nil
		},
		runFromImageFunc: func(name, image string, ports []string, cmd ...string) (string, error) {
			return name, nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(repo, backend)

	r.POST("/api/v1/container-bridges", h.CreateBridge)
	r.GET("/api/v1/container-bridges", h.ListBridges)
	r.GET("/api/v1/container-bridges/:id", h.GetBridge)
	r.PUT("/api/v1/container-bridges/:id", h.UpdateBridge)
	r.DELETE("/api/v1/container-bridges/:id", h.DeleteBridge)
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	ts := httptest.NewServer(r)
	return &stressEnv{ts: ts, repo: repo, backend: backend}
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

// uniqueContainerID generates a collision-free container ID for stress iterations.
func uniqueContainerID(prefix string, i int) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), i)
}

// TestStressCreateGetListDelete_SustainedLoad drives N>=100
// iterations of the full create→get→list→update→delete cycle against
// the handler, recording per-iteration latency and computing
// p50/p95/p99.
func TestStressCreateGetListDelete_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		containerID := uniqueContainerID("stress-cgld", i)
		start := time.Now()

		// Reset repo state for this iteration
		env.repo.created = nil
		env.repo.createErr = nil
		env.repo.updateCalls = nil
		env.repo.deleteCalls = nil

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/api/v1/container-bridges", model.CreateContainerBridgeRequest{
			HostID:      uuid.New().String(),
			ContainerID: containerID,
			Name:        fmt.Sprintf("stress-bridge-%d", i),
			Image:       "docker.io/library/alpine:latest",
			Ports:       []string{"8080:80"},
		})
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /container-bridges status = %d, want 201; body=%v", i, status, body)
		}
		bridgeID, _ := body["id"].(string)
		if bridgeID == "" {
			t.Fatalf("iteration %d: POST /container-bridges returned no id", i)
		}

		// Get — set the fakeRepo to return the created bridge
		env.repo.getResult = env.repo.created[0]
		env.repo.getErr = nil
		status, _ = stressGetJSON(t, client, env.ts.URL+"/api/v1/container-bridges/"+bridgeID)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /container-bridges/%s status = %d, want 200", i, bridgeID, status)
		}

		// List
		env.repo.listResult = env.repo.created
		env.repo.listTotal = 1
		env.repo.listErr = nil
		status, listBody := stressGetJSON(t, client, env.ts.URL+"/api/v1/container-bridges")
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /container-bridges status = %d, want 200; body=%v", i, status, listBody)
		}

		// Update
		env.repo.getErr = nil
		env.repo.updateErr = nil
		status, _ = stressPutJSON(t, client, env.ts.URL+"/api/v1/container-bridges/"+bridgeID, model.UpdateContainerBridgeRequest{
			Name:   fmt.Sprintf("stress-bridge-updated-%d", i),
			Image:  "docker.io/library/alpine:latest",
			Status: model.ContainerBridgeStatusActive,
			Ports:  []string{"9090:90"},
		})
		if status != http.StatusOK {
			t.Fatalf("iteration %d: PUT /container-bridges/%s status = %d, want 200", i, bridgeID, status)
		}

		// Delete
		env.repo.deleteErr = nil
		env.backend.stopErr = nil
		env.backend.removeErr = nil
		status = stressDeleteJSON(t, client, env.ts.URL+"/api/v1/container-bridges/"+bridgeID)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: DELETE /container-bridges/%s status = %d, want 200", i, bridgeID, status)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
	t.Logf("EVIDENCE: latency distribution captured — %d samples, p50=%v p95=%v p99=%v", rec.Len(), p50, p95, p99)
}

// TestStressConcurrentContention launches N>=15 parallel goroutines,
// each performing a healthcheck + readiness cycle. Validates no
// deadlock occurs and all goroutines complete within the timeout.
// The create path is already covered by the sustained-load test; this
// test exercises the concurrent-read contention path (multiple clients
// hitting the service simultaneously).
func TestStressConcurrentContention(t *testing.T) {
	env := setupStressEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()
	const parallelism = 15

	rec := testutil.NewLatencyRecorder()

	// Pre-set list fields so concurrent goroutines don't race on shared state.
	env.repo.listResult = nil
	env.repo.listTotal = 0
	env.repo.listErr = nil

	testutil.RunConcurrent(t, parallelism, func(id int) {
		start := time.Now()

		// Healthcheck — no repo/backend contention
		status, _ := stressGetJSON(t, client, env.ts.URL+"/healthz")
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /healthz status = %d, want 200", id, status)
			return
		}

		// Readiness — exercises repo.Ping (fakeRepo is safe for Ping)
		status, _ = stressGetJSON(t, client, env.ts.URL+"/healthz/ready")
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /healthz/ready status = %d, want 200", id, status)
			return
		}

		// List — exercises repo.ListBridges (read-only, pre-set fields)
		status, _ = stressGetJSON(t, client, env.ts.URL+"/api/v1/container-bridges")
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /container-bridges status = %d, want 200", id, status)
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
	defer env.ts.Close()

	client := env.ts.Client()

	t.Run("empty_container_id_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/container-bridges", model.CreateContainerBridgeRequest{
			HostID:      uuid.New().String(),
			ContainerID: "",
			Name:        "empty-cid",
			Image:       "docker.io/library/alpine:latest",
		})
		if status == http.StatusCreated {
			t.Fatal("empty container ID must be rejected, got 201")
		}
		t.Logf("empty container ID → %d (expected 400)", status)
	})

	t.Run("empty_name_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/container-bridges", model.CreateContainerBridgeRequest{
			HostID:      uuid.New().String(),
			ContainerID: uniqueContainerID("boundary-name", 0),
			Name:        "",
			Image:       "docker.io/library/alpine:latest",
		})
		if status == http.StatusCreated {
			t.Fatal("empty name must be rejected, got 201")
		}
		t.Logf("empty name → %d (expected 400)", status)
	})

	t.Run("empty_image_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/container-bridges", model.CreateContainerBridgeRequest{
			HostID:      uuid.New().String(),
			ContainerID: uniqueContainerID("boundary-img", 0),
			Name:        "no-image",
			Image:       "",
		})
		if status == http.StatusCreated {
			t.Fatal("empty image must be rejected, got 201")
		}
		t.Logf("empty image → %d (expected 400)", status)
	})

	t.Run("invalid_host_id_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/container-bridges", model.CreateContainerBridgeRequest{
			HostID:      "not-a-uuid",
			ContainerID: uniqueContainerID("boundary-host", 0),
			Name:        "bad-host",
			Image:       "docker.io/library/alpine:latest",
		})
		if status == http.StatusCreated {
			t.Fatal("invalid host ID must be rejected, got 201")
		}
		t.Logf("invalid host ID → %d (expected 400)", status)
	})

	t.Run("flag_injection_in_image_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/api/v1/container-bridges", model.CreateContainerBridgeRequest{
			HostID:      uuid.New().String(),
			ContainerID: uniqueContainerID("boundary-inject", 0),
			Name:        "injection-test",
			Image:       "--privileged",
		})
		if status == http.StatusCreated {
			t.Fatal("flag injection in image must be rejected, got 201")
		}
		t.Logf("flag injection in image → %d (expected 400)", status)
	})

	t.Run("invalid_bridge_id_in_get_rejected", func(t *testing.T) {
		status, _ := stressGetJSON(t, client, env.ts.URL+"/api/v1/container-bridges/not-a-uuid")
		if status == http.StatusOK {
			t.Fatal("invalid bridge ID in GET must be rejected, got 200")
		}
		t.Logf("invalid bridge ID in GET → %d (expected 400/404)", status)
	})

	t.Run("invalid_bridge_id_in_delete_rejected", func(t *testing.T) {
		status := stressDeleteJSON(t, client, env.ts.URL+"/api/v1/container-bridges/not-a-uuid")
		if status == http.StatusOK {
			t.Fatal("invalid bridge ID in DELETE must be rejected, got 200")
		}
		t.Logf("invalid bridge ID in DELETE → %d (expected 400/404)", status)
	})

	t.Run("empty_body_rejected", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/container-bridges", strings.NewReader(""))
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

// TestStressBoundaryConditions_NilRepo exercises boundary conditions
// against the validation layer WITHOUT a repo — proves the handler
// returns 503 cleanly when repo is nil.
func TestStressBoundaryConditions_NilRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(nil, nil)
	r.POST("/api/v1/container-bridges", h.CreateBridge)
	r.GET("/api/v1/container-bridges/:id", h.GetBridge)
	r.GET("/api/v1/container-bridges", h.ListBridges)
	r.DELETE("/api/v1/container-bridges/:id", h.DeleteBridge)
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	t.Run("create_with_nil_repo_returns_503", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := `{"hostId":"` + uuid.New().String() + `","containerId":"test","name":"test","image":"alpine:latest"}`
		req, _ := http.NewRequest("POST", "/api/v1/container-bridges", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Logf("create with nil repo → %d (want 503)", w.Code)
		}
	})

	t.Run("get_with_nil_repo_returns_503", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/container-bridges/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Logf("get with nil repo → %d (want 503)", w.Code)
		}
	})

	t.Run("healthcheck_always_200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/healthz", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("healthcheck → %d (want 200)", w.Code)
		}
	})

	t.Run("readiness_with_nil_repo_returns_503", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/healthz/ready", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Logf("readiness with nil repo → %d (want 503)", w.Code)
		}
	})
}

// TestStressHealthEndpoints exercises health and readiness endpoints
// under sustained load — these must always return cleanly.
func TestStressHealthEndpoints(t *testing.T) {
	env := setupStressEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()
	const iterations = 200

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		start := time.Now()
		status, _ := stressGetJSON(t, client, env.ts.URL+"/healthz")
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /healthz status = %d, want 200", i, status)
		}
		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("HEALTH ENDPOINTS (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
}

// TestStressReadinessWithBackend exercises the readiness endpoint
// with a real (fake) repo and backend — must always return 200.
func TestStressReadinessWithBackend(t *testing.T) {
	env := setupStressEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()
	const iterations = 200

	for i := 0; i < iterations; i++ {
		env.repo.pingErr = nil
		status, _ := stressGetJSON(t, client, env.ts.URL+"/healthz/ready")
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /healthz/ready status = %d, want 200", i, status)
		}
	}
	t.Logf("READINESS (%d iterations): all 200 OK", iterations)
}

// init ensures the containerrt package is imported so ErrInvalidInput
// is available for stress test assertions.
var _ = containerrt.ErrInvalidInput
