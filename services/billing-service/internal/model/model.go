package model

import (
	"time"

	"github.com/google/uuid"
)

// BillingPlan represents a subscription plan
type BillingPlan struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Description string    `json:"description" db:"description"`
	PriceCents  int       `json:"priceCents" db:"price_cents"`
	Currency    string    `json:"currency" db:"currency"`
	Interval    string    `json:"interval" db:"interval"`
	Features    []string  `json:"features" db:"features"`
	Limits      map[string]interface{} `json:"limits,omitempty" db:"limits"`
	IsActive    bool      `json:"isActive" db:"is_active"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// Subscription represents an organization subscription
type Subscription struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OrgID       uuid.UUID `json:"orgId" db:"org_id"`
	PlanID      uuid.UUID `json:"planId" db:"plan_id"`
	Status      string    `json:"status" db:"status"`
	StartedAt   time.Time `json:"startedAt" db:"started_at"`
	EndsAt      *time.Time `json:"endsAt,omitempty" db:"ends_at"`
	CanceledAt  *time.Time `json:"canceledAt,omitempty" db:"canceled_at"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// Invoice represents a billing invoice
type Invoice struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OrgID       uuid.UUID `json:"orgId" db:"org_id"`
	SubscriptionID uuid.UUID `json:"subscriptionId" db:"subscription_id"`
	AmountCents int       `json:"amountCents" db:"amount_cents"`
	Currency    string    `json:"currency" db:"currency"`
	Status      string    `json:"status" db:"status"`
	DueDate     time.Time `json:"dueDate" db:"due_date"`
	PaidAt      *time.Time `json:"paidAt,omitempty" db:"paid_at"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// UsageRecord represents a resource usage record
type UsageRecord struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OrgID       uuid.UUID `json:"orgId" db:"org_id"`
	ResourceType string   `json:"resourceType" db:"resource_type"`
	Quantity    int       `json:"quantity" db:"quantity"`
	PeriodStart time.Time `json:"periodStart" db:"period_start"`
	PeriodEnd   time.Time `json:"periodEnd" db:"period_end"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
}

// CreateSubscriptionRequest represents a request to create a subscription
type CreateSubscriptionRequest struct {
	OrgID  string `json:"orgId" binding:"required,uuid"`
	PlanID string `json:"planId" binding:"required,uuid"`
}

// UpdateSubscriptionRequest represents a request to update a subscription
type UpdateSubscriptionRequest struct {
	PlanID *string `json:"planId,omitempty"`
	Status *string `json:"status,omitempty" binding:"omitempty,oneof=active canceled expired"`
}

// ListSubscriptionsRequest represents a request to list subscriptions.
// Deliberately has NO OrgID field (T12): the tenant filter comes
// exclusively from the caller's authenticated identity (see
// internal/handler.callerOrgID), never from client-supplied input — a
// client-controlled org filter is exactly how the cross-tenant leak
// occurred (an omitted or arbitrary value bypassed tenant scoping
// entirely).
type ListSubscriptionsRequest struct {
	Status string `form:"status"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
}

// SubscriptionResponse is the API response for a subscription
type SubscriptionResponse struct {
	ID         uuid.UUID  `json:"id"`
	OrgID      uuid.UUID  `json:"orgId"`
	PlanID     uuid.UUID  `json:"planId"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	EndsAt     *time.Time `json:"endsAt,omitempty"`
	CanceledAt *time.Time `json:"canceledAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// ListSubscriptionsResponse is the API response for listing subscriptions
type ListSubscriptionsResponse struct {
	Items  []*SubscriptionResponse `json:"items"`
	Total  int                     `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

// InvoiceResponse is the API response for an invoice
type InvoiceResponse struct {
	ID             uuid.UUID  `json:"id"`
	OrgID          uuid.UUID  `json:"orgId"`
	SubscriptionID uuid.UUID  `json:"subscriptionId"`
	AmountCents    int        `json:"amountCents"`
	Currency       string     `json:"currency"`
	Status         string     `json:"status"`
	DueDate        time.Time  `json:"dueDate"`
	PaidAt         *time.Time `json:"paidAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// UsageRecordResponse is the API response for a usage record
type UsageRecordResponse struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"orgId"`
	ResourceType string    `json:"resourceType"`
	Quantity     int       `json:"quantity"`
	PeriodStart  time.Time `json:"periodStart"`
	PeriodEnd    time.Time `json:"periodEnd"`
	CreatedAt    time.Time `json:"createdAt"`
}
