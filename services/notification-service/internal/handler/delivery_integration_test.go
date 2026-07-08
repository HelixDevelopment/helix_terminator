//go:build integration

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/notification-service/internal/delivery"
	"github.com/helixdevelopment/notification-service/internal/handler"
	"github.com/helixdevelopment/notification-service/internal/repository"
)

// This file is the FULL end-to-end wiring proof: it drives the real
// POST /api/v1/notifications HTTP handler — the exact code path an end
// user's request goes through — against a real Postgres database, a real
// MailHog SMTP sink, and a real HTTP webhook receiver, and asserts the
// persisted notification.status reflects REAL delivery outcomes rather than
// the pre-existing "always pending" bluff (fleet audit finding, operator
// decision 2026-07-07).

const e2eDBURL = "postgres://postgres:postgres@127.0.0.1:15432/notification_test?sslmode=disable"

func setupE2EPostgres(t *testing.T) *repository.Repository {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), e2eDBURL)
	if err != nil {
		t.Skipf("e2e postgres not available: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("e2e postgres not reachable: %v", err)
	}
	_, _ = pool.Exec(context.Background(), "DELETE FROM notifications")
	return repository.New(pool)
}

func startMailhogForHandler(t *testing.T) (smtpPort, httpPort string, cleanup func()) {
	t.Helper()
	name := "notif-svc-handler-mailhog-" + uuid.New().String()[:8]
	smtpPort = "12626"
	httpPort = "18126"

	cmd := exec.Command("podman", "run", "-d", "--rm",
		"--name", name,
		"-p", smtpPort+":1025",
		"-p", httpPort+":8025",
		"docker.io/mailhog/mailhog",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to start mailhog: %s", string(out))
	cleanup = func() { _ = exec.Command("podman", "rm", "-f", name).Run() }

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/api/v2/messages", httpPort))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return smtpPort, httpPort, cleanup
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	cleanup()
	t.Fatal("mailhog did not become ready")
	return "", "", nil
}

// TestCreateNotification_Email_RealDeliveryEndToEnd drives the real HTTP
// handler used by end users, with a real Postgres-backed repository and a
// real SMTP sink, and confirms via MailHog's own HTTP API that the email
// actually arrived AND that the persisted notification row's status is
// "sent" (not the old permanent "pending" bluff).
func TestCreateNotification_Email_RealDeliveryEndToEnd(t *testing.T) {
	repo := setupE2EPostgres(t)
	smtpPort, httpPort, cleanup := startMailhogForHandler(t)
	defer cleanup()

	es := delivery.NewEmailSender(delivery.SMTPConfig{
		Host: "127.0.0.1",
		Port: smtpPort,
		From: "notifications@helix-terminator.test",
	})
	ws := delivery.NewWebhookSender(5 * time.Second)
	ps := delivery.NewPushSender()
	h := handler.NewWithDelivery(repo, es, ws, ps)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	// T18: CreateNotification/GetNotification now derive the caller's
	// identity exclusively from the gin context (populated by the real
	// authMiddleware in production, internal/server/server.go) rather
	// than a client-supplied "userId" body field, so this direct
	// handler-mount test injects the same context key.
	callerID := uuid.New().String()
	r.Use(func(c *gin.Context) {
		c.Set("userID", callerID)
		c.Next()
	})
	r.POST("/api/v1/notifications", h.CreateNotification)
	r.GET("/api/v1/notifications/:id", h.GetNotification)

	to := "e2e-" + uuid.New().String()[:8] + "@example.com"
	payload := map[string]interface{}{
		"type":    "info",
		"title":   "E2E real delivery",
		"message": "End-to-end handler-level real SMTP delivery proof " + uuid.New().String(),
		"channel": "email",
		"target":  to,
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())

	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.Equal(t, "sent", created["status"], "status must reflect REAL delivery outcome, not the old permanent 'pending' bluff")

	// Independent sink-side confirmation via MailHog's own API.
	found := false
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/api/v2/messages", httpPort))
		require.NoError(t, err)
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if bytes.Contains(raw, []byte(to)) {
			found = true
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	require.True(t, found, "MailHog never reported receiving the email created via the real HTTP handler")

	// Also fetch back via the handler's GET endpoint to confirm persistence.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/"+created["id"].(string), nil)
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	var fetched map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &fetched))
	require.Equal(t, "sent", fetched["status"])
	require.NotEmpty(t, fetched["sentAt"])
}

// TestCreateNotification_Webhook_RealDeliveryEndToEnd proves the same for
// the webhook channel: a real net/http receiver actually gets the POST, and
// the persisted status is "delivered".
func TestCreateNotification_Webhook_RealDeliveryEndToEnd(t *testing.T) {
	repo := setupE2EPostgres(t)

	received := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		received <- b
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// The receiver above is a loopback httptest.Server, so this uses the
	// test-permissive constructor — the production sender used by
	// handler.New() (delivery.NewWebhookSender) correctly refuses to dial
	// loopback/private/link-local destinations (SSRF guard, see
	// internal/delivery/webhook_ssrf_test.go).
	ws := delivery.NewWebhookSenderForTesting(5 * time.Second)
	ps := delivery.NewPushSender()
	h := handler.NewWithDelivery(repo, nil, ws, ps)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
		c.Next()
	})
	r.POST("/api/v1/notifications", h.CreateNotification)

	payload := map[string]interface{}{
		"type":    "info",
		"title":   "E2E webhook delivery",
		"message": "End-to-end handler-level real webhook POST proof " + uuid.New().String(),
		"channel": "webhook",
		"target":  srv.URL,
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())

	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.Equal(t, "delivered", created["status"])

	select {
	case got := <-received:
		require.Contains(t, string(got), "E2E webhook delivery")
	case <-time.After(5 * time.Second):
		t.Fatal("webhook receiver never observed the POST")
	}
}

// TestCreateNotification_Push_HonestNotConfigured proves push channel
// notifications are created with an HONEST status, never a fabricated
// "sent"/"delivered".
func TestCreateNotification_Push_HonestNotConfigured(t *testing.T) {
	repo := setupE2EPostgres(t)

	ps := delivery.NewPushSender()
	h := handler.NewWithDelivery(repo, nil, nil, ps)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
		c.Next()
	})
	r.POST("/api/v1/notifications", h.CreateNotification)

	payload := map[string]interface{}{
		"type":    "info",
		"title":   "E2E push honest state",
		"message": "Push must never fabricate success",
		"channel": "push",
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())
	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.Equal(t, "pending_provider_unconfigured", created["status"])
}
