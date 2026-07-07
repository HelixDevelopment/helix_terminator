//go:build integration

// Package server_test real-network-hop integration coverage for
// gateway-service (queue#4, §11.4.27).
//
// The pre-existing unit test suite in server_test.go drives the gateway
// exclusively in-process via httptest.NewRecorder + Router().ServeHTTP —
// no TCP socket, no real listener, no real upstream. This file adds the
// fleet's first genuine end-to-end network test for gateway-service: it
// starts the REAL gateway HTTP server bound to a real loopback TCP port
// (mirroring cmd/gateway-service/main.go's own wiring: http.Server{Handler:
// srv.Router()} + Serve(listener)), starts a REAL, independent net/http
// upstream server on a second real port, points the gateway at it via the
// <SERVICE>_ADDR environment override added alongside the real
// reverse-proxy implementation in proxyTo, and drives a REAL http.Client
// over loopback through the gateway's full middleware chain (request-ID,
// logging, CORS, rate-limit, JWT auth) and its real reverse-proxy hop to
// the upstream and back.
//
// Run with: go test -tags integration ./internal/server/...
package server_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/gateway-service/internal/server"
)

// recordedUpstreamRequest captures what the REAL upstream process actually
// received on its own real socket — the positive evidence that gateway's
// proxy hop genuinely reached it over the network (§11.4.69).
type recordedUpstreamRequest struct {
	Method          string
	Path            string
	GatewayUpstream string
	RequestID       string
	Authorization   string
}

// startRealUpstream starts a genuine net/http server bound to a real
// loopback TCP port (a distinct OS-level socket from the gateway's own),
// standing in for a Helix Terminator upstream (host-service). It records
// every request it genuinely receives and returns a realistic JSON payload
// that could not have originated from the gateway itself.
func startRealUpstream(t *testing.T) (baseURL string, recorded func() []recordedUpstreamRequest, stop func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var mu sync.Mutex
	var seen []recordedUpstreamRequest

	mux := http.NewServeMux()
	mux.HandleFunc("/hosts", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, recordedUpstreamRequest{
			Method:          r.Method,
			Path:            r.URL.Path,
			GatewayUpstream: r.Header.Get("X-Gateway-Upstream"),
			RequestID:       r.Header.Get("X-Request-ID"),
			Authorization:   r.Header.Get("Authorization"),
		})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"real_upstream":"host-service","hosts":[{"id":"h-1","name":"prod-db-1.internal"}]}`)
	})

	httpSrv := &http.Server{Handler: mux}
	go func() {
		_ = httpSrv.Serve(ln)
	}()

	return "http://" + ln.Addr().String(),
		func() []recordedUpstreamRequest {
			mu.Lock()
			defer mu.Unlock()
			out := make([]recordedUpstreamRequest, len(seen))
			copy(out, seen)
			return out
		},
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = httpSrv.Shutdown(ctx)
		}
}

// startRealGateway starts the REAL gateway server (server.New + its actual
// Router()) bound to a real loopback TCP port, using the same
// http.Server{Handler: ...} + Serve(listener) pattern as
// cmd/gateway-service/main.go — not httptest.NewRecorder, a real listener.
func startRealGateway(t *testing.T) (baseURL string, stop func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := server.New(nil)
	httpSrv := &http.Server{Handler: srv.Router()}
	go func() {
		_ = httpSrv.Serve(ln)
	}()

	return "http://" + ln.Addr().String(), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
	}
}

// TestIntegration_GatewayProxiesRealHTTPRequestToRealUpstream is the
// queue#4 real-network-hop proof: a genuine loopback TCP client request
// through the real gateway server, through its real middleware chain
// (JWT auth), through a real reverse-proxy hop, to a real independent
// upstream process, and the real upstream's real response flowing back
// through gateway to the client.
func TestIntegration_GatewayProxiesRealHTTPRequestToRealUpstream(t *testing.T) {
	// testPublicKey / testPrivateKey / generateTestToken() are the shared
	// key pair + helper declared in server_test.go (same server_test
	// package, always compiled), reused here for consistency.
	upstreamURL, recordedRequests, stopUpstream := startRealUpstream(t)
	defer stopUpstream()

	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	t.Setenv("HOST_SERVICE_ADDR", upstreamURL)

	gatewayURL, stopGateway := startRealGateway(t)
	defer stopGateway()

	client := &http.Client{Timeout: 10 * time.Second}

	// --- (b) missing auth => real rejection over the real network, and
	// the real upstream must NOT have been contacted at all. ---
	unauthReq, err := http.NewRequest(http.MethodGet, gatewayURL+"/api/v1/hosts", nil)
	require.NoError(t, err)
	unauthResp, err := client.Do(unauthReq)
	require.NoError(t, err, "real TCP request to gateway must succeed at the transport layer")
	unauthBody, err := io.ReadAll(unauthResp.Body)
	require.NoError(t, err)
	_ = unauthResp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, unauthResp.StatusCode,
		"missing Authorization header must be rejected by the real JWT middleware")
	require.Contains(t, string(unauthBody), "missing authorization header")
	require.Empty(t, recordedRequests(),
		"the real upstream must not have been reached when auth middleware rejects the request")

	// --- invalid token => real rejection, upstream still not reached. ---
	invalidReq, err := http.NewRequest(http.MethodGet, gatewayURL+"/api/v1/hosts", nil)
	require.NoError(t, err)
	invalidReq.Header.Set("Authorization", "Bearer not-a-real-jwt")
	invalidResp, err := client.Do(invalidReq)
	require.NoError(t, err)
	invalidBody, err := io.ReadAll(invalidResp.Body)
	require.NoError(t, err)
	_ = invalidResp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, invalidResp.StatusCode)
	require.Contains(t, string(invalidBody), "invalid token")
	require.Empty(t, recordedRequests(), "an invalid token must not reach the real upstream")

	// --- (a) valid auth => real request flows all the way through: real
	// gateway middleware -> real reverse-proxy hop -> real upstream ->
	// real upstream's real response back through gateway to the client. ---
	requestID := "it-req-real-hop-001"
	token := generateTestToken()

	validReq, err := http.NewRequest(http.MethodGet, gatewayURL+"/api/v1/hosts", nil)
	require.NoError(t, err)
	validReq.Header.Set("Authorization", "Bearer "+token)
	validReq.Header.Set("X-Request-ID", requestID)

	validResp, err := client.Do(validReq)
	require.NoError(t, err, "real TCP request through the gateway to the real upstream must succeed")
	validBody, err := io.ReadAll(validResp.Body)
	require.NoError(t, err)
	_ = validResp.Body.Close()

	// The gateway's own response IS the real upstream's real response body
	// — not a gateway-fabricated stub. This is the crux of the anti-bluff
	// proof: "request routed to host-service" (the OLD stub) would fail
	// this assertion; only a genuine proxied round trip produces it.
	require.Equal(t, http.StatusOK, validResp.StatusCode)
	require.Contains(t, string(validBody), `"real_upstream":"host-service"`,
		"response body must be the REAL upstream's real payload, proxied back through gateway")
	require.Contains(t, string(validBody), "prod-db-1.internal")
	require.Equal(t, requestID, validResp.Header.Get("X-Request-ID"),
		"gateway's own request-ID middleware header must survive the proxy hop")

	// Cross-check from the OTHER side of the real network hop: the real
	// upstream process, on its own real socket, genuinely received the
	// forwarded request with the auth middleware's context intact.
	reqs := recordedRequests()
	require.Len(t, reqs, 1, "the real upstream must have received exactly one real request")
	got := reqs[0]
	require.Equal(t, http.MethodGet, got.Method)
	require.Equal(t, "/hosts", got.Path)
	require.Equal(t, "host-service", got.GatewayUpstream,
		"gateway must identify itself/the route to the real upstream")
	require.Equal(t, requestID, got.RequestID,
		"the client's X-Request-ID must be forwarded over the real network hop")
	require.Equal(t, "Bearer "+token, got.Authorization,
		"the original Authorization header must be forwarded to the real upstream")
}

// TestIntegration_GatewayRejectsUnknownUpstream proves the gateway's
// service-unavailable path is also exercised over a real network hop
// (no upstream override configured => the loopback-only default address
// is unreachable in this sandboxed environment => real 502, never a
// fabricated 200).
func TestIntegration_GatewayReturnsBadGatewayForUnreachableUpstream(t *testing.T) {
	// /api/v1/auth/login is unauthenticated (jwtValidationMiddleware skips
	// /api/v1/auth/*), so no JWT setup is needed for this test.

	// Point a real, but nobody's-listening, loopback port so the real
	// network hop genuinely fails (connection refused) rather than being
	// simulated.
	deadLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	deadAddr := "http://" + deadLn.Addr().String()
	require.NoError(t, deadLn.Close()) // free the port so nothing answers on it

	t.Setenv("AUTH_SERVICE_ADDR", deadAddr)

	gatewayURL, stopGateway := startRealGateway(t)
	defer stopGateway()

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, gatewayURL+"/api/v1/auth/login", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err, "the real TCP request to the gateway itself must still succeed")
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusBadGateway, resp.StatusCode,
		"a real, genuinely-unreachable upstream must surface as a real 502, never a fabricated 200")
	require.Contains(t, string(body), "auth-service")
}
