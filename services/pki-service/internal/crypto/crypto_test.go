package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helixdevelopment/pki-service/internal/crypto"
)

func TestGenerateCAKeyPair(t *testing.T) {
	privPEM, pubPEM, err := crypto.GenerateCAKeyPair(2048)
	require.NoError(t, err)
	assert.NotEmpty(t, privPEM)
	assert.NotEmpty(t, pubPEM)
	assert.Contains(t, privPEM, "RSA PRIVATE KEY")
	assert.Contains(t, pubPEM, "RSA PUBLIC KEY")
}

func TestGenerateCertKeyPair(t *testing.T) {
	privPEM, pubPEM, err := crypto.GenerateCertKeyPair(2048)
	require.NoError(t, err)
	assert.NotEmpty(t, privPEM)
	assert.NotEmpty(t, pubPEM)
	assert.Contains(t, privPEM, "RSA PRIVATE KEY")
	assert.Contains(t, pubPEM, "RSA PUBLIC KEY")
}

func TestCreateCACertificate(t *testing.T) {
	privPEM, _, err := crypto.GenerateCAKeyPair(2048)
	require.NoError(t, err)

	certPEM, serial, err := crypto.CreateCACertificate(privPEM, "CN=Test CA,O=Helix,C=US", 365)
	require.NoError(t, err)
	assert.NotEmpty(t, certPEM)
	assert.Contains(t, certPEM, "CERTIFICATE")
	assert.NotNil(t, serial)
	assert.True(t, serial.Sign() >= 0)

	parsed, err := crypto.ParseCertificate(certPEM)
	require.NoError(t, err)
	assert.Equal(t, "Test CA", parsed.Subject.CommonName)
	assert.True(t, parsed.IsCA)
}

func TestCreateCertificate(t *testing.T) {
	caPrivPEM, _, err := crypto.GenerateCAKeyPair(2048)
	require.NoError(t, err)

	caCertPEM, _, err := crypto.CreateCACertificate(caPrivPEM, "CN=Test CA,O=Helix,C=US", 365)
	require.NoError(t, err)

	certPrivPEM, _, err := crypto.GenerateCertKeyPair(2048)
	require.NoError(t, err)

	certPEM, err := crypto.CreateCertificate(certPrivPEM, caPrivPEM, caCertPEM, "CN=Test Cert,O=Helix,C=US", nil, 30)
	require.NoError(t, err)
	assert.NotEmpty(t, certPEM)
	assert.Contains(t, certPEM, "CERTIFICATE")

	parsed, err := crypto.ParseCertificate(certPEM)
	require.NoError(t, err)
	assert.Equal(t, "Test Cert", parsed.Subject.CommonName)
	assert.False(t, parsed.IsCA)
}

func TestParseCertificate_Invalid(t *testing.T) {
	_, err := crypto.ParseCertificate("not a certificate")
	assert.Error(t, err)
}

func TestEncryptDecryptPrivateKey(t *testing.T) {
	privPEM, _, err := crypto.GenerateCAKeyPair(2048)
	require.NoError(t, err)

	password := "super-secret-password-123"
	encrypted, err := crypto.EncryptPrivateKey(privPEM, password)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, privPEM, encrypted)

	decrypted, err := crypto.DecryptPrivateKey(encrypted, password)
	require.NoError(t, err)
	assert.Equal(t, privPEM, decrypted)
}

func TestEncryptDecryptPrivateKey_WrongPassword(t *testing.T) {
	privPEM, _, err := crypto.GenerateCAKeyPair(2048)
	require.NoError(t, err)

	password := "correct-password"
	encrypted, err := crypto.EncryptPrivateKey(privPEM, password)
	require.NoError(t, err)

	_, err = crypto.DecryptPrivateKey(encrypted, "wrong-password")
	assert.Error(t, err)
}

func TestEncryptPrivateKey_EmptyPassword(t *testing.T) {
	privPEM, _, err := crypto.GenerateCAKeyPair(2048)
	require.NoError(t, err)

	_, err = crypto.EncryptPrivateKey(privPEM, "")
	assert.Error(t, err)
}
