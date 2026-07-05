package crypto_test

import (
	"strings"
	"testing"

	"github.com/helixdevelopment/auth-service/internal/crypto"
)

func TestPasswordHasherHashAndVerify(t *testing.T) {
	h := crypto.NewPasswordHasher()
	password := "super-secret-password-123!"
	hash, err := h.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected argon2id prefix, got: %s", hash)
	}

	ok, err := h.VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword error for correct password: %v", err)
	}
	if !ok {
		t.Fatal("expected VerifyPassword to succeed for correct password")
	}

	ok, err = h.VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword error for wrong password: %v", err)
	}
	if ok {
		t.Fatal("expected VerifyPassword to fail for wrong password")
	}
}

func TestJWTManagerGenerateAndValidateAccessToken(t *testing.T) {
	jm, err := crypto.NewJWTManager()
	if err != nil {
		t.Fatalf("NewJWTManager failed: %v", err)
	}

	token, expiresAt, err := jm.GenerateAccessToken("user-123", "org-456", "user@example.com", "admin", "sess-1", []string{"read", "write"})
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.IsZero() {
		t.Fatal("expected non-zero expiry")
	}

	parsed, err := jm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if parsed.UserID != "user-123" {
		t.Fatalf("expected UserID user-123, got %s", parsed.UserID)
	}
	if parsed.OrgID != "org-456" {
		t.Fatalf("expected OrgID org-456, got %s", parsed.OrgID)
	}
	if parsed.Role != "admin" {
		t.Fatalf("expected Role admin, got %s", parsed.Role)
	}
	if parsed.Email != "user@example.com" {
		t.Fatalf("expected Email user@example.com, got %s", parsed.Email)
	}
}

func TestJWTManagerGenerateAndValidateRefreshToken(t *testing.T) {
	jm, err := crypto.NewJWTManager()
	if err != nil {
		t.Fatalf("NewJWTManager failed: %v", err)
	}

	token, expiresAt, err := jm.GenerateRefreshToken("user-123", "sess-1")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.IsZero() {
		t.Fatal("expected non-zero expiry")
	}

	parsed, err := jm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if parsed.UserID != "user-123" {
		t.Fatalf("expected UserID user-123, got %s", parsed.UserID)
	}
	if parsed.TokenType != "refresh" {
		t.Fatalf("expected TokenType refresh, got %s", parsed.TokenType)
	}
}

func TestHashToken(t *testing.T) {
	token := "my-refresh-token-value"
	hash := crypto.HashToken(token)
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == token {
		t.Fatal("hash should not equal raw token")
	}
}
