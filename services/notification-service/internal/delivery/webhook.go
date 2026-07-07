package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

const (
	// maxWebhookTargetURLLength caps the target URL to a sane length.
	// Defense-in-depth (Constitution §11.4 security-hardening audit): the
	// HTTP request layer already caps model.CreateNotificationRequest.Target
	// at 1000 characters, but WebhookSender.Send is a public API other
	// callers can invoke directly, so it enforces its own bound too.
	maxWebhookTargetURLLength = 2048

	// maxWebhookResponseBytes bounds how much of a webhook receiver's
	// response body this sender will read. Without a cap, an
	// attacker-controlled (or merely misbehaving) receiver could stream an
	// unbounded response body back at us; io.Copy(io.Discard, ...) with no
	// limit is itself a resource-exhaustion vector even though
	// http.Client.Timeout bounds the wall-clock time.
	maxWebhookResponseBytes = 1 << 20 // 1 MiB
)

// WebhookPayload is the real JSON body POSTed to a subscriber's webhook URL.
type WebhookPayload struct {
	ID      string          `json:"id"`
	UserID  string          `json:"userId"`
	Type    string          `json:"type"`
	Title   string          `json:"title"`
	Message string          `json:"message"`
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// WebhookSender delivers notifications via a real outbound HTTP POST. It
// needs no external credentials — any reachable http(s) URL works.
type WebhookSender struct {
	client *http.Client
}

// NewWebhookSender constructs a PRODUCTION WebhookSender with the given
// request timeout (defaults to 10s when timeout <= 0).
//
// Every outbound connection is guarded against SSRF (Server-Side Request
// Forgery, Constitution §11.4.133): a net.Dialer.Control hook inspects the
// RESOLVED destination IP at actual connect time — never a
// resolve-then-validate pre-check, which is vulnerable to TOCTOU/DNS-rebind
// races — and refuses to connect to loopback, RFC1918 private, link-local
// (unicast+multicast, which is what blocks the 169.254.169.254 cloud
// metadata address), multicast, unspecified, or CGN (100.64.0.0/10)
// destinations. Because the SAME guarded Transport handles redirects, a
// malicious 3xx response pointing at an internal address is ALSO blocked —
// the guard is not bypassable via redirect.
func NewWebhookSender(timeout time.Duration) *WebhookSender {
	return newWebhookSender(timeout, false)
}

// NewWebhookSenderForTesting constructs a WebhookSender with the SSRF
// destination guard DISABLED, so it can target a loopback httptest.Server
// (127.0.0.1) the way this package's real-delivery integration tests do.
//
// This constructor MUST NEVER be used in production wiring — handler.New
// always uses NewWebhookSender. It exists solely so tests that stand up a
// genuinely real HTTP receiver are not forced onto a public network.
// TestWebhookSender_ProductionSender_RefusesSSRFTargets proves the
// PRODUCTION (strict) sender still blocks the classic SSRF targets.
func NewWebhookSenderForTesting(timeout time.Duration) *WebhookSender {
	return newWebhookSender(timeout, true)
}

func newWebhookSender(timeout time.Duration, allowPrivateNetworks bool) *WebhookSender {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if !allowPrivateNetworks {
		dialer.Control = ssrfGuardDialControl
	}

	// Clone the default transport (proxy settings, TLS defaults, idle-conn
	// tuning, handshake timeouts) rather than constructing a bare
	// &http.Transport{} — only DialContext is overridden, so every other
	// hardening default Go ships stays in effect.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialer.DialContext

	return &WebhookSender{client: &http.Client{Timeout: timeout, Transport: transport}}
}

// ssrfGuardDialControl is a net.Dialer.Control hook: it runs AFTER DNS
// resolution but BEFORE the connect() syscall, for every dial the
// Transport performs — including redirect targets, since following a
// redirect is just another RoundTrip through the same guarded Transport.
// This closes both the TOCTOU window a "resolve, validate, then dial
// separately" check would have, and the redirect bypass a purely
// pre-request URL check would have.
func ssrfGuardDialControl(_ /* network */ string, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("webhook destination guard: cannot parse dial address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("webhook destination guard: dial address %q did not resolve to an IP", host)
	}
	if isBlockedWebhookDestination(ip) {
		return fmt.Errorf("webhook destination guard: refusing to connect to non-public address %s (SSRF protection)", ip)
	}
	return nil
}

// isBlockedWebhookDestination reports whether ip is a destination a webhook
// MUST NEVER be allowed to target: loopback, RFC1918 private, link-local
// (unicast — this is what blocks the 169.254.169.254 cloud-metadata
// address — and multicast), multicast, unspecified ("0.0.0.0"/"::"), or
// carrier-grade-NAT (RFC 6598, 100.64.0.0/10 — NOT covered by IsPrivate).
func isBlockedWebhookDestination(ip net.IP) bool {
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
		// 100.64.0.0/10 — shared/carrier-grade-NAT address space.
		return true
	}
	return false
}

// Send POSTs payload as JSON to targetURL and returns the response status
// code. A 2xx response is the ONLY success outcome; any transport error,
// SSRF-guard rejection, or non-2xx status is returned as an error so
// callers persist an honest "failed" status — never a fabricated
// "delivered".
func (w *WebhookSender) Send(ctx context.Context, targetURL string, payload WebhookPayload) (int, error) {
	if targetURL == "" {
		return 0, fmt.Errorf("webhook target URL is required")
	}
	if len(targetURL) > maxWebhookTargetURLLength {
		return 0, fmt.Errorf("webhook target URL exceeds maximum length of %d characters", maxWebhookTargetURLLength)
	}
	parsed, err := url.ParseRequestURI(targetURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return 0, fmt.Errorf("invalid webhook target URL: %q", targetURL)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "helix-notification-service/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("webhook POST to %s failed: %w", targetURL, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxWebhookResponseBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("webhook POST to %s returned status %d", targetURL, resp.StatusCode)
	}
	return resp.StatusCode, nil
}
