package delivery

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
)

// ErrPushProviderNotConfigured is returned by PushSender.Send/SendTo when NO
// push provider credentials are present. Push delivery (FCM/APNs) requires
// operator-supplied provider credentials (a Firebase service account or legacy
// server key, an APNs signing key + identifiers) that no environment variable
// in this deployment currently provides. This is an HONEST not-configured
// state, per operator decision — it MUST NEVER be papered over with a
// fabricated "sent"/"delivered" status (Constitution §11.4 anti-bluff
// covenant). Callers persist notification.Status =
// "pending_provider_unconfigured" when this error is returned.
var ErrPushProviderNotConfigured = errors.New(
	"push provider (fcm/apns) not configured: set FCM_SERVICE_ACCOUNT_JSON or FCM_SERVER_KEY (FCM), or APNS_KEY_PATH/APNS_KEY_ID/APNS_TEAM_ID/APNS_BUNDLE_ID (APNs) credentials to enable push delivery",
)

// ErrPushProviderNotImplemented is returned by PushSender.SendTo when the
// sender is armed with a PushConfig whose Provider value is not one this
// package builds a client for (neither FCM nor APNs). With PushConfigFromEnv
// this cannot occur — it is a defensive honest state for a hand-constructed
// PushConfig with an unknown Provider, never a fabricated delivery. The real
// FCM HTTP v1 (push_fcm.go) and APNs HTTP/2 (push_apns.go) clients are now
// implemented; a recognised provider is delivered for real, never reported
// "not implemented".
var ErrPushProviderNotImplemented = errors.New(
	"push provider credential set present but its Provider value is not a supported push backend (expected fcm or apns): no push was sent",
)

// ErrPushTokenEmpty is returned when a real send is attempted with an empty
// device/registration token. A provider call with an empty token would be a
// guaranteed provider-side rejection, so it is caught locally and surfaced as
// an honest error — never swallowed into a fabricated success.
var ErrPushTokenEmpty = errors.New("push device token is required: no push was sent")

// PushProvider identifies which push backend a PushConfig targets.
type PushProvider string

const (
	// PushProviderNone means no complete provider credential set was found.
	PushProviderNone PushProvider = ""
	// PushProviderFCM means a Firebase Cloud Messaging credential set is present.
	PushProviderFCM PushProvider = "fcm"
	// PushProviderAPNs means an Apple Push Notification service credential set is present.
	PushProviderAPNs PushProvider = "apns"
)

// PushConfig holds push-provider (FCM/APNs) credentials sourced EXCLUSIVELY
// from environment variables (Constitution §11.4.10 — never hardcoded, never
// logged). Secret FILES (the FCM service-account JSON, the APNs .p8 signing
// key) are referenced BY PATH; their contents never appear in an env var.
type PushConfig struct {
	Provider PushProvider

	// FCM (Firebase Cloud Messaging).
	FCMServiceAccountJSONPath string // FCM_SERVICE_ACCOUNT_JSON — path to the HTTP v1 service-account key file
	FCMServerKey              string // FCM_SERVER_KEY — legacy HTTP server key

	// APNs (Apple Push Notification service).
	APNsKeyPath  string // APNS_KEY_PATH — path to the .p8 signing key file
	APNsKeyID    string // APNS_KEY_ID
	APNsTeamID   string // APNS_TEAM_ID
	APNsBundleID string // APNS_BUNDLE_ID
	APNsHost     string // APNS_HOST — optional override, e.g. https://api.sandbox.push.apple.com (default: production api.push.apple.com)
}

// PushConfigFromEnv builds a PushConfig from the FCM_*/APNS_* environment
// variables. ok is true ONLY when a COMPLETE provider credential set is present
// — for FCM either FCM_SERVICE_ACCOUNT_JSON (HTTP v1) or FCM_SERVER_KEY
// (legacy); for APNs the full APNS_KEY_PATH + APNS_KEY_ID + APNS_TEAM_ID +
// APNS_BUNDLE_ID set. A partial set (e.g. an APNs key id with no team id) yields
// ok=false — an honest "not configured", never a half-armed provider. FCM takes
// precedence when both provider sets are present. Mirrors
// SMTPConfigFromEnv's ok-when-configured contract.
func PushConfigFromEnv() (PushConfig, bool) {
	fcmJSON := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
	fcmServerKey := os.Getenv("FCM_SERVER_KEY")
	if fcmJSON != "" || fcmServerKey != "" {
		return PushConfig{
			Provider:                  PushProviderFCM,
			FCMServiceAccountJSONPath: fcmJSON,
			FCMServerKey:              fcmServerKey,
		}, true
	}

	keyPath := os.Getenv("APNS_KEY_PATH")
	keyID := os.Getenv("APNS_KEY_ID")
	teamID := os.Getenv("APNS_TEAM_ID")
	bundleID := os.Getenv("APNS_BUNDLE_ID")
	if keyPath != "" && keyID != "" && teamID != "" && bundleID != "" {
		return PushConfig{
			Provider:     PushProviderAPNs,
			APNsKeyPath:  keyPath,
			APNsKeyID:    keyID,
			APNsTeamID:   teamID,
			APNsBundleID: bundleID,
			APNsHost:     os.Getenv("APNS_HOST"),
		}, true
	}

	return PushConfig{}, false
}

// PushPayload is the channel-neutral content of a push notification. The
// concrete provider clients (FCM HTTP v1, APNs HTTP/2) map it onto their own
// wire shapes (FCM message.notification, APNs aps.alert). Data carries optional
// user-defined key/value pairs; values are strings because FCM HTTP v1 requires
// the message `data` map to be string→string.
type PushPayload struct {
	Title string
	Body  string
	Data  map[string]string
}

// httpDoer is the minimal HTTP client seam the provider clients depend on.
// *http.Client satisfies it. Tests inject an httptest.Server-backed client (or
// a hand-written doer) so request construction + response handling are asserted
// against a MOCK transport — no live FCM/Google-OAuth/Apple endpoint is ever
// contacted from a unit test (Constitution §11.4.27 — real system under test,
// but the third-party push backend is the operator-gated boundary, see below).
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// PushSender is a REAL push (FCM/APNs) delivery client. The zero value / the
// NewPushSender() form is UNCONFIGURED; NewPushSenderWithConfig arms it with
// detected credentials and a real *http.Client. When armed with a recognised
// provider and a non-empty token, SendTo performs a REAL provider HTTP call and
// returns the REAL outcome — a 2xx maps to success, any transport error or
// non-2xx maps to a real wrapped error. It NEVER fabricates a "sent" status
// (Constitution §11.4 anti-bluff covenant).
//
// HONEST BOUNDARY (Constitution §11.4.10, operator-gated): the unit tests in
// push_fcm_test.go / push_apns_test.go prove this client builds the correct
// request (URL, auth header, headers, body) and correctly handles 2xx / 4xx /
// 5xx responses against a MOCK httptest transport. Real end-to-end delivery to
// a physical device still requires the operator's actual FCM service-account
// JSON / APNs .p8 signing key + a real device registration token; that live
// path is not, and cannot be, verified from an autonomous unit test.
type PushSender struct {
	cfg        PushConfig
	configured bool

	// httpClient is the transport the provider clients use. nil => a default
	// real *http.Client (see client()). net/http auto-negotiates HTTP/2 over
	// TLS, which APNs requires — no explicit http2 wiring is needed.
	httpClient httpDoer

	// now is an injectable clock for JWT iat/exp (deterministic tests). nil =>
	// time.Now (see nowFn()).
	now func() time.Time

	// Endpoint overrides. Empty => real production endpoints. Set ONLY by tests
	// to point at an httptest.Server; production leaves them empty.
	fcmBaseURL    string // default https://fcm.googleapis.com
	oauthTokenURL string // override the service-account token_uri (tests)
	apnsBaseURL   string // default: cfg.APNsHost, else https://api.push.apple.com
}

// NewPushSender constructs an UNCONFIGURED PushSender. Its Send/SendTo always
// return ErrPushProviderNotConfigured. Kept unchanged so existing call
// sites/tests expecting the honest not-configured outcome stay green.
func NewPushSender() *PushSender { return &PushSender{} }

// NewPushSenderWithConfig constructs a PushSender armed with cfg (credentials
// detected) and a real HTTP client. Its SendTo performs REAL FCM/APNs delivery
// for a recognised provider + non-empty token.
func NewPushSenderWithConfig(cfg PushConfig) *PushSender {
	return &PushSender{
		cfg:        cfg,
		configured: true,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		now:        time.Now,
	}
}

// NewPushSenderForTesting constructs a PushSender armed with cfg (FCM/APNs
// credentials) whose provider HTTP calls are dispatched through client and
// fcmBaseURL/apnsBaseURL (e.g. an httptest.Server) instead of the real
// Google/Apple endpoints. Its purpose is letting OTHER packages (e.g.
// internal/handler's tests, via handler.NewWithDelivery) wire a REAL
// PushSender — exercising the exact request-construction/response-handling
// code path production uses (sendFCM/sendAPNs) — without contacting a live
// provider. Mirrors NewWebhookSenderForTesting's role for the webhook
// sender. Constitution §11.4.27 — the mock transport stands in for the
// third-party push backend (the operator-gated boundary documented on
// PushSender above); this constructor itself is test-only scaffolding, not
// a production entry point.
func NewPushSenderForTesting(cfg PushConfig, client *http.Client, fcmBaseURL, apnsBaseURL string) *PushSender {
	return &PushSender{
		cfg:         cfg,
		configured:  true,
		httpClient:  client,
		now:         time.Now,
		fcmBaseURL:  fcmBaseURL,
		apnsBaseURL: apnsBaseURL,
	}
}

// client returns the configured transport or a default real *http.Client. The
// default carries a bounded timeout so a hung provider can never wedge a
// request goroutine.
func (p *PushSender) client() httpDoer {
	if p.httpClient != nil {
		return p.httpClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// nowFn returns the injected clock or time.Now.
func (p *PushSender) nowFn() func() time.Time {
	if p.now != nil {
		return p.now
	}
	return time.Now
}

// Send is the legacy no-argument entry point retained for existing callers and
// tests. It delegates to SendTo with an empty token: an unconfigured sender
// returns ErrPushProviderNotConfigured (checked first), an armed sender returns
// ErrPushTokenEmpty (a real send needs a device token — never a fabricated
// success). Prefer SendTo for real delivery.
func (p *PushSender) Send() error {
	return p.SendTo(context.Background(), "", PushPayload{})
}

// SendTo performs a REAL push to token via the configured provider. It NEVER
// returns nil without a genuine provider 2xx, and NEVER fabricates delivery.
//
// Ordered, honest outcomes:
//   - unconfigured sender            => ErrPushProviderNotConfigured
//   - empty token                    => ErrPushTokenEmpty
//   - provider == fcm                => real FCM HTTP v1 (or legacy) delivery
//   - provider == apns               => real APNs HTTP/2 delivery
//   - unknown provider               => ErrPushProviderNotImplemented
//   - real transport error / non-2xx => a real wrapped error (not swallowed)
func (p *PushSender) SendTo(ctx context.Context, token string, payload PushPayload) error {
	if p == nil || !p.configured {
		return ErrPushProviderNotConfigured
	}
	if strings.TrimSpace(token) == "" {
		return ErrPushTokenEmpty
	}
	switch p.cfg.Provider {
	case PushProviderFCM:
		return p.sendFCM(ctx, token, payload)
	case PushProviderAPNs:
		return p.sendAPNs(ctx, token, payload)
	default:
		return ErrPushProviderNotImplemented
	}
}
