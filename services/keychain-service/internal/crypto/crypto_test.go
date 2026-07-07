package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/keychain-service/internal/crypto"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	plaintext := "-----BEGIN OPENSSH PRIVATE KEY-----\nsuper-secret-key-material\n-----END OPENSSH PRIVATE KEY-----"
	key := "super-secret-encryption-key-123"

	encrypted, err := crypto.Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)
	assert.NotContains(t, encrypted, plaintext)

	decrypted, err := crypto.Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_WrongKeyFails(t *testing.T) {
	plaintext := "a secret passphrase"
	encrypted, err := crypto.Encrypt(plaintext, "correct-key")
	require.NoError(t, err)

	_, err = crypto.Decrypt(encrypted, "wrong-key")
	assert.Error(t, err)
}

func TestEncrypt_EmptyKeyFailsClosed(t *testing.T) {
	_, err := crypto.Encrypt("some plaintext", "")
	assert.Error(t, err)
}

func TestDecrypt_EmptyKeyFailsClosed(t *testing.T) {
	_, err := crypto.Decrypt("c29tZS1jaXBoZXJ0ZXh0", "")
	assert.Error(t, err)
}

func TestEncryptDecrypt_EmptyPlaintextRoundTrips(t *testing.T) {
	encrypted, err := crypto.Encrypt("", "some-key")
	require.NoError(t, err)
	assert.Empty(t, encrypted)

	decrypted, err := crypto.Decrypt(encrypted, "some-key")
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestEncrypt_DifferentCiphertextEachTime(t *testing.T) {
	plaintext := "same-plaintext-value"
	key := "same-key"

	c1, err := crypto.Encrypt(plaintext, key)
	require.NoError(t, err)
	c2, err := crypto.Encrypt(plaintext, key)
	require.NoError(t, err)

	// Random salt+nonce per call means encrypting the same plaintext twice
	// MUST produce different ciphertext (otherwise it would leak equality
	// of underlying secrets across rows).
	assert.NotEqual(t, c1, c2)

	d1, err := crypto.Decrypt(c1, key)
	require.NoError(t, err)
	d2, err := crypto.Decrypt(c2, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, d1)
	assert.Equal(t, plaintext, d2)
}

func TestDecrypt_InvalidBase64Fails(t *testing.T) {
	_, err := crypto.Decrypt("not-valid-base64!!!", "some-key")
	assert.Error(t, err)
}

func TestDecrypt_TooShortDataFails(t *testing.T) {
	// Valid base64 but far too short to contain a salt.
	_, err := crypto.Decrypt("YWJj", "some-key")
	assert.Error(t, err)
}
