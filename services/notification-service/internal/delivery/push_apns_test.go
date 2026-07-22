package delivery

// APNs delivery-client unit tests. Every request is served by an in-process
// httptest.Server and the sender is injected with that server's client — NO
// live Apple endpoint is ever contacted (Constitution §11.4.27: the APNs
// backend is the operator-gated boundary; these tests prove request
// construction + response handling against a MOCK transport).

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
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

// writeTestP8 generates a throwaway P-256 EC key, writes it as a PKCS#8 PEM
// (.p8, as Apple emits), and returns the file path plus its public key (used to
// verify the ES256 JWS signature the client produces).
func writeTestP8(t *testing.T) (path string, pub *ecdsa.PublicKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	path = filepath.Join(t.TempDir(), "AuthKey.p8")
	require.NoError(t, os.WriteFile(path, keyPEM, 0o600))
	return path, &key.PublicKey
}

// decodeAndVerifyAPNsJWT splits a compact JWS, decodes its header + claims, and
// verifies the ES256 signature against pub — proving the R||S JWS encoding is
// correct (not the ASN.1 DER form), i.e. a real, verifiable provider token.
func decodeAndVerifyAPNsJWT(t *testing.T, token string, pub *ecdsa.PublicKey) (header, claims map[string]any) {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "a compact JWS has three dot-separated parts")

	hRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	cRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(hRaw, &header))
	require.NoError(t, json.Unmarshal(cRaw, &claims))

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)
	require.Len(t, sig, 64, "ES256 signature is R||S, 32 bytes each")

	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	require.True(t, ecdsa.Verify(pub, digest[:], r, s), "the ES256 signature must verify against the signing key")
	return header, claims
}

func TestPushSender_APNs_Success(t *testing.T) {
	var gotPath, gotAuth, gotTopic, gotPushType, gotContentType string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("authorization")
		gotTopic = r.Header.Get("apns-topic")
		gotPushType = r.Header.Get("apns-push-type")
		gotContentType = r.Header.Get("Content-Type")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("apns-id", "00000000-0000-0000-0000-000000000000")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p8Path, pub := writeTestP8(t)
	p := &PushSender{
		cfg: PushConfig{
			Provider:     PushProviderAPNs,
			APNsKeyPath:  p8Path,
			APNsKeyID:    "ABC1234567",
			APNsTeamID:   "TEAM123456",
			APNsBundleID: "com.example.app",
		},
		configured:  true,
		httpClient:  srv.Client(),
		now:         func() time.Time { return time.Unix(1_700_000_000, 0) },
		apnsBaseURL: srv.URL,
	}

	err := p.SendTo(context.Background(), "devtoken123", PushPayload{
		Title: "Hi",
		Body:  "There",
		Data:  map[string]string{"x": "y"},
	})
	require.NoError(t, err)

	assert.Equal(t, "/3/device/devtoken123", gotPath)
	assert.Equal(t, "com.example.app", gotTopic, "apns-topic must be the bundle id")
	assert.Equal(t, "alert", gotPushType)
	assert.Equal(t, "application/json", gotContentType)
	require.True(t, strings.HasPrefix(gotAuth, "bearer "), "APNs auth is a bearer provider token, got %q", gotAuth)

	header, claims := decodeAndVerifyAPNsJWT(t, strings.TrimPrefix(gotAuth, "bearer "), pub)
	assert.Equal(t, "ES256", header["alg"])
	assert.Equal(t, "ABC1234567", header["kid"], "the JWT header kid must be the APNs key id")
	assert.Equal(t, "TEAM123456", claims["iss"], "the JWT iss claim must be the team id")
	assert.EqualValues(t, 1_700_000_000, claims["iat"], "iat must be the issued-at epoch seconds")

	aps, ok := gotBody["aps"].(map[string]any)
	require.True(t, ok, "body must carry an aps object")
	alert, ok := aps["alert"].(map[string]any)
	require.True(t, ok, "aps must carry an alert object")
	assert.Equal(t, "Hi", alert["title"])
	assert.Equal(t, "There", alert["body"])
	assert.Equal(t, "y", gotBody["x"], "custom data must ride alongside aps at the top level")
}

// TestPushSender_APNs_SandboxHostViaConfig proves the APNs host is configurable
// for the sandbox via PushConfig.APNsHost (APNS_HOST) when no test-only
// apnsBaseURL override is set — the production-vs-sandbox switch operators use.
func TestPushSender_APNs_SandboxHostViaConfig(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		assert.Equal(t, "/3/device/tok", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p8Path, _ := writeTestP8(t)
	p := &PushSender{
		cfg: PushConfig{
			Provider:     PushProviderAPNs,
			APNsKeyPath:  p8Path,
			APNsKeyID:    "ABC1234567",
			APNsTeamID:   "TEAM123456",
			APNsBundleID: "com.example.app",
			APNsHost:     srv.URL, // stands in for https://api.sandbox.push.apple.com
		},
		configured: true,
		httpClient: srv.Client(),
		now:        func() time.Time { return time.Unix(1_700_000_000, 0) },
		// apnsBaseURL deliberately empty: exercise the APNsHost path.
	}

	err := p.SendTo(context.Background(), "tok", PushPayload{Title: "T"})
	require.NoError(t, err)
	assert.True(t, hit, "the request must be routed to the configured APNs host")
}

func TestPushSender_APNs_BadDeviceToken_ReturnsRealError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"reason":"BadDeviceToken"}`))
	}))
	defer srv.Close()

	p8Path, _ := writeTestP8(t)
	p := &PushSender{
		cfg: PushConfig{
			Provider:     PushProviderAPNs,
			APNsKeyPath:  p8Path,
			APNsKeyID:    "ABC1234567",
			APNsTeamID:   "TEAM123456",
			APNsBundleID: "com.example.app",
		},
		configured:  true,
		httpClient:  srv.Client(),
		now:         func() time.Time { return time.Unix(1_700_000_000, 0) },
		apnsBaseURL: srv.URL,
	}

	err := p.SendTo(context.Background(), "devtoken123", PushPayload{Title: "T"})
	require.Error(t, err, "a 4xx from APNs must surface as a real error")
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "BadDeviceToken")
}

func TestPushSender_APNs_ServerError_ReturnsRealError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p8Path, _ := writeTestP8(t)
	p := &PushSender{
		cfg: PushConfig{
			Provider:     PushProviderAPNs,
			APNsKeyPath:  p8Path,
			APNsKeyID:    "ABC1234567",
			APNsTeamID:   "TEAM123456",
			APNsBundleID: "com.example.app",
		},
		configured:  true,
		httpClient:  srv.Client(),
		now:         func() time.Time { return time.Unix(1_700_000_000, 0) },
		apnsBaseURL: srv.URL,
	}

	err := p.SendTo(context.Background(), "devtoken123", PushPayload{Title: "T"})
	require.Error(t, err, "a 5xx from APNs must surface as a real error")
	assert.Contains(t, err.Error(), "500")
}
