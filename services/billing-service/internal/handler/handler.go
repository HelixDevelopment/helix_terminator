package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/billing-service/internal/model"
	"github.com/helixdevelopment/billing-service/internal/repository"
)

// Handler holds billing service handlers
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// callerOrgID returns the requesting tenant's org ID as established by the
// server's auth middleware (context key "orgID", populated from a
// validated JWT claim — see internal/server/server.go). It is the SOLE
// source of truth for tenant scoping on every billing read endpoint (T12):
// a client-supplied "orgId" query parameter or path segment MUST NEVER be
// trusted to select which tenant's data is served, since any caller could
// then read another tenant's subscriptions/invoices (or, if omitted,
// every tenant's data at once) by supplying an arbitrary or absent value.
// Returns ok=false when no valid identity is present, in which case the
// caller MUST reject the request (401) rather than fall back to serving
// unscoped data.
func callerOrgID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("orgID")
	if !exists {
		return uuid.Nil, false
	}
	str, ok := val.(string)
	if !ok || str == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(str)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

// CreateSubscription handles subscription creation
func (h *Handler) CreateSubscription(c *gin.Context) {
	var req model.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, _ := uuid.Parse(req.OrgID)
	planID, _ := uuid.Parse(req.PlanID)

	sub := &model.Subscription{
		ID:        uuid.New(),
		OrgID:     orgID,
		PlanID:    planID,
		Status:    "active",
		StartedAt: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := h.repo.CreateSubscription(c.Request.Context(), sub); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, toSubscriptionResponse(sub))
}

// GetSubscription handles retrieving a subscription by ID
func (h *Handler) GetSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	callerOrg, ok := callerOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid caller identity"})
		return
	}

	sub, err := h.repo.GetSubscriptionByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "subscription not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription"})
		return
	}
	// T12: a subscription that exists but belongs to a DIFFERENT tenant
	// must be indistinguishable from one that does not exist at all —
	// the same "subscription not found" response the ID-genuinely-missing
	// case already returns, so a cross-tenant probe cannot even confirm
	// another tenant's subscription ID is valid.
	if sub.OrgID != callerOrg {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	c.JSON(http.StatusOK, toSubscriptionResponse(sub))
}

// ListSubscriptions handles listing subscriptions with filtering
func (h *Handler) ListSubscriptions(c *gin.Context) {
	var req model.ListSubscriptionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// T12: the tenant filter comes EXCLUSIVELY from the authenticated
	// caller's identity, never from a client-supplied query parameter —
	// previously an omitted (or arbitrary) "orgId" query parameter meant
	// this endpoint returned every tenant's subscriptions.
	orgID, ok := callerOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid caller identity"})
		return
	}

	subs, total, err := h.repo.ListSubscriptions(c.Request.Context(), orgID, req.Status, req.Limit, req.Offset)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list subscriptions"})
		return
	}

	resp := &model.ListSubscriptionsResponse{
		Items:  make([]*model.SubscriptionResponse, len(subs)),
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}
	for i, s := range subs {
		resp.Items[i] = toSubscriptionResponse(s)
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateSubscription handles updating a subscription
func (h *Handler) UpdateSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	var req model.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.PlanID != nil {
		planID, _ := uuid.Parse(*req.PlanID)
		updates["plan_id"] = planID
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := h.repo.UpdateSubscription(c.Request.Context(), id, updates); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "subscription not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update subscription"})
		return
	}

	sub, err := h.repo.GetSubscriptionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated subscription"})
		return
	}
	c.JSON(http.StatusOK, toSubscriptionResponse(sub))
}

// CancelSubscription handles canceling a subscription
func (h *Handler) CancelSubscription(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	if err := h.repo.CancelSubscription(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "subscription not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel subscription"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// GetInvoice handles retrieving an invoice by ID
func (h *Handler) GetInvoice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice id"})
		return
	}

	callerOrg, ok := callerOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid caller identity"})
		return
	}

	inv, err := h.repo.GetInvoiceByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "invoice not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get invoice"})
		return
	}
	// T12: same not-found-for-a-different-tenant treatment as
	// GetSubscription — a cross-tenant probe must not be able to
	// distinguish "exists but not yours" from "does not exist".
	if inv.OrgID != callerOrg {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}
	c.JSON(http.StatusOK, toInvoiceResponse(inv))
}

// ListInvoices handles listing invoices for the caller's own org
func (h *Handler) ListInvoices(c *gin.Context) {
	// T12: the tenant filter comes EXCLUSIVELY from the authenticated
	// caller's identity, never from the client-supplied "orgId" query
	// parameter this endpoint previously required and trusted verbatim —
	// any caller could read any other tenant's invoices by supplying that
	// tenant's org ID.
	orgID, ok := callerOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid caller identity"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	invoices, total, err := h.repo.ListInvoices(c.Request.Context(), orgID, limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invoices"})
		return
	}

	resp := make([]model.InvoiceResponse, len(invoices))
	for i, inv := range invoices {
		resp[i] = toInvoiceResponse(inv)
	}
	c.JSON(http.StatusOK, gin.H{"invoices": resp, "total": total, "limit": limit, "offset": offset})
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "billing-service", "timestamp": time.Now().UTC()})
}

// ReadinessCheck returns service readiness status
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": "database not available"})
		return
	}
	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "reason": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "billing-service"})
}

func toSubscriptionResponse(sub *model.Subscription) *model.SubscriptionResponse {
	return &model.SubscriptionResponse{
		ID:         sub.ID,
		OrgID:      sub.OrgID,
		PlanID:     sub.PlanID,
		Status:     sub.Status,
		StartedAt:  sub.StartedAt,
		EndsAt:     sub.EndsAt,
		CanceledAt: sub.CanceledAt,
		CreatedAt:  sub.CreatedAt,
	}
}

func toInvoiceResponse(inv *model.Invoice) model.InvoiceResponse {
	return model.InvoiceResponse{
		ID:             inv.ID,
		OrgID:          inv.OrgID,
		SubscriptionID: inv.SubscriptionID,
		AmountCents:    inv.AmountCents,
		Currency:       inv.Currency,
		Status:         inv.Status,
		DueDate:        inv.DueDate,
		PaidAt:         inv.PaidAt,
		CreatedAt:      inv.CreatedAt,
	}
}
