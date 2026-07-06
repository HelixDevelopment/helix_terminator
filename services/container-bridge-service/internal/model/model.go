package model

import (
	"time"

	"github.com/google/uuid"
)

// ContainerBridgeStatus constants
const (
	ContainerBridgeStatusActive   = "active"
	ContainerBridgeStatusInactive = "inactive"
	ContainerBridgeStatusError    = "error"
)

// ContainerBridge represents a container bridge connection
type ContainerBridge struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	HostID      uuid.UUID  `json:"hostId" db:"host_id"`
	ContainerID string     `json:"containerId" db:"container_id"`
	Name        string     `json:"name" db:"name"`
	Image       string     `json:"image" db:"image"`
	Status      string     `json:"status" db:"status"`
	Ports       []string   `json:"ports" db:"ports"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
}

// CreateContainerBridgeRequest represents a request to create a bridge
type CreateContainerBridgeRequest struct {
	HostID      string   `json:"hostId" binding:"required,uuid"`
	ContainerID string   `json:"containerId" binding:"required,max=255"`
	Name        string   `json:"name" binding:"required,max=255"`
	Image       string   `json:"image" binding:"required,max=255"`
	Ports       []string `json:"ports"`
}

// UpdateContainerBridgeRequest represents a request to update a bridge
type UpdateContainerBridgeRequest struct {
	Name   string   `json:"name" binding:"max=255"`
	Image  string   `json:"image" binding:"max=255"`
	Status string   `json:"status" binding:"oneof=active inactive error"`
	Ports  []string `json:"ports"`
}

// ContainerBridgeResponse is the API response
type ContainerBridgeResponse struct {
	ID          uuid.UUID `json:"id"`
	HostID      uuid.UUID `json:"hostId"`
	ContainerID string    `json:"containerId"`
	Name        string    `json:"name"`
	Image       string    `json:"image"`
	Status      string    `json:"status"`
	Ports       []string  `json:"ports"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ListContainerBridgesResponse is the API response for listing
type ListContainerBridgesResponse struct {
	Items  []*ContainerBridgeResponse `json:"items"`
	Total  int                        `json:"total"`
	Limit  int                        `json:"limit"`
	Offset int                        `json:"offset"`
}
