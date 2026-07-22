// Push (FCM HTTP v1 + APNs-via-FCM) delivery, implemented entirely with the
// Go standard library (crypto/rsa, crypto/x509, encoding/pem, encoding/json,
// net/http) — no third-party Google/Firebase SDK is vendored, matching this
// package's existing house style (see email.go's net/smtp-only
// implementation).
//
// FCM's HTTP v1 endpoint is a SINGLE unified transport: a device
// registration token obtained via the Firebase SDK on Android, iOS, or Web
// all send through the same "messages:send" call, and Firebase's own
// infrastructure bridges iOS-registered tokens to APNs behind the scenes —
// this file's "apns" override block is exactly that bridge's
// platform-specific customization hook, so there is no separate raw APNs
// HTTP/2 client to implement.
//
// Every endpoint URL, JSON field name, and the OAuth2 JWT-bearer
// service-account flow below were verified 2026-07-22 against the
// following official Google/Firebase sources (Constitution §11.4.99
// latest-source cross-reference — see the field-level comments for which
// source backs which detail):
//   - https://firebase.google.com/docs/reference/fcm/rest/v1/projects.messages/send
//   - https://firebase.google.com/docs/cloud-messaging/auth-server
//   - https://developers.google.com/identity/protocols/oauth2/service-account
//
// Constitution §11.4 anti-bluff covenant: Send MUST NOT report success
// unless FCM's HTTP v1 endpoint actually accepted the message (a genuine
// 2xx response) — see NewPushSender's doc comment for the "not configured"
// honest-failure state this package uses when no credentials are present.
package delivery

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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ErrPushProviderNotConfigured is returned by PushSender.Send when this
// PushSender was built with delivery.NewPushSender() — the zero-credential,
// honest "not configured" state. Push delivery (FCM/APNs) requires
// operator-supplied provider credentials (a Firebase/GCP service-account
// JSON with Cloud Messaging send permission — see
// scripts/firebase/firebase_setup.sh) that no environment variable in this
// deployment currently provides. This is an HONEST not-yet-configured
// state, per operator decision — it MUST NEVER be papered over with a
// fabricated "sent"/"delivered" status (Constitution §11.4 anti-bluff
// covenant). Callers persist notification.Status =
// "pending_provider_unconfigured" when this error is returned.
var ErrPushProviderNotConfigured = errors.New(
	"push provider (fcm/apns) not configured: set FCM_SERVICE_ACCOUNT_JSON (path to a Firebase service-account key, see scripts/firebase/firebase_setup.sh) to enable push delivery",
)

const (
	// fcmMessagingScope is the OAuth2 scope FCM HTTP v1 requires. Verified
	// 2026-07-22 against https://firebase.google.com/docs/cloud-messaging/auth-server
	// ("To authorize access to FCM, request the scope
	// https://www.googleapis.com/auth/firebase.messaging.") — the
	// least-privilege scope, deliberately NOT the broader
	// "cloud-platform" scope some generic examples use.
	fcmMessagingScope = "https://www.googleapis.com/auth/firebase.messaging"

	// googleOAuthTokenURL is Google's OAuth2 token endpoint for the
	// JWT-bearer service-account grant. Verified 2026-07-22 against
	// https://developers.google.com/identity/protocols/oauth2/service-account.
	googleOAuthTokenURL = "https://oauth2.googleapis.com/token"

	// fcmSendURLTemplate is the FCM HTTP v1 send endpoint (project id
	// substituted via fmt.Sprintf + url.PathEscape). Verified 2026-07-22
	// against
	// https://firebase.google.com/docs/reference/fcm/rest/v1/projects.messages/send.
	fcmSendURLTemplate = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

	// accessTokenRefreshSkew renews the cached OAuth2 access token this
	// long before its reported expiry, so a Send() call never races an
	// about-to-expire token.
	accessTokenRefreshSkew = 60 * time.Second

	// maxFCMResponseBytes bounds how much of a response body this sender
	// reads — the same resource-exhaustion guard webhook.go applies to
	// arbitrary receiver responses (here: Google's own endpoints, but the
	// discipline is cheap and consistent).
	maxFCMResponseBytes = 1 << 20 // 1 MiB
)

// serviceAccountKey mirrors the JSON key file Firebase/Google Cloud
// generates for a service account — the "Generate new private key"
// download from Firebase Console → Project Settings → Service Accounts, or
// `gcloud iam service-accounts keys create` (see
// scripts/firebase/firebase_setup.sh). Only the fields the JWT-bearer
// OAuth2 flow needs are parsed; unrecognised fields are ignored.
type serviceAccountKey struct {
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
}

// PushConfig names the environment-sourced inputs that configure push
// (FCM/APNs) delivery for this deployment.
type PushConfig struct {
	// ServiceAccountJSONPath is the filesystem path to a Firebase/GCP
	// service-account JSON key with Cloud Messaging send permission
	// (permission cloudmessaging.messages.create — Firebase's
	// auto-provisioned "firebase-adminsdk-*" service account already
	// carries it out of the box), as produced by
	// scripts/firebase/firebase_setup.sh.
	ServiceAccountJSONPath string
	// ProjectID overrides the project id read from the service account
	// JSON's own "project_id" field. Optional — leave empty to use the
	// key's own project.
	ProjectID string
}

// PushConfigFromEnv reads FCM_SERVICE_ACCOUNT_JSON (required to enable) and
// FCM_PROJECT_ID (optional override) from the environment. ok is false when
// FCM_SERVICE_ACCOUNT_JSON is unset, meaning push is not configured for
// this deployment — an honest "not configured" state, mirroring
// SMTPConfigFromEnv's SMTP_HOST gate in email.go (Constitution §11.4
// anti-bluff: callers must not fabricate delivery when this returns
// false).
func PushConfigFromEnv() (PushConfig, bool) {
	path := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
	if path == "" {
		return PushConfig{}, false
	}
	return PushConfig{
		ServiceAccountJSONPath: path,
		ProjectID:              os.Getenv("FCM_PROJECT_ID"),
	}, true
}

// PushMessage is the platform-neutral push payload Send() delivers.
type PushMessage struct {
	Title string
	Body  string
	// Data is delivered as FCM's "data" field (arbitrary string key/value
	// pairs the receiving app's SDK reads on wake/foreground).
	Data map[string]string
	// Sound, if non-empty, is mapped to Android's notification.sound field
	// AND (via FCM's built-in APNs bridge) the apns payload's aps.sound
	// field, so one PushMessage covers a sound cue on both platforms
	// without the caller needing to know which platform a given device
	// token belongs to.
	Sound string
	// Badge sets iOS's aps.badge count via the FCM->APNs bridge. Ignored
	// on Android, which has no equivalent unified field. nil means "leave
	// unset" (as opposed to 0, which explicitly clears the badge).
	Badge *int
}

// PushSender delivers push notifications via FCM HTTP v1. The zero value
// returned by NewPushSender() is deliberately UNCONFIGURED — Send() always
// returns ErrPushProviderNotConfigured — so a deployment with no FCM
// credentials never fabricates a "sent" status (Constitution §11.4
// anti-bluff covenant).
type PushSender struct {
	projectID   string
	tokenSource *serviceAccountTokenSource // nil => unconfigured
	httpClient  *http.Client
	sendURL     string // real FCM endpoint, or an httptest.Server URL in tests
}

// NewPushSender constructs an UNCONFIGURED PushSender — the honest
// not-yet-provisioned state. Every Send() call returns
// ErrPushProviderNotConfigured, regardless of arguments.
func NewPushSender() *PushSender { return &PushSender{} }

// NewPushSenderFromEnv builds a PushSender from PushConfigFromEnv's output.
//
// ok mirrors PushConfigFromEnv: false means FCM_SERVICE_ACCOUNT_JSON is
// unset — push is honestly not configured for this deployment (sender is
// nil; callers should fall back to NewPushSender()).
//
// When ok is true but err is non-nil, the operator SET
// FCM_SERVICE_ACCOUNT_JSON but the referenced file could not be read or
// parsed as a valid Google service-account key. Callers MUST NOT silently
// treat this the same as "not configured" — that would hide a real
// operator misconfiguration behind the honest-unconfigured status. The
// correct behaviour (see handler.New) is to log the error and still fall
// back to an unconfigured sender so the service starts, while the
// operator's push-configuration intent — and the fact that it is broken —
// surfaces in the startup log rather than being swallowed.
func NewPushSenderFromEnv() (sender *PushSender, ok bool, err error) {
	cfg, ok := PushConfigFromEnv()
	if !ok {
		return nil, false, nil
	}
	sender, err = NewConfiguredPushSender(cfg)
	return sender, true, err
}

// NewConfiguredPushSender builds a real, credentialed PushSender from cfg.
// Returns a non-nil error if the service-account JSON cannot be read or
// parsed as a valid Google service-account key.
func NewConfiguredPushSender(cfg PushConfig) (*PushSender, error) {
	raw, err := os.ReadFile(cfg.ServiceAccountJSONPath)
	if err != nil {
		return nil, fmt.Errorf("push (FCM): cannot read service account JSON at %q: %w", cfg.ServiceAccountJSONPath, err)
	}
	return newPushSenderFromServiceAccountJSON(raw, cfg.ProjectID, googleOAuthTokenURL, "")
}

// NewPushSenderForTesting builds a fully real (non-mocked) PushSender whose
// OAuth2 token endpoint AND FCM send endpoint are overridden to point at
// caller-supplied httptest.Server URLs, so tests exercise the REAL
// JWT-signing + HTTP-call code paths (Constitution §11.4.27
// no-fakes-beyond-unit-tests — the only test double is the remote HTTP
// server itself, not this package's logic) without depending on network
// access to Google. rawServiceAccountJSON is caller-generated TEST FIXTURE
// key material (an RSA key pair generated at test time) — see push_test.go
// — and MUST NEVER be a real Google credential.
//
// This constructor MUST NEVER be used in production wiring — handler.New
// always uses NewPushSenderFromEnv / NewConfiguredPushSender.
func NewPushSenderForTesting(rawServiceAccountJSON []byte, tokenURL, sendURL string) (*PushSender, error) {
	return newPushSenderFromServiceAccountJSON(rawServiceAccountJSON, "", tokenURL, sendURL)
}

func newPushSenderFromServiceAccountJSON(raw []byte, projectIDOverride, tokenURL, sendURLOverride string) (*PushSender, error) {
	var key serviceAccountKey
	if err := json.Unmarshal(raw, &key); err != nil {
		return nil, fmt.Errorf("push (FCM): service account JSON is not valid JSON: %w", err)
	}
	if key.ClientEmail == "" || key.PrivateKey == "" {
		return nil, fmt.Errorf("push (FCM): service account JSON missing required client_email/private_key fields")
	}
	privKey, err := parsePrivateKey(key.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("push (FCM): cannot parse service account private key: %w", err)
	}

	projectID := projectIDOverride
	if projectID == "" {
		projectID = key.ProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("push (FCM): no project id available (set FCM_PROJECT_ID, or ensure the service account JSON's project_id field is present)")
	}

	effectiveTokenURL := tokenURL
	if effectiveTokenURL == "" {
		effectiveTokenURL = googleOAuthTokenURL
	}
	sendURL := sendURLOverride
	if sendURL == "" {
		sendURL = fmt.Sprintf(fcmSendURLTemplate, url.PathEscape(projectID))
	}

	return &PushSender{
		projectID: projectID,
		tokenSource: &serviceAccountTokenSource{
			clientEmail: key.ClientEmail,
			privateKey:  privKey,
			tokenURL:    effectiveTokenURL,
			httpClient:  &http.Client{Timeout: 10 * time.Second},
		},
		httpClient: &http.Client{Timeout: 10 * time.Second},
		sendURL:    sendURL,
	}, nil
}

// parsePrivateKey decodes a PEM-encoded RSA private key in either PKCS#1
// ("RSA PRIVATE KEY") or PKCS#8 ("PRIVATE KEY") form — Google's
// service-account JSON keys use PKCS#8, but PKCS#1 is accepted too since it
// is a valid PEM encoding of the same key material and costs nothing extra
// to support.
func parsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in private_key field")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("private key is neither valid PKCS1 nor PKCS8: %w", err)
	}
	rsaKey, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not an RSA key (got %T)", keyAny)
	}
	return rsaKey, nil
}

// serviceAccountTokenSource implements the Google OAuth2 JWT-bearer
// service-account flow entirely with the Go standard library. Flow
// verified 2026-07-22 against
// https://developers.google.com/identity/protocols/oauth2/service-account
// (Constitution §11.4.99): build a JWT with header {alg:RS256,typ:JWT},
// claims {iss:client_email, scope, aud:token endpoint, iat, exp<=iat+3600},
// sign with the service account's RSA private key (SHA256, PKCS1v15), POST
// it as a jwt-bearer grant to the token endpoint
// (application/x-www-form-urlencoded, grant_type=
// urn:ietf:params:oauth:grant-type:jwt-bearer, assertion=<signed JWT>), and
// cache the returned access_token until shortly before its expires_in
// elapses.
type serviceAccountTokenSource struct {
	clientEmail string
	privateKey  *rsa.PrivateKey
	tokenURL    string
	httpClient  *http.Client

	mu          sync.Mutex
	cachedToken string
	expiresAt   time.Time
}

// AccessToken returns a valid OAuth2 access token, reusing a cached one
// when it has more than accessTokenRefreshSkew left before expiry.
func (s *serviceAccountTokenSource) AccessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cachedToken != "" && time.Now().Before(s.expiresAt) {
		return s.cachedToken, nil
	}
	token, expiresIn, err := s.fetchAccessToken(ctx)
	if err != nil {
		return "", err
	}
	s.cachedToken = token
	s.expiresAt = time.Now().Add(time.Duration(expiresIn)*time.Second - accessTokenRefreshSkew)
	return token, nil
}

func (s *serviceAccountTokenSource) fetchAccessToken(ctx context.Context) (string, int, error) {
	now := time.Now()
	claims := map[string]interface{}{
		"iss":   s.clientEmail,
		"scope": fcmMessagingScope,
		"aud":   s.tokenURL,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	signed, err := signJWT(s.privateKey, claims)
	if err != nil {
		return "", 0, fmt.Errorf("push (FCM): failed to sign OAuth2 JWT: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", signed)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("push (FCM): failed to build OAuth2 token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("push (FCM): OAuth2 token request to %s failed: %w", s.tokenURL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxFCMResponseBytes))

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("push (FCM): OAuth2 token endpoint returned status %d: %s", resp.StatusCode, truncateForError(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", 0, fmt.Errorf("push (FCM): OAuth2 token response is not valid JSON: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", 0, fmt.Errorf("push (FCM): OAuth2 token response missing access_token")
	}
	if tokenResp.ExpiresIn <= 0 {
		tokenResp.ExpiresIn = 3600
	}
	return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
}

// signJWT builds and RS256-signs a compact JWT (header.payload.signature,
// base64url-no-padding per RFC 7515 §3 / RFC 7519) from claims.
func signJWT(key *rsa.PrivateKey, claims map[string]interface{}) (string, error) {
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)

	digest := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func truncateForError(b []byte) string {
	const max = 500
	if len(b) > max {
		return string(b[:max]) + "...(truncated)"
	}
	return string(b)
}

// fcmMessage is the wire shape FCM HTTP v1's messages:send expects. Field
// names verified 2026-07-22 against
// https://firebase.google.com/docs/reference/fcm/rest/v1/projects.messages/send
// (Constitution §11.4.99).
type fcmMessage struct {
	Token        string            `json:"token"`
	Notification *fcmNotification  `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Android      *fcmAndroidConfig `json:"android,omitempty"`
	Apns         *fcmApnsConfig    `json:"apns,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

type fcmAndroidConfig struct {
	Notification *fcmAndroidNotification `json:"notification,omitempty"`
}

type fcmAndroidNotification struct {
	Sound string `json:"sound,omitempty"`
}

// fcmApnsConfig is FCM's APNs-bridge override — this is the mechanism that
// makes "APNs delivery via Firebase" real: FCM forwards the token to Apple
// Push Notification service on the caller's behalf, applying this payload
// as the raw APNs `aps` dictionary.
type fcmApnsConfig struct {
	Payload *fcmApnsPayload `json:"payload,omitempty"`
}

type fcmApnsPayload struct {
	Aps fcmAps `json:"aps"`
}

type fcmAps struct {
	Sound *string `json:"sound,omitempty"`
	Badge *int    `json:"badge,omitempty"`
}

type fcmSendRequest struct {
	Message fcmMessage `json:"message"`
}

// Send delivers msg to deviceToken via FCM HTTP v1 — the SAME unified
// endpoint FCM uses to reach Android, iOS (through FCM's built-in APNs
// bridge, see fcmApnsConfig) and Web targets, keyed by the registration
// token the receiving app's SDK produced.
//
// Returns ErrPushProviderNotConfigured when this PushSender has no
// credentials (the NewPushSender() zero-value case). Returns a non-nil
// error on ANY other failure (empty token, OAuth2 token exchange failure,
// transport error, non-2xx FCM response) — callers MUST persist that as an
// honest "failed" status. Success means FCM's HTTP v1 endpoint accepted the
// message for delivery to Google's infrastructure; it does NOT by itself
// prove the device received it (delivery confirmation is a client-SDK-side
// concern outside this server's control). Never fabricates a sent/delivered
// status either way (Constitution §11.4 anti-bluff covenant).
func (p *PushSender) Send(ctx context.Context, deviceToken string, msg PushMessage) error {
	if p.tokenSource == nil {
		return ErrPushProviderNotConfigured
	}
	if deviceToken == "" {
		return fmt.Errorf("push device token (target) is required")
	}

	accessToken, err := p.tokenSource.AccessToken(ctx)
	if err != nil {
		return err
	}

	reqBody := fcmSendRequest{Message: fcmMessage{
		Token: deviceToken,
		Data:  msg.Data,
	}}
	if msg.Title != "" || msg.Body != "" {
		reqBody.Message.Notification = &fcmNotification{Title: msg.Title, Body: msg.Body}
	}
	if msg.Sound != "" {
		reqBody.Message.Android = &fcmAndroidConfig{Notification: &fcmAndroidNotification{Sound: msg.Sound}}
	}
	if msg.Sound != "" || msg.Badge != nil {
		aps := fcmAps{Badge: msg.Badge}
		if msg.Sound != "" {
			aps.Sound = &msg.Sound
		}
		reqBody.Message.Apns = &fcmApnsConfig{Payload: &fcmApnsPayload{Aps: aps}}
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("push (FCM): failed to marshal send request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.sendURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("push (FCM): failed to build send request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("push (FCM) send to %s failed: %w", p.sendURL, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxFCMResponseBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("push (FCM) send returned status %d: %s", resp.StatusCode, truncateForError(respBody))
	}
	return nil
}
