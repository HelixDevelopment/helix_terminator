package delivery

// Real Apple Push Notification service (APNs) delivery client.
//
// APNs provider-API delivery over HTTP/2:
//
//	POST https://api.push.apple.com/3/device/{device-token}
//	authorization: bearer <ES256 provider JWT>
//	apns-topic: <bundle_id>
//	apns-push-type: alert
//	Content-Type: application/json
//	{"aps":{"alert":{"title":"...","body":"..."}}, <custom data k/v> }
//
// The provider authentication token is a JWT signed with the ES256
// (ECDSA P-256 + SHA-256) algorithm using the operator's APNs .p8 signing
// key. Its header carries alg=ES256 + kid=APNS_KEY_ID; its claims carry
// iss=APNS_TEAM_ID + iat (issued-at seconds). This is built on the Go
// standard library (crypto/ecdsa, crypto/x509) — the repo's go.mod ships no
// APNs SDK, so no dependency is added (Constitution §11.4.6 no-guessing:
// the recovered FCM half likewise signs its JWT with stdlib crypto).
//
// HTTP/2 note (Constitution §11.4.6): APNs requires HTTP/2. go.mod carries
// golang.org/x/net only as an INDIRECT dependency, so this client does NOT
// import golang.org/x/net/http2 — instead it relies on net/http, whose
// default Transport auto-negotiates HTTP/2 over a real TLS https:// endpoint
// via ALPN. That is sufficient for real production delivery to
// api.push.apple.com and needs no explicit http2 wiring. Unit tests inject a
// mock transport / an httptest.Server, where HTTP/2 is not exercised (the
// request-construction + response-handling contract is what the mock proves).
//
// HONEST BOUNDARY (Constitution §11.4.10, operator-gated): the unit tests
// prove this client builds the correct APNs request (URL, bearer JWT with the
// right alg/kid/iss, apns-topic, body) and correctly maps 200 -> success and
// 4xx/5xx -> a real surfaced error against a MOCK httptest transport. REAL
// end-to-end delivery to a physical device still requires the operator's
// actual .p8 signing key + APNS_KEY_ID/APNS_TEAM_ID/APNS_BUNDLE_ID and a real
// device token; that live path is NOT (and cannot be) verified from an
// autonomous unit test — it is never claimed as delivery-verified here.
//
// Sources verified 2026-07-22:
//   - https://developer.apple.com/documentation/usernotifications/sending-notification-requests-to-apns
//   - https://developer.apple.com/documentation/usernotifications/establishing-a-token-based-connection-to-apns

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"time"
)

// apnsProdBaseURL is the production APNs provider host; overridden by
// PushSender.apnsBaseURL (tests) or PushConfig.APNsHost (sandbox).
const apnsProdBaseURL = "https://api.push.apple.com"

// apnsPushType is the value sent in the apns-push-type header. "alert" is the
// correct type for a user-visible title/body notification (required on iOS
// 13+/watchOS 6+).
const apnsPushType = "alert"

// sendAPNs delivers payload to token over the APNs provider API using a
// freshly-signed ES256 provider JWT. It NEVER fabricates delivery: only a real
// APNs 2xx returns nil; any transport error or non-2xx returns a real wrapped
// error.
func (p *PushSender) sendAPNs(ctx context.Context, token string, payload PushPayload) error {
	if p.cfg.APNsKeyPath == "" || p.cfg.APNsKeyID == "" || p.cfg.APNsTeamID == "" || p.cfg.APNsBundleID == "" {
		// PushConfigFromEnv guarantees the full set for an APNs config; this
		// guards a hand-constructed PushConfig missing a field.
		return ErrPushProviderNotConfigured
	}

	keyPEM, err := os.ReadFile(p.cfg.APNsKeyPath)
	if err != nil {
		return fmt.Errorf("apns: read signing key %s: %w", p.cfg.APNsKeyPath, err)
	}

	jwtToken, err := signAPNsJWT(keyPEM, p.cfg.APNsKeyID, p.cfg.APNsTeamID, p.nowFn()())
	if err != nil {
		return fmt.Errorf("apns: sign provider token: %w", err)
	}

	body, err := buildAPNsPayload(payload)
	if err != nil {
		return fmt.Errorf("apns: marshal payload: %w", err)
	}

	sendURL := p.apnsBaseURLOrDefault() + "/3/device/" + token
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("apns: build send request: %w", err)
	}
	req.Header.Set("authorization", "bearer "+jwtToken)
	req.Header.Set("apns-topic", p.cfg.APNsBundleID)
	req.Header.Set("apns-push-type", apnsPushType)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client().Do(req)
	if err != nil {
		return fmt.Errorf("apns: send request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("apns: send rejected: %s", statusAndBody(resp))
	}
	return nil
}

// apnsBaseURLOrDefault returns the APNs host: a test override first, then the
// operator's APNS_HOST (sandbox) override, then the production host.
func (p *PushSender) apnsBaseURLOrDefault() string {
	if p.apnsBaseURL != "" {
		return p.apnsBaseURL
	}
	if p.cfg.APNsHost != "" {
		return p.cfg.APNsHost
	}
	return apnsProdBaseURL
}

// buildAPNsPayload renders the APNs JSON body: an aps.alert dict with the
// title/body plus any custom Data key/values at the top level.
func buildAPNsPayload(payload PushPayload) ([]byte, error) {
	alert := map[string]string{}
	if payload.Title != "" {
		alert["title"] = payload.Title
	}
	if payload.Body != "" {
		alert["body"] = payload.Body
	}
	root := map[string]any{
		"aps": map[string]any{"alert": alert},
	}
	for k, v := range payload.Data {
		// Never let a custom key clobber the reserved "aps" object.
		if k == "aps" {
			continue
		}
		root[k] = v
	}
	return json.Marshal(root)
}

// signAPNsJWT builds and ES256-signs the APNs provider-authentication JWT.
// The JWS signature is the raw R||S concatenation (each big-endian, left-padded
// to the P-256 32-byte field size), base64url-encoded — the JWS ES256 encoding,
// NOT the ASN.1 DER form ecdsa produces by default.
func signAPNsJWT(keyPEM []byte, keyID, teamID string, now time.Time) (string, error) {
	key, err := parseECPrivateKey(keyPEM)
	if err != nil {
		return "", fmt.Errorf("parse APNs signing key: %w", err)
	}
	if key.Curve != elliptic.P256() {
		return "", fmt.Errorf("APNs signing key must use the P-256 curve (ES256), got %s", key.Curve.Params().Name)
	}

	header := map[string]string{"alg": "ES256", "kid": keyID, "typ": "JWT"}
	claims := map[string]any{
		"iss": teamID,
		"iat": now.Unix(),
	}

	signingInput, err := jwtSigningInput(header, claims)
	if err != nil {
		return "", err
	}

	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	// P-256: each of r and s fits in 32 bytes; FillBytes left-pads big-endian.
	const keyByteLen = 32
	sig := make([]byte, 2*keyByteLen)
	r.FillBytes(sig[0:keyByteLen])
	s.FillBytes(sig[keyByteLen : 2*keyByteLen])

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// parseECPrivateKey decodes a PEM-encoded EC private key (the APNs .p8 is
// PKCS#8, as Apple emits; SEC1 "EC PRIVATE KEY" is accepted as a fallback) into
// an *ecdsa.PrivateKey.
func parseECPrivateKey(keyPEM []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in signing key")
	}
	if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		ecKey, ok := parsed.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("signing key is not an ECDSA key (got %T)", parsed)
		}
		return ecKey, nil
	}
	ecKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("not a PKCS#8 or SEC1 EC key: %w", err)
	}
	return ecKey, nil
}
