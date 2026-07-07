package model

import (
	"time"

	"github.com/google/uuid"
)

// PortForwardStatus constants. "active" now means a REAL tunnel is up
// (listener bound + SSH connection established) — never set unconditionally
// at creation time. "pending" is the honest state for a catalog entry that
// has not been started yet; "error" reflects a real Start failure.
const (
	PortForwardStatusPending  = "pending"
	PortForwardStatusActive   = "active"
	PortForwardStatusInactive = "inactive"
	PortForwardStatusStopped  = "stopped"
	PortForwardStatusError    = "error"
	PortForwardStatusDeleted  = "deleted"
)

// Forward type constants — mirrors forwarder.ForwardType* without importing
// the forwarder package from model (keeps model dependency-free).
const (
	ForwardTypeLocal   = "local"
	ForwardTypeRemote  = "remote"
	ForwardTypeDynamic = "dynamic"
)

// Auth type constants for the SSH connection used to establish the tunnel.
const (
	AuthTypePassword = "password"
	AuthTypeKey      = "key"
	AuthTypeAgent    = "agent"
)

// PortForward represents an SSH port forwarding rule. SSHHost/SSHPort/
// SSHUsername/AuthType/ForwardType/BindAddress are persisted catalog
// metadata; the SSH secret material (password/private key) is NEVER
// persisted (Constitution §11.4.10) — it is supplied fresh on every
// StartForward call and used only in-memory to dial.
type PortForward struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	HostID      uuid.UUID  `json:"hostId" db:"host_id"`
	ForwardType string     `json:"forwardType" db:"forward_type"`
	LocalPort   int        `json:"localPort" db:"local_port"`
	RemotePort  int        `json:"remotePort" db:"remote_port"`
	RemoteHost  string     `json:"remoteHost" db:"remote_host"`
	Protocol    string     `json:"protocol" db:"protocol"`
	BindAddress string     `json:"bindAddress" db:"bind_address"`
	SSHHost     string     `json:"sshHost" db:"ssh_host"`
	SSHPort     int        `json:"sshPort" db:"ssh_port"`
	SSHUsername string     `json:"sshUsername" db:"ssh_username"`
	AuthType    string     `json:"authType" db:"auth_type"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// CreatePortForwardRequest represents a request to create a forward.
// LocalPort may be 0 to let the OS choose an ephemeral port at Start time.
// RemoteHost/RemotePort are required for local/remote forward types (the
// -L/-R target) and are ignored for dynamic (SOCKS5 has no fixed target).
type CreatePortForwardRequest struct {
	HostID      string `json:"hostId" binding:"required,uuid"`
	ForwardType string `json:"forwardType" binding:"omitempty,oneof=local remote dynamic"`
	LocalPort   int    `json:"localPort" binding:"min=0,max=65535"`
	RemotePort  int    `json:"remotePort" binding:"omitempty,min=1,max=65535"`
	RemoteHost  string `json:"remoteHost" binding:"omitempty,max=255"`
	Protocol    string `json:"protocol" binding:"required,oneof=tcp udp"`
	BindAddress string `json:"bindAddress" binding:"omitempty,max=255"`
	SSHHost     string `json:"sshHost" binding:"required,max=255"`
	SSHPort     int    `json:"sshPort" binding:"omitempty,min=1,max=65535"`
	SSHUsername string `json:"sshUsername" binding:"required,max=255"`
	AuthType    string `json:"authType" binding:"omitempty,oneof=password key agent"`
}

// UpdatePortForwardRequest represents a request to update forward metadata.
// This does NOT touch the real tunnel lifecycle — use StartForwardRequest /
// StopForward for that.
type UpdatePortForwardRequest struct {
	LocalPort  int    `json:"localPort" binding:"min=1,max=65535"`
	RemotePort int    `json:"remotePort" binding:"min=1,max=65535"`
	RemoteHost string `json:"remoteHost" binding:"max=255"`
	Protocol   string `json:"protocol" binding:"oneof=tcp udp"`
	Status     string `json:"status" binding:"oneof=active inactive"`
}

// StartForwardRequest carries the transient SSH secret material needed to
// establish the tunnel. It is NEVER persisted (Constitution §11.4.10) — used
// once, in-memory, to build the ssh.AuthMethod, then discarded. Not required
// when the forward's persisted AuthType is "agent" (SSH_AUTH_SOCK is read
// server-side instead).
type StartForwardRequest struct {
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

// StartForwardResponse reports the REAL, resolved state of the tunnel after
// a successful Start — never a fabricated echo of the request.
type StartForwardResponse struct {
	ID           uuid.UUID `json:"id"`
	Status       string    `json:"status"`
	BoundAddress string    `json:"boundAddress,omitempty"`
}

// PortForwardResponse is the API response
type PortForwardResponse struct {
	ID         uuid.UUID `json:"id"`
	HostID     uuid.UUID `json:"hostId"`
	LocalPort  int       `json:"localPort"`
	RemotePort int       `json:"remotePort"`
	RemoteHost string    `json:"remoteHost"`
	Protocol   string    `json:"protocol"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ListPortForwardsResponse is the API response for listing
type ListPortForwardsResponse struct {
	Items  []*PortForwardResponse `json:"items"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}
