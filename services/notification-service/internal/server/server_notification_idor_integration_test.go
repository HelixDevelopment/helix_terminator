//go:build integration

// Package server_test — REAL cross-user isolation proof against a real
// PostgreSQL instance and the REAL notification-service HTTP server (T18,
// §11.4.27 / §11.4.43 / §11.4.107 / §11.4.115 / §11.4.146). Excluded from
// the default `go test ./...` run (build tag `integration`). Requires:
//
//	export DATABASE_URL="postgres://postgres:postgres@127.0.0.1:15437/notification_service_t18_test?sslmode=disable"
//	GOWORK=off GOMAXPROCS=2 go test -tags integration -p 2 ./internal/server/...
//
// Forensic anchor (T18): CreateNotification, ListNotifications,
// MarkAllRead, CountUnread, GetPreference, and UpdatePreference all
// derived the target user from a client-supplied "userId" body field or
// "user_id" query parameter, never from the caller's authenticated
// identity (set into the gin context by authMiddleware, T11). Any
// authenticated caller could therefore read/create/modify another user's
// notifications and preferences simply by naming a different user_id.
// GetNotification, MarkRead, and DeleteNotification had NO ownership
// check at all — any authenticated caller could read/mutate/delete ANY
// notification by id. This file drives the REAL HTTP server
// (server.Router()) as two distinct, real users (A and B), each with
// their own genuinely-signed Ed25519 JWT, against a real Postgres
// database, and asserts user B can NEVER read, create-as, or mutate user
// A's notifications/preferences — regardless of what user_id B's
// requests claim. Run against the pre-fix handler this test's cross-user
// assertions FAIL (RED) — see the T18 agent report for the captured
// pre-fix run. Run against the fixed handler it PASSES (GREEN): every
// notification-service handler now derives "whose data" exclusively from
// the validated JWT context claim, never from client input, and
// cross-user object-id access returns the same 404 a genuinely-missing
// id would (no existence oracle), mirroring billing-service's T12/T14.
package server_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/server"
)

// t18Claims mirrors gateway-service's Claims struct (services/
// gateway-service/internal/server/server.go) — the gateway forwards the
// original signed Authorization bearer token to notification-service
// untouched, so notification-service independently validates the SAME
// token shape with the SAME claim names (T11).
type t18Claims struct {
	UserID string `json:"userId"`
	OrgID  string `json:"orgId,omitempty"`
	jwt.RegisteredClaims
}

// t18MustConnectAndReset connects to the real Postgres pointed at by
// DATABASE_URL and resets it to a clean public+notification_service
// schema state so this test is re-runnable endlessly (§11.4.98) without
// manual intervention. Skips (does not fail) when DATABASE_URL is unset
// — the correct §11.4.3 topology-appropriate behaviour for an
// integration test with no real target.
func t18MustConnectAndReset(t *testing.T) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping real-Postgres T18 IDOR integration test (§11.4.3)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to open pgxpool against DATABASE_URL")
	require.NoError(t, pool.Ping(ctx), "real Postgres at DATABASE_URL is not reachable")

	// server.New() applies the service's own embedded migrations
	// (migrations.Run) into the notification_service schema on every
	// call, which is idempotent — but a stale row set from a PRIOR test
	// run in the same disposable database would contaminate this run's
	// cross-user assertions, so truncate first.
	_, _ = pool.Exec(ctx, `TRUNCATE TABLE notification_service.notifications, notification_service.notification_preferences`)
	pool.Close()
}

// t18MustNewServer generates a real Ed25519 keypair, points
// notification-service's JWT_PUBLIC_KEY env var at the public half
// (mirroring how gateway-service/billing-service are provisioned) and
// DATABASE_URL at the real disposable Postgres, builds the real server,
// and returns a signer bound to the private half so the test can mint
// tokens exactly as auth-service would.
func t18MustNewServer(t *testing.T) (*server.Server, func(userID, orgID string) string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	prevKey, hadPrevKey := os.LookupEnv("JWT_PUBLIC_KEY")
	require.NoError(t, os.Setenv("JWT_PUBLIC_KEY", base64.StdEncoding.EncodeToString(pub)))
	t.Cleanup(func() {
		if hadPrevKey {
			os.Setenv("JWT_PUBLIC_KEY", prevKey)
		} else {
			os.Unsetenv("JWT_PUBLIC_KEY")
		}
	})

	srv, err := server.New(nil)
	require.NoError(t, err)

	sign := func(userID, orgID string) string {
		claims := t18Claims{
			UserID: userID,
			OrgID:  orgID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		signed, err := tok.SignedString(priv)
		require.NoError(t, err)
		return signed
	}
	return srv, sign
}

func t18CreateNotification(t *testing.T, r http.Handler, token string) map[string]interface{} {
	t.Helper()
	payload := map[string]interface{}{
		"type":    "info",
		"title":   "T18 fixture",
		"message": "cross-user isolation fixture " + uuid.New().String(),
		"channel": "in_app",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "fixture create failed, body: %s", w.Body.String())

	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	return created
}

// TestT18IDOR_CreateNotification_AttributedToCallerNotBody proves
// CreateNotification attributes the created notification to the CALLER
// (from the validated JWT), never to a client-supplied "userId" body
// field. Pre-fix: the notification's userId in the response equals the
// attacker-chosen otherUser value from the body. Post-fix: it always
// equals the caller's own JWT-derived identity.
func TestT18IDOR_CreateNotification_AttributedToCallerNotBody(t *testing.T) {
	t18MustConnectAndReset(t)
	srv, sign := t18MustNewServer(t)

	callerID := uuid.New().String()
	attackerChosenOther := uuid.New().String()
	token := sign(callerID, "")

	payload := map[string]interface{}{
		"userId":  attackerChosenOther,
		"type":    "info",
		"title":   "T18 spoof attempt",
		"message": "must be attributed to the caller, not the body field",
		"channel": "in_app",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())
	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	assert.Equal(t, callerID, created["userId"],
		"the created notification MUST be attributed to the caller's own JWT-derived identity, never the body's client-supplied userId")
	assert.NotEqual(t, attackerChosenOther, created["userId"],
		"the created notification MUST NOT be attributed to an attacker-chosen body userId")
}

// TestT18IDOR_ListNotifications_CannotReadAnotherUsersNotifications is
// the main-read RED→GREEN proof. User A creates a notification. User B
// (a completely different, real, validly-authenticated caller) then
// lists notifications while ALSO supplying a client-side "user_id" query
// parameter equal to A's id — the pre-fix scoping mechanism. Pre-fix:
// user B's request returns user A's notification (cross-user data leak).
// Post-fix: user B's request is scoped exclusively to user B's own
// (empty) notification set — A's notification never appears, regardless
// of the query parameter.
func TestT18IDOR_ListNotifications_CannotReadAnotherUsersNotifications(t *testing.T) {
	t18MustConnectAndReset(t)
	srv, sign := t18MustNewServer(t)

	userA := uuid.New().String()
	userB := uuid.New().String()
	tokenA := sign(userA, "")
	tokenB := sign(userB, "")

	created := t18CreateNotification(t, srv.Router(), tokenA)
	notificationAID := created["id"].(string)

	// User B attempts to read user A's notifications by supplying A's id
	// as the client-side user_id query parameter — the exact pre-fix
	// IDOR vector.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications?user_id="+userA, nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	respBody := w.Body.String()
	assert.NotContains(t, respBody, notificationAID,
		"user B must NEVER see user A's notification id, regardless of a client-supplied user_id query parameter")
	assert.Equal(t, float64(0), resp["total"],
		"user B has created nothing, so their scoped total must be 0 even though they asked for user A's data")

	// Sanity: user A CAN see their own notification via the same endpoint
	// (proves the fix scopes correctly, not that it locks everyone out).
	wA := httptest.NewRecorder()
	reqA, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	reqA.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wA, reqA)
	require.Equal(t, http.StatusOK, wA.Code)
	assert.Contains(t, wA.Body.String(), notificationAID, "user A must still see their own notification")
}

// TestT18IDOR_GetNotification_NoExistenceOracle proves GetNotification
// (an ID-based endpoint with NO client-supplied identity input at all)
// enforces ownership: user B fetching user A's notification by its real
// id gets the SAME 404 a genuinely-missing id would — never the
// notification's content, and never a distinguishable status that would
// let B confirm A's id is valid (mirrors billing-service's T12
// no-existence-oracle pattern).
func TestT18IDOR_GetNotification_NoExistenceOracle(t *testing.T) {
	t18MustConnectAndReset(t)
	srv, sign := t18MustNewServer(t)

	userA := uuid.New().String()
	userB := uuid.New().String()
	tokenA := sign(userA, "")
	tokenB := sign(userB, "")

	created := t18CreateNotification(t, srv.Router(), tokenA)
	notificationAID := created["id"].(string)

	// User B fetches A's real notification id.
	wCross := httptest.NewRecorder()
	reqCross, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/"+notificationAID, nil)
	reqCross.Header.Set("Authorization", "Bearer "+tokenB)
	srv.Router().ServeHTTP(wCross, reqCross)

	// User B fetches a genuinely-nonexistent id.
	wMissing := httptest.NewRecorder()
	reqMissing, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/"+uuid.New().String(), nil)
	reqMissing.Header.Set("Authorization", "Bearer "+tokenB)
	srv.Router().ServeHTTP(wMissing, reqMissing)

	assert.Equal(t, http.StatusNotFound, wCross.Code,
		"user B fetching user A's notification by id must get 404, never the content; body: %s", wCross.Body.String())
	assert.NotContains(t, wCross.Body.String(), "T18 fixture", "user B must never see user A's notification content")
	assert.Equal(t, wMissing.Code, wCross.Code, "a cross-user id and a genuinely-missing id must be indistinguishable (no existence oracle)")
	assert.Equal(t, wMissing.Body.String(), wCross.Body.String(), "a cross-user id and a genuinely-missing id must return the identical body (no existence oracle)")

	// Sanity: user A CAN fetch their own notification.
	wOwn := httptest.NewRecorder()
	reqOwn, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/"+notificationAID, nil)
	reqOwn.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wOwn, reqOwn)
	assert.Equal(t, http.StatusOK, wOwn.Code, "user A must still be able to fetch their own notification")
}

// TestT18IDOR_MarkRead_CannotMutateAnotherUsersNotification proves the
// write-path analogue of the GetNotification proof above: user B cannot
// mark user A's notification as read by id.
func TestT18IDOR_MarkRead_CannotMutateAnotherUsersNotification(t *testing.T) {
	t18MustConnectAndReset(t)
	srv, sign := t18MustNewServer(t)

	userA := uuid.New().String()
	userB := uuid.New().String()
	tokenA := sign(userA, "")
	tokenB := sign(userB, "")

	created := t18CreateNotification(t, srv.Router(), tokenA)
	notificationAID := created["id"].(string)

	// User B attempts to mark A's notification as read.
	wCross := httptest.NewRecorder()
	reqCross, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/"+notificationAID+"/read", nil)
	reqCross.Header.Set("Authorization", "Bearer "+tokenB)
	srv.Router().ServeHTTP(wCross, reqCross)

	assert.Equal(t, http.StatusNotFound, wCross.Code,
		"user B must NOT be able to mark user A's notification as read; body: %s", wCross.Body.String())

	// Confirm A's notification was NOT actually marked read by B's attempt.
	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/"+notificationAID, nil)
	reqGet.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wGet, reqGet)
	require.Equal(t, http.StatusOK, wGet.Code)
	var fetched map[string]interface{}
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &fetched))
	assert.Nil(t, fetched["readAt"], "user B's blocked cross-user MarkRead attempt must NOT have flipped read_at")

	// Sanity: user A CAN mark their own notification as read.
	wOwn := httptest.NewRecorder()
	reqOwn, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/"+notificationAID+"/read", nil)
	reqOwn.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wOwn, reqOwn)
	assert.Equal(t, http.StatusOK, wOwn.Code, "user A must still be able to mark their own notification as read")
}

// TestT18IDOR_DeleteNotification_CannotDeleteAnotherUsersNotification
// proves the destructive-write analogue: user B cannot delete user A's
// notification by id.
func TestT18IDOR_DeleteNotification_CannotDeleteAnotherUsersNotification(t *testing.T) {
	t18MustConnectAndReset(t)
	srv, sign := t18MustNewServer(t)

	userA := uuid.New().String()
	userB := uuid.New().String()
	tokenA := sign(userA, "")
	tokenB := sign(userB, "")

	created := t18CreateNotification(t, srv.Router(), tokenA)
	notificationAID := created["id"].(string)

	wCross := httptest.NewRecorder()
	reqCross, _ := http.NewRequest(http.MethodDelete, "/api/v1/notifications/"+notificationAID, nil)
	reqCross.Header.Set("Authorization", "Bearer "+tokenB)
	srv.Router().ServeHTTP(wCross, reqCross)
	assert.Equal(t, http.StatusNotFound, wCross.Code,
		"user B must NOT be able to delete user A's notification; body: %s", wCross.Body.String())

	// Confirm A's notification STILL exists (was not actually deleted).
	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/"+notificationAID, nil)
	reqGet.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wGet, reqGet)
	assert.Equal(t, http.StatusOK, wGet.Code, "user A's notification must still exist after user B's blocked delete attempt")
}

// TestT18IDOR_MarkAllRead_CannotMutateAnotherUsersNotifications proves
// the bulk-write analogue of the ListNotifications proof: user B cannot
// mark ALL of user A's notifications read by supplying A's id as the
// client-side user_id query parameter.
func TestT18IDOR_MarkAllRead_CannotMutateAnotherUsersNotifications(t *testing.T) {
	t18MustConnectAndReset(t)
	srv, sign := t18MustNewServer(t)

	userA := uuid.New().String()
	userB := uuid.New().String()
	tokenA := sign(userA, "")
	tokenB := sign(userB, "")

	created := t18CreateNotification(t, srv.Router(), tokenA)
	notificationAID := created["id"].(string)

	// User B attempts a bulk mark-all-read targeting user A via the
	// pre-fix client-side user_id query parameter.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/read-all?user_id="+userA, nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	// Confirm A's notification is STILL unread — B's request (even
	// though it "succeeded" against B's own, empty notification set)
	// must not have touched A's data.
	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/"+notificationAID, nil)
	reqGet.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wGet, reqGet)
	require.Equal(t, http.StatusOK, wGet.Code)
	var fetched map[string]interface{}
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &fetched))
	assert.Nil(t, fetched["readAt"], "user B's mark-all-read (targeting user A's id via query param) must NOT have marked user A's notification read")
}

// TestT18IDOR_CountUnread_CannotReadAnotherUsersCount proves the
// aggregate-read analogue: user B cannot learn user A's unread count by
// supplying A's id as the client-side user_id query parameter.
func TestT18IDOR_CountUnread_CannotReadAnotherUsersCount(t *testing.T) {
	t18MustConnectAndReset(t)
	srv, sign := t18MustNewServer(t)

	userA := uuid.New().String()
	userB := uuid.New().String()
	tokenA := sign(userA, "")
	tokenB := sign(userB, "")

	t18CreateNotification(t, srv.Router(), tokenA) // A now has 1 unread

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count?user_id="+userA, nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	srv.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["count"],
		"user B must see their OWN (zero) unread count, never user A's, regardless of the user_id query parameter")

	wA := httptest.NewRecorder()
	reqA, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	reqA.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wA, reqA)
	require.Equal(t, http.StatusOK, wA.Code)
	var respA map[string]interface{}
	require.NoError(t, json.Unmarshal(wA.Body.Bytes(), &respA))
	assert.Equal(t, float64(1), respA["count"], "user A must still correctly see their own unread count")
}

// TestT18IDOR_UpdatePreference_CannotWriteAnotherUsersPreference is the
// preference-handler write proof required by the task: user B cannot
// overwrite user A's notification preference by supplying A's id as the
// (now-removed) "userId" body field.
func TestT18IDOR_UpdatePreference_CannotWriteAnotherUsersPreference(t *testing.T) {
	t18MustConnectAndReset(t)
	srv, sign := t18MustNewServer(t)

	userA := uuid.New().String()
	userB := uuid.New().String()
	tokenA := sign(userA, "")
	tokenB := sign(userB, "")

	// User A sets their own preference (email disabled). Types is an
	// explicit empty array (not omitted) — the notification_preferences
	// table's "types" column is NOT NULL and a Go nil slice marshals to
	// SQL NULL, a pre-existing, IDOR-unrelated constraint this fixture
	// must respect to exercise the write path at all.
	payloadA := map[string]interface{}{
		"channel": "email",
		"enabled": false,
		"types":   []string{},
	}
	bodyA, _ := json.Marshal(payloadA)
	wA := httptest.NewRecorder()
	reqA, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(bodyA))
	reqA.Header.Set("Content-Type", "application/json")
	reqA.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wA, reqA)
	require.Equal(t, http.StatusOK, wA.Code, "body: %s", wA.Body.String())

	// User B attempts to overwrite A's preference by naming A's id in
	// the (removed) "userId" body field, flipping enabled=true.
	payloadAttack := map[string]interface{}{
		"userId":  userA,
		"channel": "email",
		"enabled": true,
		"types":   []string{},
	}
	bodyAttack, _ := json.Marshal(payloadAttack)
	wAttack := httptest.NewRecorder()
	reqAttack, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(bodyAttack))
	reqAttack.Header.Set("Content-Type", "application/json")
	reqAttack.Header.Set("Authorization", "Bearer "+tokenB)
	srv.Router().ServeHTTP(wAttack, reqAttack)
	require.Equal(t, http.StatusOK, wAttack.Code, "body: %s", wAttack.Body.String())

	var attackResp map[string]interface{}
	require.NoError(t, json.Unmarshal(wAttack.Body.Bytes(), &attackResp))
	assert.Equal(t, userB, attackResp["userId"], "the write must be attributed to caller B, never to the body's userId field")

	// Confirm A's own preference is UNCHANGED (still disabled).
	wGetA := httptest.NewRecorder()
	reqGetA, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/preferences?channel=email", nil)
	reqGetA.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wGetA, reqGetA)
	require.Equal(t, http.StatusOK, wGetA.Code, "body: %s", wGetA.Body.String())
	var prefA map[string]interface{}
	require.NoError(t, json.Unmarshal(wGetA.Body.Bytes(), &prefA))
	assert.Equal(t, false, prefA["enabled"], "user A's preference must remain unchanged after user B's cross-user write attempt")
}

// TestT18IDOR_GetPreference_CannotReadAnotherUsersPreference proves the
// preference-handler read proof: user B cannot read user A's preference
// by supplying A's id as the client-side user_id query parameter.
func TestT18IDOR_GetPreference_CannotReadAnotherUsersPreference(t *testing.T) {
	t18MustConnectAndReset(t)
	srv, sign := t18MustNewServer(t)

	userA := uuid.New().String()
	userB := uuid.New().String()
	tokenA := sign(userA, "")
	tokenB := sign(userB, "")

	// User A sets a distinctive preference.
	payloadA := map[string]interface{}{
		"channel": "webhook",
		"enabled": false,
		"types":   []string{},
	}
	bodyA, _ := json.Marshal(payloadA)
	wA := httptest.NewRecorder()
	reqA, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", bytes.NewReader(bodyA))
	reqA.Header.Set("Content-Type", "application/json")
	reqA.Header.Set("Authorization", "Bearer "+tokenA)
	srv.Router().ServeHTTP(wA, reqA)
	require.Equal(t, http.StatusOK, wA.Code)

	// User B attempts to read A's preference via the client-side user_id
	// query parameter.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/preferences?channel=webhook&user_id="+userA, nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	srv.Router().ServeHTTP(w, req)

	// User B has never set a webhook preference, so their OWN scoped
	// lookup must be "preference not found" — never A's real (disabled)
	// preference.
	assert.Equal(t, http.StatusNotFound, w.Code,
		"user B must see their OWN (nonexistent) preference, never user A's, regardless of the user_id query parameter; body: %s", w.Body.String())
}
