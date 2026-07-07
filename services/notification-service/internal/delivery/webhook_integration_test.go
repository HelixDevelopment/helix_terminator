//go:build integration

package delivery_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
)

// TestWebhookSender_RealHTTPDelivery_ReceiverConfirms is the rock-solid
// anti-bluff proof (Constitution §11.4.123) that WebhookSender.Send performs
// a REAL outbound HTTP POST: it stands up a real net/http receiver, sends a
// real webhook notification through WebhookSender, and asserts the receiver
// actually got the POST with the exact expected payload.
//
// This test's receiver is an httptest.Server bound to loopback (127.0.0.1),
// so it uses NewWebhookSenderForTesting — the production sender
// (NewWebhookSender) would correctly refuse to dial a loopback address
// (SSRF guard, see webhook_ssrf_test.go). This is a test-infra concession,
// not a weakening of the production path.
func TestWebhookSender_RealHTTPDelivery_ReceiverConfirms(t *testing.T) {
	var mu sync.Mutex
	var receivedBody []byte
	var receivedContentType string
	var receivedMethod string
	received := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		mu.Lock()
		receivedBody = body
		receivedContentType = r.Header.Get("Content-Type")
		receivedMethod = r.Method
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
		select {
		case received <- struct{}{}:
		default:
		}
	}))
	defer srv.Close()

	sender := delivery.NewWebhookSenderForTesting(5 * time.Second)
	payload := delivery.WebhookPayload{
		ID:      "11111111-1111-1111-1111-111111111111",
		UserID:  "22222222-2222-2222-2222-222222222222",
		Type:    "info",
		Title:   "Real webhook delivery proof",
		Message: "This is a real outbound HTTP POST integration test",
		Channel: "webhook",
		Data:    json.RawMessage(`{"k":"v"}`),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statusCode, err := sender.Send(ctx, srv.URL, payload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)

	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("receiver never observed the POST — webhook was NOT actually delivered")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, http.MethodPost, receivedMethod)
	assert.Equal(t, "application/json", receivedContentType)

	var decoded delivery.WebhookPayload
	require.NoError(t, json.Unmarshal(receivedBody, &decoded))
	assert.Equal(t, payload.ID, decoded.ID)
	assert.Equal(t, payload.UserID, decoded.UserID)
	assert.Equal(t, payload.Title, decoded.Title)
	assert.Equal(t, payload.Message, decoded.Message)
	assert.JSONEq(t, string(payload.Data), string(decoded.Data))
}

// TestWebhookSender_NonSuccessStatus_ReturnsHonestFailure proves a non-2xx
// receiver response is surfaced as a real error — never silently accepted
// as "delivered". The receiver is loopback, so this uses the test-permissive
// constructor (see TestWebhookSender_RealHTTPDelivery_ReceiverConfirms).
func TestWebhookSender_NonSuccessStatus_ReturnsHonestFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	sender := delivery.NewWebhookSenderForTesting(5 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statusCode, err := sender.Send(ctx, srv.URL, delivery.WebhookPayload{ID: "x"})
	require.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, statusCode)
}

// TestWebhookSender_UnreachableURL_ReturnsHonestFailure proves a genuinely
// unreachable-but-otherwise-permitted target surfaces as an error, never a
// fabricated success. Uses the test-permissive constructor so the failure
// this test is actually named for (connection refused on an open loopback
// port with nothing listening) is what fires — NOT the SSRF guard, which
// has its own dedicated proof in webhook_ssrf_test.go and would otherwise
// preempt this test's intent now that loopback is guarded in production.
func TestWebhookSender_UnreachableURL_ReturnsHonestFailure(t *testing.T) {
	sender := delivery.NewWebhookSenderForTesting(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := sender.Send(ctx, "http://127.0.0.1:1/nowhere", delivery.WebhookPayload{ID: "x"})
	require.Error(t, err)
}
