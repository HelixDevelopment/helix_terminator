package delivery_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
)

// TestWebhookSender_ProductionSender_RefusesSSRFTargets is the rock-solid
// anti-bluff proof (Constitution §11.4.123/§11.4.133) that the PRODUCTION
// WebhookSender (the one handler.New() actually wires into every
// CreateNotification request) genuinely refuses to dial the classic SSRF
// destination classes: loopback, RFC1918 private, and the cloud
// metadata/link-local address. No live listener is required for these
// targets — the net.Dialer.Control guard fires BEFORE the connect()
// syscall is attempted, so the test is deterministic and needs no network
// access at all.
func TestWebhookSender_ProductionSender_RefusesSSRFTargets(t *testing.T) {
	blocked := []struct {
		name string
		url  string
	}{
		{"loopback_127_0_0_1", "http://127.0.0.1:65535/hook"},
		{"loopback_localhost_literal_ip", "http://127.0.0.2:65535/hook"},
		{"loopback_ipv6", "http://[::1]:65535/hook"},
		{"cloud_metadata_169_254_169_254", "http://169.254.169.254/latest/meta-data/"},
		{"link_local_other", "http://169.254.1.1/hook"},
		{"rfc1918_10", "http://10.0.0.1/hook"},
		{"rfc1918_172_16", "http://172.16.5.5/hook"},
		{"rfc1918_192_168", "http://192.168.1.1/hook"},
		{"cgn_100_64", "http://100.64.0.1/hook"},
		{"cgn_100_127_upper_bound", "http://100.127.255.254/hook"},
		{"unspecified_v4", "http://0.0.0.0/hook"},
		{"multicast_v4", "http://224.0.0.1/hook"},
	}

	sender := delivery.NewWebhookSender(2 * time.Second)

	for _, tc := range blocked {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			_, err := sender.Send(ctx, tc.url, delivery.WebhookPayload{ID: "ssrf-probe"})
			require.Error(t, err, "production WebhookSender must refuse to dial SSRF target %s", tc.url)
			assert.Contains(t, strings.ToLower(err.Error()), "guard",
				"error for %s must come from the SSRF destination guard, not some other failure", tc.url)
		})
	}
}

// TestWebhookSender_ProductionSender_AllowsLegitimateTarget proves the SSRF
// guard is not overly broad: a target that is NOT loopback/private/
// link-local/multicast/CGN must still be dialable. Since this repo's test
// environment has no routable public IP to safely target, this test
// verifies the guard's ALLOW decision directly by confirming a definitely
// non-blocked address forwards to a real (non-SSRF-guard) network error
// rather than being pre-emptively rejected by the guard.
func TestWebhookSender_ProductionSender_AllowsLegitimateTarget(t *testing.T) {
	sender := delivery.NewWebhookSender(1 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// TEST-NET-1 (RFC 5737, 192.0.2.0/24): reserved for documentation, is
	// NOT loopback/private/link-local/multicast/unspecified/CGN, so the
	// guard must allow the dial attempt through — it will then genuinely
	// time out (nothing is reachable there), proving the failure is a
	// network-layer timeout, not an SSRF-guard rejection.
	_, err := sender.Send(ctx, "http://192.0.2.1/hook", delivery.WebhookPayload{ID: "allow-probe"})
	require.Error(t, err, "TEST-NET-1 is unreachable so Send must still fail")
	assert.NotContains(t, strings.ToLower(err.Error()), "guard",
		"a non-blocked destination must fail on network grounds, never on the SSRF guard")
}

// TestWebhookSender_TestingConstructor_AllowsLoopback proves the
// test-permissive constructor (NewWebhookSenderForTesting) — used ONLY by
// this package's own integration tests against httptest.Server receivers —
// genuinely delivers to a real loopback receiver, so the SSRF-guard tests
// above are proven against the correct baseline (the guard, not something
// else, is what blocks production Send calls).
func TestWebhookSender_TestingConstructor_AllowsLoopback(t *testing.T) {
	received := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		received <- struct{}{}
	}))
	defer srv.Close()

	sender := delivery.NewWebhookSenderForTesting(5 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := sender.Send(ctx, srv.URL, delivery.WebhookPayload{ID: "loopback-allowed"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("loopback receiver never observed the POST — test-permissive constructor did not actually deliver")
	}
}

// TestWebhookSender_TargetURLTooLong_ReturnsError proves the length-cap
// audit fix: an excessively long target URL is rejected before any network
// activity is attempted.
func TestWebhookSender_TargetURLTooLong_ReturnsError(t *testing.T) {
	sender := delivery.NewWebhookSender(time.Second)
	longURL := "http://example.com/" + strings.Repeat("a", 3000)

	_, err := sender.Send(context.Background(), longURL, delivery.WebhookPayload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum length")
}
