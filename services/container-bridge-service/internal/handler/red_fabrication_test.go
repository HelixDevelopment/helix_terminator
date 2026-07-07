package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	ctrruntime "digital.vasic.containers/pkg/runtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/container-bridge-service/internal/model"
	"github.com/stretchr/testify/assert"
)

// TestCreateBridge_DoesNotFabricateActiveStatus_WhenContainerNeverComesUp is
// the §11.4.43/§11.4.115 RED-then-GREEN test for the anti-bluff defect this
// task exists to close: CreateBridge previously wrote
// model.ContainerBridgeStatusActive to the database UNCONDITIONALLY, without
// ever creating or checking a real container (see the task brief,
// "The defect (anti-bluff §11.4/§11.4.108)").
//
// RED (pre-fix) evidence: against the original CreateBridge — which ignores
// any container backend entirely — this test FAILS: the handler returns 201
// with status "active" and a fabricated bridge row even though the fake
// backend proves (via RunFromImage/Status) the container never started.
//
// GREEN (post-fix) evidence: against the fixed CreateBridge, this test
// PASSES: the handler honestly reports failure (5xx) and persists NO bridge
// row when the runtime cannot confirm the container is running.
func TestCreateBridge_DoesNotFabricateActiveStatus_WhenContainerNeverComesUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	repo := &fakeRepo{}
	backend := &fakeBackend{
		name:      "podman",
		available: true,
		// The container never actually starts: RunFromImage "succeeds" at
		// the CLI layer (some runtimes report success even for a container
		// that immediately crashes) but the runtime's own Status call —
		// the REAL source of truth — reports it is NOT running.
		runFromImageFunc: func(name, image string, ports []string, cmd ...string) (string, error) {
			return "deadbeef0001", nil
		},
		statusFunc: func(id string) (*ctrruntime.ContainerStatus, error) {
			return nil, fmt.Errorf("no such container")
		},
	}

	h := New(repo, backend)
	router.POST("/api/v1/container-bridges", h.CreateBridge)

	body := map[string]interface{}{
		"hostId":      uuid.New().String(),
		"containerId": "bridge-under-test",
		"name":        "test-container",
		"image":       "docker.io/library/busybox:latest",
		"ports":       []string{"18080:80"},
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/container-bridges", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Positive evidence the fabrication is closed: no 2xx, no "active"
	// status anywhere in the response body, and — the strongest possible
	// check — NOTHING was ever persisted as a bridge for a container that
	// never came up.
	assert.GreaterOrEqual(t, w.Code, 500, "must not report success for a container that never started")
	assert.NotContains(t, w.Body.String(), model.ContainerBridgeStatusActive)
	assert.Empty(t, repo.created, "must not persist a bridge row for a container that never started")
}
