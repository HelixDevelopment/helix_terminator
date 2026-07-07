package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/helixdevelopment/auth-service/internal/crypto"
	"github.com/helixdevelopment/auth-service/internal/model"
	"github.com/helixdevelopment/auth-service/internal/repository"
)

// Handler holds auth service handlers
type Handler struct {
	repo          *repository.Repository
	hasher        *crypto.PasswordHasher
	jwtManager    *crypto.JWTManager
	mfaChallenges *mfaChallengeStore
}

// New returns a new Handler with dependencies
func New(repo *repository.Repository, jwtManager *crypto.JWTManager) *Handler {
	return &Handler{
		repo:          repo,
		hasher:        crypto.NewPasswordHasher(),
		jwtManager:    jwtManager,
		mfaChallenges: newMFAChallengeStore(),
	}
}

// IsAccessTokenActive reports whether accessToken is bound to a real,
// non-revoked session row in the database - i.e. whether it has NOT
// been invalidated by a /logout. JWT signature validation alone is
// stateless and cannot detect revocation (a logged-out-but-unexpired
// token still verifies cryptographically), so the jwt validation
// middleware calls this as a second, stateful gate. When the handler
// has no repository configured (in-memory/degraded mode - see
// server.New's existing no-DATABASE_URL fallback), it reports true so
// JWT signature validation remains the sole gate in that mode.
func (h *Handler) IsAccessTokenActive(ctx context.Context, accessToken string) bool {
	if h.repo == nil {
		return true
	}
	_, err := h.repo.GetSessionByTokenHash(ctx, crypto.HashToken(accessToken))
	return err == nil
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	exists, err := h.repo.EmailExists(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check email availability"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// Hash password
	passwordHash, err := h.hasher.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Create user
	user := &model.User{
		ID:            uuid.New(),
		Email:         req.Email,
		PasswordHash:  passwordHash,
		DisplayName:   req.DisplayName,
		Role:          "user",
		MFAEnabled:    false,
		EmailVerified: false,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := h.repo.CreateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Generate tokens
	sessionID := uuid.New().String()
	accessToken, _, err := h.jwtManager.GenerateAccessToken(
		user.ID.String(), "", user.Email, user.Role, sessionID, nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	refreshToken, expiresAt, err := h.jwtManager.GenerateRefreshToken(user.ID.String(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// Create session
	session := &model.Session{
		ID:               uuid.MustParse(sessionID),
		UserID:           user.ID,
		AccessTokenHash:  crypto.HashToken(accessToken),
		RefreshTokenHash: crypto.HashToken(refreshToken),
		ExpiresAt:        expiresAt,
		LastActiveAt:     time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
	}
	if err := h.repo.CreateSession(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusCreated, model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * time.Minute.Seconds()),
		User:         user,
	})
}

// Login handles user authentication
func (h *Handler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user by email
	user, err := h.repo.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Check if account is locked
	if user.LockedUntil != nil && time.Now().UTC().Before(*user.LockedUntil) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":       "account is locked due to too many failed attempts",
			"lockedUntil": user.LockedUntil,
		})
		return
	}

	// Verify password
	valid, err := h.hasher.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !valid {
		// Increment failed logins
		_ = h.repo.IncrementFailedLogins(c.Request.Context(), user.ID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Reset failed logins on successful authentication
	_ = h.repo.ResetFailedLogins(c.Request.Context(), user.ID)

	// Check if MFA is required
	if user.MFAEnabled {
		// The challenge binds this specific user to the challengeId
		// returned below, so the unauthenticated POST /mfa/verify call
		// that completes login can resolve the right user WITHOUT a
		// bearer token (none exists yet at this point in the flow).
		challengeID := h.mfaChallenges.create(user.ID)
		methods := []string{"totp"}
		if user.MFAMethod == "fido2" {
			methods = []string{"fido2"}
		}
		c.JSON(http.StatusOK, model.AuthResponse{
			MFARequired: true,
			MFAMethods:  methods,
			ChallengeID: challengeID,
			User:        user,
		})
		return
	}

	// Generate tokens
	sessionID := uuid.New().String()
	accessToken, _, err := h.jwtManager.GenerateAccessToken(
		user.ID.String(), "", user.Email, user.Role, sessionID, nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	refreshToken, expiresAt, err := h.jwtManager.GenerateRefreshToken(user.ID.String(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// Create session
	session := &model.Session{
		ID:               uuid.MustParse(sessionID),
		UserID:           user.ID,
		DeviceID:         req.DeviceID,
		DeviceName:       req.DeviceName,
		IPAddress:        c.ClientIP(),
		UserAgent:        c.Request.UserAgent(),
		AccessTokenHash:  crypto.HashToken(accessToken),
		RefreshTokenHash: crypto.HashToken(refreshToken),
		ExpiresAt:        expiresAt,
		LastActiveAt:     time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
	}
	if err := h.repo.CreateSession(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusOK, model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * time.Minute.Seconds()),
		User:         user,
	})
}

// VerifyMFA handles MFA verification
func (h *Handler) VerifyMFA(c *gin.Context) {
	var req model.MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve the user from the login-issued MFA challenge, NOT from
	// the request context. This route is deliberately reachable with no
	// bearer token (see server.New): Login() withholds real tokens for
	// an MFA-enabled user until this call succeeds, so a
	// c.Get("userID")-style lookup here can never be populated - that
	// was the root-cause bug (same context-userID class as the
	// pre-fix /logout bug), and moving this route behind the
	// auth-required middleware would "fix" it by making MFA-enabled
	// login permanently impossible instead. The challengeId is the
	// correct identity binding for this step of the flow.
	userID, ok := h.mfaChallenges.lookup(req.ChallengeID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired MFA challenge"})
		return
	}

	user, err := h.repo.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	if !user.MFAEnabled || user.MFASecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA not enabled for this user"})
		return
	}

	// Algorithm MUST match what SetupMFA generated the key with
	// (otp.AlgorithmSHA1 = 0). The literal "1" this used to read here
	// is otp.AlgorithmSHA256, not SHA1 despite its stale comment - a
	// second genuine, pre-existing defect (alongside the MFASecret
	// encoding bug fixed in SetupMFA) that made a correct TOTP code
	// from a real authenticator app fail validation.
	valid, err := totp.ValidateCustom(req.Code, user.MFASecret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    6,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid MFA code"})
		return
	}

	// Single-use: this challenge has now been redeemed for a token
	// pair and must not be replayable.
	h.mfaChallenges.consume(req.ChallengeID)

	// Generate tokens after successful MFA
	sessionID := uuid.New().String()
	accessToken, _, err := h.jwtManager.GenerateAccessToken(
		user.ID.String(), "", user.Email, user.Role, sessionID, nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	refreshToken, expiresAt, err := h.jwtManager.GenerateRefreshToken(user.ID.String(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// Create session
	session := &model.Session{
		ID:               uuid.MustParse(sessionID),
		UserID:           user.ID,
		AccessTokenHash:  crypto.HashToken(accessToken),
		RefreshTokenHash: crypto.HashToken(refreshToken),
		ExpiresAt:        expiresAt,
		LastActiveAt:     time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
	}
	if err := h.repo.CreateSession(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusOK, model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * time.Minute.Seconds()),
		User:         user,
	})
}

// SetupMFA handles MFA setup initiation
func (h *Handler) SetupMFA(c *gin.Context) {
	var req model.MFASetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from context
	userIDStr, _ := c.Get("userID")
	if userIDStr == nil || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	user, err := h.repo.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Generate cryptographically random TOTP seed material.
	seed, err := generateRandomTOTPSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate TOTP secret"})
		return
	}

	// Derive the actual TOTP key (QR code + otpauth:// URI) from that
	// seed. key.Secret() - NOT the raw seed above - is the base32-
	// encoded value the URI/QR code embeds and that a real
	// authenticator app (Google Authenticator, Authy, etc.) will
	// actually use to compute codes.
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "HelixTerminator",
		AccountName: user.Email,
		Secret:      []byte(seed),
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate TOTP key"})
		return
	}

	// Generate real recovery codes
	recoveryCodes, err := generateRecoveryCodes(10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate recovery codes"})
		return
	}

	// Persist key.Secret() (the base32 QR-code secret), not the raw
	// pre-encoding seed: user.MFASecret MUST match what the user's
	// authenticator app is actually using, or VerifyMFA's
	// totp.ValidateCustom call can never validate a real code (it
	// requires a base32 secret, and the raw seed is not one) - this was
	// a genuine, separate, pre-existing production defect this fix
	// closes (§11.4.102/§11.4.108), discovered while proving the
	// context-userID fix with a real end-to-end MFA test.
	user.MFASecret = key.Secret()
	user.MFAEnabled = true
	user.MFAMethod = "totp"
	if err := h.repo.UpdateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save MFA setup"})
		return
	}

	c.JSON(http.StatusOK, model.MFASetupResponse{
		Secret:        key.Secret(),
		QRCode:        key.URL(),
		RecoveryCodes: recoveryCodes,
	})
}

// generateRandomTOTPSecret generates a cryptographically random TOTP secret.
func generateRandomTOTPSecret() (string, error) {
	// Generate 160 bits (20 bytes) of randomness for a standard TOTP secret
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// generateRecoveryCodes generates cryptographically random recovery codes.
func generateRecoveryCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("failed to read random bytes: %w", err)
		}
		codes[i] = base64.RawStdEncoding.EncodeToString(b)
	}
	return codes, nil
}

// RefreshToken handles token refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	var req model.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate refresh token
	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	if claims.TokenType != "refresh" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
		return
	}

	// Get user
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := h.repo.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// Generate new access token
	newAccessToken, _, err := h.jwtManager.GenerateAccessToken(
		user.ID.String(), "", user.Email, user.Role, claims.SessionID, nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	// Rebind the session's revocation-lookup key to the freshly-minted
	// access token so IsAccessTokenActive keeps recognising this
	// session by whichever access token the client is now presenting.
	// If the session was already revoked (e.g. a prior /logout), this
	// fails and the refresh is correctly rejected rather than minting a
	// working token for a logged-out session.
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}
	if err := h.repo.UpdateSessionAccessTokenHash(c.Request.Context(), sessionID, crypto.HashToken(newAccessToken)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session revoked or not found"})
		return
	}

	c.JSON(http.StatusOK, model.TokenResponse{
		AccessToken: newAccessToken,
		ExpiresIn:   int64(15 * time.Minute.Seconds()),
	})
}

// Logout handles session revocation
func (h *Handler) Logout(c *gin.Context) {
	var req model.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by JWT middleware)
	userIDStr, _ := c.Get("userID")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	if req.AllSessions {
		if err := h.repo.RevokeAllUserSessions(c.Request.Context(), userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke sessions"})
			return
		}
	} else if req.SessionID != "" {
		sessionID, err := uuid.Parse(req.SessionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
			return
		}
		if err := h.repo.RevokeSession(c.Request.Context(), sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke session"})
			return
		}
	}

	c.Status(http.StatusNoContent)
}

// ValidateToken handles token validation
func (h *Handler) ValidateToken(c *gin.Context) {
	var req model.ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := h.jwtManager.ValidateToken(req.Token)
	if err != nil {
		c.JSON(http.StatusOK, model.ValidateTokenResponse{Valid: false})
		return
	}

	c.JSON(http.StatusOK, model.ValidateTokenResponse{
		Valid:       true,
		UserID:      claims.UserID,
		OrgID:       claims.OrgID,
		Permissions: claims.Permissions,
		ExpiresAt:   claims.ExpiresAt.Time,
	})
}

// HealthCheck returns service health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "auth-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck returns readiness status. Unlike HealthCheck
// (liveness - "is the process up"), readiness reports whether the
// service can genuinely serve traffic, which for auth-service means a
// reachable database: an auth service that cannot check credentials
// or sessions against its database is not ready, even though its
// process is alive. Reports 503 + ready:false the moment the database
// is unreachable, closing the T8-6 bluff where this handler previously
// returned an unconditional ready:true without ever checking the DB
// (a crashed-DB service still reported ready, defeating
// orchestrator/k8s health gating on this security-critical service).
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if h.repo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":     false,
			"service":   "auth-service",
			"reason":    "database repository not configured",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ready":     false,
			"service":   "auth-service",
			"reason":    "database unreachable: " + err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"service":   "auth-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
