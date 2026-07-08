package model

import (
	"time"

	"github.com/google/uuid"
)

// ConnectionStatus represents the state of an SSH session.
type ConnectionStatus string

const (
	StatusConnecting   ConnectionStatus = "connecting"
	StatusConnected    ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusError        ConnectionStatus = "error"
)

// SSHSession represents an active or historical SSH terminal session.
type SSHSession struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	UserID           uuid.UUID        `json:"user_id" db:"user_id"`
	HostID           uuid.UUID        `json:"host_id" db:"host_id"`
	HostAddress      string           `json:"host_address" db:"host_address"`
	Username         string           `json:"username" db:"username"`
	AuthType         string           `json:"auth_type" db:"auth_type"`
	ConnectionStatus ConnectionStatus `json:"connection_status" db:"connection_status"`
	ConnectedAt      *time.Time       `json:"connected_at,omitempty" db:"connected_at"`
	DisconnectedAt   *time.Time       `json:"disconnected_at,omitempty" db:"disconnected_at"`
	LastActivityAt   *time.Time       `json:"last_activity_at,omitempty" db:"last_activity_at"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
}

// SSHChannel represents an SSH channel opened within a session.
type SSHChannel struct {
	ID          uuid.UUID `json:"id" db:"id"`
	SessionID   uuid.UUID `json:"session_id" db:"session_id"`
	ChannelType string    `json:"channel_type" db:"channel_type"`
	LocalPort   *int      `json:"local_port,omitempty" db:"local_port"`
	RemotePort  *int      `json:"remote_port,omitempty" db:"remote_port"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// CreateSSHSessionRequest is the payload to initiate a new SSH session.
type CreateSSHSessionRequest struct {
	HostID      string `json:"host_id" binding:"required,uuid"`
	HostAddress string `json:"host_address" binding:"required"`
	Username    string `json:"username" binding:"required"`
	AuthType    string `json:"auth_type" binding:"required,oneof=password key agent"`
	Password    string `json:"password,omitempty"`
	PrivateKey  string `json:"private_key,omitempty"`
}

// SSHSessionResponse is the API representation of an SSH session.
type SSHSessionResponse struct {
	ID               uuid.UUID        `json:"id"`
	UserID           uuid.UUID        `json:"user_id"`
	HostID           uuid.UUID        `json:"host_id"`
	HostAddress      string           `json:"host_address"`
	Username         string           `json:"username"`
	AuthType         string           `json:"auth_type"`
	ConnectionStatus ConnectionStatus `json:"connection_status"`
	ConnectedAt      *time.Time       `json:"connected_at,omitempty"`
	DisconnectedAt   *time.Time       `json:"disconnected_at,omitempty"`
	LastActivityAt   *time.Time       `json:"last_activity_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

// ListSSHSessionsRequest carries pagination parameters.
type ListSSHSessionsRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	Limit  int    `json:"limit" binding:"min=1,max=100"`
	Offset int    `json:"offset" binding:"min=0"`
}

// TerminalResizeMessage is sent over WebSocket to resize the PTY.
type TerminalResizeMessage struct {
	Type   string `json:"type"`
	Cols   uint32 `json:"cols"`
	Rows   uint32 `json:"rows"`
	Width  uint32 `json:"width,omitempty"`
	Height uint32 `json:"height,omitempty"`
}

// WebSocketMessage is a generic envelope for WebSocket control messages.
type WebSocketMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
}
