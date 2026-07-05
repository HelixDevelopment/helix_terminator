package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a platform user
type User struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Email         string     `json:"email" db:"email"`
	PasswordHash  string     `json:"-" db:"password_hash"`
	DisplayName   string     `json:"displayName" db:"display_name"`
	Role          string     `json:"role" db:"role"`
	MFAEnabled    bool       `json:"mfaEnabled" db:"mfa_enabled"`
	MFAMethod     string     `json:"mfaMethod,omitempty" db:"mfa_method"`
	MFASecret     string     `json:"-" db:"mfa_secret"`
	EmailVerified bool       `json:"emailVerified" db:"email_verified"`
	FailedLogins  int        `json:"-" db:"failed_login_attempts"`
	LockedUntil   *time.Time `json:"-" db:"locked_until"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt     *time.Time `json:"-" db:"deleted_at"`
}

// Session represents an active user session
type Session struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"userId" db:"user_id"`
	DeviceID       string     `json:"deviceId,omitempty" db:"device_id"`
	DeviceName     string     `json:"deviceName,omitempty" db:"device_name"`
	DeviceType     string     `json:"deviceType,omitempty" db:"device_type"`
	IPAddress      string     `json:"ipAddress" db:"ip_address"`
	UserAgent      string     `json:"userAgent,omitempty" db:"user_agent"`
	AccessTokenHash  string   `json:"-" db:"access_token_hash"`
	RefreshTokenHash string   `json:"-" db:"refresh_token_hash"`
	ExpiresAt      time.Time  `json:"expiresAt" db:"expires_at"`
	LastActiveAt   time.Time  `json:"lastActiveAt" db:"last_active_at"`
	RevokedAt      *time.Time `json:"-" db:"revoked_at"`
	CreatedAt      time.Time  `json:"createdAt" db:"created_at"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=12"`
	DisplayName string `json:"displayName" binding:"required,max=100"`
	OrgName     string `json:"orgName,omitempty" binding:"omitempty,max=100"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	DeviceID   string `json:"deviceId,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresIn    int64     `json:"expiresIn"`
	User         *User     `json:"user"`
	MFARequired  bool      `json:"mfaRequired,omitempty"`
	MFAMethods   []string  `json:"mfaMethods,omitempty"`
	ChallengeID  string    `json:"challengeId,omitempty"`
}

// MFAVerifyRequest represents an MFA verification request
type MFAVerifyRequest struct {
	ChallengeID string `json:"challengeId" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Method      string `json:"method" binding:"required,oneof=totp fido2 backup_codes"`
}

// MFASetupRequest represents an MFA setup request
type MFASetupRequest struct {
	Method     string `json:"method" binding:"required,oneof=totp fido2"`
	DeviceName string `json:"deviceName,omitempty"`
}

// MFASetupResponse represents the response after MFA setup initiation
type MFASetupResponse struct {
	Secret        string   `json:"secret,omitempty"`
	QRCode        string   `json:"qrCode,omitempty"`
	RecoveryCodes []string `json:"recoveryCodes,omitempty"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// TokenResponse represents the response after token refresh
type TokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	SessionID   string `json:"sessionId,omitempty"`
	AllSessions bool   `json:"allSessions,omitempty"`
}

// ValidateTokenRequest represents a token validation request
type ValidateTokenRequest struct {
	Token  string `json:"token" binding:"required"`
	Scope  string `json:"scope,omitempty"`
}

// ValidateTokenResponse represents the response after token validation
type ValidateTokenResponse struct {
	Valid       bool      `json:"valid"`
	UserID      string    `json:"userId,omitempty"`
	OrgID       string    `json:"orgId,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
}
