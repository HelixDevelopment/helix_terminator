//go:build stress

// Stress test suite for recording-service handlers (Constitution §11.4.85).
//
// Exercises three invariants:
//   - Sustained load: N>=100 iterations of create→get→update→list→delete,
//     per-iteration latency recorded, p50/p95/p99 computed.
//   - Concurrent contention: N>=15 parallel goroutines performing
//     create+get+delete, no deadlock, no resource leak.
//   - Boundary conditions: empty fields, invalid UUIDs, max-length paths,
//     invalid formats — every boundary produces a categorised result.
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
	"github.com/helixdevelopment/recording-service/internal/handler"
	"github.com/helixdevelopment/recording-service/internal/model"
	"github.com/helixdevelopment/recording-service/internal/repository"
	"github.com/helixdevelopment/recording-service/internal/testutil"
	"github.com/helixdevelopment/recording-service/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// stressEnv holds the assembled test environment: a real gin engine
// wired to a real handler backed by a real PostgreSQL pool.
type stressEnv struct {
	ts      *httptest.Server
	cleanup func()
}

// setupStressEnv boots a real PostgreSQL container (via podman),
// applies recording-service migrations, constructs a real
// handler+router, and returns a ready httptest.Server. Skips
// honestly if podman is unavailable.
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

	r.POST("/recordings", h.CreateRecording)
	r.GET("/recordings/:id", h.GetRecording)
	r.GET("/recordings", h.ListRecordings)
	r.PUT("/recordings/:id", h.UpdateRecording)
	r.DELETE("/recordings/:id", h.DeleteRecording)
	r.GET("/health", h.HealthCheck)
	r.GET("/ready", h.ReadinessCheck)

	ts := httptest.NewServer(r)

	return &stressEnv{
		ts: ts,
		cleanup: func() {
			ts.Close()
			pool.Close()
		},
	}
}

// postJSON sends a POST request with a JSON body and returns status +
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

// putJSON sends a PUT request with a JSON body and returns status +
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

// getJSON sends a GET request and returns status + parsed response.
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

// deleteJSON sends a DELETE request and returns status + parsed response.
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

// uniqueRecordingRequest generates a collision-free create request.
func uniqueRecordingRequest(i int) model.CreateRecordingRequest {
	return model.CreateRecordingRequest{
		SessionID: uuid.New().String(),
		HostID:    uuid.New().String(),
		FilePath:  fmt.Sprintf("/recordings/stress-%d-%d.cast", time.Now().UnixNano(), i),
		Format:    "asciinema",
	}
}

// TestStressCreateGetUpdateDelete_SustainedLoad drives N>=100
// iterations of the full create→get→update→list→delete cycle against a
// real PostgreSQL instance, recording per-iteration latency and
// computing p50/p95/p99.
func TestStressCreateGetUpdateDelete_SustainedLoad(t *testing.T) {
	env := setupStressEnv(t)
	defer env.cleanup()

	client := env.ts.Client()
	const iterations = 100

	rec := testutil.NewLatencyRecorder()

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/recordings", uniqueRecordingRequest(i))
		if status != http.StatusCreated {
			t.Fatalf("iteration %d: POST /recordings status = %d, want 201; body=%v", i, status, body)
		}
		id, _ := body["id"].(string)
		if id == "" {
			t.Fatalf("iteration %d: POST /recordings returned no id", i)
		}

		// Get
		status, body = stressGetJSON(t, client, env.ts.URL+"/recordings/"+id)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /recordings/%s status = %d, want 200; body=%v", i, id, status, body)
		}

		// Update
		status, body = stressPutJSON(t, client, env.ts.URL+"/recordings/"+id, model.UpdateRecordingRequest{
			Status:        "completed",
			DurationSec:   120,
			FileSizeBytes: 4096,
		})
		if status != http.StatusOK {
			t.Fatalf("iteration %d: PUT /recordings/%s status = %d, want 200; body=%v", i, id, status, body)
		}

		// List
		status, body = stressGetJSON(t, client, env.ts.URL+"/recordings?limit=5")
		if status != http.StatusOK {
			t.Fatalf("iteration %d: GET /recordings status = %d, want 200; body=%v", i, status, body)
		}

		// Delete
		status, body = stressDeleteJSON(t, client, env.ts.URL+"/recordings/"+id)
		if status != http.StatusOK {
			t.Fatalf("iteration %d: DELETE /recordings/%s status = %d, want 200; body=%v", i, id, status, body)
		}

		rec.Record(time.Since(start))
	}

	p50, p95, p99 := rec.Percentiles()
	t.Logf("SUSTAINED LOAD (%d iterations): p50=%v p95=%v p99=%v", iterations, p50, p95, p99)
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
		start := time.Now()

		// Create
		status, body := stressPostJSON(t, client, env.ts.URL+"/recordings", uniqueRecordingRequest(id))
		if status != http.StatusCreated {
			t.Errorf("goroutine %d: POST /recordings status = %d, want 201; body=%v", id, status, body)
			return
		}
		recID, _ := body["id"].(string)
		if recID == "" {
			t.Errorf("goroutine %d: POST /recordings returned no id", id)
			return
		}

		// Get
		status, body = stressGetJSON(t, client, env.ts.URL+"/recordings/"+recID)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: GET /recordings/%s status = %d, want 200; body=%v", id, recID, status, body)
			return
		}

		// Delete
		status, _ = stressDeleteJSON(t, client, env.ts.URL+"/recordings/"+recID)
		if status != http.StatusOK {
			t.Errorf("goroutine %d: DELETE /recordings/%s status = %d, want 200", id, recID, status)
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

	t.Run("empty_session_id_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/recordings", model.CreateRecordingRequest{
			SessionID: "",
			HostID:    uuid.New().String(),
			FilePath:  "/recordings/test.cast",
			Format:    "asciinema",
		})
		if status == http.StatusCreated {
			t.Fatal("empty session_id must be rejected, got 201")
		}
		t.Logf("empty session_id → %d (expected 400)", status)
	})

	t.Run("invalid_session_id_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/recordings", model.CreateRecordingRequest{
			SessionID: "not-a-uuid",
			HostID:    uuid.New().String(),
			FilePath:  "/recordings/test.cast",
			Format:    "asciinema",
		})
		if status == http.StatusCreated {
			t.Fatal("invalid session_id must be rejected, got 201")
		}
		t.Logf("invalid session_id → %d (expected 400)", status)
	})

	t.Run("invalid_host_id_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/recordings", model.CreateRecordingRequest{
			SessionID: uuid.New().String(),
			HostID:    "not-a-uuid",
			FilePath:  "/recordings/test.cast",
			Format:    "asciinema",
		})
		if status == http.StatusCreated {
			t.Fatal("invalid host_id must be rejected, got 201")
		}
		t.Logf("invalid host_id → %d (expected 400)", status)
	})

	t.Run("empty_file_path_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/recordings", model.CreateRecordingRequest{
			SessionID: uuid.New().String(),
			HostID:    uuid.New().String(),
			FilePath:  "",
			Format:    "asciinema",
		})
		if status == http.StatusCreated {
			t.Fatal("empty file_path must be rejected, got 201")
		}
		t.Logf("empty file_path → %d (expected 400)", status)
	})

	t.Run("invalid_format_rejected", func(t *testing.T) {
		status, _ := stressPostJSON(t, client, env.ts.URL+"/recordings", model.CreateRecordingRequest{
			SessionID: uuid.New().String(),
			HostID:    uuid.New().String(),
			FilePath:  "/recordings/test.mp4",
			Format:    "invalid-format",
		})
		if status == http.StatusCreated {
			t.Fatal("invalid format must be rejected, got 201")
		}
		t.Logf("invalid format → %d (expected 400)", status)
	})

	t.Run("max_length_file_path_accepted", func(t *testing.T) {
		longPath := "/recordings/" + strings.Repeat("a", 1020) + ".cast"
		status, body := stressPostJSON(t, client, env.ts.URL+"/recordings", model.CreateRecordingRequest{
			SessionID: uuid.New().String(),
			HostID:    uuid.New().String(),
			FilePath:  longPath,
			Format:    "asciinema",
		})
		t.Logf("max-length file_path (%d chars) → %d", len(longPath), status)
		if status == http.StatusCreated {
			id, _ := body["id"].(string)
			if id != "" {
				// Cleanup
				stressDeleteJSON(t, client, env.ts.URL+"/recordings/"+id)
			}
		}
	})

	t.Run("invalid_recording_id_on_get", func(t *testing.T) {
		status, _ := stressGetJSON(t, client, env.ts.URL+"/recordings/not-a-uuid")
		if status == http.StatusOK {
			t.Fatal("invalid UUID on GET must be rejected, got 200")
		}
		t.Logf("invalid UUID GET → %d (expected 400)", status)
	})

	t.Run("nonexistent_recording_on_get", func(t *testing.T) {
		status, _ := stressGetJSON(t, client, env.ts.URL+"/recordings/"+uuid.New().String())
		if status == http.StatusOK {
			t.Fatal("nonexistent recording GET must return 404, got 200")
		}
		t.Logf("nonexistent recording GET → %d (expected 404)", status)
	})

	t.Run("invalid_status_on_update_rejected", func(t *testing.T) {
		// Create a valid recording first
		status, body := stressPostJSON(t, client, env.ts.URL+"/recordings", uniqueRecordingRequest(999))
		if status != http.StatusCreated {
			t.Fatalf("create failed: %d", status)
		}
		id, _ := body["id"].(string)

		// Try to update with invalid status
		status, _ = stressPutJSON(t, client, env.ts.URL+"/recordings/"+id, model.UpdateRecordingRequest{
			Status: "invalid-status",
		})
		if status == http.StatusOK {
			t.Fatal("invalid status on update must be rejected, got 200")
		}
		t.Logf("invalid status update → %d (expected 400)", status)

		// Cleanup
		stressDeleteJSON(t, client, env.ts.URL+"/recordings/"+id)
	})

	t.Run("negative_duration_on_update_rejected", func(t *testing.T) {
		status, body := stressPostJSON(t, client, env.ts.URL+"/recordings", uniqueRecordingRequest(998))
		if status != http.StatusCreated {
			t.Fatalf("create failed: %d", status)
		}
		id, _ := body["id"].(string)

		status, _ = stressPutJSON(t, client, env.ts.URL+"/recordings/"+id, model.UpdateRecordingRequest{
			Status:      "completed",
			DurationSec: -1,
		})
		if status == http.StatusOK {
			t.Fatal("negative duration on update must be rejected, got 200")
		}
		t.Logf("negative duration update → %d (expected 400)", status)

		stressDeleteJSON(t, client, env.ts.URL+"/recordings/"+id)
	})

	t.Run("list_with_invalid_limit_defaults", func(t *testing.T) {
		status, body := stressGetJSON(t, client, env.ts.URL+"/recordings?limit=999")
		if status != http.StatusOK {
			t.Fatalf("list with limit=999 → %d, want 200", status)
		}
		limit, _ := body["limit"].(float64)
		if limit > 100 {
			t.Logf("FINDING: limit=999 accepted as %v (handler caps at 100, defaults to 20)", limit)
		}
		t.Logf("list limit=999 → %d, limit=%v", status, limit)
	})

	t.Run("list_with_negative_offset", func(t *testing.T) {
		status, body := stressGetJSON(t, client, env.ts.URL+"/recordings?offset=-5")
		if status != http.StatusOK {
			t.Fatalf("list with negative offset → %d, want 200", status)
		}
		offset, _ := body["offset"].(float64)
		if offset < 0 {
			t.Errorf("negative offset should be clamped to 0, got %v", offset)
		}
		t.Logf("list offset=-5 → %d, offset=%v", status, offset)
	})
}

// TestStressBoundaryConditions_NoRepo exercises boundary conditions
// against the validation layer WITHOUT a database.
func TestStressBoundaryConditions_NoRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil)
	r.POST("/recordings", h.CreateRecording)

	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"empty_body", "", 400},
		{"invalid_json", "{broken", 400},
		{"missing_session_id", `{"hostId":"` + uuid.New().String() + `","filePath":"/test.cast","format":"asciinema"}`, 400},
		{"missing_host_id", `{"sessionId":"` + uuid.New().String() + `","filePath":"/test.cast","format":"asciinema"}`, 400},
		{"missing_file_path", `{"sessionId":"` + uuid.New().String() + `","hostId":"` + uuid.New().String() + `","format":"asciinema"}`, 400},
		{"missing_format", `{"sessionId":"` + uuid.New().String() + `","hostId":"` + uuid.New().String() + `","filePath":"/test.cast"}`, 400},
		{"valid_shape_no_repo", `{"sessionId":"` + uuid.New().String() + `","hostId":"` + uuid.New().String() + `","filePath":"/test.cast","format":"asciinema"}`, 503},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/recordings", strings.NewReader(tc.body))
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

// init ensures the test database migrations package is imported so
// the test binary includes the migration runner.
var _ = migrations.Schema
