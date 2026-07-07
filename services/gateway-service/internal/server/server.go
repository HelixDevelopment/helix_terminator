package server

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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
		// Auth routes (proxy to auth-service)
		api.POST("/auth/register", s.proxyTo("auth-service", "/register"))
		api.POST("/auth/login", s.proxyTo("auth-service", "/login"))
		api.POST("/auth/mfa/verify", s.proxyTo("auth-service", "/mfa/verify"))
		api.POST("/auth/mfa/setup", s.proxyTo("auth-service", "/mfa/setup"))
		api.POST("/auth/refresh", s.proxyTo("auth-service", "/refresh"))
		api.POST("/auth/logout", s.proxyTo("auth-service", "/logout"))

		// User routes (proxy to user-service)
		api.GET("/users/me", s.proxyTo("user-service", "/me"))
		api.PATCH("/users/me", s.proxyTo("user-service", "/me"))
		api.GET("/users/me/sessions", s.proxyTo("user-service", "/sessions"))
		api.DELETE("/users/me/sessions/:sessionId", s.proxyTo("user-service", "/sessions/:sessionId"))
		api.GET("/users/me/preferences", s.proxyTo("user-service", "/preferences"))
		api.PATCH("/users/me/preferences", s.proxyTo("user-service", "/preferences"))

		// Vault routes (proxy to vault-service)
		api.GET("/vaults", s.proxyTo("vault-service", "/vaults"))
		api.POST("/vaults", s.proxyTo("vault-service", "/vaults"))
		api.GET("/vaults/:vaultId", s.proxyTo("vault-service", "/vaults/:vaultId"))
		api.DELETE("/vaults/:vaultId", s.proxyTo("vault-service", "/vaults/:vaultId"))
		api.GET("/vaults/:vaultId/items", s.proxyTo("vault-service", "/vaults/:vaultId/items"))
		api.POST("/vaults/:vaultId/items", s.proxyTo("vault-service", "/vaults/:vaultId/items"))
		api.POST("/vaults/:vaultId/share", s.proxyTo("vault-service", "/vaults/:vaultId/share"))

		// Host routes (proxy to host-service)
		api.GET("/hosts", s.proxyTo("host-service", "/hosts"))
		api.POST("/hosts", s.proxyTo("host-service", "/hosts"))
		api.GET("/hosts/:hostId", s.proxyTo("host-service", "/hosts/:hostId"))
		api.PATCH("/hosts/:hostId", s.proxyTo("host-service", "/hosts/:hostId"))
		api.DELETE("/hosts/:hostId", s.proxyTo("host-service", "/hosts/:hostId"))
		api.POST("/hosts/:hostId/connect", s.proxyTo("host-service", "/hosts/:hostId/connect"))
		api.POST("/hosts/:hostId/test", s.proxyTo("host-service", "/hosts/:hostId/test"))

		// SSH/Session routes (proxy to ssh-proxy-service, terminal-service)
		api.GET("/sessions", s.proxyTo("ssh-proxy-service", "/sessions"))
		api.GET("/sessions/:sessionId", s.proxyTo("ssh-proxy-service", "/sessions/:sessionId"))
		api.DELETE("/sessions/:sessionId", s.proxyTo("ssh-proxy-service", "/sessions/:sessionId"))
		api.GET("/sessions/:sessionId/terminal", s.proxyTo("terminal-service", "/sessions/:sessionId/terminal"))
		api.POST("/sessions/:sessionId/share", s.proxyTo("collaboration-service", "/sessions/:sessionId/share"))
		api.POST("/sessions/:sessionId/record", s.proxyTo("recording-service", "/sessions/:sessionId/record"))

		// SFTP routes
		api.GET("/sessions/:sessionId/sftp", s.proxyTo("sftp-service", "/sessions/:sessionId/sftp"))
		api.POST("/sessions/:sessionId/sftp/download", s.proxyTo("sftp-service", "/sessions/:sessionId/sftp/download"))
		api.POST("/sessions/:sessionId/sftp/upload", s.proxyTo("sftp-service", "/sessions/:sessionId/sftp/upload"))

		// Port forwarding routes
		api.GET("/sessions/:sessionId/tunnels", s.proxyTo("port-forward-service", "/sessions/:sessionId/tunnels"))
		api.POST("/sessions/:sessionId/tunnels", s.proxyTo("port-forward-service", "/sessions/:sessionId/tunnels"))
		api.DELETE("/sessions/:sessionId/tunnels/:tunnelId", s.proxyTo("port-forward-service", "/sessions/:sessionId/tunnels/:tunnelId"))

		// Snippet routes
		api.GET("/snippets", s.proxyTo("snippet-service", "/snippets"))
		api.POST("/snippets", s.proxyTo("snippet-service", "/snippets"))
		api.GET("/snippets/:snippetId", s.proxyTo("snippet-service", "/snippets/:snippetId"))
		api.PATCH("/snippets/:snippetId", s.proxyTo("snippet-service", "/snippets/:snippetId"))
		api.DELETE("/snippets/:snippetId", s.proxyTo("snippet-service", "/snippets/:snippetId"))
		api.POST("/snippets/:snippetId/execute", s.proxyTo("snippet-service", "/snippets/:snippetId/execute"))

		// Keychain routes
		api.GET("/keychains", s.proxyTo("keychain-service", "/keychains"))
		api.POST("/keychains", s.proxyTo("keychain-service", "/keychains"))
		api.GET("/keychains/:keyId", s.proxyTo("keychain-service", "/keychains/:keyId"))
		api.DELETE("/keychains/:keyId", s.proxyTo("keychain-service", "/keychains/:keyId"))

		// Workspace routes
		api.GET("/workspaces", s.proxyTo("workspace-service", "/workspaces"))
		api.POST("/workspaces", s.proxyTo("workspace-service", "/workspaces"))
		api.GET("/workspaces/:workspaceId", s.proxyTo("workspace-service", "/workspaces/:workspaceId"))
		api.PATCH("/workspaces/:workspaceId", s.proxyTo("workspace-service", "/workspaces/:workspaceId"))
		api.DELETE("/workspaces/:workspaceId", s.proxyTo("workspace-service", "/workspaces/:workspaceId"))

		// Recording routes
		api.GET("/recordings", s.proxyTo("recording-service", "/recordings"))
		api.GET("/recordings/:recordingId", s.proxyTo("recording-service", "/recordings/:recordingId"))
		api.GET("/recordings/:recordingId/playback", s.proxyTo("recording-service", "/recordings/:recordingId/playback"))
		api.POST("/recordings/:recordingId/export", s.proxyTo("recording-service", "/recordings/:recordingId/export"))

		// Audit routes
		api.GET("/audit", s.proxyTo("audit-service", "/audit"))

		// Analytics routes
		api.GET("/analytics/usage", s.proxyTo("analytics-service", "/analytics/usage"))

		// AI routes
		api.POST("/ai/autocomplete", s.proxyTo("ai-service", "/ai/autocomplete"))
		api.POST("/ai/explain", s.proxyTo("ai-service", "/ai/explain"))

		// Notification routes
		api.GET("/notifications", s.proxyTo("notification-service", "/notifications"))
		api.POST("/notifications", s.proxyTo("notification-service", "/notifications"))
		api.POST("/notifications/:notificationId/read", s.proxyTo("notification-service", "/notifications/:notificationId/read"))

		// Billing routes
		api.GET("/billing/subscription", s.proxyTo("billing-service", "/billing/subscription"))
		api.GET("/billing/usage", s.proxyTo("billing-service", "/billing/usage"))
		api.GET("/billing/invoices", s.proxyTo("billing-service", "/billing/invoices"))

		// PKI routes
		api.POST("/pki/certificates", s.proxyTo("pki-service", "/pki/certificates"))
		api.POST("/pki/certificates/:certId/revoke", s.proxyTo("pki-service", "/pki/certificates/:certId/revoke"))

		// Config routes
		api.GET("/config", s.proxyTo("config-service", "/config"))

		// System routes
		api.GET("/system/status", s.proxyTo("health-service", "/system/status"))
		api.GET("/system/maintenance", s.proxyTo("health-service", "/system/maintenance"))
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
		services[name] = gin.H{
			"status":  "healthy",
			"latency": 0,
			"version": "1.0.0",
		}
		if !upstream.IsHealthy() {
			services[name]["status"] = "unhealthy"
		}
	}
	s.upstreamsMu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"version":   "1.0.0",
		"uptime":    0,
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

// hopByHopHeaders are per-connection headers that MUST NOT be forwarded
// verbatim across a proxy hop (RFC 7230 §6.1).
var hopByHopHeaders = []string{
	"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
	"Te", "Trailer", "Transfer-Encoding", "Upgrade",
}

// resolvePathParams substitutes gin route params (":name") in the given
// upstream path template with their real values from the current request,
// e.g. "/hosts/:hostId" + c.Param("hostId")=="h-1" -> "/hosts/h-1".
func resolvePathParams(template string, c *gin.Context) string {
	if !strings.Contains(template, ":") {
		return template
	}
	segments := strings.Split(template, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			if val := c.Param(seg[1:]); val != "" {
				segments[i] = val
			}
		}
	}
	return strings.Join(segments, "/")
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

		targetPath := resolvePathParams(path, c)
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
