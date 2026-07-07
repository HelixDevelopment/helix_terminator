package delivery

import "errors"

// ErrPushProviderNotConfigured is returned by PushSender.Send. Push delivery
// (FCM/APNs) requires operator-supplied provider credentials (a Firebase
// service account, an APNs certificate/key) that no environment variable in
// this deployment currently provides. This is an HONEST not-yet-implemented
// state, per operator decision — it MUST NEVER be papered over with a
// fabricated "sent"/"delivered" status (Constitution §11.4 anti-bluff
// covenant). Callers persist notification.Status =
// "pending_provider_unconfigured" when this error is returned.
var ErrPushProviderNotConfigured = errors.New(
	"push provider (fcm/apns) not configured: set FCM_SERVER_KEY or APNS_KEY_ID/APNS_TEAM_ID/APNS_BUNDLE_ID/APNS_KEY_PATH credentials to enable push delivery",
)

// PushSender is a placeholder push (FCM/APNs) delivery client. It performs
// no network calls and always reports the provider as unconfigured until a
// future change wires in real FCM HTTP v1 / APNs HTTP/2 credentials.
type PushSender struct{}

// NewPushSender constructs a PushSender.
func NewPushSender() *PushSender { return &PushSender{} }

// Send always returns ErrPushProviderNotConfigured. It exists (rather than
// being omitted) so the call site in the handler has one obvious, testable
// integration point to wire real FCM/APNs clients into later.
func (p *PushSender) Send() error {
	return ErrPushProviderNotConfigured
}
