package checker_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/health-service/internal/checker"
	"github.com/helixdevelopment/health-service/internal/model"
)

func TestNew_DefaultTimeout(t *testing.T) {
	c := checker.New(map[string]string{}, 0)
	require.NotNil(t, c)
}

func TestCheckService_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	c := checker.New(map[string]string{}, 5*time.Second)
	sh, err := c.CheckService("test-service", server.URL)
	require.NoError(t, err)
	assert.Equal(t, "test-service", sh.Name)
	assert.Equal(t, model.StatusHealthy, sh.Status)
	assert.GreaterOrEqual(t, sh.ResponseTimeMs, int64(0))
	assert.Empty(t, sh.ErrorMessage)
}

func TestCheckService_Degraded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	c := checker.New(map[string]string{}, 5*time.Second)
	sh, err := c.CheckService("test-service", server.URL)
	require.NoError(t, err)
	assert.Equal(t, model.StatusDegraded, sh.Status)
}

func TestCheckService_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := checker.New(map[string]string{}, 5*time.Second)
	sh, err := c.CheckService("test-service", server.URL)
	require.NoError(t, err)
	assert.Equal(t, model.StatusUnhealthy, sh.Status)
}

func TestCheckService_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := checker.New(map[string]string{}, 50*time.Millisecond)
	sh, err := c.CheckService("slow-service", server.URL)
	require.NoError(t, err)
	assert.Equal(t, model.StatusUnhealthy, sh.Status)
	assert.Contains(t, sh.ErrorMessage, "request failed")
}

func TestCheckService_ConnectionRefused(t *testing.T) {
	c := checker.New(map[string]string{}, 1*time.Second)
	sh, err := c.CheckService("down-service", "http://localhost:59999/healthz")
	require.NoError(t, err)
	assert.Equal(t, model.StatusUnhealthy, sh.Status)
	assert.NotEmpty(t, sh.ErrorMessage)
}

func TestCheckAll(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()

	endpoints := map[string]string{
		"good-service": good.URL,
		"bad-service":  bad.URL,
	}

	c := checker.New(endpoints, 5*time.Second)
	sys, err := c.CheckAll()
	require.NoError(t, err)
	assert.Equal(t, model.StatusUnhealthy, sys.OverallStatus)
	assert.Len(t, sys.Services, 2)
}

func TestCheckAll_AllHealthy(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()

	endpoints := map[string]string{
		"svc1": good.URL,
		"svc2": good.URL,
	}

	c := checker.New(endpoints, 5*time.Second)
	sys, err := c.CheckAll()
	require.NoError(t, err)
	assert.Equal(t, model.StatusHealthy, sys.OverallStatus)
	assert.Len(t, sys.Services, 2)
}

func TestCheckServices(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()

	endpoints := map[string]string{
		"svc1": good.URL,
		"svc2": good.URL,
	}

	c := checker.New(endpoints, 5*time.Second)
	sys, err := c.CheckServices([]string{"svc1"})
	require.NoError(t, err)
	assert.Equal(t, model.StatusHealthy, sys.OverallStatus)
	assert.Len(t, sys.Services, 1)
	assert.Equal(t, "svc1", sys.Services[0].Name)
}

func TestCheckServices_UnknownService(t *testing.T) {
	c := checker.New(map[string]string{}, 5*time.Second)
	sys, err := c.CheckServices([]string{"unknown"})
	require.NoError(t, err)
	assert.Equal(t, model.StatusUnhealthy, sys.OverallStatus)
	assert.Len(t, sys.Services, 1)
	assert.Equal(t, "unknown", sys.Services[0].Name)
	assert.Contains(t, sys.Services[0].ErrorMessage, "not configured")
}
