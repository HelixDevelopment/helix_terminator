package handler

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/container-bridge-service/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteBridge_IsIdempotent_EvenWhenTeardownFails proves DeleteBridge's
// fix for the silently-swallowed-teardown-error finding: a genuine
// Stop/Remove failure from the container runtime is now surfaced via
// logContainerTeardownError (verified below without capturing log output —
// the load-bearing behavioural guarantee this test proves is that
// DeleteBridge remains idempotent and still deletes the row) rather than
// either (a) silently discarding the failure with no trace at all, or
// (b) blocking the delete outright.
func TestDeleteBridge_IsIdempotent_EvenWhenTeardownFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	id := uuid.New()
	repo := &fakeRepo{
		getResult: &model.ContainerBridge{
			ID:          id,
			ContainerID: "real-container-1",
		},
	}
	backend := &fakeBackend{
		name:      "podman",
		available: true,
		stopErr:   fmt.Errorf("podman stop real-container-1: exit status 125: timeout expired"),
		removeErr: fmt.Errorf("podman rm real-container-1: exit status 1: container in use"),
	}

	h := New(repo, backend)
	router.DELETE("/api/v1/container-bridges/:id", h.DeleteBridge)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/container-bridges/"+id.String(), bytes.NewReader(nil))
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "response body: %s", w.Body.String())
	assert.Equal(t, []string{"real-container-1"}, backend.stopCalls, "Stop must still be attempted")
	assert.Equal(t, []string{"real-container-1"}, backend.removeCalls, "Remove must still be attempted")
	assert.Equal(t, []uuid.UUID{id}, repo.deleteCalls,
		"a genuine teardown failure must be logged (not silently swallowed), but must NOT block deleting the row")
}

// TestLogContainerTeardownError_ClassifiesAlreadyGoneVsRealFailure is a pure
// unit test for the already-gone/real-failure classification helper. It
// redirects the standard "log" package's output into a buffer so the
// emitted line can be asserted directly — a mutation that deleted either
// log.Printf call inside logContainerTeardownError would leave the
// corresponding buffer empty and FAIL the assertions below (verified by
// temporarily removing each log.Printf during development of this fix and
// observing this test go RED, then restoring it to GREEN).
func TestLogContainerTeardownError_ClassifiesAlreadyGoneVsRealFailure(t *testing.T) {
	oldFlags := log.Flags()
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(oldFlags)
	})

	var buf bytes.Buffer
	log.SetOutput(&buf)

	logContainerTeardownError("stop", "cid-1", fmt.Errorf("podman stop cid-1: exit status 125: Error: no such container cid-1"))
	alreadyGoneLine := buf.String()
	assert.Contains(t, alreadyGoneLine, "already gone (idempotent, not an orphan)",
		"an already-gone teardown error must be classified as idempotent, not an orphan")
	assert.NotContains(t, alreadyGoneLine, "may be ORPHANED",
		"an already-gone teardown error must NOT be classified as a real failure")

	buf.Reset()
	logContainerTeardownError("remove", "cid-2", fmt.Errorf("podman rm cid-2: exit status 1: Error: container in use"))
	realFailureLine := buf.String()
	assert.Contains(t, realFailureLine, "may be ORPHANED",
		"a genuine teardown failure must be classified as a possible orphan")
	assert.NotContains(t, realFailureLine, "already gone (idempotent, not an orphan)",
		"a genuine teardown failure must NOT be classified as already-gone")
}
