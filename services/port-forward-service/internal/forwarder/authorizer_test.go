package forwarder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthorizer_DefaultDeny proves the blast-radius gate is default-deny
// (Constitution §11.4.21 / §11.4.133): with no configuration, local
// forwarding is allowed but remote/-R and dynamic/-D are refused.
func TestAuthorizer_DefaultDeny(t *testing.T) {
	t.Setenv("PORT_FORWARD_ALLOW_REMOTE", "")
	t.Setenv("PORT_FORWARD_ALLOW_DYNAMIC", "")
	t.Setenv("PORT_FORWARD_HIGH_RISK_SSH_HOST_ALLOWLIST", "")
	a := NewAuthorizerFromEnv()

	assert.NoError(t, a.Authorize(ForwardTypeLocal, "any.host"))
	assert.NoError(t, a.Authorize("", "any.host"), "empty type defaults to local")
	assert.ErrorIs(t, a.Authorize(ForwardTypeRemote, "any.host"), ErrForbidden)
	assert.ErrorIs(t, a.Authorize(ForwardTypeDynamic, "any.host"), ErrForbidden)
}

// TestAuthorizer_EnabledTypes proves the gate is a real two-way switch:
// once the operator enables a high-risk type via config, it is allowed.
func TestAuthorizer_EnabledTypes(t *testing.T) {
	t.Setenv("PORT_FORWARD_ALLOW_REMOTE", "true")
	t.Setenv("PORT_FORWARD_ALLOW_DYNAMIC", "1")
	t.Setenv("PORT_FORWARD_HIGH_RISK_SSH_HOST_ALLOWLIST", "")
	a := NewAuthorizerFromEnv()

	assert.NoError(t, a.Authorize(ForwardTypeRemote, "any.host"))
	assert.NoError(t, a.Authorize(ForwardTypeDynamic, "any.host"))
}

// TestAuthorizer_HostAllowlist proves that even when a high-risk type is
// enabled, an explicit SSH-host allow-list further restricts it — a host
// not on the list is refused.
func TestAuthorizer_HostAllowlist(t *testing.T) {
	t.Setenv("PORT_FORWARD_ALLOW_REMOTE", "true")
	t.Setenv("PORT_FORWARD_ALLOW_DYNAMIC", "true")
	t.Setenv("PORT_FORWARD_HIGH_RISK_SSH_HOST_ALLOWLIST", "bastion.example.com, jump.example.com")
	a := NewAuthorizerFromEnv()

	assert.NoError(t, a.Authorize(ForwardTypeRemote, "bastion.example.com"))
	assert.NoError(t, a.Authorize(ForwardTypeDynamic, "jump.example.com"))
	assert.ErrorIs(t, a.Authorize(ForwardTypeRemote, "attacker.example.com"), ErrForbidden)
	assert.ErrorIs(t, a.Authorize(ForwardTypeDynamic, "attacker.example.com"), ErrForbidden)
	// local is never gated by the high-risk host allow-list.
	assert.NoError(t, a.Authorize(ForwardTypeLocal, "attacker.example.com"))
}

// TestAuthorizer_UnknownType proves an unrecognised forward type is
// rejected explicitly (Constitution §11.4.6 — never guessed into "allow").
func TestAuthorizer_UnknownType(t *testing.T) {
	a := NewAuthorizerFromEnv()
	err := a.Authorize("bananas", "any.host")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedForwardType)
}

// TestAuthorizer_MalformedBoolIsDeny proves a malformed boolean env value
// fails closed (deny), never accidentally enabling a high-risk type.
func TestAuthorizer_MalformedBoolIsDeny(t *testing.T) {
	t.Setenv("PORT_FORWARD_ALLOW_REMOTE", "yes-please")
	a := NewAuthorizerFromEnv()
	assert.ErrorIs(t, a.Authorize(ForwardTypeRemote, "any.host"), ErrForbidden)
}
