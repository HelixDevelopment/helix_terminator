package delivery

// FCM delivery-client unit tests. Every request is served by an in-process
// httptest.Server and the sender is injected with that server's client — NO
// live Google OAuth / FCM endpoint is ever contacted (Constitution §11.4.27:
// the third-party push backend is the operator-gated boundary; these tests
// prove request construction + response handling against a MOCK transport).

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTestServiceAccount generates a throwaway RSA key, wraps it in a minimal
// service-account JSON file, and returns the file path. tokenURI is embedded in
// the JSON (may be empty when the sender's oauthTokenURL override is used).
func writeTestServiceAccount(t *testing.T, tokenURI string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	sa := map[string]string{
		"type":         "service_account",
		"project_id":   "test-project",
		"private_key":  string(keyPEM),
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"token_uri":    tokenURI,
	}
	raw, err := json.Marshal(sa)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "sa.json")
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	return path
}

func TestPushSender_FCMHTTPv1_Success(t *testing.T) {
	var tokenHit, sendHit bool
	var gotSendPath, gotAuth, gotAssertion string
	var gotGrant string
	var gotMsg fcmV1Message

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			tokenHit = true
			require.NoError(t, r.ParseForm())
			gotGrant = r.PostForm.Get("grant_type")
			gotAssertion = r.PostForm.Get("assertion")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`))
		case strings.HasSuffix(r.URL.Path, "/messages:send"):
			sendHit = true
			gotSendPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")
			require.NoError(t, json.NewDecoder(r.Body).Decode(&gotMsg))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name":"projects/test-project/messages/1"}`))
		default:
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := &PushSender{
		cfg:           PushConfig{Provider: PushProviderFCM, FCMServiceAccountJSONPath: writeTestServiceAccount(t, "")},
		configured:    true,
		httpClient:    srv.Client(),
		now:           func() time.Time { return time.Unix(1_700_000_000, 0) },
		fcmBaseURL:    srv.URL,
		oauthTokenURL: srv.URL + "/token",
	}

	err := p.SendTo(context.Background(), "device-token-abc", PushPayload{
		Title: "Hello",
		Body:  "World",
		Data:  map[string]string{"k": "v"},
	})
	require.NoError(t, err)

	assert.True(t, tokenHit, "OAuth token endpoint must be called")
	assert.True(t, sendHit, "FCM send endpoint must be called")
	assert.Equal(t, "urn:ietf:params:oauth:grant-type:jwt-bearer", gotGrant)
	assert.NotEmpty(t, gotAssertion, "a signed JWT assertion must be posted")
	assert.Equal(t, "/v1/projects/test-project/messages:send", gotSendPath)
	assert.Equal(t, "Bearer test-access-token", gotAuth, "the minted access token must be the bearer credential")
	assert.Equal(t, "device-token-abc", gotMsg.Message.Token)
	assert.Equal(t, "Hello", gotMsg.Message.Notification.Title)
	assert.Equal(t, "World", gotMsg.Message.Notification.Body)
	assert.Equal(t, "v", gotMsg.Message.Data["k"])
}

func TestPushSender_FCMHTTPv1_SendRejected_ReturnsRealError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":3600}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"status":"UNAVAILABLE"}}`))
	}))
	defer srv.Close()

	p := &PushSender{
		cfg:           PushConfig{Provider: PushProviderFCM, FCMServiceAccountJSONPath: writeTestServiceAccount(t, "")},
		configured:    true,
		httpClient:    srv.Client(),
		now:           func() time.Time { return time.Unix(1_700_000_000, 0) },
		fcmBaseURL:    srv.URL,
		oauthTokenURL: srv.URL + "/token",
	}

	err := p.SendTo(context.Background(), "device-token-abc", PushPayload{Title: "T", Body: "B"})
	require.Error(t, err, "a 5xx from FCM must surface as a real error, never a fabricated success")
	assert.Contains(t, err.Error(), "503")
}

func TestPushSender_FCMHTTPv1_TokenEndpointRejected_ReturnsRealError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	p := &PushSender{
		cfg:           PushConfig{Provider: PushProviderFCM, FCMServiceAccountJSONPath: writeTestServiceAccount(t, "")},
		configured:    true,
		httpClient:    srv.Client(),
		now:           func() time.Time { return time.Unix(1_700_000_000, 0) },
		fcmBaseURL:    srv.URL,
		oauthTokenURL: srv.URL + "/token",
	}

	err := p.SendTo(context.Background(), "device-token-abc", PushPayload{Title: "T"})
	require.Error(t, err, "an OAuth token rejection must surface as a real error")
	assert.Contains(t, err.Error(), "access token")
}

func TestPushSender_FCMLegacy_Success(t *testing.T) {
	var gotPath, gotAuth string
	var gotReq fcmLegacyRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotReq))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":1,"failure":0,"results":[{"message_id":"1:abc"}]}`))
	}))
	defer srv.Close()

	p := &PushSender{
		cfg:        PushConfig{Provider: PushProviderFCM, FCMServerKey: "server-key-xyz"},
		configured: true,
		httpClient: srv.Client(),
		fcmBaseURL: srv.URL,
	}

	err := p.SendTo(context.Background(), "device-token-abc", PushPayload{Title: "Hi", Body: "Yo", Data: map[string]string{"a": "b"}})
	require.NoError(t, err)

	assert.Equal(t, "/fcm/send", gotPath)
	assert.Equal(t, "key=server-key-xyz", gotAuth, "the legacy server key must be the auth credential")
	assert.Equal(t, "device-token-abc", gotReq.To)
	assert.Equal(t, "Hi", gotReq.Notification.Title)
	assert.Equal(t, "b", gotReq.Data["a"])
}

func TestPushSender_FCMLegacy_PerMessageFailure_ReturnsRealError(t *testing.T) {
	// The legacy endpoint returns HTTP 200 even when the message failed; the
	// body's success/failure counters must be inspected so a per-message
	// failure is surfaced as a real error, never read as a fabricated "sent".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":0,"failure":1,"results":[{"error":"InvalidRegistration"}]}`))
	}))
	defer srv.Close()

	p := &PushSender{
		cfg:        PushConfig{Provider: PushProviderFCM, FCMServerKey: "server-key-xyz"},
		configured: true,
		httpClient: srv.Client(),
		fcmBaseURL: srv.URL,
	}

	err := p.SendTo(context.Background(), "device-token-abc", PushPayload{Title: "T"})
	require.Error(t, err, "a 200 with failure=1 must surface as a real error")
	assert.Contains(t, err.Error(), "InvalidRegistration")
}
