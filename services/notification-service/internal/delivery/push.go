package delivery

import (
	"errors"
	"os"
)

// ErrPushProviderNotConfigured is returned by PushSender.Send when NO push
// provider credentials are present. Push delivery (FCM/APNs) requires
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

// ErrPushProviderNotImplemented is returned by PushSender.Send when push
// provider credentials ARE present (the sender is armed) but the real FCM
// HTTP v1 / APNs HTTP/2 delivery client has not been built yet. This is an
// HONEST not-yet-implemented state (Constitution §11.4 anti-bluff covenant):
// detecting credentials MUST NEVER be reported as a delivered push — no message
// is actually sent until the real provider client lands. Callers persist
// notification.Status = "pending_provider_unconfigured" for BOTH errors: in
// neither case did a push actually leave this process.
var ErrPushProviderNotImplemented = errors.New(
	"push provider credentials present but the FCM HTTP v1 / APNs HTTP/2 delivery client is not yet implemented: no push was sent",
)

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
		}, true
	}

	return PushConfig{}, false
}

// PushSender is a push (FCM/APNs) delivery client. The zero value / the
// NewPushSender() form is UNCONFIGURED; NewPushSenderWithConfig arms it with
// detected credentials. In BOTH forms Send performs NO network call and NEVER
// fabricates a "sent" status — the real FCM HTTP v1 / APNs HTTP/2 client is not
// yet built (see ErrPushProviderNotImplemented).
type PushSender struct {
	cfg        PushConfig
	configured bool
}

// NewPushSender constructs an UNCONFIGURED PushSender. Its Send always returns
// ErrPushProviderNotConfigured. Kept unchanged so existing call sites/tests
// expecting the honest not-configured outcome stay green.
func NewPushSender() *PushSender { return &PushSender{} }

// NewPushSenderWithConfig constructs a PushSender armed with cfg (credentials
// detected). Its Send returns ErrPushProviderNotImplemented — the credentials
// are present but the real delivery client is not yet built (honest boundary).
func NewPushSenderWithConfig(cfg PushConfig) *PushSender {
	return &PushSender{cfg: cfg, configured: true}
}

// Send NEVER performs a real push and NEVER returns nil. When the sender is
// unconfigured it returns ErrPushProviderNotConfigured; when it is armed with
// credentials it returns ErrPushProviderNotImplemented. It exists (rather than
// being omitted) so the handler has one obvious, testable integration point to
// wire real FCM/APNs clients into later (Constitution §11.4 anti-bluff
// covenant — no fabricated delivery).
func (p *PushSender) Send() error {
	if p == nil || !p.configured {
		return ErrPushProviderNotConfigured
	}
	return ErrPushProviderNotImplemented
}
