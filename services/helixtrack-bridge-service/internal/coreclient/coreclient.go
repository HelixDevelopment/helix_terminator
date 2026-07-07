// Package coreclient authenticates against a real HelixTrack Core server's
// unified /do endpoint (see submodules/helixtrack-core Application README /
// CLAUDE.md "API Structure") and caches the resulting JWT using the owned-org
// digital.vasic.auth/pkg/tokenmanager library. It replaces the previous
// fabricated-status behaviour in internal/handler.CreateBridge, which set
// Status "active" unconditionally without ever contacting a real Core.
package coreclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"digital.vasic.auth/pkg/tokenmanager"
)

const (
	// defaultTokenTTL mirrors HelixTrack Core's known JWT expiry (HS256,
	// 24h) documented in the task brief. pkg/jwt.Validate is intentionally
	// NOT used here since it requires Core's signing secret, which this
	// client does not have; the cached-TTL discipline below is the
	// deliberate substitute (re-authenticate once the cached window lapses).
	defaultTokenTTL = 24 * time.Hour

	// successErrorCode is HelixTrack Core's "no error" sentinel for its
	// unified /do response envelope ({"errorCode":-1, "data": {...}}).
	successErrorCode = -1

	actionAuthenticate = "authenticate"

	// serviceName scopes the cached token inside tokenmanager.Manager.
	serviceName = "helixtrack-core"
)

// memoryStorage is a minimal in-memory tokenmanager.Storage adapter.
// digital.vasic.auth ships no production Storage implementation (by design
// - callers own their persistence backend); this mirrors the memoryStorage
// test template in pkg/tokenmanager/tokenmanager_test.go. It is intentionally
// process-local: token caching only needs to survive within one running
// bridge-service instance between requests, not across restarts.
type memoryStorage struct {
	data map[string]string
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{data: make(map[string]string)}
}

func (m *memoryStorage) Store(key, value string) error {
	m.data[key] = value
	return nil
}

func (m *memoryStorage) Retrieve(key string) (string, error) {
	return m.data[key], nil
}

func (m *memoryStorage) Delete(key string) error {
	delete(m.data, key)
	return nil
}

// Client authenticates against a real HelixTrack Core instance.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	tokens     *tokenmanager.Manager
}

// New creates a Client targeting the given HelixTrack Core base URL (e.g.
// "http://127.0.0.1:18080", no trailing slash) authenticating as username /
// password. baseURL/username/password MUST be injected by the caller (e.g.
// from HELIXTRACK_CORE_BASE_URL / HELIXTRACK_CORE_USERNAME /
// HELIXTRACK_CORE_PASSWORD) — never hardcoded per §11.4.28(B).
func New(baseURL, username, password string) *Client {
	return &Client{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		tokens:     tokenmanager.New(serviceName, newMemoryStorage()),
	}
}

type doRequest struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// doResponse mirrors HelixTrack Core's unified /do response envelope:
// {"errorCode": -1, "errorMessage": "...", "data": {...}}.
type doResponse struct {
	ErrorCode    int             `json:"errorCode"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	Data         json.RawMessage `json:"data,omitempty"`
}

type authenticateData struct {
	Token string `json:"token"`
}

// Authenticate performs a real POST {baseURL}/do {"action":"authenticate"}
// against HelixTrack Core, caches the returned JWT (tokenmanager,
// defaultTokenTTL), and returns it. Any non-success outcome (network
// failure, non-2xx HTTP status, errorCode != -1, or an empty token) is
// returned as an error — there is no fallback that fabricates a token.
func (c *Client) Authenticate(ctx context.Context) (string, error) {
	payload, err := json.Marshal(doRequest{
		Action: actionAuthenticate,
		Data: map[string]interface{}{
			"username": c.username,
			"password": c.password,
		},
	})
	if err != nil {
		return "", fmt.Errorf("coreclient: marshal authenticate request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/do", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("coreclient: build authenticate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("coreclient: authenticate request to %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("coreclient: read authenticate response: %w", err)
	}

	var parsed doResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("coreclient: decode authenticate response (HTTP %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK || parsed.ErrorCode != successErrorCode {
		msg := parsed.ErrorMessage
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return "", fmt.Errorf("coreclient: authenticate rejected by core: %s", msg)
	}

	var authData authenticateData
	if err := json.Unmarshal(parsed.Data, &authData); err != nil {
		return "", fmt.Errorf("coreclient: decode authenticate data: %w", err)
	}
	if authData.Token == "" {
		return "", fmt.Errorf("coreclient: authenticate succeeded but response carried no token")
	}

	if err := c.tokens.StoreTokenInfo(authData.Token, "", defaultTokenTTL); err != nil {
		return "", fmt.Errorf("coreclient: cache access token: %w", err)
	}

	return authData.Token, nil
}

// EnsureAuthenticated is nil if a cached, non-expired token already exists
// (tokens.HasValidToken()); otherwise it re-authenticates against Core via
// Authenticate. Handlers MUST call this (or AccessToken after calling this)
// before treating any HelixTrack Core-backed resource as active/available.
func (c *Client) EnsureAuthenticated(ctx context.Context) error {
	valid, err := c.tokens.HasValidToken()
	if err == nil && valid {
		return nil
	}
	_, err = c.Authenticate(ctx)
	return err
}

// AccessToken returns the currently cached access token, if any. Callers
// should call EnsureAuthenticated first to guarantee freshness.
func (c *Client) AccessToken() (string, error) {
	return c.tokens.GetAccessToken()
}
