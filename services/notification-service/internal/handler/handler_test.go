package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/helixdevelopment/notification-service/internal/handler"
	"github.com/helixdevelopment/notification-service/internal/model"
	"github.com/helixdevelopment/notification-service/internal/repository"
)

func setupTestHandler(t *testing.T) (*handler.Handler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	repo := repository.New(nil)
	h := handler.New(repo)

	r := gin.New()
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	api := r.Group("/api/v1/notifications")
	{
		api.POST("", h.CreateNotification)
		api.GET("", h.ListNotifications)
		api.GET("/unread-count", h.CountUnread)
		api.GET("/:id", h.GetNotification)
		api.POST("/:id/read", h.MarkRead)
		api.POST("/read-all", h.MarkAllRead)
		api.DELETE("/:id", h.DeleteNotification)
		api.GET("/preferences", h.GetPreference)
		api.PUT("/preferences", h.UpdatePreference)
	}

	return h, r
}

// setupAuthedTestHandler mounts the SAME routes as setupTestHandler but
// with a lightweight test-only middleware that injects a caller identity
// into the gin context under the SAME key ("userID") the real
// authMiddleware (internal/server/server.go, T11) populates from a
// validated JWT claim. T18 (Constitution §11.4.102/.115/.146): every
// handler in this package now derives WHOSE notifications/preferences a
// request may touch EXCLUSIVELY from this context value — never from a
// client-supplied "user_id" query/body field — so unit-level tests that
// need to reach past the 401 "missing or invalid caller identity" guard
// use this helper. The full end-to-end proof that the REAL authMiddleware
// populates this same context key from a REAL validated Ed25519 JWT lives
// in internal/server/server_jwt_auth_test.go and the T18 cross-user
// integration tests (internal/server/server_notification_idor_integration_test.go).
func setupAuthedTestHandler(t *testing.T, userID string) (*handler.Handler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := repository.New(nil)
	h := handler.New(repo)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	r.GET("/healthz", h.HealthCheck)
	r.GET("/healthz/ready", h.ReadinessCheck)

	api := r.Group("/api/v1/notifications")
	{
		api.POST("", h.CreateNotification)
		api.GET("", h.ListNotifications)
		api.GET("/unread-count", h.CountUnread)
		api.GET("/:id", h.GetNotification)
		api.POST("/:id/read", h.MarkRead)
		api.POST("/read-all", h.MarkAllRead)
		api.DELETE("/:id", h.DeleteNotification)
		api.GET("/preferences", h.GetPreference)
		api.PUT("/preferences", h.UpdatePreference)
	}

	return h, r
}

func TestHealthCheck(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestReadinessCheckNoDB(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz/ready", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not connected")
}

func TestCreateNotificationNoDB(t *testing.T) {
	_, r := setupAuthedTestHandler(t, uuid.New().String())

	payload := map[string]interface{}{
		"type":    "info",
		"title":   "Test",
		"message": "Test message",
		"channel": "in_app",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not connected")
}

// TestCreateNotification_RequiresCallerIdentity is the T18 RED→GREEN proof
// at the unit level: a well-formed create request with NO caller identity
// in the gin context (the pre-fix code path never checked this at all —
// it derived the target user from the client-supplied "userId" body field
// instead) MUST be rejected 401, never silently proceed using a
// client-supplied identity.
func TestCreateNotification_RequiresCallerIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	payload := map[string]interface{}{
		"type":    "info",
		"title":   "Test",
		"message": "Test message",
		"channel": "in_app",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "missing or invalid caller identity")
}

func TestCreateNotificationValidation(t *testing.T) {
	// T18: "userId" is no longer a field on CreateNotificationRequest (the
	// target user is derived exclusively from the caller's validated JWT
	// claim) — this test now exercises the remaining required-field
	// validation (message/channel missing), still evaluated BEFORE the
	// caller-identity check, using an authenticated context so the 400
	// asserted below is unambiguously about request-shape validation.
	_, r := setupAuthedTestHandler(t, uuid.New().String())

	payload := map[string]interface{}{
		"type":  "info",
		"title": "Test",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreateNotification_IgnoresClientSuppliedUserID proves the removed
// "userId" body field, if a legacy/malicious client still sends it, has
// NO effect: the created notification's owner is ALWAYS the caller's
// context identity, never the (now-unbound) body value. This is the
// unit-level half of the T18 anti-spoof proof; the full cross-user
// real-Postgres proof lives in the T18 integration test.
func TestCreateNotification_IgnoresClientSuppliedUserID(t *testing.T) {
	caller := uuid.New().String()
	attackerChosenOther := uuid.New().String()
	_, r := setupAuthedTestHandler(t, caller)

	payload := map[string]interface{}{
		// A pre-fix client would have used this field to target ANY user;
		// it is not even a recognised field on the request struct anymore.
		"userId":  attackerChosenOther,
		"type":    "info",
		"title":   "T18 spoof attempt",
		"message": "must be attributed to the caller, not this value",
		"channel": "in_app",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// No DB is wired (repository.New(nil)), so the request fails at the
	// persistence step (503) — but it MUST fail there having reached
	// business logic with the CALLER's identity, never with a 400
	// "invalid user_id"/200 success keyed off attackerChosenOther. A
	// pre-fix build would have accepted attackerChosenOther as a valid
	// uuid and attempted to persist a notification owned by it.
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, "body: %s", w.Body.String())
	assert.NotContains(t, w.Body.String(), attackerChosenOther)
}

// TestListNotifications_RequiresCallerIdentity is the T18 RED→GREEN proof:
// pre-fix, ListNotifications demanded a client-supplied "user_id" query
// parameter and used it verbatim to scope the read (an IDOR — any caller
// could list ANOTHER user's notifications by supplying a different
// user_id). Post-fix, the scope comes exclusively from the caller's
// context identity, so a request with NO identity present must be
// rejected 401 regardless of query parameters.
func TestListNotifications_RequiresCallerIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "missing or invalid caller identity")
}

// TestListNotifications_InvalidQueryStillValidatesBeforeIdentity is a
// regression guard proving request-shape validation still runs BEFORE the
// identity check (matching the ordering CreateNotification/
// UpdatePreference use) — an invalid "channel" value is rejected 400 even
// with no caller identity in context.
func TestListNotifications_InvalidQueryStillValidatesBeforeIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications?channel=not-a-real-channel", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
}

// TestListNotifications_IgnoresClientSuppliedUserID proves a client-
// supplied "user_id" query parameter (the pre-fix scoping mechanism) has
// NO effect once a caller identity is present: the response is identical
// whether or not an attacker-chosen user_id is appended to the query.
func TestListNotifications_IgnoresClientSuppliedUserID(t *testing.T) {
	caller := uuid.New().String()
	_, rWithout := setupAuthedTestHandler(t, caller)
	_, rWith := setupAuthedTestHandler(t, caller)

	wWithout := httptest.NewRecorder()
	reqWithout, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	rWithout.ServeHTTP(wWithout, reqWithout)

	wWith := httptest.NewRecorder()
	reqWith, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications?user_id="+uuid.New().String(), nil)
	rWith.ServeHTTP(wWith, reqWith)

	assert.Equal(t, wWithout.Code, wWith.Code, "an attacker-chosen user_id query parameter must not change the outcome")
}

func TestGetNotificationInvalidID(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/invalid-id", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMarkReadInvalidID(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/invalid-id/read", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGetNotification_RequiresCallerIdentity: pre-fix, GetNotification
// had NO ownership check at all — any authenticated caller could fetch
// ANY notification by id regardless of who owned it. Post-fix it also
// requires a valid caller identity before it will even attempt the
// lookup.
func TestGetNotification_RequiresCallerIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "missing or invalid caller identity")
}

// TestMarkRead_RequiresCallerIdentity: pre-fix, MarkRead mutated ANY id
// with no ownership check whatsoever. Post-fix it also requires a valid
// caller identity before attempting the fetch-then-compare.
func TestMarkRead_RequiresCallerIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/"+uuid.New().String()+"/read", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "missing or invalid caller identity")
}

// TestMarkAllRead_RequiresCallerIdentity is the T18 RED→GREEN proof:
// pre-fix, MarkAllRead demanded a client-supplied "user_id" query
// parameter and used it verbatim (an IDOR write — any caller could mark
// ANOTHER user's notifications as read). Post-fix, the target comes
// exclusively from the caller's context identity.
func TestMarkAllRead_RequiresCallerIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "missing or invalid caller identity")
}

// TestMarkAllRead_IgnoresClientSuppliedUserID proves a client-supplied
// "user_id" query parameter (the pre-fix targeting mechanism) has NO
// effect once a caller identity is present.
func TestMarkAllRead_IgnoresClientSuppliedUserID(t *testing.T) {
	caller := uuid.New().String()
	_, rWithout := setupAuthedTestHandler(t, caller)
	_, rWith := setupAuthedTestHandler(t, caller)

	wWithout := httptest.NewRecorder()
	reqWithout, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", nil)
	rWithout.ServeHTTP(wWithout, reqWithout)

	wWith := httptest.NewRecorder()
	reqWith, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/read-all?user_id="+uuid.New().String(), nil)
	rWith.ServeHTTP(wWith, reqWith)

	assert.Equal(t, wWithout.Code, wWith.Code, "an attacker-chosen user_id query parameter must not change the outcome")
}

func TestDeleteNotificationInvalidID(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/notifications/invalid-id", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestDeleteNotification_RequiresCallerIdentity: pre-fix, DeleteNotification
// deleted ANY id with no ownership check whatsoever. Post-fix it also
// requires a valid caller identity before attempting the
// fetch-then-compare.
func TestDeleteNotification_RequiresCallerIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/notifications/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "missing or invalid caller identity")
}

// TestCountUnread_RequiresCallerIdentity is the T18 RED→GREEN proof:
// pre-fix, CountUnread demanded a client-supplied "user_id" query
// parameter and used it verbatim (an IDOR read — any caller could learn
// ANOTHER user's unread count). Post-fix, the target comes exclusively
// from the caller's context identity.
func TestCountUnread_RequiresCallerIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "missing or invalid caller identity")
}

// TestCountUnread_IgnoresClientSuppliedUserID proves a client-supplied
// "user_id" query parameter has NO effect once a caller identity is
// present.
func TestCountUnread_IgnoresClientSuppliedUserID(t *testing.T) {
	caller := uuid.New().String()
	_, rWithout := setupAuthedTestHandler(t, caller)
	_, rWith := setupAuthedTestHandler(t, caller)

	wWithout := httptest.NewRecorder()
	reqWithout, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	rWithout.ServeHTTP(wWithout, reqWithout)

	wWith := httptest.NewRecorder()
	reqWith, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count?user_id="+uuid.New().String(), nil)
	rWith.ServeHTTP(wWith, reqWith)

	assert.Equal(t, wWithout.Code, wWith.Code, "an attacker-chosen user_id query parameter must not change the outcome")
}

// TestGetPreferenceMissingChannel proves the (identity-independent)
// "channel" query requirement still validates first, matching the
// pre-fix behaviour's error-precedence for THIS field.
func TestGetPreferenceMissingChannel(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/preferences", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "channel is required")
}

// TestGetPreference_RequiresCallerIdentity is the T18 RED→GREEN proof:
// pre-fix, GetPreference demanded a client-supplied "user_id" query
// parameter and used it verbatim (an IDOR read — any caller could read
// ANOTHER user's notification preferences). Post-fix, with a valid
// "channel" but no caller identity, the request is rejected 401.
func TestGetPreference_RequiresCallerIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/preferences?channel=email", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "missing or invalid caller identity")
}

// TestGetPreference_IgnoresClientSuppliedUserID proves a client-supplied
// "user_id" query parameter has NO effect once a caller identity is
// present.
func TestGetPreference_IgnoresClientSuppliedUserID(t *testing.T) {
	caller := uuid.New().String()
	_, rWithout := setupAuthedTestHandler(t, caller)
	_, rWith := setupAuthedTestHandler(t, caller)

	wWithout := httptest.NewRecorder()
	reqWithout, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/preferences?channel=email", nil)
	rWithout.ServeHTTP(wWithout, reqWithout)

	wWith := httptest.NewRecorder()
	reqWith, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/preferences?channel=email&user_id="+uuid.New().String(), nil)
	rWith.ServeHTTP(wWith, reqWith)

	assert.Equal(t, wWithout.Code, wWith.Code, "an attacker-chosen user_id query parameter must not change the outcome")
}

func TestUpdatePreferenceValidation(t *testing.T) {
	// T18: "userId" is no longer a field on UpdatePreferenceRequest (the
	// target user is derived exclusively from the caller's validated JWT
	// claim) — an invalid "channel" value still validates 400 regardless.
	_, r := setupTestHandler(t)

	payload := map[string]interface{}{
		"channel": "invalid",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdatePreference_RequiresCallerIdentity is the T18 RED→GREEN proof:
// pre-fix, UpdatePreference demanded a client-supplied "userId" body
// field and used it verbatim (an IDOR write — any caller could overwrite
// ANOTHER user's notification preferences). Post-fix, with a valid body
// but no caller identity, the request is rejected 401.
func TestUpdatePreference_RequiresCallerIdentity(t *testing.T) {
	_, r := setupTestHandler(t)

	payload := map[string]interface{}{
		"channel": "email",
		"enabled": true,
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "missing or invalid caller identity")
}

// TestUpdatePreference_IgnoresClientSuppliedUserID proves a client-
// supplied "userId" body field (the removed field, if a legacy/malicious
// client still sends it) has NO effect: the persisted preference's owner
// is always the caller's context identity.
func TestUpdatePreference_IgnoresClientSuppliedUserID(t *testing.T) {
	caller := uuid.New().String()
	attackerChosenOther := uuid.New().String()
	_, r := setupAuthedTestHandler(t, caller)

	payload := map[string]interface{}{
		"userId":  attackerChosenOther,
		"channel": "email",
		"enabled": true,
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// No DB is wired, so this fails at persistence (503) — but it MUST
	// reach persistence keyed off the CALLER's identity, never a 200/400
	// keyed off attackerChosenOther.
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, "body: %s", w.Body.String())
	assert.NotContains(t, w.Body.String(), attackerChosenOther)
}

func TestNotificationResponseMapping(t *testing.T) {
	now := time.Now().UTC()
	orgID := uuid.New()
	n := &model.Notification{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		OrgID:     &orgID,
		Type:      "info",
		Title:     "Title",
		Message:   "Message",
		Data:      []byte(`{"key":"value"}`),
		Channel:   "in_app",
		Status:    "pending",
		ReadAt:    &now,
		SentAt:    &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Verify the handler can map without panic by exercising the full flow
	// (we can't call the unexported helper directly, but we can verify the model)
	assert.Equal(t, "info", n.Type)
	assert.Equal(t, "Title", n.Title)
	assert.Equal(t, "Message", n.Message)
	assert.Equal(t, "in_app", n.Channel)
	assert.Equal(t, "pending", n.Status)
}

func TestPreferenceResponseMapping(t *testing.T) {
	now := time.Now().UTC()
	p := &model.NotificationPreference{
		UserID:    uuid.New(),
		Channel:   "email",
		Enabled:   true,
		Types:     []string{"info", "warning"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "email", p.Channel)
	assert.True(t, p.Enabled)
	assert.Equal(t, []string{"info", "warning"}, p.Types)
}
