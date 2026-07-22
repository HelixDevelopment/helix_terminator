package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
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
	slackSender   *delivery.SlackSender
}

// New returns a new Handler with dependencies. Delivery clients are built
// from environment configuration (Constitution §11.4.10 — never hardcoded):
// email requires SMTP_HOST to be set (see delivery.SMTPConfigFromEnv); the
// webhook sender needs no external configuration; push requires
// FCM_SERVICE_ACCOUNT_JSON to be set (see delivery.PushConfigFromEnv +
// scripts/firebase/firebase_setup.sh); slack requires HERALD_SLACK_BOT_TOKEN
// to be set (see delivery.SlackConfigFromEnv + internal/delivery/slack.go)
// — until then each honestly reports "not configured" rather than
// fabricating delivery.
func New(repo *repository.Repository) *Handler {
	h := &Handler{
		repo:          repo,
		webhookSender: delivery.NewWebhookSender(10 * time.Second),
		pushSender:    delivery.NewPushSender(),
		slackSender:   delivery.NewSlackSender(),
	}
	if cfg, ok := delivery.SMTPConfigFromEnv(); ok {
		h.emailSender = delivery.NewEmailSender(cfg)
	}
	// FCM_SERVICE_ACCOUNT_JSON unset => ok is false, h.pushSender stays the
	// honest NewPushSender() zero value above. Set-but-broken (ok true,
	// err non-nil) is deliberately NOT collapsed into "not configured" —
	// Constitution §11.4 anti-bluff: a real operator misconfiguration must
	// surface (here, a startup log line naming the broken path) rather
	// than be silently swallowed into the same status a genuinely
	// unconfigured deployment reports.
	if sender, ok, err := delivery.NewPushSenderFromEnv(); ok {
		if err != nil {
			log.Printf("[notify] push (FCM) configuration error, falling back to honest not-configured state: %v", err)
		} else {
			h.pushSender = sender
		}
	}
	// HERALD_SLACK_BOT_TOKEN unset => ok is false, h.slackSender stays the
	// honest NewSlackSender() zero value above. Set-but-broken mirrors the
	// push case exactly — see internal/delivery/slack.go's package doc
	// comment for the specific "built without -tags heraldslack" instance
	// of this branch that fires in every default build today.
	if sender, ok, err := delivery.NewSlackSenderFromEnv(); ok {
		if err != nil {
			log.Printf("[notify] slack (Herald) configuration error, falling back to honest not-configured state: %v", err)
		} else {
			h.slackSender = sender
		}
	}
	return h
}

// NewWithDelivery returns a new Handler with explicitly supplied delivery
// clients. This is the constructor tests use to point real senders at real
// test infrastructure (a real SMTP sink, a real HTTP receiver) — per
// Constitution §11.4.27 no fakes/mocks are used beyond unit tests, so tests
// exercising this handler wire REAL delivery.EmailSender / WebhookSender /
// PushSender / SlackSender instances, never a mock double.
func NewWithDelivery(repo *repository.Repository, emailSender *delivery.EmailSender, webhookSender *delivery.WebhookSender, pushSender *delivery.PushSender, slackSender *delivery.SlackSender) *Handler {
	return &Handler{repo: repo, emailSender: emailSender, webhookSender: webhookSender, pushSender: pushSender, slackSender: slackSender}
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
	case "slack":
		if req.Target == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target (Slack channel ID) is required for channel=slack"})
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
	case "slack":
		h.deliverSlack(c.Request.Context(), notification)
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

// deliverPush attempts REAL FCM HTTP v1 delivery to n.Target (the device
// registration token — see model.Notification.Target) when a credentialed
// delivery.PushSender is wired (FCM_SERVICE_ACCOUNT_JSON set, see
// scripts/firebase/firebase_setup.sh). It NEVER fabricates a
// "sent"/"delivered" status: an unconfigured sender yields the honest
// "pending_provider_unconfigured" status (distinct from a real send
// failure, e.g. an invalid/expired device token or a network error, which
// yields "failed") — never conflating the two per Constitution §11.4.
func (h *Handler) deliverPush(ctx context.Context, n *model.Notification) {
	if h.pushSender == nil {
		n.Status = "pending_provider_unconfigured"
		return
	}
	msg := delivery.PushMessage{
		Title: n.Title,
		Body:  n.Message,
		Data:  notificationDataToPushData(n.Data),
	}
	if err := h.pushSender.Send(ctx, n.Target, msg); err != nil {
		if errors.Is(err, delivery.ErrPushProviderNotConfigured) {
			n.Status = "pending_provider_unconfigured"
		} else {
			n.Status = "failed"
		}
		return
	}
	sentAt := time.Now().UTC()
	n.Status = "sent"
	n.SentAt = &sentAt
}

// deliverSlack attempts REAL Slack delivery (via Herald's Slack channel
// adapter, see internal/delivery/slack.go) to n.Target (the destination
// Slack channel ID — see model.Notification.Target) when a credentialed
// delivery.SlackSender is wired (HERALD_SLACK_BOT_TOKEN set). It NEVER
// fabricates a "sent" status: an unconfigured sender yields the honest
// "pending_provider_unconfigured" status (distinct from a real send
// failure, which yields "failed") — never conflating the two, exactly
// mirroring deliverPush above. The message text sent to Slack combines
// n.Title and n.Message (Slack has no separate title/body fields, unlike
// FCM's notification payload).
func (h *Handler) deliverSlack(ctx context.Context, n *model.Notification) {
	if h.slackSender == nil {
		n.Status = "pending_provider_unconfigured"
		return
	}
	text := n.Title
	if n.Message != "" {
		if text != "" {
			text += "\n"
		}
		text += n.Message
	}
	if err := h.slackSender.Send(ctx, n.Target, text); err != nil {
		if errors.Is(err, delivery.ErrSlackProviderNotConfigured) {
			n.Status = "pending_provider_unconfigured"
		} else {
			n.Status = "failed"
		}
		return
	}
	sentAt := time.Now().UTC()
	// Herald's Slack adapter evidence ceiling is DeliveryRouted ("platform
	// stored & broadcast", i.e. Slack accepted + routed the message) — the
	// SAME evidence class webhook.go's 2xx response represents, so "sent"
	// (not "delivered", which this service reserves for webhook's
	// stronger receiver-side 2xx confirmation) is the honest status here,
	// matching push's own DeliveryRouted-equivalent "sent" mapping.
	n.Status = "sent"
	n.SentAt = &sentAt
}

// notificationDataToPushData converts a Notification's free-form JSON Data
// payload into FCM's required map[string]string data-field shape. Non-
// string values are re-marshalled to their JSON text form (FCM's data
// payload only carries strings — this mirrors what the Firebase Admin SDKs
// themselves do for non-string values). Malformed or empty raw JSON yields
// nil (push still proceeds with title/body only) rather than blocking
// delivery on a data-field concern.
func notificationDataToPushData(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var generic map[string]interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil
	}
	out := make(map[string]string, len(generic))
	for k, v := range generic {
		if s, ok := v.(string); ok {
			out[k] = s
			continue
		}
		if b, err := json.Marshal(v); err == nil {
			out[k] = string(b)
		}
	}
	return out
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
		Target:    n.Target,
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
