package delivery_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
)

func TestSMTPConfigFromEnv_NotConfigured(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	_, ok := delivery.SMTPConfigFromEnv()
	assert.False(t, ok, "SMTP must be reported as not-configured when SMTP_HOST is unset")
}

func TestSMTPConfigFromEnv_Configured(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_FROM", "noreply@example.com")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")

	cfg, ok := delivery.SMTPConfigFromEnv()
	require.True(t, ok)
	assert.Equal(t, "smtp.example.com", cfg.Host)
	assert.Equal(t, "2525", cfg.Port)
	assert.Equal(t, "noreply@example.com", cfg.From)
}

func TestSMTPConfigFromEnv_Defaults(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_FROM", "")

	cfg, ok := delivery.SMTPConfigFromEnv()
	require.True(t, ok)
	assert.Equal(t, "25", cfg.Port)
	assert.Equal(t, "notifications@localhost", cfg.From)
}

func TestEmailSender_EmptyRecipient_ReturnsError(t *testing.T) {
	sender := delivery.NewEmailSender(delivery.SMTPConfig{Host: "localhost", Port: "25", From: "a@b.com"})
	err := sender.Send(context.Background(), "", "subject", "body")
	require.Error(t, err)
}

func TestEmailSender_NotConfigured_ReturnsError(t *testing.T) {
	sender := delivery.NewEmailSender(delivery.SMTPConfig{})
	err := sender.Send(context.Background(), "to@example.com", "subject", "body")
	require.Error(t, err, "sending with an empty SMTP host must never silently succeed")
}

func TestWebhookSender_EmptyURL_ReturnsError(t *testing.T) {
	sender := delivery.NewWebhookSender(time.Second)
	_, err := sender.Send(context.Background(), "", delivery.WebhookPayload{})
	require.Error(t, err)
}

func TestWebhookSender_InvalidURL_ReturnsError(t *testing.T) {
	sender := delivery.NewWebhookSender(time.Second)
	for _, bad := range []string{"not-a-url", "ftp://example.com/hook", "javascript:alert(1)"} {
		_, err := sender.Send(context.Background(), bad, delivery.WebhookPayload{})
		require.Error(t, err, "URL %q must be rejected", bad)
	}
}

// TestPushSender_AlwaysHonestlyUnconfigured proves push delivery NEVER
// fabricates success — Constitution §11.4 anti-bluff covenant: a
// not-yet-implemented provider must surface as an honest error, not a
// silent "sent".
func TestPushSender_AlwaysHonestlyUnconfigured(t *testing.T) {
	sender := delivery.NewPushSender()
	err := sender.Send()
	require.ErrorIs(t, err, delivery.ErrPushProviderNotConfigured)
}

func TestMain(m *testing.M) {
	// Ensure ambient SMTP_* env vars from the host/dev environment never
	// leak into the deterministic env-parsing tests above.
	for _, k := range []string{"SMTP_HOST", "SMTP_PORT", "SMTP_FROM", "SMTP_USERNAME", "SMTP_PASSWORD"} {
		_ = os.Unsetenv(k)
	}
	os.Exit(m.Run())
}
