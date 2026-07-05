package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/config-service/internal/server"
)

func TestServerHealthEndpoints(t *testing.T) {
	srv, err := server.New(nil)
	require.NoError(t, err)

	router := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/healthz/ready", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestServerCORSMiddleware(t *testing.T) {
	srv, err := server.New(nil)
	require.NoError(t, err)

	router := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/v1/configs", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestServerRequestIDMiddleware(t *testing.T) {
	srv, err := server.New(nil)
	require.NoError(t, err)

	router := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	req.Header.Set("X-Request-ID", "test-req-id")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-req-id", w.Header().Get("X-Request-ID"))
}
