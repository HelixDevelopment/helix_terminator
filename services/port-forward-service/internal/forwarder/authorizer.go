package forwarder

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// ErrForbidden is returned when a forward request fails the blast-radius
// authorization gate (Constitution §11.4.21 / §11.4.133 — high-blast-radius
// actions are gated, default-deny).
var ErrForbidden = errors.New("forward type is not authorized")

// ErrUnsupportedForwardType is returned for a forward type outside the
// closed set {local, remote, dynamic}.
var ErrUnsupportedForwardType = errors.New("unsupported forward type")

const (
	// ForwardTypeLocal is a classic "-L" forward: listen locally, dial the
	// target THROUGH the SSH connection. Low blast-radius (bind is
	// operator-controlled, target is explicit) — allowed by default.
	ForwardTypeLocal = "local"
	// ForwardTypeRemote is a classic "-R" forward: ask the SSH server to
	// listen on its side and forward inbound connections back to us.
	// High blast-radius (exposes a listener on a third-party host) —
	// gated, default-deny.
	ForwardTypeRemote = "remote"
	// ForwardTypeDynamic is a "-D" SOCKS5 forward: an open-ended proxy that
	// can reach ANY destination the SSH server can reach. Highest
	// blast-radius (open-relay risk) — gated, default-deny.
	ForwardTypeDynamic = "dynamic"
)

// Authorizer implements the mandatory config-driven allow-list gate for
// high-blast-radius forward types. It is entirely config/env-driven
// (Constitution §11.4.10 — no hardcoded credentials or bypass switches) and
// is DEFAULT-DENY: unless the operator has explicitly enabled a forward type
// (and, optionally, allow-listed the specific SSH host), Authorize returns
// ErrForbidden.
type Authorizer struct {
	allowRemote  bool
	allowDynamic bool
	// allowedSSHHosts, when non-empty, restricts remote/dynamic forwards to
	// this explicit set of SSH hosts even when the forward type itself is
	// enabled. Empty set == "any host" once the type is enabled.
	allowedSSHHosts map[string]struct{}
}

// NewAuthorizerFromEnv builds an Authorizer from environment configuration:
//
//   - PORT_FORWARD_ALLOW_REMOTE  (bool, default false)  — enable "-R" forwards.
//   - PORT_FORWARD_ALLOW_DYNAMIC (bool, default false)  — enable "-D" (SOCKS5) forwards.
//   - PORT_FORWARD_HIGH_RISK_SSH_HOST_ALLOWLIST (comma-separated hostnames,
//     default empty == "any host allowed once the type above is enabled").
//
// Absent configuration means everything high-blast-radius is DENIED — the
// safe default. Only local forwarding is allowed out of the box.
func NewAuthorizerFromEnv() *Authorizer {
	return &Authorizer{
		allowRemote:     envBool("PORT_FORWARD_ALLOW_REMOTE"),
		allowDynamic:    envBool("PORT_FORWARD_ALLOW_DYNAMIC"),
		allowedSSHHosts: envSet("PORT_FORWARD_HIGH_RISK_SSH_HOST_ALLOWLIST"),
	}
}

// Authorize decides whether the given forward type may be established
// against the given SSH host. It NEVER guesses (Constitution §11.4.6): an
// unrecognised forward type is rejected explicitly rather than defaulted to
// "allow".
func (a *Authorizer) Authorize(forwardType, sshHost string) error {
	switch forwardType {
	case ForwardTypeLocal, "":
		return nil
	case ForwardTypeRemote:
		if a == nil || !a.allowRemote {
			return ErrForbidden
		}
	case ForwardTypeDynamic:
		if a == nil || !a.allowDynamic {
			return ErrForbidden
		}
	default:
		return ErrUnsupportedForwardType
	}
	if len(a.allowedSSHHosts) > 0 {
		if _, ok := a.allowedSSHHosts[sshHost]; !ok {
			return ErrForbidden
		}
	}
	return nil
}

func envBool(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

func envSet(key string) map[string]struct{} {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	out := make(map[string]struct{})
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out[part] = struct{}{}
	}
	return out
}
