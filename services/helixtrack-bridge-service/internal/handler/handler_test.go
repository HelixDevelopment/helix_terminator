package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helixdevelopment/helixtrack-bridge-service/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() (*gin.Engine, *Handler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := New(nil, nil)
	router.GET("/healthz", h.HealthCheck)
	router.GET("/healthz/ready", h.ReadinessCheck)
	return router, h
}

// spyAuthenticator is a unit-test-layer stand-in for coreclient.Client
// (§11.4.27) that records whether/how many times CreateBridge consulted
// real Core authentication before deciding a bridge's status, and returns a
// configurable outcome (nil = auth succeeded, non-nil = auth rejected).
type spyAuthenticator struct {
	calls int
	err   error
}

func (s *spyAuthenticator) EnsureAuthenticated(ctx context.Context) error {
	s.calls++
	return s.err
}

// unreachablePool returns a repository pointing at a syntactically-valid but
// definitely-unreachable Postgres DSN. pgxpool.New parses config and starts
// background connection management WITHOUT dialing synchronously, so this
// succeeds immediately and gives CreateBridge a non-nil *repository.Repository
// (passing the "database not available" nil-check) without a real Postgres
// instance. Any query issued against it (e.g. repo.CreateBridge's INSERT)
// fails fast with a connection error — used below to prove CreateBridge
// reaches (or does not reach) the DB-write step depending on the auth outcome.
func unreachablePool(t *testing.T) *repository.Repository {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://u:p@127.0.0.1:1/nonexistent?sslmode=disable&connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return repository.New(pool)
}

func TestHealthCheck(t *testing.T) {
	router, _ := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestReadinessCheck_NoDB(t *testing.T) {
	router, _ := setupTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz/ready", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "not ready")
}

func TestCreateBridge_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.POST("/api/v1/helixtrack-bridges", h.CreateBridge)

	body := map[string]interface{}{
		"integrationId": "integration-123",
		"orgId":         uuid.New().String(),
		"name":          "test-integration",
		"config":        map[string]string{"key": "value"},
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/helixtrack-bridges", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "database not available")
}

func TestGetBridge_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/helixtrack-bridges/:id", h.GetBridge)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/helixtrack-bridges/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListBridges_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/helixtrack-bridges", h.ListBridges)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/helixtrack-bridges", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateBridge_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.PUT("/api/v1/helixtrack-bridges/:id", h.UpdateBridge)

	body := map[string]interface{}{"status": "inactive"}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/helixtrack-bridges/"+uuid.New().String(), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteBridge_NoDB(t *testing.T) {
	router, h := setupTestRouter()
	router.DELETE("/api/v1/helixtrack-bridges/:id", h.DeleteBridge)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/helixtrack-bridges/"+uuid.New().String(), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// TestCreateBridge_ConsultsCoreAuthBeforeActive is the anti-bluff RED/GREEN
// test for the fabricated-status defect (§11.4.108 / §11.4.43 / §11.4.115):
// CreateBridge previously set Status "active" unconditionally, without ever
// authenticating against a real HelixTrack Core. It MUST consult
// h.core.EnsureAuthenticated exactly once before deciding the bridge's fate.
//
// RED (pre-fix, captured 2026-07-08): with the Authenticator wired but NOT
// yet consulted by CreateBridge, this assertion FAILed with "0 != 1" — proof
// the fabrication was real, not a synthetic strawman (§11.4.115).
// GREEN (post-fix): CreateBridge calls h.authenticateCore(ctx) exactly once.
func TestCreateBridge_ConsultsCoreAuthBeforeActive(t *testing.T) {
	spy := &spyAuthenticator{err: nil}
	repo := unreachablePool(t)
	h := New(repo, spy)

	router := gin.New()
	gin.SetMode(gin.TestMode)
	router.POST("/api/v1/helixtrack-bridges", h.CreateBridge)

	body := map[string]interface{}{
		"integrationId": "integration-123",
		"orgId":         uuid.New().String(),
		"name":          "test-integration",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/helixtrack-bridges", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 1, spy.calls, "CreateBridge MUST consult real Core authentication exactly once before deciding bridge status — a fabricated 'active' status without an auth call is a §11.4.108 PASS-bluff")
}

// TestCreateBridge_CoreAuthFailure_NeverTouchesDB proves the auth-failure
// path short-circuits BEFORE any database write is attempted: the response
// MUST be 503 with an "error" status (never 500 "failed to create bridge",
// which would mean CreateBridge reached the DB layer despite a rejected
// Core authentication — the exact fabrication this fix closes).
func TestCreateBridge_CoreAuthFailure_NeverTouchesDB(t *testing.T) {
	spy := &spyAuthenticator{err: errors.New("coreclient: authenticate rejected by core: Invalid username or password")}
	repo := unreachablePool(t)
	h := New(repo, spy)

	router := gin.New()
	gin.SetMode(gin.TestMode)
	router.POST("/api/v1/helixtrack-bridges", h.CreateBridge)

	body := map[string]interface{}{
		"integrationId": "integration-123",
		"orgId":         uuid.New().String(),
		"name":          "test-integration",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/helixtrack-bridges", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code, "a rejected Core authentication MUST yield 503, never a fabricated 201")
	assert.Contains(t, w.Body.String(), "error")
	assert.NotContains(t, w.Body.String(), "failed to create bridge", "response must not carry a DB-layer error — the auth gate must short-circuit before any repo.CreateBridge call")
	assert.Equal(t, 1, spy.calls)
}

// TestCreateBridge_CoreAuthSuccess_ReachesDBLayer proves the inverse: when
// Core authentication succeeds, CreateBridge proceeds to attempt the DB
// write (observable here as a DB-layer 500, since the pool is unreachable —
// there is no real Postgres in this unit-test environment). A DB-layer error
// (not an auth-layer 503) is the oracle proving the auth gate was passed.
func TestCreateBridge_CoreAuthSuccess_ReachesDBLayer(t *testing.T) {
	spy := &spyAuthenticator{err: nil}
	repo := unreachablePool(t)
	h := New(repo, spy)

	router := gin.New()
	gin.SetMode(gin.TestMode)
	router.POST("/api/v1/helixtrack-bridges", h.CreateBridge)

	body := map[string]interface{}{
		"integrationId": "integration-123",
		"orgId":         uuid.New().String(),
		"name":          "test-integration",
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/helixtrack-bridges", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "with auth passed and DB unreachable, CreateBridge must reach (and fail at) the DB layer, proving it did not short-circuit on auth")
	assert.Contains(t, w.Body.String(), "failed to create bridge")
	assert.Equal(t, 1, spy.calls)
}

func TestListBridges_Pagination(t *testing.T) {
	router, h := setupTestRouter()
	router.GET("/api/v1/helixtrack-bridges", h.ListBridges)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/helixtrack-bridges?limit=101&offset=-1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
