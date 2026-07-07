package server

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents custom JWT claims for gateway validation.
type Claims struct {
	UserID      string   `json:"userId"`
	OrgID       string   `json:"orgId,omitempty"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions,omitempty"`
	SessionID   string   `json:"sessionId"`
	TokenType   string   `json:"tokenType"`
	jwt.RegisteredClaims
}

// upstreamService represents a backend service for routing
type upstreamService struct {
	Name    string
	Address string
	Healthy bool
	mu      sync.RWMutex
}

func (u *upstreamService) SetHealthy(h bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Healthy = h
}

func (u *upstreamService) IsHealthy() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.Healthy
}

// gatewayHealthCheckIntervalEnv is the environment variable used to
// configure how often the background upstream health-check prober (T8-8)
// re-checks every registered upstream's real reachability. Unset or
// invalid falls back to defaultHealthCheckInterval.
const gatewayHealthCheckIntervalEnv = "GATEWAY_HEALTHCHECK_INTERVAL"

// defaultHealthCheckInterval is the sane default cadence for the
// background upstream health-check prober when
// GATEWAY_HEALTHCHECK_INTERVAL is unset: frequent enough that a genuinely
// down upstream is reflected in /healthz within a reasonable window,
// infrequent enough not to hammer 24 registered upstreams with probe
// traffic every few seconds.
const defaultHealthCheckInterval = 15 * time.Second

// healthCheckProbeTimeout bounds a single upstream health probe so one
// slow-to-respond (not yet fully unreachable) upstream can never stall the
// whole probe sweep or the shutdown path.
const healthCheckProbeTimeout = 3 * time.Second

// healthCheckIntervalFromEnv resolves the configured probe interval,
// falling back to defaultHealthCheckInterval on an unset or invalid value
// (never guessed — an invalid duration is treated the same as unset,
// §11.4.6).
func healthCheckIntervalFromEnv() time.Duration {
	if raw := strings.TrimSpace(os.Getenv(gatewayHealthCheckIntervalEnv)); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}
	return defaultHealthCheckInterval
}

// rateLimiter implements a simple token bucket rate limiter per key
type rateLimiter struct {
	requests map[string]int
	window   time.Duration
	mu       sync.Mutex
}

func newRateLimiter(window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string]int),
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) Allow(key string, maxRequests int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.requests[key] >= maxRequests {
		return false
	}
	rl.requests[key]++
	return true
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	for range ticker.C {
		rl.mu.Lock()
		rl.requests = make(map[string]int)
		rl.mu.Unlock()
	}
}

// circuitBreaker implements a simple circuit breaker
type circuitBreaker struct {
	failures    int
	threshold   int
	state       string // closed, open, half-open
	lastFailure time.Time
	mu          sync.RWMutex
}

func newCircuitBreaker(threshold int) *circuitBreaker {
	return &circuitBreaker{
		threshold: threshold,
		state:     "closed",
	}
}

func (cb *circuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	if cb.state == "open" {
		if time.Since(cb.lastFailure) > 30*time.Second {
			cb.state = "half-open"
		} else {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker is open")
		}
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.threshold {
			cb.state = "open"
		}
	} else {
		cb.failures = 0
		cb.state = "closed"
	}
	return err
}

// Server wraps the Gin engine with gateway functionality
type Server struct {
	router          *gin.Engine
	logger          Logger
	upstreams       map[string]*upstreamService
	upstreamsMu     sync.RWMutex
	userLimiter     *rateLimiter
	ipLimiter       *rateLimiter
	endpointLimiter *rateLimiter
	breakers        map[string]*circuitBreaker
	breakersMu      sync.RWMutex
	jwtPublicKey    ed25519.PublicKey
	httpClient      *http.Client
	// startTime records the real process/server boot instant (§11.4.108
	// anti-bluff: /healthz "uptime" MUST be a genuine elapsed-time
	// measurement derived from this timestamp, never a hardcoded literal).
	startTime time.Time
	// healthCheckStopCh / healthCheckWG / healthCheckStopOnce govern the
	// lifecycle of the background upstream health-check prober started by
	// startHealthChecks (T8-8): closing healthCheckStopCh signals the
	// prober goroutine to exit, healthCheckWG lets Stop block until it
	// genuinely has, and healthCheckStopOnce makes Stop safe to call more
	// than once (e.g. once explicitly during shutdown and once via a
	// deferred test cleanup) without a double-close panic.
	healthCheckStopCh   chan struct{}
	healthCheckWG       sync.WaitGroup
	healthCheckStopOnce sync.Once
}

// Logger interface for logging
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type defaultLogger struct{}

func (d *defaultLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

func (d *defaultLogger) Println(v ...interface{}) {
	fmt.Println(v...)
}

// New creates a new Gateway Server with full middleware chain
func New(logger Logger) *Server {
	if logger == nil {
		logger = &defaultLogger{}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	s := &Server{
		router:          r,
		logger:          logger,
		upstreams:       make(map[string]*upstreamService),
		userLimiter:     newRateLimiter(time.Minute),
		ipLimiter:       newRateLimiter(time.Minute),
		endpointLimiter: newRateLimiter(time.Minute),
		breakers:        make(map[string]*circuitBreaker),
		httpClient:      &http.Client{Timeout: 15 * time.Second},
		startTime:       time.Now(),
	}

	// Load JWT public key from environment
	if pubKeyB64 := os.Getenv("JWT_PUBLIC_KEY"); pubKeyB64 != "" {
		pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyB64)
		if err != nil {
			logger.Printf("warning: failed to decode JWT_PUBLIC_KEY: %v", err)
		} else if len(pubKeyBytes) != ed25519.PublicKeySize {
			logger.Printf("warning: invalid JWT_PUBLIC_KEY size: expected %d, got %d", ed25519.PublicKeySize, len(pubKeyBytes))
		} else {
			s.jwtPublicKey = ed25519.PublicKey(pubKeyBytes)
		}
	}

	// Global middleware
	r.Use(s.recoveryMiddleware())
	r.Use(s.requestIDMiddleware())
	r.Use(s.loggingMiddleware())
	r.Use(s.corsMiddleware())

	// Rate limiting middleware
	r.Use(s.rateLimitMiddleware())

	// Register upstream services
	s.registerUpstreams()

	// Start the background per-upstream health-check prober (T8-8). This
	// must never block server boot: startHealthChecks only launches a
	// goroutine and returns immediately — the first real probe sweep runs
	// asynchronously inside that goroutine.
	s.startHealthChecks()

	// Health endpoints (no auth required)
	r.GET("/healthz/live", s.livenessHandler)
	r.GET("/healthz/ready", s.readinessHandler)
	r.GET("/healthz", s.fullHealthHandler)

	// Metrics endpoint (Prometheus format)
	r.GET("/metrics", s.metricsHandler)

	// OpenAPI spec serving
	r.GET("/api/v1/openapi.json", s.openapiHandler)

	// API routes with JWT validation
	api := r.Group("/api/v1")
	api.Use(s.jwtValidationMiddleware())
	{
		// Auth routes (proxy to auth-service). auth-service registers all
		// six of these at bare root paths (services/auth-service/internal/
		// server/server.go:126-146) — re-verified current (T8-1): these
		// already matched the real registrations and are unchanged.
		api.POST("/auth/register", s.proxyTo("auth-service", "/register"))
		api.POST("/auth/login", s.proxyTo("auth-service", "/login"))
		api.POST("/auth/mfa/verify", s.proxyTo("auth-service", "/mfa/verify"))
		api.POST("/auth/mfa/setup", s.proxyTo("auth-service", "/mfa/setup"))
		api.POST("/auth/refresh", s.proxyTo("auth-service", "/refresh"))
		api.POST("/auth/logout", s.proxyTo("auth-service", "/logout"))

		// User routes. user-service (services/user-service/internal/
		// server/server.go:44-53) exposes only /api/v1/users,
		// /api/v1/users/:id, /api/v1/users/:id/profile and
		// /api/v1/users/by-email — there is NO self ("/me") lookup, no
		// sessions concept and no preferences concept anywhere in the
		// service. auth-service independently generates its own user ID
		// (services/auth-service/internal/handler/handler.go Register) and
		// user-service independently generates its own uuid.New() at
		// CreateUser (services/user-service/internal/handler/handler.go:41)
		// — there is no code anywhere establishing these two ID spaces are
		// the same, so resolving "/me" by injecting the JWT userID as a
		// user-service :id would be an unverified guess (§11.4.6); if the
		// ID spaces really are disjoint (confirmed: they are, both call
		// uuid.New() independently with no cross-service linkage) it would
		// never resolve to real data. Honest 501 gaps, not fabricated
		// routes to nowhere.
		api.GET("/users/me", s.notImplemented("users.me",
			"user-service exposes no self/\"me\" endpoint; its user IDs are independently generated from auth-service's with no verified cross-service identity link"))
		api.PATCH("/users/me", s.notImplemented("users.me",
			"user-service exposes no self/\"me\" endpoint; see GET /users/me"))
		// auth-service DOES track sessions in its repository
		// (ListActiveSessions / RevokeSession, services/auth-service/
		// internal/repository/repository.go:262-297) but no handler or
		// route anywhere exposes them over HTTP — confirmed: auth-service's
		// handler set has no caller of ListActiveSessions or RevokeSession.
		// Wiring that up is an auth-service change, out of gateway-service's
		// scope; tracked as a gap here.
		api.GET("/users/me/sessions", s.notImplemented("users.me.sessions",
			"no HTTP route anywhere exposes session listing; auth-service has the repository method (ListActiveSessions) but never wires it to a handler/route"))
		api.DELETE("/users/me/sessions/:sessionId", s.notImplemented("users.me.sessions.delete",
			"no HTTP route anywhere exposes session revoke-by-id; auth-service has the repository method (RevokeSession) but never wires it to a handler/route"))
		api.GET("/users/me/preferences", s.notImplemented("users.me.preferences",
			"no preferences concept exists anywhere in user-service (only \"profile\", a different resource)"))
		api.PATCH("/users/me/preferences", s.notImplemented("users.me.preferences",
			"no preferences concept exists anywhere in user-service; see GET /users/me/preferences"))

		// Vault routes. vault-service (services/vault-service/internal/
		// server/server.go:88-109) is a flat secrets store mounted at
		// /api/v1/vault/secrets — there is no "vault" grouping/container
		// concept and no "/share" action anywhere. The gateway's own
		// client-facing "/vaults" contract is kept; the proxied upstream
		// path is corrected to the real "secrets" resource.
		api.GET("/vaults", s.proxyTo("vault-service", "/api/v1/vault/secrets"))
		api.POST("/vaults", s.proxyTo("vault-service", "/api/v1/vault/secrets"))
		api.GET("/vaults/:vaultId", s.proxyTo("vault-service", "/api/v1/vault/secrets/:vaultId"))
		api.DELETE("/vaults/:vaultId", s.proxyTo("vault-service", "/api/v1/vault/secrets/:vaultId"))
		// "items within a vault" would need a two-level vault->item
		// hierarchy; vault-service has no such nesting (Secret is the only,
		// flat resource — already reachable via GET /vaults above). Mapping
		// these onto the same flat collection would silently discard the
		// :vaultId scope and return an unrelated result set, which is
		// worse than an honest gap.
		api.GET("/vaults/:vaultId/items", s.notImplemented("vaults.items",
			"vault-service has no vault->item nesting; secrets are already flat and reachable via GET /vaults"))
		api.POST("/vaults/:vaultId/items", s.notImplemented("vaults.items",
			"vault-service has no vault->item nesting; see GET /vaults/:vaultId/items"))
		api.POST("/vaults/:vaultId/share", s.notImplemented("vaults.share",
			"vault-service has no share/access-grant action anywhere"))

		// Host routes. host-service (services/host-service/internal/
		// server/server.go:90-96) registers PUT (not PATCH) for update, so
		// the gateway's own route verb is corrected to PUT to actually
		// reach it. Its connectivity check is "/test-connection" — there is
		// no "/connect" action; host-service never opens a live connection,
		// only tests reachability.
		api.GET("/hosts", s.proxyTo("host-service", "/api/v1/hosts"))
		api.POST("/hosts", s.proxyTo("host-service", "/api/v1/hosts"))
		api.GET("/hosts/:hostId", s.proxyTo("host-service", "/api/v1/hosts/:hostId"))
		api.PUT("/hosts/:hostId", s.proxyTo("host-service", "/api/v1/hosts/:hostId"))
		api.DELETE("/hosts/:hostId", s.proxyTo("host-service", "/api/v1/hosts/:hostId"))
		api.POST("/hosts/:hostId/connect", s.notImplemented("hosts.connect",
			"host-service has no live-connect action; only a connectivity probe exists (see /hosts/:hostId/test)"))
		api.POST("/hosts/:hostId/test", s.proxyTo("host-service", "/api/v1/hosts/:hostId/test-connection"))

		// SSH/Session routes. ssh-proxy-service (services/ssh-proxy-service/
		// internal/server/server.go:89-93) mounts sessions at
		// /api/v1/ssh/sessions.
		api.GET("/sessions", s.proxyTo("ssh-proxy-service", "/api/v1/ssh/sessions"))
		api.GET("/sessions/:sessionId", s.proxyTo("ssh-proxy-service", "/api/v1/ssh/sessions/:sessionId"))
		api.DELETE("/sessions/:sessionId", s.proxyTo("ssh-proxy-service", "/api/v1/ssh/sessions/:sessionId"))
		// terminal-service (services/terminal-service/internal/server/
		// server.go:90-103) generates its OWN uuid.New() terminal-session ID
		// (CreateTerminalSession, internal/handler/handler.go:56) — it is
		// NOT the ssh-proxy-service session ID, and nothing in the codebase
		// links the two. There is no way to honestly resolve "the terminal
		// for ssh session :sessionId".
		api.GET("/sessions/:sessionId/terminal", s.notImplemented("sessions.terminal",
			"terminal-service's own session IDs are independently generated and not linked to ssh-proxy-service's session IDs anywhere in the codebase"))
		// collaboration-service (services/collaboration-service/internal/
		// server/server.go:34-41) only has join/leave/end actions; no
		// "share" (generate/return an access grant) action exists.
		api.POST("/sessions/:sessionId/share", s.notImplemented("sessions.share",
			"collaboration-service has no share action (only join/leave/end)"))
		// recording-service's CreateRecording explicitly requires a
		// sessionId field in its own JSON request body
		// (binding:"required,uuid" — services/recording-service/internal/
		// model/model.go:41), independently confirming session-linkage is
		// part of its real design; the flat /api/v1/recordings collection
		// is the real "start recording this session" endpoint (the
		// gateway's :sessionId path segment is not itself forwarded — the
		// client is expected to supply the same sessionId in the body,
		// which recording-service's own contract already requires).
		api.POST("/sessions/:sessionId/record", s.proxyTo("recording-service", "/api/v1/recordings"))

		// SFTP routes. sftp-service (services/sftp-service/internal/
		// server/server.go:34-40) is a flat, HOST-scoped transfer-session
		// resource (CreateSFTPSessionRequest has hostId/remotePath/
		// localPath/direction — no sessionId field anywhere). There is no
		// download/upload action route, and no way to scope by an ssh
		// sessionId.
		api.GET("/sessions/:sessionId/sftp", s.notImplemented("sessions.sftp",
			"sftp-service has no per-ssh-session scoping (its sessions are host-scoped only)"))
		api.POST("/sessions/:sessionId/sftp/download", s.notImplemented("sessions.sftp.download",
			"sftp-service has no download/upload action route and no sessionId field in its create request"))
		api.POST("/sessions/:sessionId/sftp/upload", s.notImplemented("sessions.sftp.upload",
			"sftp-service has no download/upload action route and no sessionId field in its create request"))

		// Port forwarding routes. port-forward-service (services/
		// port-forward-service/internal/server/server.go:34-43) is also a
		// flat, HOST-scoped resource (CreatePortForwardRequest has no
		// sessionId field) — listing/creating "for this session" cannot be
		// honestly resolved. Deleting a specific tunnel BY ITS OWN ID needs
		// no session scoping at all: :tunnelId already addresses one exact
		// resource, so that one is a genuine, low-risk fix.
		api.GET("/sessions/:sessionId/tunnels", s.notImplemented("sessions.tunnels",
			"port-forward-service has no per-ssh-session scoping (its forwards are host-scoped only)"))
		api.POST("/sessions/:sessionId/tunnels", s.notImplemented("sessions.tunnels.create",
			"port-forward-service's create request has no sessionId field; forwards are host-scoped only"))
		api.DELETE("/sessions/:sessionId/tunnels/:tunnelId", s.proxyTo("port-forward-service", "/api/v1/forwards/:tunnelId"))

		// Snippet routes. snippet-service (services/snippet-service/
		// internal/server/server.go:34-40) registers PUT (not PATCH) for
		// update, so the gateway's own route verb is corrected to PUT. It
		// has no execute action anywhere — confirmed: no ExecuteSnippet
		// handler exists in the service at all.
		api.GET("/snippets", s.proxyTo("snippet-service", "/api/v1/snippets"))
		api.POST("/snippets", s.proxyTo("snippet-service", "/api/v1/snippets"))
		api.GET("/snippets/:snippetId", s.proxyTo("snippet-service", "/api/v1/snippets/:snippetId"))
		api.PUT("/snippets/:snippetId", s.proxyTo("snippet-service", "/api/v1/snippets/:snippetId"))
		api.DELETE("/snippets/:snippetId", s.proxyTo("snippet-service", "/api/v1/snippets/:snippetId"))
		api.POST("/snippets/:snippetId/execute", s.notImplemented("snippets.execute",
			"snippet-service has no execute handler or route anywhere"))

		// Keychain routes. keychain-service (services/keychain-service/
		// internal/server/server.go:41-47) mounts the SINGULAR "/keychain"
		// resource (not plural "/keychains").
		api.GET("/keychains", s.proxyTo("keychain-service", "/api/v1/keychain"))
		api.POST("/keychains", s.proxyTo("keychain-service", "/api/v1/keychain"))
		api.GET("/keychains/:keyId", s.proxyTo("keychain-service", "/api/v1/keychain/:keyId"))
		api.DELETE("/keychains/:keyId", s.proxyTo("keychain-service", "/api/v1/keychain/:keyId"))

		// Workspace routes. workspace-service (services/workspace-service/
		// internal/server/server.go:90-97) registers PUT (not PATCH) for
		// update, so the gateway's own route verb is corrected to PUT.
		api.GET("/workspaces", s.proxyTo("workspace-service", "/api/v1/workspaces"))
		api.POST("/workspaces", s.proxyTo("workspace-service", "/api/v1/workspaces"))
		api.GET("/workspaces/:workspaceId", s.proxyTo("workspace-service", "/api/v1/workspaces/:workspaceId"))
		api.PUT("/workspaces/:workspaceId", s.proxyTo("workspace-service", "/api/v1/workspaces/:workspaceId"))
		api.DELETE("/workspaces/:workspaceId", s.proxyTo("workspace-service", "/api/v1/workspaces/:workspaceId"))

		// Recording routes. recording-service (services/recording-service/
		// internal/server/server.go:34-40) has no playback or export
		// action — GetPlayback exists but is wired only under
		// terminal-service, keyed by a DIFFERENT (terminal-session) ID, not
		// recording-service's own ID.
		api.GET("/recordings", s.proxyTo("recording-service", "/api/v1/recordings"))
		api.GET("/recordings/:recordingId", s.proxyTo("recording-service", "/api/v1/recordings/:recordingId"))
		api.GET("/recordings/:recordingId/playback", s.notImplemented("recordings.playback",
			"recording-service has no playback route; GetPlayback exists only under terminal-service, keyed by a different (terminal-session) ID"))
		api.POST("/recordings/:recordingId/export", s.notImplemented("recordings.export",
			"recording-service has no export route anywhere"))

		// Audit routes. audit-service (services/audit-service/internal/
		// server/server.go:89-93) mounts the list under
		// "/api/v1/audit/logs".
		api.GET("/audit", s.proxyTo("audit-service", "/api/v1/audit/logs"))

		// Analytics routes. analytics-service (services/analytics-service/
		// internal/server/server.go:34-39) has no "usage" endpoint
		// anywhere (only event CRUD and a stats/event-types breakdown — a
		// different concept from usage/quota metrics).
		api.GET("/analytics/usage", s.notImplemented("analytics.usage",
			"analytics-service has no usage/quota endpoint anywhere"))

		// AI routes. ai-service (services/ai-service/internal/server/
		// server.go:41-45) only has a generic, undifferentiated
		// CreateAIRequest (prompt/context/model — no type/kind field). There
		// is no autocomplete- or explain-specific prompting logic anywhere;
		// silently routing these through the generic endpoint would fake a
		// feature that doesn't exist.
		api.POST("/ai/autocomplete", s.notImplemented("ai.autocomplete",
			"ai-service has no autocomplete-specific endpoint or prompting logic; only a generic, undifferentiated AI request resource exists"))
		api.POST("/ai/explain", s.notImplemented("ai.explain",
			"ai-service has no explain-specific endpoint or prompting logic; only a generic, undifferentiated AI request resource exists"))

		// Notification routes. notification-service (services/
		// notification-service/internal/server/server.go:96-107) mounts
		// under "/api/v1/notifications".
		api.GET("/notifications", s.proxyTo("notification-service", "/api/v1/notifications"))
		api.POST("/notifications", s.proxyTo("notification-service", "/api/v1/notifications"))
		api.POST("/notifications/:notificationId/read", s.proxyTo("notification-service", "/api/v1/notifications/:notificationId/read"))

		// Billing routes. billing-service (services/billing-service/
		// internal/server/server.go:41-49) has no "usage" endpoint
		// anywhere. Its ListSubscriptions (services/billing-service/
		// internal/handler/handler.go:124-168) is now (T12) scoped
		// EXCLUSIVELY to the caller's tenant, derived from the caller's own
		// validated JWT "orgId" claim (see billing-service's authMiddleware
		// + callerOrgID, services/billing-service/internal/server/
		// server.go + internal/handler/handler.go:25-50) — a
		// client-supplied orgId query parameter is no longer accepted as a
		// scoping input at all (model.ListSubscriptionsRequest deliberately
		// carries no OrgID field), so the prior cross-tenant-leak risk this
		// route's 501 originally guarded against is closed. The 501 stays,
		// though, for a different, still-real reason: ListSubscriptions
		// returns a collection (potentially several subscriptions across a
		// tenant's history), and billing-service has no dedicated
		// single-object "my current subscription" endpoint anywhere;
		// proxying this singular-noun route straight to the list would
		// return an array under "current subscription" semantics — a
		// response-shape mismatch, not a security gap. Re-evaluate if
		// billing-service ever adds a real "current subscription" endpoint.
		api.GET("/billing/subscription", s.notImplemented("billing.subscription",
			"billing-service has no self-scoped, single-object \"current subscription\" endpoint; its ListSubscriptions is caller-org-scoped (T12) but returns a list, so mapping straight to it would be a response-shape mismatch, not the security gap this comment previously described"))
		api.GET("/billing/usage", s.notImplemented("billing.usage",
			"billing-service has no usage endpoint anywhere"))
		api.GET("/billing/invoices", s.proxyTo("billing-service", "/api/v1/invoices"))

		// PKI routes. pki-service (services/pki-service/internal/server/
		// server.go:92-99) requires a CA id as the SOLE source of the
		// issuing CA (c.Param("id") only, no body fallback — internal/
		// handler/handler.go:200-205); certificate creation is necessarily
		// CA-scoped. The gateway's flat "/pki/certificates" route had no
		// way to carry that id at all, so the gateway ROUTE itself is
		// corrected (not just the proxied path) to add the required
		// :caId segment.
		api.POST("/pki/ca/:caId/certs", s.proxyTo("pki-service", "/api/v1/pki/ca/:caId/certs"))
		api.POST("/pki/certificates/:certId/revoke", s.proxyTo("pki-service", "/api/v1/pki/certs/:certId/revoke"))

		// Config routes. config-service (services/config-service/internal/
		// server/server.go:88-96) mounts the PLURAL "/api/v1/configs".
		api.GET("/config", s.proxyTo("config-service", "/api/v1/configs"))

		// System routes. health-service (services/health-service/internal/
		// server/server.go:64-68) has no dedicated "/system/status" path,
		// but GetSystemHealth (mounted at "/api/v1/health/system") IS a
		// genuine, real system-wide status rollup (calls
		// checker.CheckAll(), returns an OverallStatus field) — a real,
		// direct match, not a stretch. There is no maintenance-mode concept
		// anywhere in health-service.
		api.GET("/system/status", s.proxyTo("health-service", "/api/v1/health/system"))
		api.GET("/system/maintenance", s.notImplemented("system.maintenance",
			"health-service has no maintenance-mode concept anywhere"))
	}

	// WebSocket terminal endpoint
	r.GET("/ws/terminal/:sessionId", s.terminalWebSocketHandler)

	// SSO routes (no auth required)
	r.GET("/auth/sso/:provider", s.ssoHandler)
	r.GET("/auth/sso/callback", s.ssoCallbackHandler)

	return s
}

func (s *Server) registerUpstreams() {
	s.upstreamsMu.Lock()
	defer s.upstreamsMu.Unlock()

	services := []struct {
		name    string
		address string
	}{
		{"auth-service", "http://auth-service:8080"},
		{"user-service", "http://user-service:8080"},
		{"vault-service", "http://vault-service:8080"},
		{"host-service", "http://host-service:8080"},
		{"ssh-proxy-service", "http://ssh-proxy-service:8080"},
		{"terminal-service", "http://terminal-service:8080"},
		{"sftp-service", "http://sftp-service:8080"},
		{"port-forward-service", "http://port-forward-service:8080"},
		{"snippet-service", "http://snippet-service:8080"},
		{"keychain-service", "http://keychain-service:8080"},
		{"workspace-service", "http://workspace-service:8080"},
		{"collaboration-service", "http://collaboration-service:8080"},
		{"notification-service", "http://notification-service:8080"},
		{"audit-service", "http://audit-service:8080"},
		{"analytics-service", "http://analytics-service:8080"},
		{"ai-service", "http://ai-service:8080"},
		{"recording-service", "http://recording-service:8080"},
		{"pki-service", "http://pki-service:8080"},
		{"org-service", "http://org-service:8080"},
		{"billing-service", "http://billing-service:8080"},
		{"config-service", "http://config-service:8080"},
		{"health-service", "http://health-service:8080"},
		{"container-bridge-service", "http://container-bridge-service:8080"},
		{"helixtrack-bridge-service", "http://helixtrack-bridge-service:8080"},
	}

	for _, svc := range services {
		addr := svc.address
		// Allow per-service upstream address override via environment
		// variable (e.g. HOST_SERVICE_ADDR=http://127.0.0.1:41123) so the
		// gateway can be pointed at a real upstream instance — used both
		// for real deployments (service discovery / config injection)
		// and for integration tests that spin up a real loopback upstream.
		if override := strings.TrimSpace(os.Getenv(envKeyForService(svc.name))); override != "" {
			addr = override
		}
		s.upstreams[svc.name] = &upstreamService{
			Name:    svc.name,
			Address: addr,
			Healthy: true,
		}
	}
}

// envKeyForService derives the environment variable name used to override
// an upstream service's address, e.g. "host-service" -> "HOST_SERVICE_ADDR".
func envKeyForService(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_ADDR"
}

// startHealthChecks launches the background goroutine that periodically
// probes every registered upstream's REAL reachability and calls
// SetHealthy with the genuine outcome (T8-8 anti-bluff fix: before this,
// upstreamService.SetHealthy was defined but never called anywhere in the
// codebase, and every upstream was constructed with Healthy: true
// hardcoded in registerUpstreams and never flipped — /healthz's
// per-service status could therefore never report a service unhealthy,
// no matter how broken the real upstream was).
//
// This does not block server boot: the first probe sweep runs
// asynchronously inside the spawned goroutine, not synchronously here.
func (s *Server) startHealthChecks() {
	s.healthCheckStopCh = make(chan struct{})
	interval := healthCheckIntervalFromEnv()

	s.healthCheckWG.Add(1)
	go func() {
		defer s.healthCheckWG.Done()

		// Run an initial probe sweep immediately so /healthz reflects real
		// upstream reachability as soon as possible after boot, rather
		// than waiting a full interval for the first tick.
		s.probeAllUpstreams()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.probeAllUpstreams()
			case <-s.healthCheckStopCh:
				return
			}
		}
	}()
}

// Stop shuts down the background upstream health-check prober started by
// New/startHealthChecks and blocks until it has genuinely exited. It is
// safe to call more than once (a no-op after the first call) and safe to
// call concurrently with the prober's own probe sweeps (SetHealthy /
// IsHealthy are independently mutex-guarded per upstream). Callers (the
// process's graceful-shutdown path in cmd/gateway-service/main.go, and
// tests via t.Cleanup) MUST call Stop so the goroutine started in New
// does not leak past the Server's intended lifetime.
func (s *Server) Stop() {
	s.healthCheckStopOnce.Do(func() {
		if s.healthCheckStopCh != nil {
			close(s.healthCheckStopCh)
		}
		s.healthCheckWG.Wait()
	})
}

// probeAllUpstreams snapshots the current upstream set under the existing
// upstreamsMu read lock (the same lock every other reader of s.upstreams
// uses, e.g. proxyTo/readinessHandler/fullHealthHandler) and then probes
// every one's real reachability CONCURRENTLY outside the lock. Probing
// concurrently (rather than one-at-a-time) bounds a single sweep's total
// wall-clock cost to roughly healthCheckProbeTimeout regardless of how
// many upstreams are registered, instead of the sum of every individual
// probe's latency — important because a single slow-to-resolve or
// slow-to-connect upstream must never delay how quickly every OTHER
// upstream's state is refreshed. s.httpClient (a single shared
// *http.Client) is safe for concurrent use by multiple goroutines per the
// net/http documentation, and each upstreamService's own Healthy flag is
// independently mutex-guarded (SetHealthy/IsHealthy), so concurrent
// probes introduce no data race (verified under go test -race).
func (s *Server) probeAllUpstreams() {
	s.upstreamsMu.RLock()
	targets := make([]*upstreamService, 0, len(s.upstreams))
	for _, u := range s.upstreams {
		targets = append(targets, u)
	}
	s.upstreamsMu.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(targets))
	for _, u := range targets {
		go func(u *upstreamService) {
			defer wg.Done()
			u.SetHealthy(s.probeUpstreamHealth(u))
		}(u)
	}
	wg.Wait()
}

// probeUpstreamHealth performs a REAL, network-bound reachability check
// against a single upstream: an HTTP GET of its /healthz endpoint (the
// convention every upstream service in this fleet exposes — see e.g. this
// same gateway's own livenessHandler, and the equivalent /healthz
// registrations in auth-service, billing-service, etc.) bounded by
// healthCheckProbeTimeout so one hung upstream can never stall the sweep.
// A transport-level failure (connection refused, timeout, DNS failure —
// i.e. genuinely unreachable) or a 5xx response is reported as unhealthy;
// any response that was actually received with a non-5xx status is
// reported healthy. Nothing here is simulated: an unreachable upstream
// produces a genuine unhealthy result, never a fabricated healthy one.
func (s *Server) probeUpstreamHealth(u *upstreamService) bool {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(u.Address, "/")+"/healthz", nil)
	if err != nil {
		return false
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < http.StatusInternalServerError
}

// Router exposes the underlying engine for testing
func (s *Server) Router() http.Handler {
	return s.router
}

// --- Middleware ---

func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.Recovery()
}

func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		s.logger.Printf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n",
			start.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	allowedOrigins := parseCORSAllowedOrigins()
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if isAllowedOrigin(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-API-Key")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func parseCORSAllowedOrigins() []string {
	env := os.Getenv("CORS_ALLOWED_ORIGINS")
	if env == "" {
		return nil
	}
	var origins []string
	for _, o := range strings.Split(env, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

func isAllowedOrigin(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}

func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip health endpoints
		if strings.HasPrefix(c.Request.URL.Path, "/healthz") ||
			c.Request.URL.Path == "/metrics" ||
			strings.HasPrefix(c.Request.URL.Path, "/auth/sso") {
			c.Next()
			return
		}

		// IP-based rate limit
		clientIP := c.ClientIP()
		if !s.ipLimiter.Allow(clientIP, 1000) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded for IP"})
			c.Abort()
			return
		}

		// Per-endpoint rate limit
		endpoint := c.Request.Method + ":" + c.Request.URL.Path
		if !s.endpointLimiter.Allow(endpoint, 500) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded for endpoint"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (s *Server) jwtValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth endpoints that don't require authentication
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth/") {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "empty token"})
			c.Abort()
			return
		}

		if s.jwtPublicKey == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "JWT validation not configured"})
			c.Abort()
			return
		}

		claims, err := s.validateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("orgID", claims.OrgID)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Next()
	}
}

func (s *Server) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtPublicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

// --- Handlers ---

func (s *Server) livenessHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) readinessHandler(c *gin.Context) {
	// Check upstream services
	allHealthy := true
	services := make(map[string]string)

	s.upstreamsMu.RLock()
	for name, upstream := range s.upstreams {
		if upstream.IsHealthy() {
			services[name] = "healthy"
		} else {
			services[name] = "unhealthy"
			allHealthy = false
		}
	}
	s.upstreamsMu.RUnlock()

	if !allHealthy {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "degraded",
			"services":  services,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"services":  services,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) fullHealthHandler(c *gin.Context) {
	s.upstreamsMu.RLock()
	services := make(map[string]gin.H)
	for name, upstream := range s.upstreams {
		// NOTE (§11.4.6/§11.4.108 anti-bluff): IsHealthy() IS now derived
		// from a real, periodic, timed network probe (T8-8's
		// startHealthChecks/probeUpstreamHealth — a genuine GET of the
		// upstream's /healthz over the network, on GATEWAY_HEALTHCHECK_
		// INTERVAL cadence), not a static flag. What is still intentionally
		// omitted is a per-call numeric "latency" field: the prober records
		// only the healthy/unhealthy outcome, not a timing measurement, so
		// emitting a "latency" number here would still be an invented
		// value. Add real per-probe timing capture before reintroducing
		// that field.
		services[name] = gin.H{
			"status":  "healthy",
			"version": "1.0.0",
		}
		if !upstream.IsHealthy() {
			services[name]["status"] = "unhealthy"
		}
	}
	s.upstreamsMu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": "1.0.0",
		// uptime is the real elapsed wall-clock time since this Server was
		// constructed (s.startTime, set in New()) — genuine process/server
		// uptime, never a hardcoded literal (§11.4/§11.4.108 anti-bluff:
		// this field previously always reported 0).
		"uptime":    time.Since(s.startTime).Seconds(),
		"services":  services,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) metricsHandler(c *gin.Context) {
	// TODO: integrate with Prometheus client
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "# Gateway metrics\n# TODO: implement Prometheus metrics\n")
}

func (s *Server) openapiHandler(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.File("docs/research/mvp/final/implementation/03-api/openapi/openapi.yaml")
}

func (s *Server) terminalWebSocketHandler(c *gin.Context) {
	// TODO: implement WebSocket proxy to terminal-service
	c.JSON(http.StatusNotImplemented, gin.H{"error": "WebSocket terminal not yet implemented"})
}

func (s *Server) ssoHandler(c *gin.Context) {
	provider := c.Param("provider")
	// TODO: implement SSO redirect to identity provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "SSO not yet implemented for provider: " + provider})
}

func (s *Server) ssoCallbackHandler(c *gin.Context) {
	// TODO: implement SSO callback handling
	c.JSON(http.StatusNotImplemented, gin.H{"error": "SSO callback not yet implemented"})
}

// notImplemented returns a handler for a gateway route that has no
// corresponding real upstream capability anywhere in the fleet (re-verified
// per §11.4.6 against every upstream service's current route registration
// and request-model contract, not merely inherited from a prior audit).
// This is the honest alternative to either (a) silently proxying to a
// upstream path that would 404 at the upstream's own router before any
// real handler runs, or (b) fabricating a mapping onto an unrelated
// endpoint that would produce a misleading result. feature is a short,
// stable machine-readable identifier (dot-separated, mirrors the route);
// reason is the human-readable evidence for why no real mapping exists,
// intended to point a future implementer at exactly what would need to be
// built (composes §11.4.3 SKIP-with-reason, applied to routing).
func (s *Server) notImplemented(feature, reason string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "not implemented",
			"feature": feature,
			"reason":  reason,
		})
	}
}

// hopByHopHeaders are per-connection headers that MUST NOT be forwarded
// verbatim across a proxy hop (RFC 7230 §6.1).
var hopByHopHeaders = []string{
	"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
	"Te", "Trailer", "Transfer-Encoding", "Upgrade",
}

// resolvePathParams substitutes gin route params (":name") in the given
// upstream path template with their real values from the current request,
// e.g. "/hosts/:hostId" + c.Param("hostId")=="h-1" -> "/hosts/h-1".
func resolvePathParams(template string, c *gin.Context) (string, bool) {
	if !strings.Contains(template, ":") {
		return template, true
	}
	segments := strings.Split(template, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			val := c.Param(seg[1:])
			if val == "" {
				continue
			}
			// A route param occupies exactly one path segment. Reject values that
			// could break out of it — path/query/fragment separators and traversal
			// sequences — so a caller can never inject into the upstream host, path,
			// or query. Then URL-escape the remainder so it stays confined to its
			// own path component (defends against SSRF / path-injection).
			if strings.ContainsAny(val, "/?#") || strings.Contains(val, "..") {
				return "", false
			}
			segments[i] = url.PathEscape(val)
		}
	}
	return strings.Join(segments, "/"), true
}

// proxyTo returns a handler that performs a REAL reverse-proxy hop to the
// named upstream service over the network: it builds a new outbound HTTP
// request against the upstream's configured address, forwards method,
// headers, query string and body, executes it over a real TCP connection
// via s.httpClient, and streams the upstream's real status/headers/body
// back to the original caller. Nothing here is simulated — an unreachable
// or misbehaving upstream surfaces as a genuine 502, never a fabricated
// 200.
func (s *Server) proxyTo(serviceName, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		s.upstreamsMu.RLock()
		upstream, ok := s.upstreams[serviceName]
		s.upstreamsMu.RUnlock()

		if !ok || !upstream.IsHealthy() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "service unavailable",
				"service": serviceName,
			})
			return
		}

		targetPath, okPath := resolvePathParams(path, c)
		if !okPath {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid path parameter",
				"service": serviceName,
			})
			return
		}
		targetURL := strings.TrimRight(upstream.Address, "/") + targetPath
		if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
			targetURL += "?" + rawQuery
		}

		proxyReq, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":   "failed to build upstream request",
				"service": serviceName,
			})
			return
		}

		proxyReq.Header = c.Request.Header.Clone()
		for _, h := range hopByHopHeaders {
			proxyReq.Header.Del(h)
		}
		proxyReq.ContentLength = c.Request.ContentLength
		proxyReq.Header.Set("X-Forwarded-For", c.ClientIP())
		proxyReq.Header.Set("X-Forwarded-Host", c.Request.Host)
		proxyReq.Header.Set("X-Gateway-Upstream", serviceName)
		proxyReq.Header.Set("X-Request-ID", c.GetString("requestID"))

		resp, err := s.httpClient.Do(proxyReq)
		if err != nil {
			s.logger.Printf("gateway: upstream request to %s (%s) failed: %v", serviceName, targetURL, err)
			c.JSON(http.StatusBadGateway, gin.H{
				"error":   "upstream request failed",
				"service": serviceName,
			})
			return
		}
		defer resp.Body.Close()

		for k, vv := range resp.Header {
			for _, v := range vv {
				c.Writer.Header().Add(k, v)
			}
		}
		c.Writer.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(c.Writer, resp.Body); err != nil {
			s.logger.Printf("gateway: failed streaming upstream response from %s: %v", serviceName, err)
		}
	}
}
