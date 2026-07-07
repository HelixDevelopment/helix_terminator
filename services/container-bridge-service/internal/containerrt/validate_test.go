package containerrt

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateRunFromImageInputs is the §11.4.115 RED-then-GREEN pure
// (no-exec) table test for the flag/argument-injection guard: every value
// that reaches the podman/docker/nerdctl `run` argv unsanitized (name /
// image / ports / cmd) MUST be validated BEFORE exec, closing the hole where
// a value beginning with "-" (e.g. Image "--privileged") sitting in a
// positional argv slot gets parsed as a FLAG to `run` instead — host
// privilege escalation. This test is deliberately runnable WITHOUT invoking
// podman: ValidateRunFromImageInputs is pure.
func TestValidateRunFromImageInputs(t *testing.T) {
	cases := []struct {
		name    string
		ctrName string
		image   string
		ports   []string
		cmd     []string
		wantErr bool
	}{
		{
			name:    "valid minimal",
			ctrName: "bridge-abc123",
			image:   "nginx:latest",
			wantErr: false,
		},
		{
			name:    "valid fully-qualified image with registry+digest",
			ctrName: "bridge-abc123",
			image:   "ghcr.io/library/busybox@sha256:" + fortyByte(),
			wantErr: false,
		},
		{
			name:    "valid with host:port registry and tag",
			ctrName: "bridge-abc123",
			image:   "localhost:5000/myimage:1.0",
			wantErr: false,
		},
		{
			name:    "valid ports and cmd",
			ctrName: "bridge-abc123",
			image:   "redis:7",
			ports:   []string{"8080:80", "127.0.0.1:9090:90"},
			cmd:     []string{"redis-server", "--appendonly", "yes"},
			wantErr: false,
		},
		{
			name:    "empty name is allowed (caller auto-generates one)",
			ctrName: "",
			image:   "nginx:latest",
			wantErr: false,
		},
		{
			name:    "malicious image flag injection",
			ctrName: "bridge-abc123",
			image:   "--privileged",
			wantErr: true,
		},
		{
			name:    "malicious image with embedded flag-looking segment",
			ctrName: "bridge-abc123",
			image:   "--security-opt=seccomp=unconfined",
			wantErr: true,
		},
		{
			name:    "malicious containerID/name flag injection",
			ctrName: "--privileged",
			image:   "nginx:latest",
			wantErr: true,
		},
		{
			name:    "malicious ports leading dash",
			ctrName: "bridge-abc123",
			image:   "nginx:latest",
			ports:   []string{"-9090:80"},
			wantErr: true,
		},
		{
			name:    "malicious ports non-numeric",
			ctrName: "bridge-abc123",
			image:   "nginx:latest",
			ports:   []string{"not-a-port:80"},
			wantErr: true,
		},
		{
			// Blank ports elements are REJECTED, not silently skipped — see
			// the "Contract" paragraph on ValidateRunFromImageInputs' doc
			// comment. RunFromImage's argv-building loop forwards every
			// validated ports element straight to `-p <value>` with no
			// further filtering, so this strict-reject contract is what
			// makes that unconditional forwarding safe.
			name:    "blank ports element is rejected, not silently skipped",
			ctrName: "bridge-abc123",
			image:   "nginx:latest",
			ports:   []string{""},
			wantErr: true,
		},
		{
			name:    "whitespace-only ports element is rejected, not silently skipped",
			ctrName: "bridge-abc123",
			image:   "nginx:latest",
			ports:   []string{"   "},
			wantErr: true,
		},
		{
			name:    "one valid and one blank ports element still rejects the whole call",
			ctrName: "bridge-abc123",
			image:   "nginx:latest",
			ports:   []string{"8080:80", ""},
			wantErr: true,
		},
		{
			// cmd is intentionally NOT subject to the leading-dash rejection
			// (see the doc comment on ValidateRunFromImageInputs): it lands
			// strictly after the already-validated image in argv, in the
			// documented `IMAGE [COMMAND] [ARG...]` region where
			// podman/docker/nerdctl pass tokens straight through as the
			// container's own entrypoint override rather than parsing them
			// as `run` flags (confirmed against Docker's own reference docs,
			// e.g. `docker run nginx -g 'daemon off;'`). This exact pattern
			// is exercised for real by this service's own real-Podman
			// integration test (`["sh", "-c", "sleep 300"]`) — rejecting it
			// here would regress that legitimate, already-shipped feature
			// without closing any actual injection vector.
			name:    "cmd elements with a leading dash are a legitimate entrypoint override, not rejected",
			ctrName: "bridge-abc123",
			image:   "nginx:latest",
			cmd:     []string{"sh", "-c", "sleep 300"},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRunFromImageInputs(tc.ctrName, tc.image, tc.ports, tc.cmd)
			if tc.wantErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidInput),
					"validation failures must wrap ErrInvalidInput so callers can map to HTTP 400: %v", err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func fortyByte() string {
	// A syntactically-valid 64-hex-char sha256 digest body for the
	// "valid fully-qualified image with registry+digest" case.
	s := ""
	for i := 0; i < 64; i++ {
		s += "a"
	}
	return s
}

// TestCLIBackend_RunFromImage_RejectsMaliciousInputs_WithoutInvokingRuntime
// proves the validation guard runs INSIDE cliBackend.RunFromImage itself —
// the actual exec boundary — protecting every caller (not only the HTTP
// handler above it), and that the binary is NEVER invoked for a rejected
// input (b.binary is deliberately a nonexistent command so any exec attempt
// would surface as a DIFFERENT error than the validation error).
func TestCLIBackend_RunFromImage_RejectsMaliciousInputs_WithoutInvokingRuntime(t *testing.T) {
	b := &cliBackend{binary: "definitely-not-a-real-container-runtime-binary-xyz"}

	id, err := b.RunFromImage(context.Background(), "bridge-1", "--privileged", []string{"8080:80"})
	assert.Empty(t, id)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput),
		"malicious image must be rejected by validation, not surfaced as an exec/CLI error: %v", err)
}
