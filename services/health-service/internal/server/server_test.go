package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/health-service/internal/server"
)

func TestNew(t *testing.T) {
	endpoints := map[string]string{
		"svc1": "http://localhost:9999",
	}
	srv := server.New(nil, endpoints, 5*time.Second)
	require.NotNil(t, srv)
	require.NotNil(t, srv.Router())
}

func TestRoutes_Healthz(t *testing.T) {
	srv := server.New(nil, nil, 5*time.Second)
	router := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestRoutes_Readiness(t *testing.T) {
	srv := server.New(nil, nil, 5*time.Second)
	router := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}

func TestRoutes_SystemHealth(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()

	endpoints := map[string]string{
		"svc1": good.URL,
	}

	srv := server.New(nil, endpoints, 5*time.Second)
	router := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health/system", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "overall_status")
}

func TestRoutes_ServiceHealth(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()

	endpoints := map[string]string{
		"svc1": good.URL,
	}

	srv := server.New(nil, endpoints, 5*time.Second)
	router := srv.Router()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health/services/svc1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "svc1")
}
