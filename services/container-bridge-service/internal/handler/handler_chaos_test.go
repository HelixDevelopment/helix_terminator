//go:build chaos

// Chaos test suite for container-bridge-service handlers (Constitution §11.4.85).
//
// Exercises three chaos dimensions:
//   - Input-corruption: corrupt JSON, malformed request bodies,
//     binary garbage — detected and reported cleanly (no panic).
//   - Resource-exhaustion: rapid-fire requests, verify graceful
//     degradation under pressure.
//   - Boundary conditions: nil body, empty JSON, extremely large
//     payloads, zero-value structs.
//
// Run:
//
//	go test -race -tags chaos -run TestChaos -v -timeout 120s ./internal/handler/
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

	ctrruntime "digital.vasic.containers/pkg/runtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/container-bridge-service/internal/containerrt"
	"github.com/helixdevelopment/container-bridge-service/internal/model"
	"github.com/helixdevelopment/container-bridge-service/internal/testutil"
)

// chaosEnv holds the assembled test environment for chaos tests.
type chaosEnv struct {
	ts      *httptest.Server
	repo    *fakeRepo
	backend *fakeBackend
}

// setupChaosEnv constructs a handler with a fakeRepo + fakeBackend
// and returns a ready httptest.Server.
func setupChaosEnv(t *testing.T) *chaosEnv {
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
	return &chaosEnv{ts: ts, repo: repo, backend: backend}
}

// chaosPostRaw sends a POST request with a raw byte body and returns
// the status code + raw response body. Unlike stressPostJSON, this
// does NOT assume the body is valid JSON — it sends whatever bytes
// are provided.
func chaosPostRaw(t *testing.T, client *http.Client, url string, contentType string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

// chaosGetRaw sends a GET request and returns status + raw body.
func chaosGetRaw(t *testing.T, client *http.Client, url string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

// chaosPutRaw sends a PUT request with a raw byte body and returns
// the status code + raw response body.
func chaosPutRaw(t *testing.T, client *http.Client, url string, contentType string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

// chaosDeleteRaw sends a DELETE request and returns status + raw body.
func chaosDeleteRaw(t *testing.T, client *http.Client, url string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
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
	env := setupChaosEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()

	t.Run("malformed_json_bodies", func(t *testing.T) {
		malformedBodies := []string{
			"",
			"{",
			"{}",
			"null",
			"[]",
			"42",
			`{"hostId":}`,
			`{"hostId":"not-a-uuid","containerId":"test","name":"test","image":"alpine:latest"}`,
			`{"hostId":null,"containerId":null,"name":null,"image":null}`,
			"{broken json here",
			strings.Repeat("{", 100),                // deeply nested
			`"` + strings.Repeat("x", 100000) + `"`, // huge string value
		}

		endpoints := []string{
			"/api/v1/container-bridges",
		}
		for _, ep := range endpoints {
			for i, body := range malformedBodies {
				status, _ := chaosPostRaw(t, client, env.ts.URL+ep, "application/json", []byte(body))
				if status == 0 {
					t.Logf("malformed body %d to %s: connection failed (acceptable)", i, ep)
					continue
				}
				if status >= 500 {
					t.Errorf("malformed body %d to %s: got %d — expected 400 for bad input", i, ep, status)
				}
			}
		}
		t.Logf("tested %d malformed bodies across %d endpoints", len(malformedBodies), len(endpoints))
	})

	t.Run("wrong_content_type", func(t *testing.T) {
		// gin's ShouldBindJSON parses the body as JSON regardless of
		// Content-Type header — valid JSON with wrong Content-Type is
		// still accepted. This is expected behavior, not a bug.
		contentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"",
			"application/octet-stream",
		}
		for _, ct := range contentTypes {
			body := fmt.Sprintf(`{"hostId":"%s","containerId":"test-ct","name":"test","image":"alpine:latest"}`, uuid.New().String())
			status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", ct, []byte(body))
			// Valid JSON is accepted regardless of Content-Type — no 500
			if status >= 500 {
				t.Errorf("content-type %q: got %d — expected non-server-error", ct, status)
			}
			t.Logf("content-type %q → %d (gin accepts valid JSON regardless of Content-Type)", ct, status)
		}
	})

	t.Run("binary_garbage_body", func(t *testing.T) {
		garbage := []byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01, 0x02, 0x03}
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", garbage)
		if status >= 500 {
			t.Errorf("binary garbage: got %d — expected 400", status)
		}
		t.Logf("binary garbage → %d", status)
	})

	t.Run("corrupt_uuid_in_get", func(t *testing.T) {
		// Note: empty string is excluded — it matches the ListBridges
		// route (GET /api/v1/container-bridges/) which returns 200 with
		// an empty list, not a corrupt-UUID error.
		corruptIDs := []string{
			"not-a-uuid",
			"null",
			"undefined",
			strings.Repeat("x", 1000),
			"../../../etc/passwd",
			"'; DROP TABLE bridges; --",
		}

		for i, id := range corruptIDs {
			status, raw := chaosGetRaw(t, client, env.ts.URL+"/api/v1/container-bridges/"+id)
			if status >= 500 {
				t.Errorf("corrupt UUID %d (%q): got %d — expected 400/404", i, truncate(id, 30), status)
			}
			t.Logf("corrupt UUID %d (%q) → %d: %s", i, truncate(id, 30), status, truncate(string(raw), 100))
		}
	})

	t.Run("corrupt_uuid_in_delete", func(t *testing.T) {
		corruptIDs := []string{
			"not-a-uuid",
			"",
			"null",
			strings.Repeat("y", 500),
		}

		for i, id := range corruptIDs {
			status, raw := chaosDeleteRaw(t, client, env.ts.URL+"/api/v1/container-bridges/"+id)
			if status >= 500 {
				t.Errorf("corrupt UUID delete %d (%q): got %d — expected 400/404", i, truncate(id, 30), status)
			}
			t.Logf("corrupt UUID delete %d (%q) → %d: %s", i, truncate(id, 30), status, truncate(string(raw), 100))
		}
	})

	t.Run("corrupt_body_in_update", func(t *testing.T) {
		corruptBodies := []string{
			"",
			"{broken",
			"null",
			"[]",
			`{"status":"fabricated"}`, // status field is ignored per §11.4.108
		}

		validID := uuid.New().String()
		env.repo.getErr = nil
		env.repo.getResult = &model.ContainerBridge{
			ID:          uuid.MustParse(validID),
			ContainerID: "test-container",
			Status:      model.ContainerBridgeStatusActive,
		}

		for i, body := range corruptBodies {
			status, _ := chaosPutRaw(t, client, env.ts.URL+"/api/v1/container-bridges/"+validID, "application/json", []byte(body))
			if status >= 500 {
				t.Errorf("corrupt update body %d: got %d — expected 400", i, status)
			}
			t.Logf("corrupt update body %d → %d", i, status)
		}
	})
}

// TestChaosResourceExhaustion drives rapid-fire requests to verify
// the service degrades gracefully under pressure — no goroutine
// leaks, no deadlocks, no panics.
func TestChaosResourceExhaustion(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()

	t.Run("rapid_fire_healthcheck", func(t *testing.T) {
		const burst = 100

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/healthz")
			if status != http.StatusOK {
				t.Errorf("healthcheck %d: got %d — expected 200", id, status)
			}
		})
	})

	t.Run("rapid_fire_readiness", func(t *testing.T) {
		const burst = 100

		testutil.RunConcurrent(t, burst, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/healthz/ready")
			if status != http.StatusOK {
				t.Errorf("readiness %d: got %d — expected 200", id, status)
			}
		})
	})

	t.Run("rapid_fire_get_nonexistent", func(t *testing.T) {
		// Hammer GET with non-existent IDs — must not panic.
		// Pre-set repo state before the loop to avoid races.
		env.repo.getErr = fmt.Errorf("not found")
		env.repo.getResult = nil
		const burst = 30

		testutil.RunConcurrent(t, burst, func(id int) {
			fakeID := uuid.New().String()
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/container-bridges/"+fakeID)
			if status >= 500 {
				t.Errorf("get nonexistent %d: got %d — expected 404", id, status)
			}
		})
	})

	t.Run("concurrent_list_empty", func(t *testing.T) {
		// Multiple goroutines listing simultaneously — must not deadlock.
		// Pre-set list state before the loop to avoid races.
		env.repo.listResult = nil
		env.repo.listTotal = 0
		env.repo.listErr = nil
		const parallel = 15

		testutil.RunConcurrent(t, parallel, func(id int) {
			status, _ := chaosGetRaw(t, client, env.ts.URL+"/api/v1/container-bridges")
			if status != http.StatusOK {
				t.Errorf("concurrent list %d: got %d — expected 200", id, status)
			}
		})
	})
}

// TestChaosBoundaryConditions exercises extreme boundary values
// that stress the parsing, validation, and serialization layers.
func TestChaosBoundaryConditions(t *testing.T) {
	env := setupChaosEnv(t)
	defer env.ts.Close()

	client := env.ts.Client()

	t.Run("nil_body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", env.ts.URL+"/api/v1/container-bridges", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("nil body request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			t.Errorf("nil body: got %d — expected 400", resp.StatusCode)
		}
		t.Logf("nil body → %d", resp.StatusCode)
	})

	t.Run("empty_json_object", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte("{}"))
		if status >= 500 {
			t.Errorf("empty JSON: got %d — expected 400", status)
		}
		t.Logf("empty JSON → %d", status)
	})

	t.Run("empty_json_array", func(t *testing.T) {
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte("[]"))
		if status >= 500 {
			t.Errorf("empty array: got %d — expected 400", status)
		}
		t.Logf("empty array → %d", status)
	})

	t.Run("extremely_large_payload", func(t *testing.T) {
		// 1MB payload — must not panic, must return an error.
		largeID := strings.Repeat("a", 500000)
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"%s","name":"%s","image":"%s"}`,
			uuid.New().String(), largeID, strings.Repeat("n", 500000), strings.Repeat("i", 500000))
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		if status == 0 {
			t.Fatal("1MB payload: connection failed entirely")
		}
		if status < 400 {
			t.Errorf("1MB payload: got %d — expected error (4xx or 5xx)", status)
		}
		t.Logf("1MB payload → %d (handler does not panic)", status)
	})

	t.Run("zero_value_struct_fields", func(t *testing.T) {
		payload := `{"hostId":"","containerId":"","name":"","image":""}`
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("zero-value fields: got %d — expected 400", status)
		}
		t.Logf("zero-value fields → %d", status)
	})

	t.Run("unicode_in_all_fields", func(t *testing.T) {
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"テストコンテナ","name":"日本語ブリッジ","image":"alpine:latest"}`,
			uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		// Either accepted (201) or rejected (400) — never 500
		if status >= 500 {
			t.Errorf("unicode fields: got %d — expected non-server-error", status)
		}
		t.Logf("unicode fields → %d", status)
	})

	t.Run("sql_injection_in_name", func(t *testing.T) {
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"test","name":"'; DROP TABLE bridges; --","image":"alpine:latest"}`,
			uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("SQL injection attempt: got %d — expected non-server-error", status)
		}
		t.Logf("SQL injection attempt → %d", status)
	})

	t.Run("xss_in_name", func(t *testing.T) {
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"test","name":"<script>alert('xss')</script>","image":"alpine:latest"}`,
			uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		if status >= 500 {
			t.Errorf("XSS in name: got %d — expected non-server-error", status)
		}
		t.Logf("XSS in name → %d", status)
	})

	t.Run("flag_injection_in_container_id", func(t *testing.T) {
		// Attempt to inject CLI flags via ContainerID — must be rejected
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"--privileged","name":"injection","image":"alpine:latest"}`,
			uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		if status == http.StatusCreated {
			t.Error("flag injection in containerId must NOT be accepted, got 201")
		}
		t.Logf("flag injection in containerId → %d", status)
	})

	t.Run("flag_injection_in_image", func(t *testing.T) {
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"test-inject","name":"injection","image":"--network=host"}`,
			uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		if status == http.StatusCreated {
			t.Error("flag injection in image must NOT be accepted, got 201")
		}
		t.Logf("flag injection in image → %d", status)
	})

	t.Run("path_traversal_in_container_id", func(t *testing.T) {
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"../../../etc/passwd","name":"traversal","image":"alpine:latest"}`,
			uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		if status == http.StatusCreated {
			t.Error("path traversal in containerId must NOT be accepted, got 201")
		}
		t.Logf("path traversal in containerId → %d", status)
	})

	t.Run("extremely_long_ports_array", func(t *testing.T) {
		// Generate a ports array with 1000 entries
		ports := make([]string, 1000)
		for i := range ports {
			ports[i] = fmt.Sprintf("%d:%d", 1000+i, 2000+i)
		}
		portsJSON, _ := json.Marshal(ports)
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"many-ports","name":"many-ports","image":"alpine:latest","ports":%s}`,
			uuid.New().String(), string(portsJSON))
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		// Must not panic — either accepted or rejected cleanly
		if status == 0 {
			t.Fatal("extremely long ports: connection failed entirely")
		}
		t.Logf("extremely long ports (1000 entries) → %d", status)
	})

	t.Run("negative_port_numbers", func(t *testing.T) {
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"neg-ports","name":"neg-ports","image":"alpine:latest","ports":["-1:-1"]}`,
			uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		if status == http.StatusCreated {
			t.Error("negative port numbers must NOT be accepted, got 201")
		}
		t.Logf("negative port numbers → %d", status)
	})

	t.Run("port_number_overflow", func(t *testing.T) {
		payload := fmt.Sprintf(`{"hostId":"%s","containerId":"overflow-ports","name":"overflow","image":"alpine:latest","ports":["99999:99999"]}`,
			uuid.New().String())
		status, _ := chaosPostRaw(t, client, env.ts.URL+"/api/v1/container-bridges", "application/json", []byte(payload))
		if status == http.StatusCreated {
			t.Error("port overflow (99999) must NOT be accepted, got 201")
		}
		t.Logf("port overflow → %d", status)
	})
}

// init ensures the containerrt package is imported so ErrInvalidInput
// is available for chaos test assertions.
var _ = containerrt.ErrInvalidInput
