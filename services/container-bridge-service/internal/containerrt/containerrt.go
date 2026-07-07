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
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
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
	// cmd is an OPTIONAL trailing command+args override (e.g. for a minimal
	// image like busybox whose default entrypoint does not stay running);
	// most real application images (nginx, redis, postgres, ...) need no
	// override — omit cmd entirely.
	RunFromImage(ctx context.Context, name, image string, ports []string, cmd ...string) (string, error)
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

// ErrInvalidInput is the sentinel every ValidateRunFromImageInputs failure
// wraps (via %w), letting callers — in particular the HTTP handler — detect
// a caller-input defect with errors.Is and map it to HTTP 400, distinctly
// from a genuine container-runtime failure (which surfaces as 502).
var ErrInvalidInput = errors.New("containerrt: invalid input")

// containerNamePattern matches the docker/podman container-name grammar
// (`[a-zA-Z0-9][a-zA-Z0-9_.-]*`). It intentionally REJECTS a leading '-' (or
// any other non-alphanumeric first character) — the exact shape a
// flag/argument-injection payload like "--privileged" would need to be
// parsed as a CLI flag instead of the `--name` value / a harmless string.
var containerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// The following build an OCI/docker image-reference pattern from the
// documented grammar (OCI Distribution Spec / docker/distribution
// `reference` package grammar — a public specification, reimplemented here
// rather than importing an extra dependency): optional
// registry-domain[:port] + '/' separated lowercase path components + optional
// ":tag" + optional "@digest". Every component MUST start with an
// alphanumeric character, which — same as containerNamePattern above —
// structurally forbids a leading '-' anywhere a flag-injection payload could
// hide.
var (
	imageNameComponent    = `[a-z0-9]+(?:(?:[._]|__|[-]+)[a-z0-9]+)*`
	imageDomainComponent  = `[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?`
	imageDomain           = imageDomainComponent + `(?:\.` + imageDomainComponent + `)*(?::[0-9]+)?`
	imagePath             = imageNameComponent + `(?:/` + imageNameComponent + `)*`
	imageTag              = `[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}`
	imageDigest           = `[A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*:[0-9a-fA-F]{32,}`
	imageReferencePattern = regexp.MustCompile(
		`^(?:` + imageDomain + `/)?` + imagePath +
			`(?::` + imageTag + `)?` +
			`(?:@` + imageDigest + `)?$`,
	)
)

// portMappingPattern matches "hostPort:containerPort", optionally prefixed
// with a host IP ("hostIP:hostPort:containerPort") and optionally suffixed
// with "/tcp" or "/udp". Port numbers are range-checked separately (1-65535)
// so an out-of-range numeric value is still cleanly rejected rather than
// silently truncated by the regex.
var portMappingPattern = regexp.MustCompile(
	`^(?:[0-9]{1,3}(?:\.[0-9]{1,3}){3}:)?([0-9]{1,5}):([0-9]{1,5})(?:/(?:tcp|udp))?$`,
)

// ValidateRunFromImageInputs rejects any caller-controlled value that could
// reach the podman/docker/nerdctl `run` CLI unsanitized and be parsed as a
// FLAG instead of the positional/value slot it is meant to occupy — e.g. an
// Image of "--privileged" sitting in the IMAGE positional, or a Ports
// element like "-v" sitting in a `-p <value>` slot, both scanned by the
// underlying CLI's flag parser before (or in place of) the intended
// positional argument. It is intentionally the SOLE gate: relying only on
// an allow-listed runtime binary (runBinary) does not restrict what
// arguments reach that binary.
//
// Contract: every element of ports MUST be non-blank and match the
// documented hostPort:containerPort grammar — a blank or whitespace-only
// element is REJECTED (wrapping ErrInvalidInput), not silently skipped.
// This is deliberately strict rather than permissive: RunFromImage forwards
// every validated ports element straight to `-p <value>` with no further
// filtering, so the caller gets an explicit 400 for a malformed request
// instead of the runtime silently receiving one fewer port mapping than
// requested.
//
// It is pure — no exec, no I/O — so it is unit-testable in a table test
// without a real container runtime, and it is called from BOTH
// cliBackend.RunFromImage (the actual exec boundary, protecting every
// caller) and the HTTP handler's bringUp (so the rejection surfaces as a
// clean 400 before any backend call is attempted, per §11.4.108 — a caller
// mistake is never conflated with a genuine runtime failure).
//
// name may be "" (the caller has not yet decided a name / will auto-generate
// one); every other non-empty value is validated.
//
// cmd is deliberately NOT rejected for a leading '-': in the argv this
// package builds (`run -d --name <name> [-p <port>]* <image> <cmd...>`),
// every cmd element lands strictly AFTER the already-validated image
// positional — the documented `IMAGE [COMMAND] [ARG...]` region where
// podman/docker/nerdctl stop treating tokens as `run` options and instead
// pass them straight through as the container's entrypoint override
// (verified against Docker's own reference docs, e.g. `docker run nginx -g
// 'daemon off;'`, and exercised by this service's own real-Podman
// integration test's `["sh", "-c", "sleep 300"]`). Rejecting a leading '-'
// there would regress that legitimate, already-shipped capability without
// closing any actual injection vector — cmd can never be mistaken for the
// name/image/ports slots that DO precede it in argv, which are the ones
// this validator protects.
func ValidateRunFromImageInputs(name, image string, ports []string, cmd []string) error {
	if name != "" && !containerNamePattern.MatchString(name) {
		return fmt.Errorf("%w: name %q must match %s (docker/podman container-name grammar, no leading '-')",
			ErrInvalidInput, name, containerNamePattern.String())
	}
	if !imageReferencePattern.MatchString(image) {
		return fmt.Errorf("%w: image %q is not a valid OCI image reference (registry/name[:tag][@digest], no leading '-')",
			ErrInvalidInput, image)
	}
	for i, p := range ports {
		if err := validatePortMapping(p); err != nil {
			return fmt.Errorf("%w: ports[%d] %q: %v", ErrInvalidInput, i, p, err)
		}
	}
	_ = cmd // intentionally not validated here — see doc comment above.
	return nil
}

// validatePortMapping enforces the documented "hostPort:containerPort"
// syntax (optionally "hostIP:hostPort:containerPort", optionally suffixed
// "/tcp"|"/udp") with both port numbers range-checked to 1-65535.
func validatePortMapping(p string) error {
	m := portMappingPattern.FindStringSubmatch(p)
	if m == nil {
		return fmt.Errorf(
			"must match hostPort:containerPort (optionally hostIP:hostPort:containerPort, optional /tcp or /udp suffix)")
	}
	for _, portStr := range m[1:3] {
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("port %q out of range 1-65535", portStr)
		}
	}
	return nil
}

// RunFromImage implements Backend by shelling out to the detected runtime's
// CLI, mirroring digital.vasic.containers/pkg/brokertest.StartNATS.
func (b *cliBackend) RunFromImage(
	ctx context.Context, name, image string, ports []string, cmd ...string,
) (string, error) {
	if err := ValidateRunFromImageInputs(name, image, ports, cmd); err != nil {
		return "", err
	}
	// Every element of ports has already been validated (non-blank,
	// hostPort:containerPort grammar) by ValidateRunFromImageInputs above —
	// a blank/whitespace-only element is rejected there with ErrInvalidInput
	// before this loop ever runs, so no defensive trim/skip is needed or
	// performed here. See ValidateRunFromImageInputs' doc comment for the
	// strict reject-blank-ports contract this loop relies on.
	args := []string{"run", "-d", "--name", name}
	for _, p := range ports {
		args = append(args, "-p", p)
	}
	args = append(args, image)
	args = append(args, cmd...)

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
