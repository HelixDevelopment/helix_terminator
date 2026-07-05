package model

import (
	"time"

	"github.com/google/uuid"
)

// ContainerBridge represents a container bridge connection
type ContainerBridge struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"userId" db:"user_id"`
	OrgID       *uuid.UUID `json:"orgId,omitempty" db:"org_id"`
	Name        string    `json:"name" db:"name"`
	HostID      uuid.UUID `json:"hostId" db:"host_id"`
	ContainerID string    `json:"containerId" db:"container_id"`
	Image       string    `json:"image" db:"image"`
	Status      string    `json:"status" db:"status"`
	Ports       map[string]interface{} `json:"ports,omitempty" db:"ports"`
	Env         map[string]interface{} `json:"env,omitempty" db:"env"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// CreateContainerBridgeRequest represents a request to create a bridge
type CreateContainerBridgeRequest struct {
	Name        string                 `json:"name" binding:"required,max=255"`
	HostID      string                 `json:"hostId" binding:"required,uuid"`
	ContainerID string                 `json:"containerId" binding:"required"`
	Image       string                 `json:"image" binding:"required"`
	Ports       map[string]interface{} `json:"ports,omitempty"`
	Env         map[string]interface{} `json:"env,omitempty"`
}

// ContainerBridgeResponse is the API response
type ContainerBridgeResponse struct {
	ID          uuid.UUID              `json:"id"`
	UserID      uuid.UUID              `json:"userId"`
	OrgID       *uuid.UUID             `json:"orgId,omitempty"`
	Name        string                 `json:"name"`
	HostID      uuid.UUID              `json:"hostId"`
	ContainerID string                 `json:"containerId"`
	Image       string                 `json:"image"`
	Status      string                 `json:"status"`
	Ports       map[string]interface{} `json:"ports,omitempty"`
	Env         map[string]interface{} `json:"env,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// ListContainerBridgesResponse is the API response for listing
type ListContainerBridgesResponse struct {
	Items  []*ContainerBridgeResponse `json:"items"`
	Total  int                       `json:"total"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}
