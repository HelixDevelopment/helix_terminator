package delivery

// Real Firebase Cloud Messaging delivery client.
//
// Two paths, in precedence order:
//  1. FCM HTTP v1 (preferred) — POST
//     https://fcm.googleapis.com/v1/projects/{project_id}/messages:send with an
//     OAuth2 bearer access token obtained from the service-account JSON via the
//     RSA-SHA256 JWT-bearer assertion flow (no google.golang.org/api or
//     golang.org/x/oauth2 dependency — the repo's go.mod ships neither, so the
//     flow is built on the Go standard library per the "don't add a heavy dep"
//     guidance and Constitution §11.4.6 no-guessing).
//  2. Legacy HTTP — POST https://fcm.googleapis.com/fcm/send with the
//     Authorization: key=<FCM_SERVER_KEY> header (used only when no service
//     account JSON is configured).
//
// Sources verified 2026-07-22:
//   - https://firebase.google.com/docs/cloud-messaging/send/v1-api
//   - https://developers.google.com/identity/protocols/oauth2/service-account

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// fcmScope is the OAuth2 scope FCM HTTP v1 requires.
const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// fcmProdBaseURL is the production FCM host; overridden by PushSender.fcmBaseURL
// in tests.
const fcmProdBaseURL = "https://fcm.googleapis.com"

// googleTokenURL is the default OAuth2 token endpoint used when the service
// account JSON does not carry its own token_uri.
const googleTokenURL = "https://oauth2.googleapis.com/token"

// maxErrBodyBytes caps how much of a non-2xx provider response body is read
// into an error message (defense against an oversized error body).
const maxErrBodyBytes = 4096

// serviceAccount is the subset of a Google service-account JSON key file this
// client needs. The private_key is a PEM-encoded RSA key; it is used to sign
// the JWT assertion and is NEVER logged (Constitution §11.4.10).
type serviceAccount struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

// fcmV1Notification is the FCM HTTP v1 message.notification object.
type fcmV1Notification struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

// fcmV1Message is the FCM HTTP v1 request body ({"message": {...}}).
type fcmV1Message struct {
	Message struct {
		Token        string            `json:"token"`
		Notification fcmV1Notification `json:"notification"`
		Data         map[string]string `json:"data,omitempty"`
	} `json:"message"`
}

// sendFCM delivers payload to token over FCM, choosing the HTTP v1 path when a
// service-account JSON is configured and the legacy path otherwise.
func (p *PushSender) sendFCM(ctx context.Context, token string, payload PushPayload) error {
	if p.cfg.FCMServiceAccountJSONPath != "" {
		return p.sendFCMHTTPv1(ctx, token, payload)
	}
	if p.cfg.FCMServerKey != "" {
		return p.sendFCMLegacy(ctx, token, payload)
	}
	// PushConfigFromEnv guarantees one of the two is set for a FCM config; this
	// guards a hand-constructed PushConfig with neither credential.
	return ErrPushProviderNotConfigured
}

// fcmBaseURLOrDefault returns the FCM host (test override or production).
func (p *PushSender) fcmBaseURLOrDefault() string {
	if p.fcmBaseURL != "" {
		return p.fcmBaseURL
	}
	return fcmProdBaseURL
}

// sendFCMHTTPv1 runs the full HTTP v1 flow: load the service account, mint an
// OAuth2 access token from it, then POST the message.
func (p *PushSender) sendFCMHTTPv1(ctx context.Context, token string, payload PushPayload) error {
	sa, err := loadServiceAccount(p.cfg.FCMServiceAccountJSONPath)
	if err != nil {
		return fmt.Errorf("fcm: load service account: %w", err)
	}
	if sa.ProjectID == "" {
		return fmt.Errorf("fcm: service account JSON is missing project_id")
	}

	accessToken, err := p.fetchFCMAccessToken(ctx, sa)
	if err != nil {
		return fmt.Errorf("fcm: obtain access token: %w", err)
	}

	var msg fcmV1Message
	msg.Message.Token = token
	msg.Message.Notification = fcmV1Notification{Title: payload.Title, Body: payload.Body}
	msg.Message.Data = payload.Data
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("fcm: marshal message: %w", err)
	}

	sendURL := fmt.Sprintf("%s/v1/projects/%s/messages:send", p.fcmBaseURLOrDefault(), sa.ProjectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("fcm: build send request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client().Do(req)
	if err != nil {
		return fmt.Errorf("fcm: send request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fcm: send rejected: %s", statusAndBody(resp))
	}
	return nil
}

// fetchFCMAccessToken mints a short-lived OAuth2 access token from the service
// account via the JWT-bearer assertion flow.
func (p *PushSender) fetchFCMAccessToken(ctx context.Context, sa serviceAccount) (string, error) {
	tokenURL := p.oauthTokenURL
	if tokenURL == "" {
		tokenURL = sa.TokenURI
	}
	if tokenURL == "" {
		tokenURL = googleTokenURL
	}

	assertion, err := signServiceAccountJWT(sa, tokenURL, p.nowFn()())
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint rejected assertion: %s", statusAndBody(resp))
	}

	var tr struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("token response contained no access_token")
	}
	return tr.AccessToken, nil
}

// signServiceAccountJWT builds and RS256-signs the OAuth2 assertion JWT.
func signServiceAccountJWT(sa serviceAccount, audience string, now time.Time) (string, error) {
	if sa.ClientEmail == "" {
		return "", fmt.Errorf("service account JSON is missing client_email")
	}
	key, err := parseRSAPrivateKey(sa.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("parse service account private_key: %w", err)
	}

	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iss":   sa.ClientEmail,
		"scope": fcmScope,
		"aud":   audience,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}

	signingInput, err := jwtSigningInput(header, claims)
	if err != nil {
		return "", err
	}

	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// parseRSAPrivateKey decodes a PEM RSA private key (PKCS#8, as Google emits, or
// PKCS#1) into an *rsa.PrivateKey.
func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("not a PKCS#1 or PKCS#8 key: %w", err)
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA (got %T)", parsed)
	}
	return rsaKey, nil
}

// fcmLegacyRequest is the legacy HTTP request body.
type fcmLegacyRequest struct {
	To           string            `json:"to"`
	Notification fcmV1Notification `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
}

// fcmLegacyResponse is the subset of the legacy response used to detect a
// per-message failure the 200 status alone would hide.
type fcmLegacyResponse struct {
	Success int `json:"success"`
	Failure int `json:"failure"`
	Results []struct {
		Error string `json:"error"`
	} `json:"results"`
}

// sendFCMLegacy delivers via the legacy FCM HTTP endpoint. The legacy endpoint
// returns HTTP 200 even when the individual message failed, so the body's
// success/failure counters are inspected — a failure with zero successes is
// surfaced as a real error, never a fabricated "sent".
func (p *PushSender) sendFCMLegacy(ctx context.Context, token string, payload PushPayload) error {
	reqBody := fcmLegacyRequest{
		To:           token,
		Notification: fcmV1Notification{Title: payload.Title, Body: payload.Body},
		Data:         payload.Data,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("fcm-legacy: marshal message: %w", err)
	}

	sendURL := p.fcmBaseURLOrDefault() + "/fcm/send"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("fcm-legacy: build request: %w", err)
	}
	req.Header.Set("Authorization", "key="+p.cfg.FCMServerKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client().Do(req)
	if err != nil {
		return fmt.Errorf("fcm-legacy: send request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fcm-legacy: send rejected: %s", statusAndBody(resp))
	}

	var lr fcmLegacyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxErrBodyBytes)).Decode(&lr); err != nil {
		// A 2xx with an unparseable body cannot be confirmed as delivered.
		return fmt.Errorf("fcm-legacy: could not parse success response: %w", err)
	}
	if lr.Success == 0 {
		reason := "unknown"
		if len(lr.Results) > 0 && lr.Results[0].Error != "" {
			reason = lr.Results[0].Error
		}
		return fmt.Errorf("fcm-legacy: message not delivered (failure=%d): %s", lr.Failure, reason)
	}
	return nil
}

// loadServiceAccount reads and parses the service-account JSON key file.
func loadServiceAccount(path string) (serviceAccount, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return serviceAccount{}, fmt.Errorf("read %s: %w", path, err)
	}
	var sa serviceAccount
	if err := json.Unmarshal(raw, &sa); err != nil {
		return serviceAccount{}, fmt.Errorf("parse service account JSON: %w", err)
	}
	return sa, nil
}

// jwtSigningInput renders base64url(header) + "." + base64url(claims).
func jwtSigningInput(header map[string]string, claims map[string]any) (string, error) {
	h, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal JWT header: %w", err)
	}
	c, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal JWT claims: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(c), nil
}

// statusAndBody renders "<status>: <bounded body>" for an error message without
// leaking unbounded provider output.
func statusAndBody(resp *http.Response) string {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBodyBytes))
	return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
}
