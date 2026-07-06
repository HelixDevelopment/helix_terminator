package model

import (
	"time"
)

// HealthStatus represents the health status of a service.
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

// ServiceHealth represents the health status of a single service.
type ServiceHealth struct {
	Name           string     `json:"name"`
	Status         HealthStatus `json:"status"`
	LastCheckAt    time.Time  `json:"last_check_at"`
	ResponseTimeMs int64      `json:"response_time_ms"`
	ErrorMessage   string     `json:"error_message,omitempty"`
}

// SystemHealth represents the aggregated health status of all services.
type SystemHealth struct {
	OverallStatus HealthStatus    `json:"overall_status"`
	Services      []ServiceHealth `json:"services"`
	CheckedAt     time.Time       `json:"checked_at"`
}

// HealthCheckRequest represents a request to check specific services.
type HealthCheckRequest struct {
	Services []string `json:"services" binding:"required"`
}

// HealthCheckResponse represents the response from a health check operation.
type HealthCheckResponse struct {
	Status    HealthStatus    `json:"status"`
	Services  []ServiceHealth `json:"services"`
	CheckedAt time.Time       `json:"checked_at"`
}
