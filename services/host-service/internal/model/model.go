package model

import (
	"time"

	"github.com/google/uuid"
)

// AuthType represents the SSH authentication method.
type AuthType string

const (
	AuthTypePassword AuthType = "password"
	AuthTypeKey      AuthType = "key"
	AuthTypeAgent    AuthType = "agent"
	AuthTypeVaultKey AuthType = "vault_key"
)

// ConnectionStatus represents the host connection status.
type ConnectionStatus string

const (
	StatusUnknown  ConnectionStatus = "unknown"
	StatusOnline   ConnectionStatus = "online"
	StatusOffline  ConnectionStatus = "offline"
	StatusError    ConnectionStatus = "error"
)

// Host represents an SSH host managed by the platform.
type Host struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	UserID           uuid.UUID        `json:"user_id" db:"user_id"`
	OrgID            uuid.UUID        `json:"org_id" db:"org_id"`
	Name             string           `json:"name" db:"name"`
	Hostname         string           `json:"hostname" db:"hostname"`
	Port             int              `json:"port" db:"port"`
	Username         string           `json:"username" db:"username"`
	AuthType         AuthType         `json:"auth_type" db:"auth_type"`
	VaultSecretID    *uuid.UUID       `json:"vault_secret_id,omitempty" db:"vault_secret_id"`
	ConnectionParams map[string]interface{} `json:"connection_params,omitempty" db:"connection_params"`
	Tags             []string         `json:"tags,omitempty" db:"tags"`
	LastConnectedAt  *time.Time       `json:"last_connected_at,omitempty" db:"last_connected_at"`
	ConnectionStatus ConnectionStatus `json:"connection_status" db:"connection_status"`
	LastError        string           `json:"last_error,omitempty" db:"last_error"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at" db:"updated_at"`
	DeletedAt        *time.Time       `json:"-" db:"deleted_at"`
}

// HostConnectionLog represents an event log for host connections.
type HostConnectionLog struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	HostID    uuid.UUID              `json:"host_id" db:"host_id"`
	EventType string                 `json:"event_type" db:"event_type"`
	Details   map[string]interface{} `json:"details,omitempty" db:"details"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

// CreateHostRequest represents a request to create a new host.
type CreateHostRequest struct {
	Name             string                 `json:"name" binding:"required,max=255"`
	Hostname         string                 `json:"hostname" binding:"required,max=255"`
	Port             int                    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username         string                 `json:"username" binding:"required,max=255"`
	AuthType         AuthType               `json:"auth_type" binding:"required,oneof=password key agent vault_key"`
	VaultSecretID    *uuid.UUID             `json:"vault_secret_id,omitempty"`
	ConnectionParams map[string]interface{} `json:"connection_params,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
}

// UpdateHostRequest represents a request to update an existing host.
type UpdateHostRequest struct {
	Name             string                 `json:"name,omitempty" binding:"omitempty,max=255"`
	Hostname         string                 `json:"hostname,omitempty" binding:"omitempty,max=255"`
	Port             int                    `json:"port,omitempty" binding:"omitempty,min=1,max=65535"`
	Username         string                 `json:"username,omitempty" binding:"omitempty,max=255"`
	AuthType         AuthType               `json:"auth_type,omitempty" binding:"omitempty,oneof=password key agent vault_key"`
	VaultSecretID    *uuid.UUID             `json:"vault_secret_id,omitempty"`
	ConnectionParams map[string]interface{} `json:"connection_params,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
}

// ListHostsRequest represents query parameters for listing hosts.
type ListHostsRequest struct {
	Tags   []string `json:"tags,omitempty" form:"tags"`
	Status string   `json:"status,omitempty" form:"status"`
	Limit  int      `json:"limit,omitempty" form:"limit" binding:"min=1,max=100"`
	Offset int      `json:"offset,omitempty" form:"offset" binding:"min=0"`
}

// HostResponse wraps a Host for API responses.
type HostResponse struct {
	Host
}

// TestConnectionRequest represents a request to test a host connection.
type TestConnectionRequest struct {
	// Optional override fields for testing
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// TestConnectionResponse represents the result of a connection test.
type TestConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Latency int64  `json:"latency_ms,omitempty"`
}
