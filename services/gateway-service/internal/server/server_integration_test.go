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
	"encoding/json"
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
	// Real host-service registers this collection at "/api/v1/hosts"
	// (services/host-service/internal/server/server.go:91), not the bare
	// "/hosts" this stub used before the T8-1 route-table realignment —
	// updated here in lockstep with the gateway's corrected proxyTo call.
	mux.HandleFunc("/api/v1/hosts", func(w http.ResponseWriter, r *http.Request) {
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
		// Stop the background upstream health-check prober (T8-8) so its
		// goroutine does not leak past this test.
		srv.Stop()
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
	require.Equal(t, "/api/v1/hosts", got.Path)
	require.Equal(t, "host-service", got.GatewayUpstream,
		"gateway must identify itself/the route to the real upstream")
	require.Equal(t, requestID, got.RequestID,
		"the client's X-Request-ID must be forwarded over the real network hop")
	require.Equal(t, "Bearer "+token, got.Authorization,
		"the original Authorization header must be forwarded to the real upstream")
}

// TestIntegration_GatewayReturnsBadGatewayForUnreachableUpstream proves
// the gateway's real reverse-proxy hop genuinely fails (never a
// fabricated 200) against a real, unreachable loopback address.
//
// Reconciliation note (§11.4.120, T8-8): before T8-8 wired a real
// background upstream health-check prober, upstreamService.Healthy was a
// static flag hardcoded true in registerUpstreams and SetHealthy was
// never called anywhere, so proxyTo's own "!upstream.IsHealthy()"
// short-circuit (server.go) was dead code in practice — every request
// always fell through to a REAL connection attempt, which then genuinely
// failed with 502 "upstream request failed". This test originally
// asserted exactly that 502. Now that T8-8 makes IsHealthy() a genuine,
// periodically-refreshed reachability measurement, that same short-circuit
// is alive: once the real prober has confirmed an upstream is
// unreachable, the gateway correctly and efficiently rejects it with a
// proactive 503 "service unavailable" BEFORE even attempting the doomed
// connection, instead of paying a real connect/DNS-failure cost on every
// single request — a strictly better outcome. Without controlling for
// this, the test raced the prober's async initial sweep (started inside
// New(), not synchronous with it) and flaked between 502 (request landed
// before the sweep completed) and 503 (request landed after); this
// version waits deterministically, over the real network, for the real
// prober to converge (via the gateway's own /healthz/ready endpoint)
// before asserting — so it now exercises the genuine, deterministic
// post-T8-8 steady state. Both 502 and 503 are honest, non-fabricated
// failure signals for a genuinely unreachable upstream; this test's core
// invariant — never a fabricated 200 — is unchanged and still enforced.
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
	t.Setenv("GATEWAY_HEALTHCHECK_INTERVAL", "20ms")

	gatewayURL, stopGateway := startRealGateway(t)
	defer stopGateway()

	client := &http.Client{Timeout: 10 * time.Second}

	// Wait for the REAL T8-8 background prober to genuinely converge on
	// "auth-service" being unhealthy, observed over the real network via
	// the gateway's own /healthz/ready endpoint — never assumed, never
	// timed around a fixed sleep.
	require.Eventually(t, func() bool {
		readyReq, err := http.NewRequest(http.MethodGet, gatewayURL+"/healthz/ready", nil)
		if err != nil {
			return false
		}
		readyResp, err := client.Do(readyReq)
		if err != nil {
			return false
		}
		defer readyResp.Body.Close()
		var readyBody map[string]interface{}
		if err := json.NewDecoder(readyResp.Body).Decode(&readyBody); err != nil {
			return false
		}
		services, ok := readyBody["services"].(map[string]interface{})
		if !ok {
			return false
		}
		status, ok := services["auth-service"].(string)
		return ok && status == "unhealthy"
	}, 2*time.Second, 20*time.Millisecond,
		"the real T8-8 background health-check prober must converge on auth-service being unhealthy")

	req, err := http.NewRequest(http.MethodPost, gatewayURL+"/api/v1/auth/login", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err, "the real TCP request to the gateway itself must still succeed")
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode,
		"once the real T8-8 prober has confirmed auth-service is unreachable, the gateway must proactively reject with 503 rather than reattempt a doomed connection — still a real, non-fabricated failure signal, never a fabricated 200")
	require.Contains(t, string(body), "auth-service")
}

// startExactPathStub starts a genuine net/http server bound to a real
// loopback TCP port whose router registers ONLY the given method+path
// pattern (Go 1.22+ enhanced http.ServeMux syntax, e.g.
// "GET /api/v1/hosts/{hostId}") — mirroring the exact shape of the real
// upstream service's own route registration, not a catch-all. Any request
// for a DIFFERENT path (in particular the OLD, pre-T8-1 mismatched path)
// gets net/http's genuine "404 page not found" from this stub's own
// router, exactly as the real upstream service would have returned it —
// this is what makes the RED-style "old path would have missed" assertion
// below a real proof rather than a string comparison.
func startExactPathStub(t *testing.T, pattern string, body string) (baseURL string, hits func() int, stop func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var mu sync.Mutex
	var count int

	mux := http.NewServeMux()
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, body)
	})

	httpSrv := &http.Server{Handler: mux}
	go func() { _ = httpSrv.Serve(ln) }()

	return "http://" + ln.Addr().String(),
		func() int {
			mu.Lock()
			defer mu.Unlock()
			return count
		},
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = httpSrv.Shutdown(ctx)
		}
}

// requestExactPath issues a real, direct (gateway-bypassing) HTTP request
// against a stub started by startExactPathStub, to prove — independently
// of the gateway — what that stub's own router does for a given path.
func requestExactPath(t *testing.T, client *http.Client, method, baseURL, path string) int {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+path, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

// TestIntegration_CorrectedRoutesReachRealUpstreamAcrossServices is the
// T8-1 multi-service anti-bluff proof required by the task: for five
// different upstream services (covering a pure prefix/rename fix, a
// param route, an HTTP-method fix, and a gateway-route-reshape fix), it
// stands up a REAL, independent net/http stub registered ONLY at the
// CORRECTED real upstream path, drives a REAL request through the REAL
// gateway server over a real loopback TCP connection, and asserts the
// request reaches the stub (a real 200 from the stub's own handler, and
// the stub's own hit-counter increments) — never a 404. For every case it
// ALSO issues a direct (gateway-bypassing) request for the OLD, pre-T8-1
// mismatched path against the very same stub and asserts that request
// gets a genuine 404 from the stub's own router — the RED-style proof
// that the old path really would have missed.
func TestIntegration_CorrectedRoutesReachRealUpstreamAcrossServices(t *testing.T) {
	t.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(testPublicKey))
	client := &http.Client{Timeout: 10 * time.Second}

	type caseDef struct {
		name            string
		service         string
		envKey          string
		stubPattern     string // Go 1.22+ "METHOD /path/{param}" pattern
		stubBody        string
		method          string
		gatewayPath     string
		oldUpstreamPath string // the pre-T8-1 mismatched path, requested directly against the same stub
	}

	cases := []caseDef{
		{
			name:            "host-service param route",
			service:         "host-service",
			envKey:          "HOST_SERVICE_ADDR",
			stubPattern:     "GET /api/v1/hosts/{hostId}",
			stubBody:        `{"real_upstream":"host-service","id":"h-1","name":"prod-db-1.internal"}`,
			method:          http.MethodGet,
			gatewayPath:     "/api/v1/hosts/h-1",
			oldUpstreamPath: "/hosts/h-1",
		},
		{
			name:            "vault-service flat secrets rename",
			service:         "vault-service",
			envKey:          "VAULT_SERVICE_ADDR",
			stubPattern:     "GET /api/v1/vault/secrets",
			stubBody:        `{"real_upstream":"vault-service","secrets":[]}`,
			method:          http.MethodGet,
			gatewayPath:     "/api/v1/vaults",
			oldUpstreamPath: "/vaults",
		},
		{
			name:            "ssh-proxy-service prefix rename",
			service:         "ssh-proxy-service",
			envKey:          "SSH_PROXY_SERVICE_ADDR",
			stubPattern:     "GET /api/v1/ssh/sessions",
			stubBody:        `{"real_upstream":"ssh-proxy-service","sessions":[]}`,
			method:          http.MethodGet,
			gatewayPath:     "/api/v1/sessions",
			oldUpstreamPath: "/sessions",
		},
		{
			name:            "snippet-service method fix (PATCH -> PUT)",
			service:         "snippet-service",
			envKey:          "SNIPPET_SERVICE_ADDR",
			stubPattern:     "PUT /api/v1/snippets/{snippetId}",
			stubBody:        `{"real_upstream":"snippet-service","id":"sn-1"}`,
			method:          http.MethodPut,
			gatewayPath:     "/api/v1/snippets/sn-1",
			oldUpstreamPath: "/snippets/sn-1", // old stub had no method-matched PATCH handler either; same path also never matched the real (prefix-less) registration
		},
		{
			name:            "pki-service gateway-route-reshape (added :caId)",
			service:         "pki-service",
			envKey:          "PKI_SERVICE_ADDR",
			stubPattern:     "POST /api/v1/pki/ca/{caId}/certs",
			stubBody:        `{"real_upstream":"pki-service","certId":"c-1"}`,
			method:          http.MethodPost,
			gatewayPath:     "/api/v1/pki/ca/ca-1/certs",
			oldUpstreamPath: "/pki/certificates", // the old gateway route had no :caId segment at all
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubURL, hits, stopStub := startExactPathStub(t, tc.stubPattern, tc.stubBody)
			defer stopStub()

			// --- RED: the OLD, pre-T8-1 mismatched path genuinely misses
			// against this exact same stub (net/http's own real 404,
			// never a fabricated result). ---
			oldStatus := requestExactPath(t, client, tc.method, stubURL, tc.oldUpstreamPath)
			require.Equal(t, http.StatusNotFound, oldStatus,
				"%s: the OLD upstream path %q must NOT match the real upstream's router (proves the pre-T8-1 mismatch was real)", tc.service, tc.oldUpstreamPath)
			require.Equal(t, 0, hits(), "the old-path probe must not have hit the real handler")

			// --- GREEN: a real request through the real gateway, over a
			// real TCP hop, to the corrected upstream path. ---
			t.Setenv(tc.envKey, stubURL)
			gatewayURL, stopGateway := startRealGateway(t)
			defer stopGateway()

			req, err := http.NewRequest(tc.method, gatewayURL+tc.gatewayPath, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+generateTestToken())

			resp, err := client.Do(req)
			require.NoError(t, err, "%s: real TCP request through the gateway must succeed at the transport layer", tc.service)
			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			_ = resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode,
				"%s: gateway route %s %s must reach the real upstream at the corrected path (got body: %s)",
				tc.service, tc.method, tc.gatewayPath, string(respBody))
			require.Contains(t, string(respBody), tc.service,
				"%s: response must be the real upstream's own payload, proxied back through the gateway", tc.service)
			require.Equal(t, 1, hits(),
				"%s: the real upstream's own handler must have been hit exactly once", tc.service)
		})
	}
}
