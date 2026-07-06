package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/health-service/internal/model"
)

func TestHealthStatusConstants(t *testing.T) {
	assert.Equal(t, model.HealthStatus("healthy"), model.StatusHealthy)
	assert.Equal(t, model.HealthStatus("degraded"), model.StatusDegraded)
	assert.Equal(t, model.HealthStatus("unhealthy"), model.StatusUnhealthy)
}

func TestServiceHealth(t *testing.T) {
	now := time.Now().UTC()
	sh := model.ServiceHealth{
		Name:           "auth-service",
		Status:         model.StatusHealthy,
		LastCheckAt:    now,
		ResponseTimeMs: 42,
		ErrorMessage:   "",
	}

	assert.Equal(t, "auth-service", sh.Name)
	assert.Equal(t, model.StatusHealthy, sh.Status)
	assert.Equal(t, int64(42), sh.ResponseTimeMs)
	assert.Empty(t, sh.ErrorMessage)
}

func TestSystemHealth(t *testing.T) {
	now := time.Now().UTC()
	sh := model.ServiceHealth{
		Name:           "auth-service",
		Status:         model.StatusHealthy,
		LastCheckAt:    now,
		ResponseTimeMs: 10,
	}

	sys := model.SystemHealth{
		OverallStatus: model.StatusHealthy,
		Services:      []model.ServiceHealth{sh},
		CheckedAt:     now,
	}

	assert.Equal(t, model.StatusHealthy, sys.OverallStatus)
	assert.Len(t, sys.Services, 1)
	assert.Equal(t, "auth-service", sys.Services[0].Name)
}

func TestHealthCheckRequest(t *testing.T) {
	req := model.HealthCheckRequest{
		Services: []string{"auth-service", "gateway-service"},
	}
	assert.Len(t, req.Services, 2)
	assert.Contains(t, req.Services, "auth-service")
}

func TestHealthCheckResponse(t *testing.T) {
	now := time.Now().UTC()
	resp := model.HealthCheckResponse{
		Status:    model.StatusDegraded,
		Services:  []model.ServiceHealth{},
		CheckedAt: now,
	}
	assert.Equal(t, model.StatusDegraded, resp.Status)
	assert.Empty(t, resp.Services)
}
