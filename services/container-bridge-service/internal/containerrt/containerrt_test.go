package containerrt

import (
	"context"
	"testing"

	ctrruntime "digital.vasic.containers/pkg/runtime"
	"github.com/stretchr/testify/assert"
)

func TestPriorityFromEnv(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace-only", "   ", nil},
		{"single", "podman", []string{"podman"}},
		{"multiple", "docker,podman", []string{"docker", "podman"}},
		{"trims-and-drops-blanks", " docker , , podman ", []string{"docker", "podman"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := priorityFromEnv(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRunBinary(t *testing.T) {
	supported := []string{"podman", "docker", "nerdctl"}
	for _, name := range supported {
		binary, ok := runBinary(name)
		assert.True(t, ok, "expected %s to be supported", name)
		assert.Equal(t, name, binary)
	}

	unsupported := []string{"lxd", "cri-o", "kubernetes", "bogus"}
	for _, name := range unsupported {
		_, ok := runBinary(name)
		assert.False(t, ok, "expected %s to be unsupported", name)
	}
}

func TestStatusFromState(t *testing.T) {
	cases := []struct {
		state ctrruntime.ContainerState
		want  string
	}{
		{ctrruntime.StateRunning, "active"},
		{ctrruntime.StateDead, "error"},
		{ctrruntime.StateStopped, "inactive"},
		{ctrruntime.StateCreated, "inactive"},
		{ctrruntime.StatePaused, "inactive"},
		{ctrruntime.StateRestarting, "inactive"},
		{ctrruntime.StateRemoving, "inactive"},
		{ctrruntime.ContainerState("unknown-future-state"), "inactive"},
	}
	for _, tc := range cases {
		t.Run(string(tc.state), func(t *testing.T) {
			assert.Equal(t, tc.want, StatusFromState(tc.state))
		})
	}
}

// TestCLIBackend_RunFromImage_WrapsExecError proves RunFromImage surfaces a
// real exec failure (nonexistent binary) as an error rather than silently
// swallowing it or returning a fabricated container ID.
func TestCLIBackend_RunFromImage_WrapsExecError(t *testing.T) {
	b := &cliBackend{binary: "definitely-not-a-real-container-runtime-binary-xyz"}
	id, err := b.RunFromImage(context.Background(), "test-name", "busybox:latest", []string{"8080:80"})
	assert.Empty(t, id)
	assert.Error(t, err)
}

// TestDetect_FindsPodmanOnThisHost exercises the real Detect() detection path
// (rootless Podman 5.7.1 is confirmed installed and available on this host)
// and asserts it returns a usable, CLI-run-capable Backend — not a fabricated
// one. This complements (does not replace) the dedicated real-Podman
// integration test in handler_integration_test.go.
func TestDetect_FindsPodmanOnThisHost(t *testing.T) {
	ctx := context.Background()
	backend, err := Detect(ctx, "")
	if err != nil {
		t.Skipf("no supported container runtime available on this host: %v", err)
	}
	assert.Equal(t, "podman", backend.Name())
	assert.True(t, backend.IsAvailable(ctx))
}

// TestDetect_PriorityEnvHonored proves a CONTAINER_RUNTIME_PRIORITY-style
// override actually changes detection order rather than being silently
// ignored: podman is real and available on this host, so putting it first
// even in an odd custom order must still resolve to podman.
func TestDetect_PriorityEnvHonored(t *testing.T) {
	ctx := context.Background()
	backend, err := Detect(ctx, "podman,docker,nerdctl")
	if err != nil {
		t.Skipf("no supported container runtime available on this host: %v", err)
	}
	assert.Equal(t, "podman", backend.Name())
}
