// Package delivery implements REAL outbound delivery for notification
// channels (email over SMTP, webhook over HTTP). Push (FCM/APNs) is an
// honest not-yet-configured placeholder — see push.go.
//
// Constitution §11.4 anti-bluff covenant: CreateNotification MUST NOT
// persist a "sent"/"delivered" status without actually attempting delivery.
// This package is the real delivery client that closes that gap.
package delivery

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
)

// SMTPConfig holds SMTP server connection settings. Values are sourced from
// environment variables only — credentials are never hardcoded in source
// (Constitution §11.4.10 credentials-handling mandate).
type SMTPConfig struct {
	Host     string
	Port     string
	From     string
	Username string
	Password string
}

// SMTPConfigFromEnv builds an SMTPConfig from the standard SMTP_* environment
// variables. ok is false when SMTP_HOST is unset, meaning SMTP is not
// configured for this deployment (an honest "not configured" state, not an
// error — callers must not fabricate delivery in that case).
//
// Recognised variables: SMTP_HOST (required to enable), SMTP_PORT (default
// "25"), SMTP_FROM (default "notifications@localhost"), SMTP_USERNAME,
// SMTP_PASSWORD (both optional; PLAIN auth is used only when a username is
// set — MailHog/Mailpit-class test sinks do not require auth).
func SMTPConfigFromEnv() (SMTPConfig, bool) {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return SMTPConfig{}, false
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "25"
	}
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = "notifications@localhost"
	}
	return SMTPConfig{
		Host:     host,
		Port:     port,
		From:     from,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
	}, true
}

// EmailSender delivers notification emails over real SMTP using only the Go
// standard library (net/smtp) — no third-party mail library is reimplemented.
type EmailSender struct {
	cfg SMTPConfig
}

// NewEmailSender constructs an EmailSender bound to the given SMTP config.
func NewEmailSender(cfg SMTPConfig) *EmailSender {
	return &EmailSender{cfg: cfg}
}

// Send connects to the configured SMTP server and sends a real email to to.
// It returns a non-nil error on ANY failure (dial, STARTTLS, auth, envelope,
// data) — callers MUST persist that as an honest "failed" status. Success
// means the configured SMTP server accepted the message for delivery; it
// does NOT by itself prove the recipient received it — that confirmation
// comes from the downstream mailbox/sink (see the integration tests in this
// package, which assert against a real MailHog/Mailpit inbox).
func (s *EmailSender) Send(ctx context.Context, to, subject, body string) error {
	if to == "" {
		return fmt.Errorf("email recipient (target) is required")
	}
	if s.cfg.Host == "" {
		return fmt.Errorf("smtp not configured: SMTP_HOST is unset")
	}

	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	msg := buildMessage(s.cfg.From, to, subject, body)

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, s.cfg.From, []string{to}, msg)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("smtp send to %s via %s failed: %w", to, addr, err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// buildMessage renders a minimal, valid RFC 5322 plaintext email.
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return []byte(b.String())
}
