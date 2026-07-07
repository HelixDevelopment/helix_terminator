//go:build integration

// Package handler integration test — §11.4.27 (no fakes beyond unit tests):
// this test drives a REAL rootless-Podman container end-to-end through the
// HTTP handlers (no fake ContainerRuntime here). Run with:
//
//	GOWORK=off GOMAXPROCS=2 go test -tags=integration -p 2 -v -run TestIntegration_ContainerBridge_RealPodman ./internal/handler/...
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/container-bridge-service/internal/containerrt"
	"github.com/helixdevelopment/container-bridge-service/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// busyboxImage is a tiny real image (§11.4.27) pulled from the public
// registry. sleep keeps it running past the container-runtime's default
// (busybox's bare `sh` exits immediately on closed stdin under `-d` — a FACT
// verified on this host before authoring this test, not guessed per §11.4.6).
const busyboxImage = "docker.io/library/busybox:latest"

// TestIntegration_ContainerBridge_RealPodman drives CreateBridge → GetBridge
// → DeleteBridge through the real HTTP handlers wired to a REAL, auto-
// detected container runtime (rootless Podman 5.7.1 confirmed on this host).
// It asserts the API-reported status matches the REAL `podman inspect` state
// at each step, and cleans up the container on every exit path (§11.4.14).
func TestIntegration_ContainerBridge_RealPodman(t *testing.T) {
	ctx := t.Context()

	backend, err := containerrt.Detect(ctx, "")
	if err != nil {
		t.Skipf("§11.4.3 topology SKIP: no supported container runtime detected: %v", err)
	}
	if !backend.IsAvailable(ctx) {
		t.Skip("§11.4.3 topology SKIP: detected runtime reports unavailable")
	}
	if backend.Name() != "podman" {
		t.Skipf("§11.4.3 topology SKIP: expected podman on this host, detected %q", backend.Name())
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	repo := &fakeRepo{}
	h := New(repo, backend)
	router.POST("/api/v1/container-bridges", h.CreateBridge)
	router.GET("/api/v1/container-bridges/:id", h.GetBridge)
	router.DELETE("/api/v1/container-bridges/:id", h.DeleteBridge)

	containerName := "cbs-integration-" + uuid.New().String()[:8]

	// §11.4.14 test playback cleanup: force-remove the container on every
	// exit path, even if an assertion fails partway through.
	t.Cleanup(func() {
		_ = exec.Command("podman", "rm", "-f", containerName).Run()
	})

	// --- CreateBridge: create a brand-new container from a real image ---
	createBody := map[string]interface{}{
		"hostId":      uuid.New().String(),
		"containerId": containerName,
		"name":        "integration-test-bridge",
		"image":       busyboxImage,
		"ports":       []string{},
		"command":     []string{"sh", "-c", "sleep 300"},
	}
	b, _ := json.Marshal(createBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/container-bridges", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "CreateBridge response: %s", w.Body.String())

	var created model.ContainerBridge
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, model.ContainerBridgeStatusActive, created.Status,
		"API MUST report active only when the real container is confirmed running")
	assert.Equal(t, containerName, created.ContainerID)

	// Positive sink-side evidence (§11.4.13/§11.4.69): cross-check directly
	// against the REAL podman state, not just our own API's claim about it.
	realState := podmanState(t, containerName)
	assert.Equal(t, "running", realState,
		"real `podman inspect` state MUST be running right after CreateBridge reports active")

	// --- GetBridge: reconciled status still matches the real, running state ---
	repo.getResult = &created
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/container-bridges/"+created.ID.String(), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var fetched model.ContainerBridge
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fetched))
	assert.Equal(t, model.ContainerBridgeStatusActive, fetched.Status)

	// --- Stop the container out-of-band (simulating an external stop) and
	// confirm GetBridge's reconciliation catches the REAL state change ---
	stopOut, stopErr := exec.Command("podman", "stop", "-t", "2", containerName).CombinedOutput()
	require.NoError(t, stopErr, "podman stop: %s", string(stopOut))

	repo.getResult = &created // still stores stale "active" in the fake row
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/container-bridges/"+created.ID.String(), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var afterStop model.ContainerBridge
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &afterStop))
	assert.NotEqual(t, model.ContainerBridgeStatusActive, afterStop.Status,
		"a stopped real container MUST NOT still be reported active — this is the anti-bluff reconciliation")
	assert.Equal(t, "exited", podmanState(t, containerName))

	// --- DeleteBridge: real Stop+Remove via the runtime, then the row ---
	// Re-create so DeleteBridge has a real running-then-stopped container to
	// exercise its own Stop+Remove path end-to-end (independent of the
	// manual `podman stop` above).
	recreateOut, recreateErr := exec.Command("podman", "start", containerName).CombinedOutput()
	require.NoError(t, recreateErr, "podman start (re-arm for delete path): %s", string(recreateOut))

	repo.getResult = &created
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/v1/container-bridges/"+created.ID.String(), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "DeleteBridge response: %s", w.Body.String())

	// Positive sink-side evidence: the container must genuinely be gone from
	// `podman ps -a`, not merely that our handler returned 200.
	deadline := time.Now().Add(10 * time.Second)
	var lastList string
	for time.Now().Before(deadline) {
		out, _ := exec.Command("podman", "ps", "-a", "--filter", "name="+containerName, "--format", "{{.Names}}").CombinedOutput()
		lastList = string(out)
		if lastList == "" || lastList == "\n" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	assert.Empty(t, lastList, "DeleteBridge MUST actually remove the real container (podman ps -a evidence)")
}

// podmanState returns the REAL `podman inspect .State.Status` for name — the
// sink-side ground truth this test's assertions are cross-checked against
// (§11.4.13/§11.4.69), never trusting the API's own claim in isolation.
func podmanState(t *testing.T, name string) string {
	t.Helper()
	out, err := exec.Command(
		"podman", "inspect", "--format", "{{.State.Status}}", name,
	).CombinedOutput()
	require.NoError(t, err, "podman inspect: %s", string(out))
	s := string(out)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
