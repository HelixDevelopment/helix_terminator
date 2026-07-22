//go:build integration

// This file is the rock-solid anti-bluff proof (Constitution §11.4.123 /
// §11.4.27(B)) that SlackSender.Send performs a REAL call into Herald's
// Slack channel adapter — no mock of Herald's own code, and no bypass of
// notification-service's own production wrapper. It drives the exact
// public API handler.go's deliverSlack uses in production
// (delivery.NewSlackSenderForTesting mirrors NewConfiguredSlackSender's
// construction, only substituting an httptest.Server base URL for the
// live Slack Web API — see slack_herald.go) against a real HTTP receiver
// standing in for Slack's chat.postMessage endpoint, reusing Herald's own
// already-built hermetic-but-real test seam (heraldslack.NewWithBaseURL,
// the identical mechanism Herald's own send_test.go/
// send_integration_test.go use) rather than reinventing one.
//
// Requires the `integration` build tag (Herald's real adapter, unlike in
// an earlier round of this change, now compiles by DEFAULT — see slack.go
// and slack_herald.go's package doc comments — so no separate build tag
// is needed to reach it, only the standard integration-test gate):
//
//	go test -tags integration -run TestSlackSender_RealHeraldWireDelivery -v ./internal/delivery/
package delivery_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
)

// TestSlackSender_RealHeraldWireDelivery drives notification-service's
// public SlackSender.Send — the SAME call handler.go's deliverSlack makes
// — against a real HTTP receiver standing in for Slack's chat.postMessage
// endpoint, and asserts the receiver actually observed a real
// chat.postMessage call carrying the expected channel + text, with a real
// `ts` accepted back through the full production code path with no
// error.
func TestSlackSender_RealHeraldWireDelivery(t *testing.T) {
	type postMessageCall struct {
		Channel string
		Text    string
	}
	received := make(chan postMessageCall, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.postMessage" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		require.NoError(t, r.ParseForm())
		call := postMessageCall{Channel: r.FormValue("channel"), Text: r.FormValue("text")}
		select {
		case received <- call:
		default:
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"channel":"` + call.Channel + `","ts":"1234567890.123456"}`))
	}))
	defer srv.Close()

	sender, err := delivery.NewSlackSenderForTesting("xoxb-test-token", srv.URL)
	require.NoError(t, err)
	require.NotNil(t, sender)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = sender.Send(ctx, "C0123ABCD", "Real Herald wire delivery proof")
	require.NoError(t, err, "SlackSender.Send must succeed against a receiver that returns a real ts")

	select {
	case call := <-received:
		assert.Equal(t, "C0123ABCD", call.Channel)
		assert.Equal(t, "Real Herald wire delivery proof", call.Text)
	default:
		t.Fatal("chat.postMessage receiver never observed the call — Slack delivery was NOT actually attempted")
	}
}

// TestSlackSender_RealHeraldWireDelivery_EmptyTsIsAnError proves Herald's
// own §107 bluff guard (send.go: "empty ts in chat.postMessage response")
// propagates as a real error through notification-service's wrapper —
// i.e. a technically-200-OK-but-degenerate Slack response is NOT
// misreported as a successful send.
func TestSlackSender_RealHeraldWireDelivery_EmptyTsIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"channel":"C0123ABCD","ts":""}`))
	}))
	defer srv.Close()

	sender, err := delivery.NewSlackSenderForTesting("xoxb-test-token", srv.URL)
	require.NoError(t, err)

	err = sender.Send(context.Background(), "C0123ABCD", "should not report success")
	require.Error(t, err, "an empty ts must surface as an error, never a fabricated success")
}
