package delivery_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
)

// TestPushSender_Send_LiveFCM is the REAL-send proof: when live FCM
// credentials AND a real device registration token are present in the
// environment, it performs a genuine FCM HTTP v1 send against Google's
// actual endpoint (https://fcm.googleapis.com) and requires it to succeed.
//
// When either is absent it SKIPs with an explicit, honest reason
// (Constitution §11.4.3 — SKIP-by-design when the required topology/
// credentials are absent is the correct fallback; a disguised PASS is
// forbidden). This is intentionally NOT gated behind a build tag: it is
// part of the normal `go test ./...` run, self-skips instantly (no network
// attempt) when unconfigured, and only reaches the network when an
// operator has deliberately set both variables to drive a real send.
//
// FCM_TEST_DEVICE_TOKEN must be a real registration token from a device (or
// emulator) that has the target Firebase app installed and has granted
// notification permission — obtain one by running the client app once and
// logging the token the Firebase SDK reports on registration.
func TestPushSender_Send_LiveFCM(t *testing.T) {
	path := os.Getenv("FCM_SERVICE_ACCOUNT_JSON")
	token := os.Getenv("FCM_TEST_DEVICE_TOKEN")
	if path == "" || token == "" {
		t.Skip("SKIP (§11.4.3): FCM_SERVICE_ACCOUNT_JSON and/or FCM_TEST_DEVICE_TOKEN not set in the environment " +
			"— no live FCM credentials / real device registration token available to drive a genuine send. " +
			"Set FCM_SERVICE_ACCOUNT_JSON (see scripts/firebase/firebase_setup.sh) and FCM_TEST_DEVICE_TOKEN " +
			"(a real token from a device running the target Firebase app) to exercise this test against the " +
			"real FCM HTTP v1 endpoint.")
	}

	sender, err := delivery.NewConfiguredPushSender(delivery.PushConfig{ServiceAccountJSONPath: path})
	require.NoError(t, err, "FCM_SERVICE_ACCOUNT_JSON is set but the service account JSON failed to load")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = sender.Send(ctx, token, delivery.PushMessage{
		Title: "helix_terminator live FCM test",
		Body:  "Constitution §11.4.98 real send proof " + time.Now().UTC().Format(time.RFC3339),
		Data:  map[string]string{"source": "push_live_test"},
	})
	require.NoError(t, err, "live FCM send must succeed with valid credentials and a valid device token")
}
