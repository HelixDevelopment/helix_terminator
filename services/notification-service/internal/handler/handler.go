package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/helixdevelopment/notification-service/internal/delivery"
	"github.com/helixdevelopment/notification-service/internal/model"
	"github.com/helixdevelopment/notification-service/internal/repository"
)

// Handler holds notification service handlers
type Handler struct {
	repo          *repository.Repository
	emailSender   *delivery.EmailSender
	webhookSender *delivery.WebhookSender
	pushSender    *delivery.PushSender
}

// New returns a new Handler with dependencies. Delivery clients are built
// from environment configuration (Constitution §11.4.10 — never hardcoded):
// email requires SMTP_HOST to be set (see delivery.SMTPConfigFromEnv); the
// webhook sender needs no external configuration; push has no real provider
// wired yet and always reports an honest "not configured" outcome.
func New(repo *repository.Repository) *Handler {
	h := &Handler{
		repo:          repo,
		webhookSender: delivery.NewWebhookSender(10 * time.Second),
		pushSender:    delivery.NewPushSender(),
	}
	if cfg, ok := delivery.SMTPConfigFromEnv(); ok {
		h.emailSender = delivery.NewEmailSender(cfg)
	}
	// Push (FCM/APNs): arm the sender only when a COMPLETE provider credential
	// set is present (env-sourced, never hardcoded — §11.4.10). The default
	// stays the unconfigured NewPushSender() above; an armed sender still makes
	// no network call and never fabricates delivery (see delivery.PushSender).
	if cfg, ok := delivery.PushConfigFromEnv(); ok {
		h.pushSender = delivery.NewPushSenderWithConfig(cfg)
	}
	return h
}

// NewWithDelivery returns a new Handler with explicitly supplied delivery
// clients. This is the constructor tests use to point real senders at real
// test infrastructure (a real SMTP sink, a real HTTP receiver) — per
// Constitution §11.4.27 no fakes/mocks are used beyond unit tests, so tests
// exercising this handler wire REAL delivery.EmailSender / WebhookSender /
// PushSender instances, never a mock double.
func NewWithDelivery(repo *repository.Repository, emailSender *delivery.EmailSender, webhookSender *delivery.WebhookSender, pushSender *delivery.PushSender) *Handler {
	return &Handler{repo: repo, emailSender: emailSender, webhookSender: webhookSender, pushSender: pushSender}
}

// callerUserID returns the requesting caller's user ID as established by
// the server's auth middleware (context key "userID", populated from a
// validated JWT claim — see internal/server/server.go authMiddleware,
// T11). It is the SOLE source of truth for WHOSE notifications/
// preferences a request may read or write (T18 — Constitution
// §11.4.102/.115/.146): a client-supplied "user_id" query parameter or
// "userId" body field MUST NEVER be trusted to select which user's data
// is served or mutated, since any authenticated caller could then
// read/create/modify/delete another user's notifications or preferences
// simply by supplying a different user_id (IDOR). Mirrors billing-
// service's T12/T14 callerOrgID helper. Returns ok=false when no valid
// identity is present in the context, in which case the caller MUST
// reject the request (401) rather than fall back to unscoped or
// client-supplied behaviour.
func callerUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("userID")
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

// callerOrgID returns the requesting caller's org ID (if any) as
// established by the auth middleware (context key "orgID"). Unlike
// callerUserID it is optional — the JWT's orgId claim itself is
// `omitempty` — and returns nil (never an error) when absent. Used by
// CreateNotification so a created notification's org tag reflects the
// caller's OWN org claim, never a client-supplied "orgId" body field
// (T18: the prior body field let any caller tag a notification under an
// arbitrary org, including one they do not belong to).
func callerOrgID(c *gin.Context) *uuid.UUID {
	val, exists := c.Get("orgID")
	if !exists {
		return nil
	}
	str, ok := val.(string)
	if !ok || str == "" {
		return nil
	}
	id, err := uuid.Parse(str)
	if err != nil || id == uuid.Nil {
		return nil
	}
	return &id
}

// CreateNotification handles POST /api/v1/notifications
func (h *Handler) CreateNotification(c *gin.Context) {
	var req model.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := callerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid caller identity"})
		return
	}

	orgID := callerOrgID(c)

	// Channel-specific target validation — fail fast instead of silently
	// persisting a notification that can never be delivered.
	switch req.Channel {
	case "email":
		if req.Target == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target (recipient email address) is required for channel=email"})
			return
		}
	case "webhook":
		if req.Target == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target (webhook URL) is required for channel=webhook"})
			return
		}
	}

	status := req.Status
	if status == "" {
		status = "pending"
	}

	now := time.Now().UTC()
	notification := &model.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		OrgID:     orgID,
		Type:      req.Type,
		Title:     req.Title,
		Message:   req.Message,
		Data:      req.Data,
		Channel:   req.Channel,
		Target:    req.Target,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// REAL delivery attempt — the persisted status reflects what actually
	// happened, never a fabricated success (Constitution §11.4 anti-bluff
	// covenant). in_app has no external transport so it keeps whatever
	// status was requested (default "pending").
	switch req.Channel {
	case "email":
		h.deliverEmail(c.Request.Context(), notification)
	case "webhook":
		h.deliverWebhook(c.Request.Context(), notification)
	case "push":
		h.deliverPush(c.Request.Context(), notification)
	}

	if err := h.repo.CreateNotification(c.Request.Context(), notification); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toNotificationResponse(notification))
}

// deliverEmail attempts real SMTP delivery and sets notification.Status to
// the REAL outcome: "sent" only if the configured SMTP server actually
// accepted the message, "failed" (with the real error preserved in Data via
// the caller's logs — the notification row itself only carries the status,
// per the existing schema) otherwise. Never fabricates success.
func (h *Handler) deliverEmail(ctx context.Context, n *model.Notification) {
	if h.emailSender == nil {
		n.Status = "failed"
		return
	}
	if err := h.emailSender.Send(ctx, n.Target, n.Title, n.Message); err != nil {
		n.Status = "failed"
		return
	}
	sentAt := time.Now().UTC()
	n.Status = "sent"
	n.SentAt = &sentAt
}

// deliverWebhook attempts a real outbound HTTP POST of the notification
// payload to n.Target. A 2xx response yields "delivered"; anything else
// (transport error, timeout, non-2xx status) yields "failed".
func (h *Handler) deliverWebhook(ctx context.Context, n *model.Notification) {
	if h.webhookSender == nil {
		n.Status = "failed"
		return
	}
	payload := delivery.WebhookPayload{
		ID:      n.ID.String(),
		UserID:  n.UserID.String(),
		Type:    n.Type,
		Title:   n.Title,
		Message: n.Message,
		Channel: n.Channel,
		Data:    json.RawMessage(n.Data),
	}
	if _, err := h.webhookSender.Send(ctx, n.Target, payload); err != nil {
		n.Status = "failed"
		return
	}
	sentAt := time.Now().UTC()
	n.Status = "delivered"
	n.SentAt = &sentAt
}

// deliverPush attempts a REAL FCM/APNs push via h.pushSender and sets
// notification.Status to the REAL outcome — it NEVER fabricates a
// "sent"/"delivered" status for a channel with no real backend wired in,
// AND it never silently no-ops when a real backend IS wired in.
//
// T-PUSH-WIRING (PR #8 review, Important finding): this previously called
// the arg-less PushSender.Send(), which always dispatches
// SendTo(ctx, "", PushPayload{}) — an empty device token — so even an ARMED
// provider (real FCM/APNs credentials present) short-circuited to
// ErrPushTokenEmpty BEFORE sendFCM/sendAPNs ever ran, and every push
// notification was persisted as "pending_provider_unconfigured" regardless
// of whether a provider was actually configured (armed credentials produced
// ZERO real sends — Constitution §11.4.108 present-but-not-wired /
// §11.4.197). It now calls SendTo with the notification's REAL device
// token (n.Target, mirroring the email/webhook channels' use of n.Target as
// the delivery destination) and content, so an armed provider genuinely
// dispatches through sendFCM/sendAPNs.
func (h *Handler) deliverPush(ctx context.Context, n *model.Notification) {
	if h.pushSender == nil {
		n.Status = "pending_provider_unconfigured"
		return
	}

	err := h.pushSender.SendTo(ctx, n.Target, delivery.PushPayload{
		Title: n.Title,
		Body:  n.Message,
		Data:  pushDataFromRaw(n.Data),
	})

	switch {
	case err == nil:
		sentAt := time.Now().UTC()
		n.Status = "sent"
		n.SentAt = &sentAt
	case errors.Is(err, delivery.ErrPushProviderNotConfigured):
		// No provider armed (covers both h.pushSender == NewPushSender()
		// zero-value AND a hand-constructed unconfigured sender) — the
		// honest not-configured state, unchanged from before this fix.
		n.Status = "pending_provider_unconfigured"
	case errors.Is(err, delivery.ErrPushTokenEmpty):
		// A provider IS configured, but this notification carries no
		// device/registration token to send to — distinct from
		// "pending_provider_unconfigured", which would now be INACCURATE
		// (it would claim no provider is armed when one genuinely is).
		n.Status = "failed_missing_target"
	default:
		// A configured provider's REAL send call returned a real error
		// (transport failure, non-2xx rejection, provider-reported
		// per-message failure) — surfaced honestly, never a fabricated
		// "sent".
		n.Status = "failed"
	}
}

// pushDataFromRaw converts a notification's raw JSON Data payload
// (model.Notification.Data, arbitrary operator-supplied JSON) into the flat
// string map FCM HTTP v1 / the legacy FCM endpoint require for
// delivery.PushPayload.Data (FCM mandates a data map of string values;
// APNs's custom-data payload uses the same shape here for parity). When
// Data is empty, absent, or not shaped as a flat JSON object of string
// values, it is omitted from the push (best-effort) rather than failing the
// entire send — Title/Body remain the push's primary content either way.
func pushDataFromRaw(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// ListNotifications handles GET /api/v1/notifications
func (h *Handler) ListNotifications(c *gin.Context) {
	var req model.ListNotificationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := callerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid caller identity"})
		return
	}

	var orgID *uuid.UUID
	if req.OrgID != "" {
		parsed, err := uuid.Parse(req.OrgID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
			return
		}
		orgID = &parsed
	}

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	notifications, total, err := h.repo.ListNotifications(c.Request.Context(), userID, orgID, req.Status, req.Channel, limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	var responses []model.NotificationResponse
	for _, n := range notifications {
		responses = append(responses, toNotificationResponse(n))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   responses,
		"total":  total,
		"limit":  limit,
		"offset": req.Offset,
	})
}

// GetNotification handles GET /api/v1/notifications/:id
func (h *Handler) GetNotification(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	callerID, ok := callerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid caller identity"})
		return
	}

	notification, err := h.repo.GetNotificationByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	// T18 (mirrors billing-service's T12 no-existence-oracle pattern): a
	// notification belonging to a DIFFERENT caller MUST get the SAME
	// "notification not found" response a genuinely-missing id would —
	// never a distinct status/message that would let a caller confirm
	// another user's notification ID exists.
	if notification.UserID != callerID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	c.JSON(http.StatusOK, toNotificationResponse(notification))
}

// MarkRead handles POST /api/v1/notifications/:id/read
func (h *Handler) MarkRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	callerID, ok := callerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid caller identity"})
		return
	}

	// T18: fetch-then-compare BEFORE mutating — the pre-fix handler
	// mutated ANY id with no ownership check whatsoever, letting any
	// authenticated caller mark another user's notification as read
	// merely by learning/guessing its id. A cross-caller target returns
	// the SAME "notification not found" a genuinely-missing id would.
	existing, err := h.repo.GetNotificationByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if existing.UserID != callerID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	// T18 follow-up (Constitution §11.4.134 review finding): the repo
	// mutation ALSO scopes WHERE id = $1 AND user_id = $2 (defense in
	// depth on top of the fetch-then-compare above) — a mismatch here
	// mirrors the same "notification not found" the fetch-then-compare
	// check above would have already produced, never a distinct status.
	if err := h.repo.MarkRead(c.Request.Context(), id, callerID); err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read"})
}

// MarkAllRead handles POST /api/v1/notifications/read-all
func (h *Handler) MarkAllRead(c *gin.Context) {
	userID, ok := callerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid caller identity"})
		return
	}

	if err := h.repo.MarkAllRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all notifications marked as read"})
}

// DeleteNotification handles DELETE /api/v1/notifications/:id
func (h *Handler) DeleteNotification(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	callerID, ok := callerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid caller identity"})
		return
	}

	// T18: fetch-then-compare BEFORE deleting — mirrors MarkRead above.
	existing, err := h.repo.GetNotificationByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if existing.UserID != callerID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}

	// T18 follow-up (Constitution §11.4.134 review finding): defense in
	// depth, same rationale as MarkRead above.
	if err := h.repo.DeleteNotification(c.Request.Context(), id, callerID); err != nil {
		if err.Error() == "notification not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CountUnread handles GET /api/v1/notifications/unread-count
func (h *Handler) CountUnread(c *gin.Context) {
	userID, ok := callerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid caller identity"})
		return
	}

	count, err := h.repo.CountUnread(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetPreference handles GET /api/v1/notifications/preferences
func (h *Handler) GetPreference(c *gin.Context) {
	channel := c.Query("channel")
	if channel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel is required"})
		return
	}

	userID, ok := callerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid caller identity"})
		return
	}

	pref, err := h.repo.GetPreference(c.Request.Context(), userID, channel)
	if err != nil {
		if err.Error() == "preference not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toPreferenceResponse(pref))
}

// UpdatePreference handles PUT /api/v1/notifications/preferences
func (h *Handler) UpdatePreference(c *gin.Context) {
	var req model.UpdatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := callerUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing or invalid caller identity"})
		return
	}

	// T24: when the client omits "types", the slice is nil/empty — the DB
	// column is NOT NULL, so the upsert would fail with a constraint
	// violation → 503. Default to ["all"] so a minimal request succeeds.
	if len(req.Types) == 0 {
		req.Types = []string{"all"}
	}

	pref := &model.NotificationPreference{
		UserID:  userID,
		Channel: req.Channel,
		Enabled: req.Enabled,
		Types:   req.Types,
	}

	if err := h.repo.UpdatePreference(c.Request.Context(), pref); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toPreferenceResponse(pref))
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "notification-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"service": "notification-service",
			"error":   "database not connected",
		})
		return
	}

	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":   false,
			"service": "notification-service",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"service":   "notification-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func toNotificationResponse(n *model.Notification) model.NotificationResponse {
	var data json.RawMessage
	if n.Data != nil {
		data = json.RawMessage(n.Data)
	}
	return model.NotificationResponse{
		ID:        n.ID,
		UserID:    n.UserID,
		OrgID:     n.OrgID,
		Type:      n.Type,
		Title:     n.Title,
		Message:   n.Message,
		Data:      data,
		Channel:   n.Channel,
		Status:    n.Status,
		ReadAt:    n.ReadAt,
		SentAt:    n.SentAt,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}
}

func toPreferenceResponse(p *model.NotificationPreference) model.PreferenceResponse {
	return model.PreferenceResponse{
		UserID:    p.UserID,
		Channel:   p.Channel,
		Enabled:   p.Enabled,
		Types:     p.Types,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
