package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of action performed
type AuditAction string

const (
	ActionCreate AuditAction = "create"
	ActionRead   AuditAction = "read"
	ActionUpdate AuditAction = "update"
	ActionDelete AuditAction = "delete"
	ActionLogin  AuditAction = "login"
	ActionLogout AuditAction = "logout"
	ActionExport AuditAction = "export"
)

// AuditResourceType represents the type of resource acted upon
type AuditResourceType string

const (
	ResourceTypeUser      AuditResourceType = "user"
	ResourceTypeHost      AuditResourceType = "host"
	ResourceTypeOrg       AuditResourceType = "org"
	ResourceTypeVault     AuditResourceType = "vault"
	ResourceTypeWorkspace AuditResourceType = "workspace"
)

// AuditSeverity represents the severity level of the audit log
type AuditSeverity string

const (
	SeverityInfo     AuditSeverity = "info"
	SeverityWarning  AuditSeverity = "warning"
	SeverityError    AuditSeverity = "error"
	SeverityCritical AuditSeverity = "critical"
)

// AuditLog represents a single audit log entry
type AuditLog struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	OrgID        *uuid.UUID      `json:"orgId,omitempty" db:"org_id"`
	UserID       *uuid.UUID      `json:"userId,omitempty" db:"user_id"`
	Action       AuditAction     `json:"action" db:"action"`
	ResourceType AuditResourceType `json:"resourceType" db:"resource_type"`
	ResourceID   *uuid.UUID      `json:"resourceId,omitempty" db:"resource_id"`
	Details      []byte          `json:"details,omitempty" db:"details"`
	IPAddress    string          `json:"ipAddress" db:"ip_address"`
	UserAgent    string          `json:"userAgent,omitempty" db:"user_agent"`
	Timestamp    time.Time       `json:"timestamp" db:"timestamp"`
	Severity     AuditSeverity   `json:"severity" db:"severity"`
}

// CreateAuditLogRequest represents a request to create an audit log
type CreateAuditLogRequest struct {
	OrgID        *uuid.UUID          `json:"orgId,omitempty"`
	UserID       *uuid.UUID          `json:"userId,omitempty"`
	Action       AuditAction         `json:"action" binding:"required,oneof=create read update delete login logout export"`
	ResourceType AuditResourceType   `json:"resourceType" binding:"required,oneof=user host org vault workspace"`
	ResourceID   *uuid.UUID          `json:"resourceId,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	IPAddress    string              `json:"ipAddress,omitempty"`
	UserAgent    string              `json:"userAgent,omitempty"`
	Severity     AuditSeverity       `json:"severity" binding:"required,oneof=info warning error critical"`
}

// ListAuditLogsRequest represents query parameters for listing audit logs
type ListAuditLogsRequest struct {
	OrgID        string        `form:"org_id"`
	UserID       string        `form:"user_id"`
	Action       AuditAction   `form:"action"`
	ResourceType AuditResourceType `form:"resource_type"`
	Severity     AuditSeverity `form:"severity"`
	Start        string        `form:"start"`
	End          string        `form:"end"`
	Limit        int           `form:"limit,default=20"`
	Offset       int           `form:"offset,default=0"`
}

// AuditLogResponse represents a single audit log in responses
type AuditLogResponse struct {
	ID           uuid.UUID       `json:"id"`
	OrgID        *uuid.UUID      `json:"orgId,omitempty"`
	UserID       *uuid.UUID      `json:"userId,omitempty"`
	Action       AuditAction     `json:"action"`
	ResourceType AuditResourceType `json:"resourceType"`
	ResourceID   *uuid.UUID      `json:"resourceId,omitempty"`
	Details      interface{}     `json:"details,omitempty"`
	IPAddress    string          `json:"ipAddress"`
	UserAgent    string          `json:"userAgent,omitempty"`
	Timestamp    time.Time       `json:"timestamp"`
	Severity     AuditSeverity   `json:"severity"`
}

// ListAuditLogsResponse represents the response for listing audit logs
type ListAuditLogsResponse struct {
	Logs   []*AuditLogResponse `json:"logs"`
	Total  int                 `json:"total"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

// CountResponse represents a count aggregation response
type CountResponse struct {
	Counts map[string]int `json:"counts"`
}
