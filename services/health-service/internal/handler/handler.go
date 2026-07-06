package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/helixdevelopment/health-service/internal/checker"
	"github.com/helixdevelopment/health-service/internal/model"
)

// Handler holds health service handlers.
type Handler struct {
	checker *checker.HealthChecker
}

// New returns a new Handler with the given checker.
func New(chk *checker.HealthChecker) *Handler {
	return &Handler{checker: chk}
}

// HealthCheck returns the health-service's own health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "health-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status.
func (h *Handler) ReadinessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"service":   "health-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetSystemHealth checks all configured services and returns aggregated status.
func (h *Handler) GetSystemHealth(c *gin.Context) {
	if h.checker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "health checker not configured"})
		return
	}

	systemHealth, err := h.checker.CheckAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	statusCode := http.StatusOK
	if systemHealth.OverallStatus == model.StatusDegraded {
		statusCode = http.StatusOK
	} else if systemHealth.OverallStatus == model.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, systemHealth)
}

// GetServiceHealth checks a single service by name.
func (h *Handler) GetServiceHealth(c *gin.Context) {
	if h.checker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "health checker not configured"})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service name is required"})
		return
	}

	// We need to access the endpoints map; since HealthChecker doesn't expose it,
	// we use CheckServices with a single name which handles unknown services gracefully.
	systemHealth, err := h.checker.CheckServices([]string{name})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(systemHealth.Services) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}

	svc := systemHealth.Services[0]
	statusCode := http.StatusOK
	if svc.Status == model.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, svc)
}

// RunHealthCheck accepts a list of service names and checks them.
func (h *Handler) RunHealthCheck(c *gin.Context) {
	if h.checker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "health checker not configured"})
		return
	}

	var req model.HealthCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Services) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one service name is required"})
		return
	}

	systemHealth, err := h.checker.CheckServices(req.Services)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := model.HealthCheckResponse{
		Status:    systemHealth.OverallStatus,
		Services:  systemHealth.Services,
		CheckedAt: systemHealth.CheckedAt,
	}

	statusCode := http.StatusOK
	if resp.Status == model.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, resp)
}
