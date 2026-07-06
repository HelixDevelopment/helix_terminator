package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/server"
)

func TestServerNew(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)
	require.NotNil(t, srv)

	r := srv.Router()
	require.NotNil(t, r)
}

func TestHealthEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	r := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/healthz/live", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not connected")
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	r := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv, err := server.New(nil)
	require.NoError(t, err)

	r := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-request-id", w.Header().Get("X-Request-ID"))
}
