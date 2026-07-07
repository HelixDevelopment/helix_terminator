package delivery_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
)

// TestEmailSender_RecipientHeaderInjection_Rejected is the rock-solid
// anti-bluff proof (Constitution §11.4.123) that a recipient value carrying
// CRLF-delimited extra header lines (the classic "Bcc:" smuggling attack)
// is rejected BEFORE any SMTP connection is attempted — never silently
// stripped and sent, never allowed to reach the hand-built header block in
// buildMessage.
func TestEmailSender_RecipientHeaderInjection_Rejected(t *testing.T) {
	// A real SMTP server MUST NOT even be configured here — a bug in the
	// guard would otherwise attempt to dial "localhost:25", and if that
	// somehow degraded to a false PASS via a swallowed dial error, this
	// test would still catch it because the assertion is on the error
	// class ("must not contain CR/LF"), not merely "an error occurred".
	sender := delivery.NewEmailSender(delivery.SMTPConfig{Host: "127.0.0.1", Port: "1", From: "notifications@example.com"})

	malicious := []string{
		"victim@example.com\r\nBcc: attacker@evil.com",
		"victim@example.com\nBcc: attacker@evil.com",
		"victim@example.com\r\nX-Injected: true\r\n\r\nInjected body",
		"victim@example.com%0d%0aBcc:attacker@evil.com", // literal %0d%0a, NOT decoded — still an invalid address
	}

	for _, to := range malicious {
		t.Run("", func(t *testing.T) {
			err := sender.Send(context.Background(), to, "subject", "body")
			require.Error(t, err, "recipient %q must be rejected, never silently delivered", to)
		})
	}
}

// TestEmailSender_RecipientRawCRLF_RejectedByCRLFGuardSpecifically pins the
// exact rejection reason for a raw CR/LF recipient (as opposed to merely
// being an invalid address for some unrelated reason) so a future
// regression that silently swaps the guard for a weaker check is caught.
func TestEmailSender_RecipientRawCRLF_RejectedByCRLFGuardSpecifically(t *testing.T) {
	sender := delivery.NewEmailSender(delivery.SMTPConfig{Host: "127.0.0.1", Port: "1", From: "notifications@example.com"})

	err := sender.Send(context.Background(), "victim@example.com\r\nBcc: attacker@evil.com", "subject", "body")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cr/lf")
}

// TestEmailSender_SubjectHeaderInjection_Rejected proves a malicious
// Subject cannot smuggle extra headers into the message — the concrete
// finding: "recipient/subject/from/body built into SMTP headers from user
// input... CRLF in the recipient address or subject injects extra
// headers".
func TestEmailSender_SubjectHeaderInjection_Rejected(t *testing.T) {
	sender := delivery.NewEmailSender(delivery.SMTPConfig{Host: "127.0.0.1", Port: "1", From: "notifications@example.com"})

	malicious := []string{
		"Innocent subject\r\nBcc: attacker@evil.com",
		"Innocent subject\nX-Injected: true",
		"Line one\r\nLine two",
	}

	for _, subject := range malicious {
		t.Run("", func(t *testing.T) {
			err := sender.Send(context.Background(), "victim@example.com", subject, "body")
			require.Error(t, err, "subject %q must be rejected, never silently delivered", subject)
			assert.Contains(t, strings.ToLower(err.Error()), "cr/lf")
		})
	}
}

// TestEmailSender_LegitimateSingleLineValues_NotRejectedByInjectionGuard
// proves the new guard is not overly broad: an ordinary, well-formed
// recipient/subject must still pass validation. The subsequent failure
// (unreachable SMTP host: 127.0.0.1:1) proves the value cleared the
// injection guard and reached the actual SMTP dial attempt.
func TestEmailSender_LegitimateSingleLineValues_NotRejectedByInjectionGuard(t *testing.T) {
	sender := delivery.NewEmailSender(delivery.SMTPConfig{Host: "127.0.0.1", Port: "1", From: "notifications@example.com"})

	err := sender.Send(context.Background(), "victim@example.com", "A perfectly normal subject line", "body")
	require.Error(t, err, "127.0.0.1:1 has nothing listening, so Send must still fail")
	assert.NotContains(t, strings.ToLower(err.Error()), "cr/lf",
		"a legitimate single-line recipient/subject must not be rejected by the injection guard")
	assert.NotContains(t, strings.ToLower(err.Error()), "invalid email recipient",
		"a well-formed recipient must not be rejected as malformed")
}

// TestEmailSender_MalformedAddressList_Rejected proves a comma-separated
// address list (a common header-injection / BCC-smuggling vector even
// without raw CRLF) is rejected — Send requires a SINGLE well-formed
// mailbox, per net/mail.ParseAddress semantics.
func TestEmailSender_MalformedAddressList_Rejected(t *testing.T) {
	sender := delivery.NewEmailSender(delivery.SMTPConfig{Host: "127.0.0.1", Port: "1", From: "notifications@example.com"})

	err := sender.Send(context.Background(), "victim@example.com, attacker@evil.com", "subject", "body")
	require.Error(t, err, "a multi-address recipient value must be rejected")
}

// TestEmailSender_SubjectTooLong_Rejected proves the RFC 5322 header-line
// length cap (audit fix: cap subject length to prevent abuse) is enforced.
func TestEmailSender_SubjectTooLong_Rejected(t *testing.T) {
	sender := delivery.NewEmailSender(delivery.SMTPConfig{Host: "127.0.0.1", Port: "1", From: "notifications@example.com"})

	huge := strings.Repeat("a", 5000)
	err := sender.Send(context.Background(), "victim@example.com", huge, "body")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "maximum length")
}

// TestEmailSender_RecipientTooLong_Rejected proves the RFC 5321 recipient
// address length cap is enforced.
func TestEmailSender_RecipientTooLong_Rejected(t *testing.T) {
	sender := delivery.NewEmailSender(delivery.SMTPConfig{Host: "127.0.0.1", Port: "1", From: "notifications@example.com"})

	huge := strings.Repeat("a", 300) + "@example.com"
	err := sender.Send(context.Background(), huge, "subject", "body")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "maximum length")
}
