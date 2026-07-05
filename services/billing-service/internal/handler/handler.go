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

	var orgID uuid.UUID
	if req.OrgID != "" {
		orgID, _ = uuid.Parse(req.OrgID)
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
	c.JSON(http.StatusOK, toInvoiceResponse(inv))
}

// ListInvoices handles listing invoices for an org
func (h *Handler) ListInvoices(c *gin.Context) {
	orgIDStr := c.Query("orgId")
	if orgIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orgId query parameter required"})
		return
	}
	orgID, _ := uuid.Parse(orgIDStr)

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
