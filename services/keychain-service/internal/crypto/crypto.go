// Package crypto provides encryption-at-rest for secret material stored by
// keychain-service (private_key, passphrase).
//
// This mirrors pki-service's internal/crypto EncryptPrivateKey /
// DecryptPrivateKey pattern (§11.4.74 extend-don't-reimplement, T10):
// AES-256-GCM (authenticated encryption) with a PBKDF2-SHA256-derived key,
// salt || nonce || ciphertext+tag concatenated and base64-encoded as a
// single string so it fits the existing TEXT columns without a schema
// migration. The encryption key itself is never hardcoded — callers MUST
// supply it (production: from the KEYCHAIN_ENCRYPTION_KEY environment
// variable, §11.4.10; tests: a test-only key).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// pbkdf2Iterations matches pki-service's internal/crypto convention.
	pbkdf2Iterations = 100000
	// keyLenBytes is 32 bytes, i.e. AES-256.
	keyLenBytes = 32
	// saltLenBytes is the PBKDF2 salt length, matching pki-service.
	saltLenBytes = 16
)

// Encrypt encrypts plaintext with AES-256-GCM under a key derived from key
// via PBKDF2-HMAC-SHA256. The result is base64(salt || nonce ||
// ciphertext+tag). Returns an error (fail-closed, never a silent plaintext
// fallback) if key is empty. An empty plaintext short-circuits to an empty
// ciphertext — there is nothing sensitive to protect for an unset optional
// field (e.g. no passphrase), and it lets Decrypt round-trip it back to
// "" symmetrically without invoking AES-GCM on zero-length input.
func Encrypt(plaintext, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("encryption key cannot be empty")
	}
	if plaintext == "" {
		return "", nil
	}

	salt := make([]byte, saltLenBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	derivedKey := pbkdf2.Key([]byte(key), salt, pbkdf2Iterations, keyLenBytes, sha256.New)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	result := append(salt, sealed...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt reverses Encrypt. Returns an error (fail-closed) if key is
// empty. An empty ciphertext decrypts to "" (the symmetric counterpart of
// Encrypt's empty-plaintext short-circuit).
func Decrypt(ciphertext, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("encryption key cannot be empty")
	}
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	if len(data) < saltLenBytes {
		return "", fmt.Errorf("invalid encrypted data")
	}

	salt := data[:saltLenBytes]
	sealed := data[saltLenBytes:]

	derivedKey := pbkdf2.Key([]byte(key), salt, pbkdf2Iterations, keyLenBytes, sha256.New)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	if len(sealed) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, encrypted := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
