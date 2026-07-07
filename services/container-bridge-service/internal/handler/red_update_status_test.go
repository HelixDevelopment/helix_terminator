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
	"github.com/stretchr/testify/require"
)

// TestUpdateBridge_DoesNotFabricateClientAssertedStatus is the §11.4.43/
// §11.4.115 RED-then-GREEN test for the anti-bluff defect this task exists
// to close: UpdateBridge previously persisted req.Status VERBATIM — a
// client PUT of {"status":"active"} was written straight to the database
// with no cross-check against the real container-runtime state at all.
//
// RED (pre-fix) evidence: against the original UpdateBridge, the update map
// passed to the repository carries the client-asserted "active" even though
// the fake backend proves (via Status) the real container is NOT running —
// a fabricated row.
//
// GREEN (post-fix) evidence: the persisted status is the REAL,
// runtime-confirmed status ("inactive" here), never the client-supplied
// "active".
func TestUpdateBridge_DoesNotFabricateClientAssertedStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	id := uuid.New()
	repo := &fakeRepo{
		getResult: &model.ContainerBridge{
			ID:          id,
			ContainerID: "real-container-1",
			Name:        "old-name",
			Image:       "old-image:latest",
			Status:      model.ContainerBridgeStatusInactive,
		},
	}
	backend := &fakeBackend{
		name:      "podman",
		available: true,
		// The REAL runtime reports this container is NOT running — the
		// client's PUT asserting "active" must not be trusted over this.
		statusFunc: func(cid string) (*ctrruntime.ContainerStatus, error) {
			return nil, fmt.Errorf("no such container: %s", cid)
		},
	}

	h := New(repo, backend)
	router.PUT("/api/v1/container-bridges/:id", h.UpdateBridge)

	body := map[string]interface{}{
		"name":   "old-name",
		"image":  "old-image:latest",
		"status": "active",
		"ports":  []string{},
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/container-bridges/"+id.String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "response body: %s", w.Body.String())
	require.NotEmpty(t, repo.updateCalls, "UpdateBridge must persist SOMETHING")

	last := repo.updateCalls[len(repo.updateCalls)-1]
	assert.NotEqual(t, model.ContainerBridgeStatusActive, last.updates["status"],
		"a client PUT asserting \"active\" must never be persisted verbatim when the real container is not running")
	assert.Equal(t, model.ContainerBridgeStatusInactive, last.updates["status"],
		"the persisted status must be the REAL, runtime-confirmed state")
}
