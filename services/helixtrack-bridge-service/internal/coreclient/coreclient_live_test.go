package coreclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLive_Authenticate_RealCore is the §11.4.27 real-integration test: it
// authenticates against a genuinely running HelixTrack Core instance (see
// the recipe in the task brief / services/helixtrack-bridge-service/README.md
// "HelixTrack Core authentication") — NO fake/mocked HTTP server here.
//
// §11.4.3 topology dispatch: SKIPs with an explicit reason when
// HELIXTRACK_CORE_BASE_URL is unset (the live Core sandbox is not running) —
// this is the honest fallback, never a fabricated PASS.
func TestLive_Authenticate_RealCore(t *testing.T) {
	baseURL := os.Getenv("HELIXTRACK_CORE_BASE_URL")
	if baseURL == "" {
		t.Skip("SKIP (§11.4.3): HELIXTRACK_CORE_BASE_URL not set — live HelixTrack Core sandbox not running in this environment")
	}
	username := envOrDefault("HELIXTRACK_CORE_USERNAME", "admin_user")
	password := envOrDefault("HELIXTRACK_CORE_PASSWORD", "Admin@123456")

	t.Run("correct credentials return a real HS256 24h JWT", func(t *testing.T) {
		c := New(baseURL, username, password)
		token, err := c.Authenticate(context.Background())
		require.NoError(t, err, "real Core authenticate must succeed with the seeded test-fixture credentials")
		require.NotEmpty(t, token)

		// Structural JWT proof (never fabricated): 3 dot-separated base64url
		// segments, header decodes to HS256 per the task brief.
		parts := strings.Split(token, ".")
		require.Len(t, parts, 3, "a JWT MUST have 3 dot-separated segments (header.payload.signature)")
		headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
		require.NoError(t, err)
		var header struct {
			Alg string `json:"alg"`
			Typ string `json:"typ"`
		}
		require.NoError(t, json.Unmarshal(headerJSON, &header))
		assert.Equal(t, "HS256", header.Alg)
		assert.Equal(t, "JWT", header.Typ)

		payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)
		var payload struct {
			Username string `json:"username"`
			Exp      int64  `json:"exp"`
			Iat      int64  `json:"iat"`
		}
		require.NoError(t, json.Unmarshal(payloadJSON, &payload))
		assert.Equal(t, username, payload.Username)
		assert.Equal(t, int64(24*60*60), payload.Exp-payload.Iat, "Core's JWT expiry must be the documented 24h window")

		cached, err := c.AccessToken()
		require.NoError(t, err)
		assert.Equal(t, token, cached, "Authenticate must cache the real token via tokenmanager")

		require.NoError(t, c.EnsureAuthenticated(context.Background()), "a freshly-cached valid token must satisfy EnsureAuthenticated")
	})

	t.Run("wrong password yields a non-active auth error, no token cached", func(t *testing.T) {
		c := New(baseURL, username, "definitely-wrong-password")
		token, err := c.Authenticate(context.Background())
		assert.Error(t, err, "a wrong password MUST be rejected by the real Core")
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "Invalid username or password")

		cached, cerr := c.AccessToken()
		require.NoError(t, cerr)
		assert.Empty(t, cached, "a rejected authenticate MUST NOT cache a token")

		ensureErr := c.EnsureAuthenticated(context.Background())
		assert.Error(t, ensureErr, "EnsureAuthenticated must propagate the real auth failure")
	})
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
