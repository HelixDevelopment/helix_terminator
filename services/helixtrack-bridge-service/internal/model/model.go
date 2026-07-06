package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// HelixTrackBridgeStatus constants
const (
	HelixTrackBridgeStatusActive   = "active"
	HelixTrackBridgeStatusInactive = "inactive"
	HelixTrackBridgeStatusError    = "error"
)

// HelixTrackBridge represents a HelixTrack integration bridge
type HelixTrackBridge struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	IntegrationID string     `json:"integrationId" db:"integration_id"`
	OrgID         uuid.UUID  `json:"orgId" db:"org_id"`
	Name          string     `json:"name" db:"name"`
	Status        string     `json:"status" db:"status"`
	Config        []byte     `json:"config" db:"config"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
}

// CreateHelixTrackBridgeRequest represents a request to create a bridge
type CreateHelixTrackBridgeRequest struct {
	IntegrationID string          `json:"integrationId" binding:"required,max=255"`
	OrgID         string          `json:"orgId" binding:"required,uuid"`
	Name          string          `json:"name" binding:"required,max=255"`
	Config        json.RawMessage `json:"config"`
}

// UpdateHelixTrackBridgeRequest represents a request to update a bridge
type UpdateHelixTrackBridgeRequest struct {
	Name   string          `json:"name" binding:"max=255"`
	Status string          `json:"status" binding:"oneof=active inactive error"`
	Config json.RawMessage `json:"config"`
}

// HelixTrackBridgeResponse is the API response
type HelixTrackBridgeResponse struct {
	ID            uuid.UUID       `json:"id"`
	IntegrationID string          `json:"integrationId"`
	OrgID         uuid.UUID       `json:"orgId"`
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	Config        json.RawMessage `json:"config"`
	CreatedAt     time.Time       `json:"createdAt"`
}

// ListHelixTrackBridgesResponse is the API response for listing
type ListHelixTrackBridgesResponse struct {
	Items  []*HelixTrackBridgeResponse `json:"items"`
	Total  int                         `json:"total"`
	Limit  int                         `json:"limit"`
	Offset int                         `json:"offset"`
}
