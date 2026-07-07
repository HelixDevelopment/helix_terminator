package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters (OWASP recommended)
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16

	// JWT expiration times
	accessTokenExpiry  = 15 * time.Minute
	refreshTokenExpiry = 7 * 24 * time.Hour
)

// PasswordHasher handles password hashing with Argon2id
type PasswordHasher struct{}

// NewPasswordHasher creates a new password hasher
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

// HashPassword hashes a password using Argon2id
func (h *PasswordHasher) HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Encode salt + hash as base64
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads, encodedSalt, encodedHash), nil
}

// VerifyPassword verifies a password against a hash
func (h *PasswordHasher) VerifyPassword(password, encodedHash string) (bool, error) {
	// Parse the encoded hash using strings.Split for robustness
	// Format: $argon2id$v={version}$m={memory},t={time},p={threads}${salt}${hash}
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format: expected 6 parts, got %d", len(parts))
	}
	if parts[1] != "argon2id" {
		return false, fmt.Errorf("invalid hash algorithm: %s", parts[1])
	}

	var version, memory, time, threads int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("invalid version format: %w", err)
	}
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("invalid parameters format: %w", err)
	}

	saltB64 := parts[4]
	hashB64 := parts[5]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(password), salt, uint32(time), uint32(memory), uint8(threads), uint32(len(expectedHash)))

	// Constant-time comparison
	if len(computedHash) != len(expectedHash) {
		return false, nil
	}
	var result byte
	for i := range computedHash {
		result |= computedHash[i] ^ expectedHash[i]
	}
	return result == 0, nil
}

// JWTManager handles JWT signing and validation with Ed25519
type JWTManager struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewJWTManager creates a new JWT manager with a generated Ed25519 key pair
func NewJWTManager() (*JWTManager, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}
	return &JWTManager{
		privateKey: privateKey,
		publicKey:  privateKey.Public().(ed25519.PublicKey),
	}, nil
}

// NewJWTManagerWithKey creates a new JWT manager with an existing key
func NewJWTManagerWithKey(privateKey ed25519.PrivateKey) *JWTManager {
	return &JWTManager{
		privateKey: privateKey,
		publicKey:  privateKey.Public().(ed25519.PublicKey),
	}
}

// NewJWTManagerFromKey builds a JWTManager whose Ed25519 signing key is
// decoded from privateKeyB64 - the base64 (standard encoding, via
// encoding/base64 StdEncoding, NOT RawStdEncoding/URL-safe) representation
// of exactly ed25519.PrivateKeySize (64) raw bytes. This is the SAME
// encoding convention gateway-service and billing-service already use to
// decode JWT_PUBLIC_KEY (services/gateway-service/internal/server/
// server.go, services/billing-service/internal/server/server.go), so a
// single JWT_PRIVATE_KEY / JWT_PUBLIC_KEY pair generated together decodes
// identically everywhere.
//
// This is the production/persisted-key path: unlike NewJWTManager (which
// generates a fresh, ephemeral, process-local key every call), the key
// here is supplied by the caller - typically read once from the
// JWT_PRIVATE_KEY environment variable, itself sourced from a mounted
// Kubernetes Secret (see infrastructure/kubernetes/base/services/
// auth-service/deployment.yaml and docs/guides/JWT_KEY_PROVISIONING.md).
// Because the SAME key material is used across process restarts and
// across every auth-service replica, a token this manager issues
// validates identically after a restart and against any other service
// that independently loads the paired public key - closing the
// cross-service/cross-restart validation gap NewJWTManager cannot close
// by itself.
//
// If publicKeyB64 is non-empty, it is decoded the same way and MUST
// byte-for-byte match the public key derived from privateKeyB64. A
// provisioned key pair whose distributed public half cannot verify its
// own private half's signatures is a fail-closed configuration error,
// not a warning: every gateway-service/billing-service instance
// validating against that mismatched JWT_PUBLIC_KEY would reject every
// real token this manager ever issues, silently re-creating the exact
// production outage this function exists to prevent.
func NewJWTManagerFromKey(privateKeyB64, publicKeyB64 string) (*JWTManager, error) {
	rawPriv, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("JWT_PRIVATE_KEY is not valid base64: %w", err)
	}
	if len(rawPriv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf(
			"JWT_PRIVATE_KEY has invalid size: expected %d raw bytes (base64-decoded), got %d",
			ed25519.PrivateKeySize, len(rawPriv),
		)
	}
	privateKey := ed25519.PrivateKey(rawPriv)
	derivedPublicKey := privateKey.Public().(ed25519.PublicKey)

	if publicKeyB64 != "" {
		rawPub, err := base64.StdEncoding.DecodeString(publicKeyB64)
		if err != nil {
			return nil, fmt.Errorf("JWT_PUBLIC_KEY is not valid base64: %w", err)
		}
		if len(rawPub) != ed25519.PublicKeySize {
			return nil, fmt.Errorf(
				"JWT_PUBLIC_KEY has invalid size: expected %d raw bytes (base64-decoded), got %d",
				ed25519.PublicKeySize, len(rawPub),
			)
		}
		if !bytes.Equal(rawPub, derivedPublicKey) {
			return nil, fmt.Errorf(
				"JWT_PUBLIC_KEY does not match the public key derived from JWT_PRIVATE_KEY - " +
					"the provisioned key pair is internally inconsistent",
			)
		}
	}

	return &JWTManager{
		privateKey: privateKey,
		publicKey:  derivedPublicKey,
	}, nil
}

// PublicKey returns this manager's Ed25519 public key - the value an
// independent verifier (gateway-service, billing-service, or a test
// simulating either) needs to validate tokens this manager signs
// without ever holding the private key.
func (m *JWTManager) PublicKey() ed25519.PublicKey {
	return m.publicKey
}

// Claims represents custom JWT claims
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

// GenerateAccessToken generates a new access token using the default
// access-token TTL.
func (m *JWTManager) GenerateAccessToken(userID, orgID, email, role, sessionID string, permissions []string) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(accessTokenExpiry)
	tokenString, err := m.GenerateAccessTokenWithExpiry(userID, orgID, email, role, sessionID, permissions, expiresAt)
	if err != nil {
		return "", time.Time{}, err
	}
	return tokenString, expiresAt, nil
}

// GenerateAccessTokenWithExpiry generates a new access token signed by
// this exact manager with an explicit expiry instead of the default
// access-token TTL. GenerateAccessToken is a thin convenience wrapper
// around this function. Exported so callers needing a non-default TTL
// (e.g. a security test proving expired tokens are genuinely rejected
// by this manager's own key) can do so without duplicating the claims
// construction logic.
func (m *JWTManager) GenerateAccessTokenWithExpiry(userID, orgID, email, role, sessionID string, permissions []string, expiresAt time.Time) (string, error) {
	claims := Claims{
		UserID:      userID,
		OrgID:       orgID,
		Email:       email,
		Role:        role,
		Permissions: permissions,
		SessionID:   sessionID,
		TokenType:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			NotBefore: jwt.NewNumericDate(time.Now().UTC()),
			Subject:   userID,
			Issuer:    "helixterminator",
			Audience:  jwt.ClaimStrings{"helixterminator"},
			// A fresh per-token identifier (RFC 7519 "jti"). EdDSA
			// signing is deterministic: two access tokens issued for
			// the same session with every other claim identical (e.g.
			// a login immediately followed by a /refresh within the
			// same wall-clock second, which NumericDate encodes at
			// second granularity) would otherwise serialize to the
			// exact same signed JWT string. A unique jti guarantees
			// every issuance is genuinely distinct regardless of
			// timing.
			ID: uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenString, err := token.SignedString(m.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a new refresh token
func (m *JWTManager) GenerateRefreshToken(userID, sessionID string) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(refreshTokenExpiry)
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Subject:   userID,
			Issuer:    "helixterminator",
			ID:        sessionID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenString, err := token.SignedString(m.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
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

// HashToken creates a SHA-256 hash of a token for storage
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawStdEncoding.EncodeToString(h[:])
}
