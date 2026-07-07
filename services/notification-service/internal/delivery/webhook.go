package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
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

// NewWebhookSender constructs a WebhookSender with the given request
// timeout (defaults to 10s when timeout <= 0).
func NewWebhookSender(timeout time.Duration) *WebhookSender {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &WebhookSender{client: &http.Client{Timeout: timeout}}
}

// Send POSTs payload as JSON to targetURL and returns the response status
// code. A 2xx response is the ONLY success outcome; any transport error or
// non-2xx status is returned as an error so callers persist an honest
// "failed" status — never a fabricated "delivered".
func (w *WebhookSender) Send(ctx context.Context, targetURL string, payload WebhookPayload) (int, error) {
	if targetURL == "" {
		return 0, fmt.Errorf("webhook target URL is required")
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
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("webhook POST to %s returned status %d", targetURL, resp.StatusCode)
	}
	return resp.StatusCode, nil
}
