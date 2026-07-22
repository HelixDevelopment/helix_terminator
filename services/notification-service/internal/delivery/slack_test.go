package delivery_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
)

func TestSlackConfigFromEnv_NotConfigured(t *testing.T) {
	t.Setenv("HERALD_SLACK_BOT_TOKEN", "")
	_, ok := delivery.SlackConfigFromEnv()
	assert.False(t, ok, "slack must be reported as not-configured when HERALD_SLACK_BOT_TOKEN is unset")
}

func TestSlackConfigFromEnv_Configured(t *testing.T) {
	t.Setenv("HERALD_SLACK_BOT_TOKEN", "xoxb-test-token")
	cfg, ok := delivery.SlackConfigFromEnv()
	require.True(t, ok)
	assert.Equal(t, "xoxb-test-token", cfg.BotToken)
}

// TestSlackSender_ZeroValue_NeverFabricatesSuccess proves the honest
// not-configured state: NewSlackSender()'s zero value MUST return
// ErrSlackProviderNotConfigured for every Send() call, never a fabricated
// success, mirroring PushSender's equivalent guarantee (push_test.go).
func TestSlackSender_ZeroValue_NeverFabricatesSuccess(t *testing.T) {
	sender := delivery.NewSlackSender()
	err := sender.Send(context.Background(), "C0123ABCD", "hello")
	require.Error(t, err)
	assert.ErrorIs(t, err, delivery.ErrSlackProviderNotConfigured)
}

// TestSlackSender_NilReceiver_NeverFabricatesSuccess covers the
// h.slackSender == nil path handler.go's deliverSlack relies on directly
// (mirrors deliverPush's h.pushSender == nil check for push.go).
func TestSlackSender_NilReceiver_NeverFabricatesSuccess(t *testing.T) {
	var sender *delivery.SlackSender
	err := sender.Send(context.Background(), "C0123ABCD", "hello")
	require.Error(t, err)
	assert.ErrorIs(t, err, delivery.ErrSlackProviderNotConfigured)
}

func TestNewConfiguredSlackSender_EmptyBotToken(t *testing.T) {
	_, err := delivery.NewConfiguredSlackSender(delivery.SlackConfig{})
	require.Error(t, err, "an empty BotToken must be rejected, never silently accepted")
}

// TestNewSlackSenderFromEnv_NotConfigured proves the honest ok=false path:
// no HERALD_SLACK_BOT_TOKEN => sender is nil, ok is false, err is nil —
// exactly mirroring NewPushSenderFromEnv's documented contract.
func TestNewSlackSenderFromEnv_NotConfigured(t *testing.T) {
	t.Setenv("HERALD_SLACK_BOT_TOKEN", "")
	sender, ok, err := delivery.NewSlackSenderFromEnv()
	assert.Nil(t, sender)
	assert.False(t, ok)
	assert.NoError(t, err)
}

// TestNewSlackSenderFromEnv_Configured_BuildsRealSender proves the
// ok=true/err=nil "operator set real credentials" branch constructs a
// genuine, non-nil SlackSender backed by Herald's real adapter (real by
// default — see slack.go's package doc comment). Herald's adapter
// constructor performs NO network call (see slack_herald.go's
// newHeraldSlackTransport doc comment), so a syntactically-present token
// always succeeds here; an actually-invalid token is only discovered at
// Send() time (mapped to the honest "failed" status by handler.go's
// deliverSlack), exactly like a real Slack API client would report it.
func TestNewSlackSenderFromEnv_Configured_BuildsRealSender(t *testing.T) {
	t.Setenv("HERALD_SLACK_BOT_TOKEN", "xoxb-test-token")
	sender, ok, err := delivery.NewSlackSenderFromEnv()
	require.True(t, ok, "HERALD_SLACK_BOT_TOKEN is set, so ok must be true")
	require.NoError(t, err)
	require.NotNil(t, sender)
}
