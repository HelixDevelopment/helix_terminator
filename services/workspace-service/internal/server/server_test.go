package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/workspace-service/internal/server"
)

func TestServerHealthEndpoints(t *testing.T) {
	srv, err := server.New(nil)
	require.NoError(t, err)

	router := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	router.ServeHTTP(w, req)
	// Without DB, readiness returns 503
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
