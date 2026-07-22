package model

import (
	"time"

	"github.com/google/uuid"
)

// BillingPlan represents a subscription plan
type BillingPlan struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Slug        string                 `json:"slug" db:"slug"`
	Description string                 `json:"description" db:"description"`
	PriceCents  int                    `json:"priceCents" db:"price_cents"`
	Currency    string                 `json:"currency" db:"currency"`
	Interval    string                 `json:"interval" db:"interval"`
	Features    []string               `json:"features" db:"features"`
	Limits      map[string]interface{} `json:"limits,omitempty" db:"limits"`
	IsActive    bool                   `json:"isActive" db:"is_active"`
	CreatedAt   time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time              `json:"updatedAt" db:"updated_at"`
}

// Subscription represents an organization subscription.
//
// Provider/ExternalSubscriptionID/ExternalCustomerID (migration
// 002_payment_provider) are the anti-bluff proof-of-real-call columns
// (Constitution §11.4 — see internal/billing/provider.go): Provider
// defaults to "none" for any row that was never created through a real
// payment processor (either created before this migration, or created
// while no PaymentProvider was configured is now structurally
// impossible — see internal/handler.CreateSubscription — but "none"
// stays the honest default for defense-in-depth). A row with
// Provider != "none" carries the REAL processor's own subscription id
// (ExternalSubscriptionID) and customer id (ExternalCustomerID),
// evidence a real API call actually happened.
type Subscription struct {
	ID                     uuid.UUID  `json:"id" db:"id"`
	OrgID                  uuid.UUID  `json:"orgId" db:"org_id"`
	PlanID                 uuid.UUID  `json:"planId" db:"plan_id"`
	Status                 string     `json:"status" db:"status"`
	StartedAt              time.Time  `json:"startedAt" db:"started_at"`
	EndsAt                 *time.Time `json:"endsAt,omitempty" db:"ends_at"`
	CanceledAt             *time.Time `json:"canceledAt,omitempty" db:"canceled_at"`
	CreatedAt              time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt              time.Time  `json:"updatedAt" db:"updated_at"`
	Provider               string     `json:"provider" db:"provider"`
	ExternalSubscriptionID string     `json:"externalSubscriptionId,omitempty" db:"external_subscription_id"`
	ExternalCustomerID     string     `json:"externalCustomerId,omitempty" db:"external_customer_id"`
}

// Invoice represents a billing invoice
type Invoice struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrgID          uuid.UUID  `json:"orgId" db:"org_id"`
	SubscriptionID uuid.UUID  `json:"subscriptionId" db:"subscription_id"`
	AmountCents    int        `json:"amountCents" db:"amount_cents"`
	Currency       string     `json:"currency" db:"currency"`
	Status         string     `json:"status" db:"status"`
	DueDate        time.Time  `json:"dueDate" db:"due_date"`
	PaidAt         *time.Time `json:"paidAt,omitempty" db:"paid_at"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time  `json:"updatedAt" db:"updated_at"`
}

// UsageRecord represents a resource usage record
type UsageRecord struct {
	ID           uuid.UUID `json:"id" db:"id"`
	OrgID        uuid.UUID `json:"orgId" db:"org_id"`
	ResourceType string    `json:"resourceType" db:"resource_type"`
	Quantity     int       `json:"quantity" db:"quantity"`
	PeriodStart  time.Time `json:"periodStart" db:"period_start"`
	PeriodEnd    time.Time `json:"periodEnd" db:"period_end"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

// CreateSubscriptionRequest represents a request to create a subscription.
// Deliberately has NO OrgID field (T14): the subscription's org comes
// exclusively from the caller's authenticated identity (see
// internal/handler.callerOrgID), never from client-supplied input — a
// client-controlled org field is exactly how a caller could previously
// create a subscription attributed to an ARBITRARY org, including
// another tenant's (the same cross-tenant IDOR root cause T12 closed
// for the read endpoints, left open here until T14).
//
// StripePriceID (Constitution §11.4 anti-bluff — see
// internal/billing/provider.go): the Stripe Price object id
// ("price_...") the new subscription should be created against. It is
// REQUIRED whenever a PaymentProvider is configured (STRIPE_SECRET_KEY
// set) — the handler layer rejects a missing value with 400 rather
// than silently proceeding without one. It is validated at the
// business-logic layer (internal/handler.CreateSubscription), not via
// a static binding tag, because "required" here is CONDITIONAL on
// runtime provider configuration, something a static tag cannot
// express. See docs/guides/BILLING.md "Creating a subscription" for a
// worked example.
type CreateSubscriptionRequest struct {
	PlanID        string  `json:"planId" binding:"required,uuid"`
	StripePriceID *string `json:"stripePriceId,omitempty"`
}

// UpdateSubscriptionRequest represents a request to update a subscription.
//
// Status's allowed values deliberately EXCLUDE "active" (Constitution
// §11.4 anti-bluff): reactivating/marking a subscription active is a
// REAL processor-side billing event (a resumed subscription, a paid
// invoice) that must be learned from the processor (via a verified
// webhook, or the real result of CreateSubscription/UpdateSubscription)
// — never asserted directly by a PUT request with no processor
// involved whatsoever, which is exactly how the original bluff this
// service is being fixed for worked. "canceled"/"expired" remain
// client-settable as LOCAL bookkeeping (e.g. reconciling state learned
// out-of-band); the real end-user cancel path is the dedicated
// POST .../cancel endpoint, which DOES call the configured
// PaymentProvider.
//
// StripePriceID, like CreateSubscriptionRequest's, is REQUIRED (checked
// at the business-logic layer) whenever PlanID is also supplied and a
// PaymentProvider is configured — a plan change is a real price change
// against the processor.
type UpdateSubscriptionRequest struct {
	PlanID        *string `json:"planId,omitempty"`
	StripePriceID *string `json:"stripePriceId,omitempty"`
	Status        *string `json:"status,omitempty" binding:"omitempty,oneof=canceled expired"`
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

// SubscriptionResponse is the API response for a subscription. Provider
// and ExternalSubscriptionID are surfaced so an API consumer can itself
// verify (e.g. against the Stripe dashboard) that "status" reflects a
// real processor-side object rather than taking it on faith —
// Constitution §11.4 anti-bluff transparency.
type SubscriptionResponse struct {
	ID                     uuid.UUID  `json:"id"`
	OrgID                  uuid.UUID  `json:"orgId"`
	PlanID                 uuid.UUID  `json:"planId"`
	Status                 string     `json:"status"`
	StartedAt              time.Time  `json:"startedAt"`
	EndsAt                 *time.Time `json:"endsAt,omitempty"`
	CanceledAt             *time.Time `json:"canceledAt,omitempty"`
	CreatedAt              time.Time  `json:"createdAt"`
	Provider               string     `json:"provider"`
	ExternalSubscriptionID string     `json:"externalSubscriptionId,omitempty"`
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
