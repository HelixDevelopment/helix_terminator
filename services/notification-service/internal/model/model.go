package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Notification represents a notification sent to a user
type Notification struct {
	ID      uuid.UUID  `json:"id" db:"id"`
	UserID  uuid.UUID  `json:"userId" db:"user_id"`
	OrgID   *uuid.UUID `json:"orgId,omitempty" db:"org_id"`
	Type    string     `json:"type" db:"type"`
	Title   string     `json:"title" db:"title"`
	Message string     `json:"message" db:"message"`
	Data    []byte     `json:"data,omitempty" db:"data"`
	Channel string     `json:"channel" db:"channel"`
	// Target is the delivery destination: recipient email address for
	// channel=email, destination URL for channel=webhook. Unused for
	// in_app/push.
	Target    string     `json:"target,omitempty" db:"target"`
	Status    string     `json:"status" db:"status"`
	ReadAt    *time.Time `json:"readAt,omitempty" db:"read_at"`
	SentAt    *time.Time `json:"sentAt,omitempty" db:"sent_at"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
}

// NotificationPreference represents a user's notification preferences for a channel
type NotificationPreference struct {
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	Channel   string    `json:"channel" db:"channel"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	Types     []string  `json:"types" db:"types"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// CreateNotificationRequest represents a request to create a notification.
//
// T18 (Constitution §11.4.102/.115/.146): this request previously carried
// client-supplied "userId"/"orgId" fields that the handler trusted
// verbatim to decide WHOSE notification inbox to write into — an IDOR
// (any authenticated caller could create/spoof a notification into
// ANOTHER user's inbox by simply naming a different userId). The target
// user and org are now derived EXCLUSIVELY from the caller's validated
// JWT claims (internal/server/server.go authMiddleware, T11), so these
// fields no longer exist here — mirrors billing-service's T14
// CreateSubscriptionRequest, which dropped its OrgID field for the same
// reason.
type CreateNotificationRequest struct {
	Type    string          `json:"type" binding:"required,oneof=info warning error success"`
	Title   string          `json:"title" binding:"required,max=255"`
	Message string          `json:"message" binding:"required,max=2000"`
	Data    json.RawMessage `json:"data,omitempty" binding:"omitempty,max=65536"`
	Channel string          `json:"channel" binding:"required,oneof=email in_app push webhook"`
	Status  string          `json:"status" binding:"omitempty,oneof=pending sent delivered failed"`
	// Target is the delivery destination, required for channel=email
	// (recipient email address) and channel=webhook (destination URL).
	// Ignored for in_app/push. The server always overwrites the persisted
	// status with the REAL delivery outcome for email/webhook/push — any
	// client-supplied Status above is honored only for in_app.
	Target string `json:"target,omitempty" binding:"omitempty,max=1000"`
}

// ListNotificationsRequest represents query parameters for listing
// notifications.
//
// T18: this request previously carried a client-supplied "user_id" query
// parameter that the handler trusted verbatim to decide WHOSE
// notifications to list — an IDOR (any authenticated caller could read
// ANOTHER user's notifications by supplying a different user_id). The
// scope is now derived EXCLUSIVELY from the caller's validated JWT claim
// (internal/server/server.go authMiddleware, T11), so the field no
// longer exists here. OrgID remains as an OPTIONAL further filter over
// the caller's OWN notifications (a user may belong to more than one
// org) — it can never expose another user's rows because the underlying
// query is always additionally scoped to the caller's own user_id.
type ListNotificationsRequest struct {
	OrgID string `form:"org_id" binding:"omitempty,uuid"`
	// Status vocabulary mirrors every real outcome deliverEmail/deliverWebhook/
	// deliverPush can persist: "pending" (default/in_app), "sent" (email/push
	// success), "delivered" (webhook success), "failed" (any channel's real
	// send error), "pending_provider_unconfigured" (push, no provider armed),
	// "failed_missing_target" (push, provider armed but no device token —
	// internal/handler/handler.go deliverPush).
	Status  string `form:"status" binding:"omitempty,oneof=pending sent delivered failed pending_provider_unconfigured failed_missing_target"`
	Channel string `form:"channel" binding:"omitempty,oneof=email in_app push webhook"`
	Limit   int    `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Offset  int    `form:"offset,default=0" binding:"omitempty,min=0"`
}

// MarkReadRequest represents a request to mark a notification as read
type MarkReadRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// UpdatePreferenceRequest represents a request to update notification
// preferences.
//
// T18: this request previously carried a client-supplied "userId" field
// that the handler trusted verbatim to decide WHOSE preference to write —
// an IDOR (any authenticated caller could overwrite ANOTHER user's
// notification preferences by supplying a different userId). The target
// user is now derived EXCLUSIVELY from the caller's validated JWT claim
// (internal/server/server.go authMiddleware, T11), so the field no
// longer exists here.
type UpdatePreferenceRequest struct {
	Channel string   `json:"channel" binding:"required,oneof=email in_app push webhook"`
	Enabled bool     `json:"enabled"`
	Types   []string `json:"types" binding:"omitempty,dive,oneof=info warning error success"`
}

// NotificationResponse represents a notification in API responses
type NotificationResponse struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"userId"`
	OrgID     *uuid.UUID      `json:"orgId,omitempty"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data,omitempty"`
	Channel   string          `json:"channel"`
	Target    string          `json:"target,omitempty"`
	Status    string          `json:"status"`
	ReadAt    *time.Time      `json:"readAt,omitempty"`
	SentAt    *time.Time      `json:"sentAt,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// PreferenceResponse represents a notification preference in API responses
type PreferenceResponse struct {
	UserID    uuid.UUID `json:"userId"`
	Channel   string    `json:"channel"`
	Enabled   bool      `json:"enabled"`
	Types     []string  `json:"types"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
