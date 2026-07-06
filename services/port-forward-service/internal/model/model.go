package model

import (
	"time"

	"github.com/google/uuid"
)

// PortForwardStatus constants
const (
	PortForwardStatusActive   = "active"
	PortForwardStatusInactive = "inactive"
	PortForwardStatusDeleted  = "deleted"
)

// PortForward represents an SSH port forwarding rule
type PortForward struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	HostID      uuid.UUID  `json:"hostId" db:"host_id"`
	LocalPort   int        `json:"localPort" db:"local_port"`
	RemotePort  int        `json:"remotePort" db:"remote_port"`
	RemoteHost  string     `json:"remoteHost" db:"remote_host"`
	Protocol    string     `json:"protocol" db:"protocol"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// CreatePortForwardRequest represents a request to create a forward
type CreatePortForwardRequest struct {
	HostID     string `json:"hostId" binding:"required,uuid"`
	LocalPort  int    `json:"localPort" binding:"required,min=1,max=65535"`
	RemotePort int    `json:"remotePort" binding:"required,min=1,max=65535"`
	RemoteHost string `json:"remoteHost" binding:"required,max=255"`
	Protocol   string `json:"protocol" binding:"required,oneof=tcp udp"`
}

// UpdatePortForwardRequest represents a request to update a forward
type UpdatePortForwardRequest struct {
	LocalPort  int    `json:"localPort" binding:"min=1,max=65535"`
	RemotePort int    `json:"remotePort" binding:"min=1,max=65535"`
	RemoteHost string `json:"remoteHost" binding:"max=255"`
	Protocol   string `json:"protocol" binding:"oneof=tcp udp"`
	Status     string `json:"status" binding:"oneof=active inactive"`
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
