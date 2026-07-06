package model

import (
	"time"

	"github.com/google/uuid"
)

// SessionStatus constants
const (
	SessionStatusActive = "active"
	SessionStatusEnded  = "ended"
)

// CollaborationSession represents a shared collaboration session
type CollaborationSession struct {
	ID          uuid.UUID `json:"id" db:"id"`
	HostID      uuid.UUID `json:"hostId" db:"host_id"`
	CreatedBy   uuid.UUID `json:"createdBy" db:"created_by"`
	OrgID       *uuid.UUID `json:"orgId,omitempty" db:"org_id"`
	Name        string    `json:"name" db:"name"`
	Status      string    `json:"status" db:"status"`
	Participants []uuid.UUID `json:"participants" db:"participants"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
	EndedAt     *time.Time `json:"endedAt,omitempty" db:"ended_at"`
}

// CreateCollaborationSessionRequest represents a request to create a session
type CreateCollaborationSessionRequest struct {
	HostID string `json:"hostId" binding:"required,uuid"`
	Name   string `json:"name" binding:"required,max=255"`
}

// JoinSessionRequest represents a request to join a session
type JoinSessionRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

// CollaborationSessionResponse is the API response
type CollaborationSessionResponse struct {
	ID           uuid.UUID   `json:"id"`
	HostID       uuid.UUID   `json:"hostId"`
	CreatedBy    uuid.UUID   `json:"createdBy"`
	OrgID        *uuid.UUID  `json:"orgId,omitempty"`
	Name         string      `json:"name"`
	Status       string      `json:"status"`
	Participants []uuid.UUID `json:"participants"`
	CreatedAt    time.Time   `json:"createdAt"`
}

// ListCollaborationSessionsResponse is the API response for listing
type ListCollaborationSessionsResponse struct {
	Items  []*CollaborationSessionResponse `json:"items"`
	Total  int                             `json:"total"`
	Limit  int                             `json:"limit"`
	Offset int                             `json:"offset"`
}
