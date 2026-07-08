package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AnalyticsEventType constants
const (
	AnalyticsEventTypeSession  = "session"
	AnalyticsEventTypeCommand  = "command"
	AnalyticsEventTypeTransfer = "transfer"
	AnalyticsEventTypeLogin    = "login"
	AnalyticsEventTypeError    = "error"
)

// AnalyticsEvent represents a tracked analytics event
type AnalyticsEvent struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	OrgID     *uuid.UUID `json:"orgId,omitempty" db:"org_id"`
	UserID    uuid.UUID  `json:"userId" db:"user_id"`
	HostID    *uuid.UUID `json:"hostId,omitempty" db:"host_id"`
	EventType string     `json:"eventType" db:"event_type"`
	Payload   []byte     `json:"payload" db:"payload"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
}

// CreateAnalyticsEventRequest represents a request to create an event
type CreateAnalyticsEventRequest struct {
	OrgID     string          `json:"orgId,omitempty"`
	HostID    string          `json:"hostId,omitempty"`
	EventType string          `json:"eventType" binding:"required,oneof=session command transfer login error"`
	Payload   json.RawMessage `json:"payload"`
}

// AnalyticsEventResponse is the API response
type AnalyticsEventResponse struct {
	ID        uuid.UUID       `json:"id"`
	OrgID     *uuid.UUID      `json:"orgId,omitempty"`
	UserID    uuid.UUID       `json:"userId"`
	HostID    *uuid.UUID      `json:"hostId,omitempty"`
	EventType string          `json:"eventType"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"createdAt"`
}

// ListAnalyticsEventsResponse is the API response for listing
type ListAnalyticsEventsResponse struct {
	Items  []*AnalyticsEventResponse `json:"items"`
	Total  int                       `json:"total"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

// AnalyticsSummary represents aggregated analytics data
type AnalyticsSummary struct {
	EventType string `json:"eventType"`
	Count     int    `json:"count"`
}

// CountByEventTypeResponse is the API response for counts
type CountByEventTypeResponse struct {
	Items []*AnalyticsSummary `json:"items"`
}
