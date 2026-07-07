package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/coreclient"
	"github.com/stretchr/testify/assert"
)

// TestLive_CreateBridge_ReflectsRealCoreAuth is the §11.4.27 real-integration
// proof that CreateBridge's status decision is driven by a genuinely running
// HelixTrack Core instance — NOT a fake/mocked Authenticator. Same recipe as
// coreclient's live test (see README.md "HelixTrack Core authentication").
//
// There is no live Postgres in this environment (out of scope per the task
// brief), so the DB-write step itself cannot be observed succeeding; instead
// this test uses the same auth-gate/DB-layer oracle as
// TestCreateBridge_CoreAuth{Success,Failure}* in handler_test.go: a
// DB-layer 500 (not an auth-layer 503) proves the auth gate was PASSED —
// i.e. CreateBridge would have set Status "active" for a real, live-auth
// success.
//
// §11.4.3 topology dispatch: SKIPs with an explicit reason when
// HELIXTRACK_CORE_BASE_URL is unset.
func TestLive_CreateBridge_ReflectsRealCoreAuth(t *testing.T) {
	baseURL := os.Getenv("HELIXTRACK_CORE_BASE_URL")
	if baseURL == "" {
		t.Skip("SKIP (§11.4.3): HELIXTRACK_CORE_BASE_URL not set — live HelixTrack Core sandbox not running in this environment")
	}
	username := envOrDefault("HELIXTRACK_CORE_USERNAME", "admin_user")
	password := envOrDefault("HELIXTRACK_CORE_PASSWORD", "Admin@123456")

	gin.SetMode(gin.TestMode)

	newRequest := func() (*http.Request, error) {
		body := map[string]interface{}{
			"integrationId": "integration-123",
			"orgId":         uuid.New().String(),
			"name":          "test-integration",
		}
		b, _ := json.Marshal(body)
		req, err := http.NewRequest("POST", "/api/v1/helixtrack-bridges", bytes.NewBuffer(b))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return req, err
	}

	t.Run("real correct Core credentials pass the auth gate (reach the DB layer)", func(t *testing.T) {
		core := coreclient.New(baseURL, username, password)
		repo := unreachablePool(t)
		h := New(repo, core)

		router := gin.New()
		router.POST("/api/v1/helixtrack-bridges", h.CreateBridge)

		req, err := newRequest()
		assert.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code,
			"real Core auth succeeded, so CreateBridge must proceed to (and fail only at) the DB layer — proving it would set Status active for a real, live authentication")
		assert.Contains(t, w.Body.String(), "failed to create bridge")
	})

	t.Run("real wrong Core password yields a non-active/error status, never reaches the DB layer", func(t *testing.T) {
		core := coreclient.New(baseURL, username, "definitely-wrong-password")
		repo := unreachablePool(t)
		h := New(repo, core)

		router := gin.New()
		router.POST("/api/v1/helixtrack-bridges", h.CreateBridge)

		req, err := newRequest()
		assert.NoError(t, err)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code, "a real rejected Core authentication MUST yield 503, never a fabricated 201")
		assert.Contains(t, w.Body.String(), `"status":"error"`)
		assert.Contains(t, w.Body.String(), "Invalid username or password")
		assert.NotContains(t, w.Body.String(), "failed to create bridge")
	})
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
