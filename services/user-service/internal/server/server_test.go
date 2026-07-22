package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/helixdevelopment/user-service/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_WiresRouterAndRoutes proves server.New(nil) genuinely wires a
// live gin.Engine with the real registered routes - not just a non-nil
// struct check. It drives an actual HTTP request through Router() (real
// middleware chain: recovery + request-ID + logging) and asserts on the
// real response, the same way an operator would probe the process.
func TestNew_WiresRouterAndRoutes(t *testing.T) {
	srv := server.New(nil)
	require.NotNil(t, srv)
	require.NotNil(t, srv.Router())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "real /healthz route must be registered and respond 200, got body=%s", w.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp["status"])
}

// TestNew_RequestIDMiddlewareAppliesToEveryResponse proves the real
// requestIDMiddleware wired inside server.New actually runs on the live
// router - both generating an X-Request-ID when the caller supplies none
// and echoing back a caller-supplied one, exactly as the middleware
// promises. A regression that drops the middleware from the chain (or
// wires it to the wrong group) fails this test.
func TestNew_RequestIDMiddlewareAppliesToEveryResponse(t *testing.T) {
	srv := server.New(nil)
	require.NotNil(t, srv)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.Router().ServeHTTP(w, req)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"), "middleware must generate an X-Request-ID when the caller supplies none")

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req2.Header.Set("X-Request-ID", "caller-supplied-id-42")
	srv.Router().ServeHTTP(w2, req2)
	assert.Equal(t, "caller-supplied-id-42", w2.Header().Get("X-Request-ID"), "middleware must echo back a caller-supplied X-Request-ID rather than overwrite it")
}

// TestNew_APIRoutesAreRegisteredUnderV1 proves the /api/v1 route group
// genuinely exists and dispatches to the user CRUD handlers - a request
// for a registered method+path must NOT fall through to gin's default
// 404, even though it will fail downstream (repo is nil here) once
// inside the handler. This distinguishes "route not wired" (404) from
// "route wired, handler errored on nil repo" (any non-404 status),
// proving the router in New really does register every declared route.
func TestNew_APIRoutesAreRegisteredUnderV1(t *testing.T) {
	srv := server.New(nil)
	require.NotNil(t, srv)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/users"},
		{http.MethodGet, "/api/v1/users/some-id"},
		{http.MethodGet, "/api/v1/users/by-email"},
		{http.MethodGet, "/api/v1/users/some-id/profile"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			srv.Router().ServeHTTP(w, req)
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"expected %s %s to be routed to a real handler (any non-404 status), got 404 - route missing from server.New wiring", tc.method, tc.path)
		})
	}
}

// TestNew_UnknownRouteIs404 is the negative control for
// TestNew_APIRoutesAreRegisteredUnderV1 - proves the router doesn't
// simply accept every path (which would make the positive assertions
// above vacuous).
func TestNew_UnknownRouteIs404(t *testing.T) {
	srv := server.New(nil)
	require.NotNil(t, srv)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/definitely-not-a-real-route", nil)
	srv.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
