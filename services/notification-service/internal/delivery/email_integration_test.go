//go:build integration

package delivery_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
)

// mailhogMessage is the subset of MailHog's /api/v2/messages response body
// this test needs to assert real receipt.
type mailhogMessagesResponse struct {
	Items []mailhogMessage `json:"items"`
}

type mailhogMessage struct {
	Content struct {
		Headers map[string][]string `json:"Headers"`
		Body    string              `json:"Body"`
	} `json:"Content"`
}

// startMailhog boots a real MailHog SMTP sink in a rootless podman container
// (SMTP on smtpPort, HTTP API on httpPort) and returns a cleanup func.
// Constitution §11.4.161 rootless podman mandate: plain default userns only,
// no :z / --userns=keep-id / label=disable.
func startMailhog(t *testing.T) (smtpPort, httpPort string, cleanup func()) {
	t.Helper()

	name := "notif-svc-mailhog-" + uuid.New().String()[:8]
	smtpPort = "12525"
	httpPort = "18025"

	cmd := exec.Command("podman", "run", "-d", "--rm",
		"--name", name,
		"-p", smtpPort+":1025",
		"-p", httpPort+":8025",
		"docker.io/mailhog/mailhog",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to start mailhog container: %s", string(out))

	cleanup = func() {
		_ = exec.Command("podman", "rm", "-f", name).Run()
	}

	// Wait for MailHog's HTTP API to become reachable (real readiness probe,
	// not a fixed sleep).
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/api/v2/messages", httpPort))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return smtpPort, httpPort, cleanup
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	cleanup()
	t.Fatal("mailhog did not become ready within 30s")
	return "", "", nil
}

// TestEmailSender_RealSMTPDelivery_ConfirmedByMailhogAPI is the rock-solid
// anti-bluff proof (Constitution §11.4.123) that EmailSender.Send performs a
// REAL SMTP delivery: it boots a real MailHog sink, sends a real email
// through EmailSender, then queries MailHog's real HTTP API to confirm the
// message actually arrived with the expected recipient/subject/body.
func TestEmailSender_RealSMTPDelivery_ConfirmedByMailhogAPI(t *testing.T) {
	smtpPort, httpPort, cleanup := startMailhog(t)
	defer cleanup()

	sender := delivery.NewEmailSender(delivery.SMTPConfig{
		Host: "127.0.0.1",
		Port: smtpPort,
		From: "notifications@helix-terminator.test",
	})

	to := "recipient-" + uuid.New().String()[:8] + "@example.com"
	subject := "Real SMTP delivery proof " + uuid.New().String()
	body := "This is a real end-to-end SMTP delivery test body, unique id " + uuid.New().String()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := sender.Send(ctx, to, subject, body)
	require.NoError(t, err, "EmailSender.Send must succeed against a real reachable MailHog SMTP sink")

	// Poll MailHog's real HTTP API for the message — this is the captured,
	// independent, sink-side evidence per Constitution §11.4.13/§11.4.69.
	var found *mailhogMessage
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/api/v2/messages", httpPort))
		require.NoError(t, err)
		var parsed mailhogMessagesResponse
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, json.Unmarshal(body, &parsed))
		for i := range parsed.Items {
			m := parsed.Items[i]
			if strings.Contains(m.Content.Body, "unique id") {
				found = &m
				break
			}
		}
		if found != nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	require.NotNil(t, found, "MailHog API never reported the sent message — email was NOT actually delivered")
	require.Contains(t, found.Content.Body, body, "MailHog-received body must match the sent body")
	require.Contains(t, found.Content.Headers["Subject"], subject, "MailHog-received Subject header must match")
	require.Contains(t, found.Content.Headers["To"], to, "MailHog-received To header must match the real recipient")
}

// TestEmailSender_UnreachableHost_ReturnsHonestError proves the sender
// never fabricates success when the SMTP server is unreachable — the
// §11.4.1 FAIL-bluff-equally-forbidden requirement's positive counterpart:
// a genuine failure MUST surface as a genuine error, not a silent PASS.
func TestEmailSender_UnreachableHost_ReturnsHonestError(t *testing.T) {
	sender := delivery.NewEmailSender(delivery.SMTPConfig{
		Host: "127.0.0.1",
		Port: "1", // nothing listens on port 1
		From: "notifications@helix-terminator.test",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := sender.Send(ctx, "someone@example.com", "subject", "body")
	require.Error(t, err, "Send against an unreachable SMTP host must return an error, never a silent success")
}
