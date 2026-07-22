package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/billing-service/internal/billing"
	"github.com/helixdevelopment/billing-service/internal/model"
	"github.com/helixdevelopment/billing-service/internal/repository"
)

// Handler holds billing service handlers
type Handler struct {
	repo     *repository.Repository
	provider billing.PaymentProvider
}

// Option configures a Handler at construction time.
type Option func(*Handler)

// WithProvider wires a real billing.PaymentProvider into the Handler.
// Omitting it (the zero value — provider stays nil) is the honest "no
// payment processor is configured" operating mode (Constitution §11.4
// anti-bluff): every subscription-lifecycle-mutating endpoint
// (CreateSubscription / UpdateSubscription's plan-change path /
// CancelSubscription's processor-backed path / StripeWebhook) responds
// 501 "payments provider not configured" rather than ever fabricate a
// success with no processor behind it — see internal/billing/provider.go
// for the full rationale.
func WithProvider(p billing.PaymentProvider) Option {
	return func(h *Handler) { h.provider = p }
}

// New creates a new Handler
func New(repo *repository.Repository, opts ...Option) *Handler {
	h := &Handler{repo: repo}
	for _, opt := range opts {
		opt(h)
	}
	return h
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

// idempotencyKeyFromRequest returns the client-supplied "Idempotency-Key"
// header when present (the standard REST idiom Stripe's own API itself
// uses — a client that wants retry-safety across a network timeout sets
// this header identically on every retry of the same logical request),
// or a freshly generated key otherwise. A freshly generated per-call key
// provides no retry protection on its own, but that is strictly safer
// than the alternative of deriving a key from request CONTENT (e.g. org
// + price): a content-derived key would make Stripe's idempotency layer
// silently return the FIRST subscription's result for a second,
// legitimately-different subscription create to the same price by the
// same org — a subtle, hard-to-detect variant of exactly the bluff this
// package exists to prevent (a client asking for and expecting a new
// resource, silently getting an old one back instead).
func idempotencyKeyFromRequest(c *gin.Context) string {
	if key := c.GetHeader("Idempotency-Key"); key != "" {
		return key
	}
	return uuid.NewString()
}

// CreateSubscription handles subscription creation.
//
// Constitution §11.4 anti-bluff — THE BLUFF THIS FIX CLOSES: this
// handler used to persist a new subscription row with Status:"active"
// unconditionally, with NO payment processor ever contacted — a
// fabricated success. It now REQUIRES a configured billing.PaymentProvider
// (see WithProvider / internal/billing.NewProviderFromEnv, wired from
// STRIPE_SECRET_KEY by internal/server.New) and persists ONLY the real
// status the processor actually returned. With no provider configured,
// this endpoint honestly reports 501 Not Implemented — never a
// fabricated "active".
func (h *Handler) CreateSubscription(c *gin.Context) {
	var req model.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// T14: the subscription's org is derived EXCLUSIVELY from the
	// authenticated caller's identity, never from client-supplied input —
	// previously this endpoint trusted a client-supplied "orgId" body
	// field verbatim (uuid.Parse(req.OrgID)), letting any caller create a
	// subscription attributed to an ARBITRARY org, including another
	// tenant's.
	callerOrg, ok := callerOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid caller identity"})
		return
	}

	// §11.4 anti-bluff honest feature-flag: no configured processor ⇒
	// honestly refuse rather than fabricate a subscription with nothing
	// behind it. This is checked BEFORE any DB access so the response is
	// deterministic regardless of database availability.
	if h.provider == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "payments provider not configured"})
		return
	}

	if req.StripePriceID == nil || strings.TrimSpace(*req.StripePriceID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stripePriceId is required when a payment provider is configured"})
		return
	}

	planID, _ := uuid.Parse(req.PlanID)

	// Reuse this org's existing processor-side customer record (from a
	// prior subscription, if any) rather than creating a duplicate
	// customer for every subscription the org creates.
	existingCustomerID, err := h.repo.GetLatestExternalCustomerID(c.Request.Context(), callerOrg, h.provider.Name())
	if err != nil {
		if strings.Contains(err.Error(), "database not connected") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve billing customer"})
		return
	}

	result, err := h.provider.CreateSubscription(c.Request.Context(), billing.CreateSubscriptionInput{
		OrgID:              callerOrg.String(),
		PriceID:            *req.StripePriceID,
		ExistingCustomerID: existingCustomerID,
		IdempotencyKey:     idempotencyKeyFromRequest(c),
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "payment provider rejected the subscription create", "details": err.Error()})
		return
	}

	sub := &model.Subscription{
		ID:                     uuid.New(),
		OrgID:                  callerOrg,
		PlanID:                 planID,
		Status:                 result.Status, // the REAL status the processor returned — never hardcoded
		StartedAt:              time.Now().UTC(),
		CreatedAt:              time.Now().UTC(),
		UpdatedAt:              time.Now().UTC(),
		Provider:               h.provider.Name(),
		ExternalSubscriptionID: result.ExternalSubscriptionID,
		ExternalCustomerID:     result.ExternalCustomerID,
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

// UpdateSubscription handles updating a subscription.
//
// Constitution §11.4 anti-bluff: a plan/price change on a
// processor-backed subscription (ExternalSubscriptionID != "") is a
// REAL billing event and MUST go through the configured
// billing.PaymentProvider — it is never applied as a local-only DB
// write for such a row. A subscription that was never processor-backed
// (Provider == "none" — see migration 002_payment_provider) has no
// processor truth to diverge from, so its plan_id may still be updated
// as local bookkeeping without a provider. Status updates accept only
// {canceled, expired} (model.UpdateSubscriptionRequest) — reactivating
// a subscription to "active" via this endpoint is deliberately
// impossible; that must come from a real processor result (Create /
// the plan-change path below / a verified webhook), never a bare PUT.
func (h *Handler) UpdateSubscription(c *gin.Context) {
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

	var req model.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.PlanID != nil && req.Status != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot change plan and status in the same request"})
		return
	}

	// T14: fetch-then-compare BEFORE mutating, mirroring T12's
	// GetSubscription no-existence-oracle treatment — a subscription that
	// exists but belongs to a DIFFERENT tenant returns the SAME
	// "subscription not found" response as a genuinely-missing id. This
	// runs even when the update body carries no changed fields, closing
	// an oracle an empty-body PUT would otherwise open (an empty update
	// is a repository no-op that would otherwise skip any ownership
	// check and let the post-update re-fetch below hand back another
	// tenant's current subscription state).
	existing, err := h.repo.GetSubscriptionByID(c.Request.Context(), id)
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
	if existing.OrgID != callerOrg {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	updates := make(map[string]interface{})
	if req.PlanID != nil {
		planID, _ := uuid.Parse(*req.PlanID)
		if existing.ExternalSubscriptionID != "" {
			// Processor-backed row: the price change MUST be applied
			// against the real processor before we ever write a new
			// plan_id locally — never assert a plan change locally that
			// the processor was never asked to make.
			if h.provider == nil {
				c.JSON(http.StatusNotImplemented, gin.H{"error": "payments provider not configured; cannot change a processor-backed subscription's plan"})
				return
			}
			if req.StripePriceID == nil || strings.TrimSpace(*req.StripePriceID) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "stripePriceId is required to change a processor-backed subscription's plan"})
				return
			}
			result, perr := h.provider.UpdateSubscription(c.Request.Context(), billing.UpdateSubscriptionInput{
				ExternalSubscriptionID: existing.ExternalSubscriptionID,
				NewPriceID:             *req.StripePriceID,
				IdempotencyKey:         idempotencyKeyFromRequest(c),
			})
			if perr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "payment provider rejected the subscription update", "details": perr.Error()})
				return
			}
			updates["plan_id"] = planID
			updates["status"] = result.Status // the REAL status the processor returned
			if result.ExternalCustomerID != "" {
				updates["external_customer_id"] = result.ExternalCustomerID
			}
		} else {
			// Never processor-backed (Provider == "none"): local
			// bookkeeping only, no processor truth to diverge from.
			updates["plan_id"] = planID
		}
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	// T14: the mutation itself is ALSO scoped to the caller's own org —
	// defense in depth on top of the fetch-then-compare check above, so a
	// TOCTOU window between the check and the write can never let a
	// mutation land against another tenant's row.
	if err := h.repo.UpdateSubscription(c.Request.Context(), id, callerOrg, updates); err != nil {
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

// CancelSubscription handles canceling a subscription.
//
// Constitution §11.4 anti-bluff: a processor-backed subscription
// (ExternalSubscriptionID != "") is canceled by REALLY calling the
// configured billing.PaymentProvider and persisting the REAL status it
// returns — never a hardcoded "canceled" literal, and never applied
// locally-only when no provider is configured (that would leave this
// service's record diverged from the processor's real state, a
// data-integrity variant of the same bluff). A subscription that was
// never processor-backed has no processor truth to diverge from and is
// canceled locally.
func (h *Handler) CancelSubscription(c *gin.Context) {
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

	// T14: fetch-then-compare BEFORE mutating — same no-existence-oracle
	// treatment as UpdateSubscription/GetSubscription (T12): a
	// subscription that exists but belongs to a DIFFERENT tenant returns
	// the SAME "subscription not found" response as a genuinely-missing
	// id.
	existing, err := h.repo.GetSubscriptionByID(c.Request.Context(), id)
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
	if existing.OrgID != callerOrg {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	realStatus := "canceled"
	if existing.ExternalSubscriptionID != "" {
		if h.provider == nil {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "payments provider not configured; cannot cancel a processor-backed subscription without it"})
			return
		}
		result, perr := h.provider.CancelSubscription(c.Request.Context(), billing.CancelSubscriptionInput{
			ExternalSubscriptionID: existing.ExternalSubscriptionID,
			IdempotencyKey:         idempotencyKeyFromRequest(c),
		})
		if perr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "payment provider rejected the subscription cancel", "details": perr.Error()})
			return
		}
		realStatus = result.Status
	}

	// T14: the mutation itself is ALSO scoped to the caller's own org —
	// defense in depth on top of the fetch-then-compare check above.
	if err := h.repo.CancelSubscription(c.Request.Context(), id, callerOrg, realStatus); err != nil {
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

// StripeWebhook handles inbound Stripe webhook events (Constitution
// §11.4: this is what actually wires billing.PaymentProvider.VerifyWebhook
// end-to-end — a VerifyWebhook implementation that no HTTP endpoint ever
// calls would be dead, unproven code). Mounted OUTSIDE the JWT auth
// middleware group (see internal/server/server.go) since Stripe
// authenticates a webhook delivery via its own Stripe-Signature header
// scheme, never a bearer JWT.
//
// Reconciliation scope: for customer.subscription.updated and
// customer.subscription.deleted events, the locally-stored subscription
// row (matched by provider + external_subscription_id) has its status
// reconciled to the REAL status the processor now reports — closing the
// gap where a processor-initiated change (e.g. a failed payment
// auto-canceling a subscription) would otherwise never reach this
// service's own records, leaving them silently stale (itself a
// data-integrity variant of the bluff this service exists to close).
// Other event types are acknowledged (200) without action: Stripe
// requires a 2xx response to stop retrying delivery, and doing nothing
// for an event type this service does not yet reconcile is honest — it
// never pretends to have handled something it didn't.
func (h *Handler) StripeWebhook(c *gin.Context) {
	if h.provider == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "payments provider not configured"})
		return
	}

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	event, err := h.provider.VerifyWebhook(payload, c.GetHeader("Stripe-Signature"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "webhook signature verification failed"})
		return
	}

	switch event.Type {
	case "customer.subscription.updated", "customer.subscription.deleted":
		if h.repo != nil {
			if extID, status, perr := billing.ParseSubscriptionObject(event.ObjectRaw); perr == nil && extID != "" && status != "" {
				// Best-effort reconciliation: a failure here does not
				// change the fact that the webhook's signature was
				// genuinely verified, so Stripe still gets a 2xx and
				// will not endlessly retry delivery of an event this
				// service cannot currently persist against (e.g. DB
				// unavailable) — but it IS surfaced in the response
				// body so an operator inspecting delivery logs sees it.
				if uerr := h.repo.UpdateSubscriptionStatusByExternalID(c.Request.Context(), h.provider.Name(), extID, status); uerr != nil {
					c.JSON(http.StatusOK, gin.H{"received": true, "eventId": event.ID, "reconciled": false, "reconcileError": uerr.Error()})
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"received": true, "eventId": event.ID})
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
		ID:                     sub.ID,
		OrgID:                  sub.OrgID,
		PlanID:                 sub.PlanID,
		Status:                 sub.Status,
		StartedAt:              sub.StartedAt,
		EndsAt:                 sub.EndsAt,
		CanceledAt:             sub.CanceledAt,
		CreatedAt:              sub.CreatedAt,
		Provider:               sub.Provider,
		ExternalSubscriptionID: sub.ExternalSubscriptionID,
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
