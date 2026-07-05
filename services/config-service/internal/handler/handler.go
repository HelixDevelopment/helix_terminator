package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler holds service handlers.
type Handler struct{}

// New returns a new Handler.
func New() *Handler {
	return &Handler{}
}

// HealthCheck returns service health status.
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// ReadinessCheck returns readiness status.
func (h *Handler) ReadinessCheck(c *gin.Context) {
	// TODO: check DB, cache, upstream dependencies
	c.JSON(http.StatusOK, gin.H{"ready": true})
}

// TODO: add service-specific handlers
