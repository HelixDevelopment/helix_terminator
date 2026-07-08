package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// GenerateCAKeyPair generates an RSA key pair for a CA and returns PEM-encoded private and public keys.
func GenerateCAKeyPair(bits int) (privPEM, pubPEM string, err error) {
	return generateRSAKeyPair(bits)
}

// GenerateCertKeyPair generates an RSA key pair for a certificate and returns PEM-encoded private and public keys.
func GenerateCertKeyPair(bits int) (privPEM, pubPEM string, err error) {
	return generateRSAKeyPair(bits)
}

func generateRSAKeyPair(bits int) (privPEM, pubPEM string, err error) {
	if bits < 2048 {
		bits = 2048
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privPEM = string(pem.EncodeToMemory(privBlock))

	pubBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	}
	pubPEM = string(pem.EncodeToMemory(pubBlock))

	return privPEM, pubPEM, nil
}

// CreateCACertificate creates a self-signed CA certificate.
func CreateCACertificate(privPEM, subject string, validityDays int) (certPEM string, serial *big.Int, err error) {
	privKey, err := parseRSAPrivateKey(privPEM)
	if err != nil {
		return "", nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               parseSubject(subject),
		NotBefore:             time.Now().UTC(),
		NotAfter:              time.Now().UTC().Add(time.Duration(validityDays) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	certPEM = string(pem.EncodeToMemory(certBlock))
	return certPEM, serialNumber, nil
}

// CreateCertificate creates a certificate signed by a CA.
func CreateCertificate(privPEM, caPrivPEM, caCertPEM, subject string, serial *big.Int, validityDays int) (certPEM string, err error) {
	privKey, err := parseRSAPrivateKey(privPEM)
	if err != nil {
		return "", err
	}

	caPrivKey, err := parseRSAPrivateKey(caPrivPEM)
	if err != nil {
		return "", err
	}

	caCert, err := ParseCertificate(caCertPEM)
	if err != nil {
		return "", err
	}

	if serial == nil {
		serial, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
		if err != nil {
			return "", fmt.Errorf("failed to generate serial number: %w", err)
		}
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      parseSubject(subject),
		NotBefore:    time.Now().UTC(),
		NotAfter:     time.Now().UTC().Add(time.Duration(validityDays) * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, &privKey.PublicKey, caPrivKey)
	if err != nil {
		return "", fmt.Errorf("failed to create certificate: %w", err)
	}

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	certPEM = string(pem.EncodeToMemory(certBlock))
	return certPEM, nil
}

// ParseCertificate parses a PEM-encoded certificate.
func ParseCertificate(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	return cert, nil
}

// EncryptPrivateKey encrypts a PEM-encoded private key using AES-GCM with a password-derived key.
func EncryptPrivateKey(privPEM string, password string) (encrypted string, err error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	key, salt, err := deriveKey(password, nil)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
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

	ciphertext := gcm.Seal(nonce, nonce, []byte(privPEM), nil)

	// Store salt + ciphertext as base64
	result := append(salt, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// DecryptPrivateKey decrypts an AES-GCM encrypted private key.
func DecryptPrivateKey(encrypted string, password string) (privPEM string, err error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(data) < 16 {
		return "", fmt.Errorf("invalid encrypted data")
	}

	salt := data[:16]
	ciphertext := data[16:]

	key, _, err := deriveKey(password, salt)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

func parseRSAPrivateKey(privPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}
	return key, nil
}

func parseSubject(subject string) pkix.Name {
	// Simple subject parser: expects comma-separated key=value pairs
	// e.g., "CN=Test CA,O=Helix,C=US"
	name := pkix.Name{}
	parts := strings.Split(subject, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])
		switch key {
		case "CN":
			name.CommonName = val
		case "O":
			name.Organization = append(name.Organization, val)
		case "OU":
			name.OrganizationalUnit = append(name.OrganizationalUnit, val)
		case "C":
			name.Country = append(name.Country, val)
		case "ST":
			name.Province = append(name.Province, val)
		case "L":
			name.Locality = append(name.Locality, val)
		}
	}
	return name
}

func deriveKey(password string, salt []byte) (key []byte, outSalt []byte, err error) {
	if salt == nil {
		salt = make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}
	key = pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)
	return key, salt, nil
}
