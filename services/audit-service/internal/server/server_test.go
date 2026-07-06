package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/audit-service/internal/server"
)

func TestServerCreation(t *testing.T) {
	srv, err := server.New(nil)
	assert.NoError(t, err)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.Router())
}

func TestServerHealthEndpoints(t *testing.T) {
	srv, err := server.New(nil)
	assert.NoError(t, err)

	router := srv.Router()

	// Test /healthz
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test /healthz/live
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/healthz/live", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test /healthz/ready (no DB, should return 503)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/healthz/ready", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestServerAuditRoutesExist(t *testing.T) {
	srv, err := server.New(nil)
	assert.NoError(t, err)

	router := srv.Router()

	// POST /api/v1/audit/logs (will fail with 500 because no DB, but route exists)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/audit/logs", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code) // bad request because no body

	// GET /api/v1/audit/logs (will fail with 500 because no DB, but route exists)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/audit/logs", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// GET /api/v1/audit/logs/:id (will fail with 500 because no DB, but route exists)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/audit/logs/invalid", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code) // invalid UUID

	// GET /api/v1/audit/stats/actions (will fail with 500 because no DB, but route exists)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/audit/stats/actions", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// GET /api/v1/audit/stats/resources (will fail with 500 because no DB, but route exists)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/audit/stats/resources", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServerCORSMiddleware(t *testing.T) {
	srv, err := server.New(nil)
	assert.NoError(t, err)

	router := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/healthz", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}
