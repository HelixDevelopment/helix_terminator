package handler

// T-PUSH-WIRING: this file is a WHITE-BOX test (package handler, not
// handler_test) because it calls the unexported deliverPush method
// directly — the cleanest way to prove the ARMED-provider wiring without
// depending on a live Postgres/MailHog stack (which the equivalent
// handler_test-package proof, delivery_integration_test.go, requires and is
// gated behind the `integration` build tag). It uses
// delivery.NewPushSenderForTesting (a small, explicit test-only seam added
// alongside this fix) to arm a REAL PushSender against an httptest.Server
// standing in for the FCM legacy endpoint, so the request-construction +
// response-handling code path exercised is the exact one production uses
// (PushSender.SendTo -> sendFCM -> sendFCMLegacy), never a hand-rolled
// double (Constitution §11.4.27 — mocks/stubs confined to this unit test,
// the mocked transport stands in ONLY for the third-party FCM endpoint
// itself, the operator-gated boundary).
//
// PR #8 review finding (Important): pre-fix, deliverPush called the
// arg-less PushSender.Send() (always SendTo(ctx, "", PushPayload{})), so an
// armed provider's empty token short-circuited to ErrPushTokenEmpty BEFORE
// sendFCM/sendAPNs ran — an armed provider produced ZERO real sends and the
// handler reported "pending_provider_unconfigured" even when a provider WAS
// configured. TestDeliverPush_ArmedProvider_RealTargetReachesClient below is
// RED against the pre-fix arg-less-Send() body (see the mutation-check note
// at the bottom of this file) and GREEN against the fixed SendTo(ctx,
// n.Target, ...) body.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
	"github.com/helixdevelopment/notification-service/internal/model"
)

// TestDeliverPush_ArmedProvider_RealTargetReachesClient proves an ARMED push
// sender with a notification carrying a real Target REACHES the real FCM
// legacy client (the exact production code path, PushSender.SendTo ->
// sendFCM -> sendFCMLegacy) and sets Status="sent" on a genuine provider
// success — the wiring gap this fix closes.
func TestDeliverPush_ArmedProvider_RealTargetReachesClient(t *testing.T) {
	var gotAuth, gotBody string
	var requests int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":1,"failure":0,"results":[{"message_id":"m1"}]}`))
	}))
	defer ts.Close()

	cfg := delivery.PushConfig{Provider: delivery.PushProviderFCM, FCMServerKey: "test-server-key"}
	sender := delivery.NewPushSenderForTesting(cfg, ts.Client(), ts.URL, "")
	h := NewWithDelivery(nil, nil, nil, sender)

	n := &model.Notification{
		Target:  "device-token-abc-123",
		Title:   "Wiring proof",
		Message: "deliverPush must reach the real client",
		Data:    []byte(`{"k":"v"}`),
	}

	h.deliverPush(context.Background(), n)

	require.Equal(t, 1, requests, "the real FCM client must have been invoked exactly once — the wiring gap this fix closes")
	assert.Equal(t, "key=test-server-key", gotAuth, "the real provider auth header must be set — proves sendFCMLegacy actually ran")
	assert.Contains(t, gotBody, "device-token-abc-123", "the request body sent to the provider must carry the notification's REAL target token")
	assert.Contains(t, gotBody, "Wiring proof", "the request body must carry the notification's real title")
	assert.Contains(t, gotBody, "deliverPush must reach the real client", "the request body must carry the notification's real message as the push body")

	assert.Equal(t, "sent", n.Status, "a genuine provider success must be reflected as sent, never left at a not-configured/unconfigured status")
	require.NotNil(t, n.SentAt)
}

// TestDeliverPush_ArmedProvider_SendErrorIsSurfacedHonestly proves a REAL
// (mocked-transport) provider error is surfaced as "failed" — never a
// fabricated "sent" and never mislabelled "pending_provider_unconfigured"
// (which would falsely claim no provider is armed).
func TestDeliverPush_ArmedProvider_SendErrorIsSurfacedHonestly(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`upstream provider error`))
	}))
	defer ts.Close()

	cfg := delivery.PushConfig{Provider: delivery.PushProviderFCM, FCMServerKey: "test-server-key"}
	sender := delivery.NewPushSenderForTesting(cfg, ts.Client(), ts.URL, "")
	h := NewWithDelivery(nil, nil, nil, sender)

	n := &model.Notification{
		Target:  "device-token-xyz",
		Title:   "Error path",
		Message: "provider will reject this",
	}

	h.deliverPush(context.Background(), n)

	assert.Equal(t, "failed", n.Status, "a real provider-side send error must be surfaced honestly as failed")
	assert.Nil(t, n.SentAt)
}

// TestDeliverPush_ArmedProvider_EmptyTargetIsDistinctFromUnconfigured proves
// a configured provider with a notification that carries NO device token
// gets a status DISTINCT from "pending_provider_unconfigured" (which would
// now be inaccurate — a provider genuinely IS configured) and never
// contacts the provider (no token to send to).
func TestDeliverPush_ArmedProvider_EmptyTargetIsDistinctFromUnconfigured(t *testing.T) {
	var requests int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":1,"failure":0}`))
	}))
	defer ts.Close()

	cfg := delivery.PushConfig{Provider: delivery.PushProviderFCM, FCMServerKey: "test-server-key"}
	sender := delivery.NewPushSenderForTesting(cfg, ts.Client(), ts.URL, "")
	h := NewWithDelivery(nil, nil, nil, sender)

	n := &model.Notification{
		Target:  "",
		Title:   "No target",
		Message: "no device token on this notification",
	}

	h.deliverPush(context.Background(), n)

	assert.Equal(t, "failed_missing_target", n.Status)
	assert.NotEqual(t, "pending_provider_unconfigured", n.Status, "a genuinely configured provider must never be reported as unconfigured")
	assert.Equal(t, 0, requests, "an empty target must never reach the provider")
}

// TestDeliverPush_UnconfiguredProvider_HonestNotConfigured is the unchanged
// honest baseline: with NO provider armed (h.pushSender == nil), the status
// stays "pending_provider_unconfigured" — proves this fix did not regress
// the pre-existing honest not-configured path (mirrors
// delivery.TestPushSender_AlwaysHonestlyUnconfigured at the handler layer).
func TestDeliverPush_UnconfiguredProvider_HonestNotConfigured(t *testing.T) {
	h := NewWithDelivery(nil, nil, nil, nil)
	n := &model.Notification{Target: "device-token", Title: "t", Message: "m"}

	h.deliverPush(context.Background(), n)

	assert.Equal(t, "pending_provider_unconfigured", n.Status)
	assert.Nil(t, n.SentAt)
}

// TestDeliverPush_UnconfiguredNonNilSender_HonestNotConfigured proves the
// SAME honest status when h.pushSender is a non-nil-but-unconfigured
// PushSender (delivery.NewPushSender()) — the exact construction the
// pre-existing integration test TestCreateNotification_Push_HonestNotConfigured
// (delivery_integration_test.go) wires, so this fix must not change ITS
// outcome.
func TestDeliverPush_UnconfiguredNonNilSender_HonestNotConfigured(t *testing.T) {
	h := NewWithDelivery(nil, nil, nil, delivery.NewPushSender())
	n := &model.Notification{Target: "device-token", Title: "t", Message: "m"}

	h.deliverPush(context.Background(), n)

	assert.Equal(t, "pending_provider_unconfigured", n.Status)
	assert.Nil(t, n.SentAt)
}

// MUTATION-CHECK (performed manually, Constitution §1.1 / §11.4.43): reverting
// deliverPush's SendTo(ctx, n.Target, delivery.PushPayload{...}) call back to
// the pre-fix arg-less h.pushSender.Send() makes
// TestDeliverPush_ArmedProvider_RealTargetReachesClient fail — Send() always
// dispatches SendTo(ctx, "", PushPayload{}), which resolves to
// ErrPushTokenEmpty before ever reaching the mock server, so `requests` stays
// 0 (require.Equal(t, 1, requests, ...) fails) and n.Status would land on
// "pending_provider_unconfigured" (the pre-fix mapping used for ANY Send()
// error), not "sent". This was verified by temporarily reverting the call
// site and confirming the RED failure, then re-applying the fix and
// confirming GREEN — see the session report for the captured before/after
// `go test` output.
