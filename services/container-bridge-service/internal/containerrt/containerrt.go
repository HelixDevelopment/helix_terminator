// Package containerrt wires container-bridge-service to a real container
// runtime from the digital.vasic.containers submodule (pkg/runtime), and adds
// the one capability that abstraction intentionally does not expose: creating
// a brand-new container from an image.
//
// Precedent (documented in the submodule itself,
// digital.vasic.containers/pkg/brokertest/brokertest.go header comment): the
// ContainerRuntime interface only Starts/Stops/Removes EXISTING containers by
// ID or name. Running a NEW container from an image is done by shelling out
// to the detected runtime's own CLI ("<runtime> run -d --name ... <image>"),
// then handing all subsequent lifecycle (Status/Stop/Remove) back to
// pkg/runtime, which stays the single owner of teardown. This package mirrors
// that exact, already-established pattern for container-bridge-service's
// production (not test-only) use.
package containerrt

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	ctrruntime "digital.vasic.containers/pkg/runtime"

	"github.com/helixdevelopment/container-bridge-service/internal/model"
)

// Backend is the full container capability the handler needs: every method of
// runtime.ContainerRuntime (operating on EXISTING containers by ID/name) plus
// RunFromImage, the create-a-new-container step the upstream interface
// deliberately omits.
type Backend interface {
	ctrruntime.ContainerRuntime

	// RunFromImage creates and starts a brand-new container named `name` from
	// `image`, publishing `ports` (each already in CLI "host:container"
	// syntax, e.g. "8080:80"), and returns the runtime-assigned container ID.
	RunFromImage(ctx context.Context, name, image string, ports []string) (string, error)
}

// cliBackend wraps a pkg/runtime.ContainerRuntime and adds RunFromImage via
// the runtime's own CLI binary (docker/podman/nerdctl-compatible `run`
// subcommand).
type cliBackend struct {
	ctrruntime.ContainerRuntime
	binary string
}

// runBinary maps a pkg/runtime runtime name to a CLI binary that supports a
// one-off `run -d --name ... image` invocation. Mirrors
// digital.vasic.containers/pkg/brokertest's runtimeBinary allow-list: only
// the CLI-backed, docker-compatible runtimes support one-off image runs (LXD,
// Kubernetes, CRI-O do not follow the same `run` grammar).
func runBinary(name string) (string, bool) {
	switch name {
	case "podman", "docker", "nerdctl":
		return name, true
	default:
		return "", false
	}
}

// RunFromImage implements Backend by shelling out to the detected runtime's
// CLI, mirroring digital.vasic.containers/pkg/brokertest.StartNATS.
func (b *cliBackend) RunFromImage(
	ctx context.Context, name, image string, ports []string,
) (string, error) {
	args := []string{"run", "-d", "--name", name}
	for _, p := range ports {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		args = append(args, "-p", p)
	}
	args = append(args, image)

	out, err := exec.CommandContext(ctx, b.binary, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"%s run -d --name %s %s: %w: %s",
			b.binary, name, image, err, strings.TrimSpace(string(out)),
		)
	}
	id := strings.TrimSpace(string(out))
	if id == "" {
		return "", fmt.Errorf(
			"%s run -d --name %s %s: empty container ID in output",
			b.binary, name, image,
		)
	}
	return id, nil
}

// priorityFromEnv parses a comma-separated runtime-priority value (e.g.
// "docker,podman") such as the CONTAINER_RUNTIME_PRIORITY environment
// variable. It returns nil if envVal is empty/whitespace-only so callers fall
// back to the module's own default priority (Podman-first per
// digital.vasic.containers/pkg/runtime.RuntimePriority).
func priorityFromEnv(envVal string) []string {
	envVal = strings.TrimSpace(envVal)
	if envVal == "" {
		return nil
	}
	parts := strings.Split(envVal, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Detect auto-detects the local container runtime — Podman-first by the
// module's own default, or the order given by priorityEnv (a
// CONTAINER_RUNTIME_PRIORITY-style comma-separated list) — and wraps it as a
// Backend. It returns an error if no runtime is available, or the detected
// runtime does not support one-off image runs (only podman/docker/nerdctl
// do).
func Detect(ctx context.Context, priorityEnv string) (Backend, error) {
	var (
		rt  ctrruntime.ContainerRuntime
		err error
	)
	if priority := priorityFromEnv(priorityEnv); len(priority) > 0 {
		rt, err = ctrruntime.AutoDetectWithPriority(ctx, priority)
	} else {
		rt, err = ctrruntime.AutoDetect(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("containerrt: detect runtime: %w", err)
	}
	binary, ok := runBinary(rt.Name())
	if !ok {
		return nil, fmt.Errorf(
			"containerrt: runtime %q does not support one-off image runs", rt.Name(),
		)
	}
	return &cliBackend{ContainerRuntime: rt, binary: binary}, nil
}

// StatusFromState maps a runtime-reported container state to the service's
// own bridge-status vocabulary (model.ContainerBridgeStatus*). It is the
// single place that decides what "active" honestly means: a container the
// runtime confirms is StateRunning — never an unconditional default.
func StatusFromState(state ctrruntime.ContainerState) string {
	switch state {
	case ctrruntime.StateRunning:
		return model.ContainerBridgeStatusActive
	case ctrruntime.StateDead:
		return model.ContainerBridgeStatusError
	default:
		// created, stopped, paused, restarting, removing, or any future/
		// unknown state: honestly inactive, never active.
		return model.ContainerBridgeStatusInactive
	}
}
