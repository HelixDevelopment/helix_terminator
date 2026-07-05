package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/host-service/internal/server"
)

func TestMainHealthCheck(t *testing.T) {
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	srv.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestMainReadinessCheck(t *testing.T) {
	srv, err := server.New(nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	srv.Router().ServeHTTP(w, req)

	// Without DB, readiness should be 503
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
