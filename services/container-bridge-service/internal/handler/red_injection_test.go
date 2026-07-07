package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateBridge_RejectsFlagInjection_ImageStartsWithDash is the
// §11.4.115 RED-then-GREEN test for the flag/argument-injection defect: a
// caller-controlled Image value that begins with "-" (e.g. "--privileged")
// lands unsanitized in the podman/docker/nerdctl `run` argv's IMAGE
// positional slot, where it can be parsed as a FLAG to `run` instead of a
// harmless positional — host privilege escalation.
//
// RED (pre-fix) evidence: against the original CreateBridge/bringUp — which
// passes req.Image straight to containerrt.RunFromImage with no validation
// — this request would proceed to attempt a real container/CLI invocation
// (backend.RunFromImage gets called; in this test it also would have to
// reach backend.Status to report a non-201 result), NOT a clean 400 with
// zero backend calls.
//
// GREEN (post-fix) evidence: the malicious Image is rejected with 400
// BEFORE any container/CLI invocation — backend.runFromImageCalls stays
// empty and no bridge row is persisted.
func TestCreateBridge_RejectsFlagInjection_ImageStartsWithDash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	repo := &fakeRepo{}
	backend := &fakeBackend{
		name:      "podman",
		available: true,
		runFromImageFunc: func(name, image string, ports []string, cmd ...string) (string, error) {
			t.Fatalf("RunFromImage MUST NOT be invoked for a malicious Image value: name=%q image=%q", name, image)
			return "", nil
		},
	}

	h := New(repo, backend)
	router.POST("/api/v1/container-bridges", h.CreateBridge)

	body := map[string]interface{}{
		"hostId":      uuid.New().String(),
		"containerId": "bridge-under-test",
		"name":        "test-container",
		"image":       "--privileged",
		"ports":       []string{"18080:80"},
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/container-bridges", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, "response body: %s", w.Body.String())
	assert.Empty(t, backend.runFromImageCalls, "no container/CLI invocation may be attempted for a rejected input")
	assert.Empty(t, repo.created, "must not persist a bridge row for a rejected input")
}

// TestCreateBridge_RejectsFlagInjection_PortsLeadingDash mirrors the Image
// case above for a Ports element beginning with "-" (each Ports element is
// passed as a standalone `-p <value>` CLI argument, before the IMAGE
// positional is reached, so a malicious value there is squarely in the
// interspersed-flag-parsing danger zone).
func TestCreateBridge_RejectsFlagInjection_PortsLeadingDash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	repo := &fakeRepo{}
	backend := &fakeBackend{
		name:      "podman",
		available: true,
		runFromImageFunc: func(name, image string, ports []string, cmd ...string) (string, error) {
			t.Fatalf("RunFromImage MUST NOT be invoked for a malicious Ports value: ports=%v", ports)
			return "", nil
		},
	}

	h := New(repo, backend)
	router.POST("/api/v1/container-bridges", h.CreateBridge)

	body := map[string]interface{}{
		"hostId":      uuid.New().String(),
		"containerId": "bridge-under-test-2",
		"name":        "test-container",
		"image":       "docker.io/library/busybox:latest",
		"ports":       []string{"-9090:80"},
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/container-bridges", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, "response body: %s", w.Body.String())
	assert.Empty(t, backend.runFromImageCalls, "no container/CLI invocation may be attempted for a rejected input")
	assert.Empty(t, repo.created, "must not persist a bridge row for a rejected input")
}
