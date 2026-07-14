// Package sshca implements a short-lived SSH certificate authority for
// pki-service.
//
// It is the crypto substance of the capability mandated by
// docs/research/mvp/output/SERVICE_REGISTRY.md §19 ("Certificate Authority
// Service / PKI ... Issues short-lived SSH certificates (user + host), CA
// rotation, revocation checked by SSH Proxy"): the enterprise SSH-access model
// where hosts trust a CA rather than a growing pile of long-lived
// authorized_keys, and every session presents a principal-scoped certificate
// that expires in minutes.
//
// This package is deliberately decoupled from persistence, HTTP, and the
// existing x509 internal/crypto package — it does one thing (mint and verify
// OpenSSH certificates with golang.org/x/crypto/ssh) and is directly
// consumable both by pki-service's handlers and by ssh-proxy-service for
// certificate-based host authentication.
package sshca

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

// Default validity windows. User certificates are short-lived by design
// (SERVICE_REGISTRY §19 "short-lived"); host certificates rotate on a slower
// cadence. Callers may override both via SignRequest.ValidBefore.
const (
	defaultUserValidity = 1 * time.Hour
	defaultHostValidity = 90 * 24 * time.Hour
	// clockSkewBackdate backdates ValidAfter slightly so a freshly issued
	// certificate is not rejected by a verifier whose clock runs a little
	// behind the issuer's.
	clockSkewBackdate = 1 * time.Minute
)

// defaultUserExtensions are the standard OpenSSH permissions granted to an
// interactive user certificate. permit-pty is required for a terminal session
// (the core product use-case); permit-port-forwarding backs the port-forward
// service. Empty string values match OpenSSH's on-the-wire encoding.
var defaultUserExtensions = map[string]string{
	"permit-X11-forwarding":   "",
	"permit-agent-forwarding": "",
	"permit-port-forwarding":  "",
	"permit-pty":              "",
	"permit-user-rc":          "",
}

// CA holds a generated SSH certificate-authority keypair.
type CA struct {
	// PrivateKeyPEM is the CA signing key in OpenSSH PEM form
	// ("OPENSSH PRIVATE KEY"). It MUST be stored encrypted at rest by the
	// caller (pki-service reuses internal/crypto.EncryptPrivateKey).
	PrivateKeyPEM string
	// PublicKeyAuthorized is the CA public key as a single authorized_keys
	// line ("ssh-ed25519 AAAA..."). This is what hosts add to
	// TrustedUserCAKeys / clients add to @cert-authority known_hosts.
	PublicKeyAuthorized string
	// Fingerprint is the SHA256 fingerprint of the CA public key
	// ("SHA256:...", matching `ssh-keygen -lf`).
	Fingerprint string
	// KeyType is the SSH key algorithm, e.g. "ssh-ed25519".
	KeyType string
}

// SignRequest describes a certificate to issue.
type SignRequest struct {
	// PublicKeyAuthorized is the subject's SSH public key in authorized_keys
	// form (the key being certified). Required.
	PublicKeyAuthorized string
	// KeyID is a human-readable identity stamped into the certificate; sshd
	// logs it on every authentication. Recommended (e.g. an email or host FQDN).
	KeyID string
	// Principals are the usernames (user cert) or hostnames (host cert) the
	// certificate is valid for. An empty slice yields a certificate valid for
	// ALL principals — avoid for user certs.
	Principals []string
	// ValidAfter / ValidBefore bound the validity window. Zero ValidAfter
	// backdates by clockSkewBackdate; zero ValidBefore applies the type
	// default (never unbounded).
	ValidAfter  time.Time
	ValidBefore time.Time
	// Serial is the certificate serial. Zero requests a random non-zero serial.
	Serial uint64
	// CriticalOptions and Extensions override the defaults when non-nil.
	CriticalOptions map[string]string
	Extensions      map[string]string
}

// GenerateKeyPair generates a fresh Ed25519 SSH keypair and returns the
// private key in OpenSSH PEM form and the public key as an authorized_keys
// line. Useful for CA generation and for minting ephemeral subject keys to
// certify.
func GenerateKeyPair() (privateKeyPEM string, publicKeyAuthorized string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal ssh private key: %w", err)
	}
	privateKeyPEM = string(pem.EncodeToMemory(block))

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", "", fmt.Errorf("failed to build ssh public key: %w", err)
	}
	publicKeyAuthorized = string(bytes.TrimSpace(ssh.MarshalAuthorizedKey(sshPub)))

	return privateKeyPEM, publicKeyAuthorized, nil
}

// GenerateCA generates a new Ed25519 SSH certificate authority.
func GenerateCA() (*CA, error) {
	privPEM, pubAuth, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubAuth))
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated CA public key: %w", err)
	}
	return &CA{
		PrivateKeyPEM:       privPEM,
		PublicKeyAuthorized: pubAuth,
		Fingerprint:         ssh.FingerprintSHA256(pub),
		KeyType:             pub.Type(),
	}, nil
}

// SignUserCertificate issues an OpenSSH user certificate signed by the CA.
func SignUserCertificate(caPrivateKeyPEM string, req SignRequest) (certAuthorized string, serial uint64, err error) {
	return sign(caPrivateKeyPEM, req, ssh.UserCert)
}

// SignHostCertificate issues an OpenSSH host certificate signed by the CA.
func SignHostCertificate(caPrivateKeyPEM string, req SignRequest) (certAuthorized string, serial uint64, err error) {
	return sign(caPrivateKeyPEM, req, ssh.HostCert)
}

func sign(caPrivateKeyPEM string, req SignRequest, certType uint32) (string, uint64, error) {
	caSigner, err := ssh.ParsePrivateKey([]byte(caPrivateKeyPEM))
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse CA private key: %w", err)
	}

	subjectPub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(req.PublicKeyAuthorized))
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse subject public key: %w", err)
	}

	serial := req.Serial
	if serial == 0 {
		serial, err = randomSerial()
		if err != nil {
			return "", 0, err
		}
	}

	now := time.Now().UTC()
	validAfter := req.ValidAfter
	if validAfter.IsZero() {
		validAfter = now.Add(-clockSkewBackdate)
	}
	validBefore := req.ValidBefore
	if validBefore.IsZero() {
		if certType == ssh.HostCert {
			validBefore = now.Add(defaultHostValidity)
		} else {
			validBefore = now.Add(defaultUserValidity)
		}
	}
	if !validBefore.After(validAfter) {
		return "", 0, fmt.Errorf("invalid validity window: ValidBefore (%s) not after ValidAfter (%s)", validBefore, validAfter)
	}

	extensions := req.Extensions
	if extensions == nil && certType == ssh.UserCert {
		extensions = defaultUserExtensions
	}

	cert := &ssh.Certificate{
		Key:             subjectPub,
		Serial:          serial,
		CertType:        certType,
		KeyId:           req.KeyID,
		ValidPrincipals: req.Principals,
		ValidAfter:      uint64(validAfter.Unix()),
		ValidBefore:     uint64(validBefore.Unix()),
		Permissions: ssh.Permissions{
			CriticalOptions: req.CriticalOptions,
			Extensions:      extensions,
		},
	}

	if err := cert.SignCert(rand.Reader, caSigner); err != nil {
		return "", 0, fmt.Errorf("failed to sign certificate: %w", err)
	}

	return string(bytes.TrimSpace(ssh.MarshalAuthorizedKey(cert))), serial, nil
}

// VerifyCertificate parses certAuthorized and verifies that it is a valid
// OpenSSH certificate of the given certType, signed by the CA identified by
// caPublicKeyAuthorized, valid at time `at`, and (unless principal is empty
// and the certificate is valid for all principals) valid for `principal`.
//
// It is the authoritative oracle: signature, issuing-authority, validity
// window, cert-type, and principal are ALL enforced. It returns the parsed
// certificate on success.
func VerifyCertificate(certAuthorized, caPublicKeyAuthorized string, certType uint32, principal string, at time.Time) (*ssh.Certificate, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(certAuthorized))
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	cert, ok := pk.(*ssh.Certificate)
	if !ok {
		return nil, fmt.Errorf("not an SSH certificate (got %T)", pk)
	}
	if cert.CertType != certType {
		return nil, fmt.Errorf("certificate type mismatch: got %d, want %d", cert.CertType, certType)
	}

	caPub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(caPublicKeyAuthorized))
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA public key: %w", err)
	}
	isAuthority := func(auth ssh.PublicKey) bool {
		return bytes.Equal(auth.Marshal(), caPub.Marshal())
	}

	// CRITICAL: ssh.CertChecker.CheckCert verifies the certificate signature
	// against cert.SignatureKey (the key embedded in the certificate) and the
	// validity window + principal — but it does NOT check that SignatureKey is
	// a TRUSTED authority (IsUserAuthority/IsHostAuthority are only consulted
	// by Authenticate/CheckHostKey during a live handshake, verified by reading
	// golang.org/x/crypto/ssh/certs.go CheckCert). Without this explicit
	// authority check, a self-consistent certificate signed by ANY CA — an
	// attacker's — would verify TRUE. This defect was caught by the golden-bad
	// self-validation test TestVerify_RejectsCertSignedByDifferentCA
	// (§11.4.107(10)); the guard below closes it.
	if !isAuthority(cert.SignatureKey) {
		return nil, fmt.Errorf("certificate signed by unrecognized authority (not the trusted CA)")
	}

	checker := &ssh.CertChecker{
		Clock:           func() time.Time { return at },
		IsUserAuthority: isAuthority,
		IsHostAuthority: func(auth ssh.PublicKey, _ string) bool { return isAuthority(auth) },
	}
	if err := checker.CheckCert(principal, cert); err != nil {
		return nil, fmt.Errorf("certificate verification failed: %w", err)
	}
	return cert, nil
}

func randomSerial() (uint64, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, fmt.Errorf("failed to generate serial: %w", err)
	}
	serial := binary.BigEndian.Uint64(buf[:])
	if serial == 0 {
		serial = 1
	}
	return serial, nil
}
