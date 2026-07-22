package delivery

// Routing, honest-state, and env-config unit tests for the push sender. These
// exercise the ordered honest outcomes (unconfigured / empty-token / unknown
// provider) and PushConfigFromEnv's configured-vs-not contract WITHOUT any
// network call.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushSender_SendTo_Unconfigured_ReturnsNotConfigured(t *testing.T) {
	p := NewPushSender()
	err := p.SendTo(context.Background(), "some-token", PushPayload{Title: "T"})
	require.ErrorIs(t, err, ErrPushProviderNotConfigured,
		"an unconfigured sender must never fabricate delivery — honest not-configured error required")
}

func TestPushSender_SendTo_ArmedEmptyToken_ReturnsTokenEmpty(t *testing.T) {
	p := &PushSender{cfg: PushConfig{Provider: PushProviderFCM, FCMServerKey: "k"}, configured: true}
	err := p.SendTo(context.Background(), "", PushPayload{Title: "T"})
	require.ErrorIs(t, err, ErrPushTokenEmpty,
		"a real send with an empty device token must be caught locally, never sent")
}

func TestPushSender_SendTo_UnknownProvider_ReturnsNotImplemented(t *testing.T) {
	p := &PushSender{cfg: PushConfig{Provider: PushProvider("carrier-pigeon")}, configured: true}
	err := p.SendTo(context.Background(), "some-token", PushPayload{Title: "T"})
	require.ErrorIs(t, err, ErrPushProviderNotImplemented,
		"an unrecognised provider must surface honestly, never a fabricated delivery")
}

func TestPushConfigFromEnv_FCMServiceAccount(t *testing.T) {
	t.Setenv("FCM_SERVICE_ACCOUNT_JSON", "/etc/secrets/fcm-sa.json")
	t.Setenv("FCM_SERVER_KEY", "")
	t.Setenv("APNS_KEY_PATH", "")
	t.Setenv("APNS_KEY_ID", "")
	t.Setenv("APNS_TEAM_ID", "")
	t.Setenv("APNS_BUNDLE_ID", "")

	cfg, ok := PushConfigFromEnv()
	require.True(t, ok)
	assert.Equal(t, PushProviderFCM, cfg.Provider)
	assert.Equal(t, "/etc/secrets/fcm-sa.json", cfg.FCMServiceAccountJSONPath)
}

func TestPushConfigFromEnv_FCMServerKey(t *testing.T) {
	t.Setenv("FCM_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("FCM_SERVER_KEY", "legacy-key")
	t.Setenv("APNS_KEY_PATH", "")
	t.Setenv("APNS_KEY_ID", "")
	t.Setenv("APNS_TEAM_ID", "")
	t.Setenv("APNS_BUNDLE_ID", "")

	cfg, ok := PushConfigFromEnv()
	require.True(t, ok)
	assert.Equal(t, PushProviderFCM, cfg.Provider)
	assert.Equal(t, "legacy-key", cfg.FCMServerKey)
}

func TestPushConfigFromEnv_APNsComplete(t *testing.T) {
	t.Setenv("FCM_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("FCM_SERVER_KEY", "")
	t.Setenv("APNS_KEY_PATH", "/etc/secrets/AuthKey.p8")
	t.Setenv("APNS_KEY_ID", "ABC1234567")
	t.Setenv("APNS_TEAM_ID", "TEAM123456")
	t.Setenv("APNS_BUNDLE_ID", "com.example.app")
	t.Setenv("APNS_HOST", "https://api.sandbox.push.apple.com")

	cfg, ok := PushConfigFromEnv()
	require.True(t, ok)
	assert.Equal(t, PushProviderAPNs, cfg.Provider)
	assert.Equal(t, "/etc/secrets/AuthKey.p8", cfg.APNsKeyPath)
	assert.Equal(t, "ABC1234567", cfg.APNsKeyID)
	assert.Equal(t, "TEAM123456", cfg.APNsTeamID)
	assert.Equal(t, "com.example.app", cfg.APNsBundleID)
	assert.Equal(t, "https://api.sandbox.push.apple.com", cfg.APNsHost)
}

func TestPushConfigFromEnv_APNsPartial_NotConfigured(t *testing.T) {
	// A half-armed APNs set (key id with no team id / bundle id) must be
	// reported as NOT configured — never a partially-armed provider.
	t.Setenv("FCM_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("FCM_SERVER_KEY", "")
	t.Setenv("APNS_KEY_PATH", "/etc/secrets/AuthKey.p8")
	t.Setenv("APNS_KEY_ID", "ABC1234567")
	t.Setenv("APNS_TEAM_ID", "")
	t.Setenv("APNS_BUNDLE_ID", "")

	_, ok := PushConfigFromEnv()
	assert.False(t, ok, "a partial APNs credential set must be reported as not-configured")
}

func TestPushConfigFromEnv_NothingSet_NotConfigured(t *testing.T) {
	t.Setenv("FCM_SERVICE_ACCOUNT_JSON", "")
	t.Setenv("FCM_SERVER_KEY", "")
	t.Setenv("APNS_KEY_PATH", "")
	t.Setenv("APNS_KEY_ID", "")
	t.Setenv("APNS_TEAM_ID", "")
	t.Setenv("APNS_BUNDLE_ID", "")

	_, ok := PushConfigFromEnv()
	assert.False(t, ok)
}
