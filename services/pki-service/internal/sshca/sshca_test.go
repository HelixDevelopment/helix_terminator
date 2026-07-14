package sshca_test

// RED-first test (Constitution §11.4.43 / §11.4.115) for the SSH certificate
// authority capability. Before internal/sshca exists this file does not
// compile ("no required module provides package .../internal/sshca") — that
// build failure IS the reproduction of the gap: pki-service can issue x509
// TLS certificates but CANNOT issue short-lived SSH certificates, despite
// docs/research/mvp/output/SERVICE_REGISTRY.md §19 mandating exactly that
// ("Issues short-lived SSH certificates (user + host) ... revocation checked
// by SSH Proxy"). After internal/sshca is implemented the same file compiles
// and every test below asserts real, verifiable cryptographic behaviour.
//
// Anti-bluff self-validation (§11.4.107(10)): the VerifyCertificate oracle is
// proven non-tautological by a golden-BAD fixture pair — a certificate signed
// by a DIFFERENT CA, a tampered certificate, an expired certificate, and a
// wrong-principal certificate MUST all be REJECTED. An oracle that passes its
// golden-bad inputs would itself be a bluff gate.

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/helixdevelopment/pki-service/internal/sshca"
)

func mustGenCA(t *testing.T) *sshca.CA {
	t.Helper()
	ca, err := sshca.GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	if ca.PrivateKeyPEM == "" || ca.PublicKeyAuthorized == "" {
		t.Fatalf("GenerateCA returned empty key material: %+v", ca)
	}
	return ca
}

func mustGenSubject(t *testing.T) (privPEM, pubAuth string) {
	t.Helper()
	priv, pub, err := sshca.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	return priv, pub
}

// TestGenerateCA_ProducesUsableEd25519Material verifies the CA keypair is a
// real, parseable Ed25519 SSH key with a SHA256 fingerprint.
func TestGenerateCA_ProducesUsableEd25519Material(t *testing.T) {
	ca := mustGenCA(t)

	if ca.KeyType != "ssh-ed25519" {
		t.Errorf("KeyType = %q, want ssh-ed25519", ca.KeyType)
	}
	if !strings.HasPrefix(ca.Fingerprint, "SHA256:") {
		t.Errorf("Fingerprint = %q, want SHA256: prefix", ca.Fingerprint)
	}
	// CA private key must parse as an ssh.Signer.
	signer, err := ssh.ParsePrivateKey([]byte(ca.PrivateKeyPEM))
	if err != nil {
		t.Fatalf("CA private key does not parse: %v", err)
	}
	// The public authorized_keys line must parse and match the signer's public key.
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(ca.PublicKeyAuthorized))
	if err != nil {
		t.Fatalf("CA public key does not parse: %v", err)
	}
	if ssh.FingerprintSHA256(signer.PublicKey()) != ssh.FingerprintSHA256(pub) {
		t.Error("CA public authorized_keys line does not match its private key")
	}
	if ssh.FingerprintSHA256(pub) != ca.Fingerprint {
		t.Errorf("reported Fingerprint %q != computed %q", ca.Fingerprint, ssh.FingerprintSHA256(pub))
	}
}

// TestSignUserCertificate_GoldenGood: a correctly signed short-lived user
// certificate verifies TRUE against the issuing CA, for its principal, within
// its validity window, with the expected cert type, key id, and default
// user-cert extensions.
func TestSignUserCertificate_GoldenGood(t *testing.T) {
	ca := mustGenCA(t)
	_, subjPub := mustGenSubject(t)

	now := time.Now().UTC()
	req := sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "alice@helixterminator.io",
		Principals:          []string{"alice", "deploy"},
		ValidAfter:          now.Add(-1 * time.Minute),
		ValidBefore:         now.Add(15 * time.Minute), // short-lived
	}
	certAuth, serial, err := sshca.SignUserCertificate(ca.PrivateKeyPEM, req)
	if err != nil {
		t.Fatalf("SignUserCertificate: %v", err)
	}
	if serial == 0 {
		t.Error("serial is 0, want a non-zero assigned serial")
	}

	cert, err := sshca.VerifyCertificate(certAuth, ca.PublicKeyAuthorized, ssh.UserCert, "alice", now)
	if err != nil {
		t.Fatalf("golden-good user cert failed verification: %v", err)
	}
	if cert.CertType != ssh.UserCert {
		t.Errorf("CertType = %d, want UserCert(%d)", cert.CertType, ssh.UserCert)
	}
	if cert.KeyId != "alice@helixterminator.io" {
		t.Errorf("KeyId = %q, want alice@helixterminator.io", cert.KeyId)
	}
	if got := cert.ValidPrincipals; len(got) != 2 || got[0] != "alice" || got[1] != "deploy" {
		t.Errorf("ValidPrincipals = %v, want [alice deploy]", got)
	}
	// Default user-cert extensions must be present (sshd needs permit-pty for
	// an interactive terminal — the core product use-case).
	if _, ok := cert.Extensions["permit-pty"]; !ok {
		t.Error("user cert missing default extension permit-pty")
	}
	if _, ok := cert.Extensions["permit-port-forwarding"]; !ok {
		t.Error("user cert missing default extension permit-port-forwarding")
	}
	// Also valid for the second principal.
	if _, err := sshca.VerifyCertificate(certAuth, ca.PublicKeyAuthorized, ssh.UserCert, "deploy", now); err != nil {
		t.Errorf("cert should be valid for principal deploy: %v", err)
	}
}

// TestSignHostCertificate_GoldenGood: a host certificate verifies as a host
// cert for its hostname principal.
func TestSignHostCertificate_GoldenGood(t *testing.T) {
	ca := mustGenCA(t)
	_, subjPub := mustGenSubject(t)

	now := time.Now().UTC()
	req := sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "web01.helixterminator.io",
		Principals:          []string{"web01.helixterminator.io", "10.0.0.11"},
		ValidBefore:         now.Add(90 * 24 * time.Hour),
	}
	certAuth, _, err := sshca.SignHostCertificate(ca.PrivateKeyPEM, req)
	if err != nil {
		t.Fatalf("SignHostCertificate: %v", err)
	}
	cert, err := sshca.VerifyCertificate(certAuth, ca.PublicKeyAuthorized, ssh.HostCert, "web01.helixterminator.io", now)
	if err != nil {
		t.Fatalf("golden-good host cert failed verification: %v", err)
	}
	if cert.CertType != ssh.HostCert {
		t.Errorf("CertType = %d, want HostCert(%d)", cert.CertType, ssh.HostCert)
	}
	// Host certs must NOT carry user-cert extensions like permit-pty.
	if len(cert.Extensions) != 0 {
		t.Errorf("host cert should have no extensions, got %v", cert.Extensions)
	}
}

// --- Anti-bluff GOLDEN-BAD suite: VerifyCertificate MUST reject all of these.
// If any of these "verify" TRUE the oracle is a bluff (§11.4.107(10)).

func TestVerify_RejectsCertSignedByDifferentCA(t *testing.T) {
	realCA := mustGenCA(t)
	attackerCA := mustGenCA(t)
	_, subjPub := mustGenSubject(t)

	now := time.Now().UTC()
	req := sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "mallory",
		Principals:          []string{"root"},
		ValidBefore:         now.Add(time.Hour),
	}
	// Signed by the ATTACKER's CA...
	certAuth, _, err := sshca.SignUserCertificate(attackerCA.PrivateKeyPEM, req)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// ...but verified against the REAL CA — MUST be rejected.
	if _, err := sshca.VerifyCertificate(certAuth, realCA.PublicKeyAuthorized, ssh.UserCert, "root", now); err == nil {
		t.Fatal("SECURITY: cert signed by a different CA was accepted — verify oracle is a bluff")
	}
}

func TestVerify_RejectsExpiredCert(t *testing.T) {
	ca := mustGenCA(t)
	_, subjPub := mustGenSubject(t)

	now := time.Now().UTC()
	req := sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "alice",
		Principals:          []string{"alice"},
		ValidAfter:          now.Add(-2 * time.Hour),
		ValidBefore:         now.Add(-1 * time.Hour), // already expired
	}
	certAuth, _, err := sshca.SignUserCertificate(ca.PrivateKeyPEM, req)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := sshca.VerifyCertificate(certAuth, ca.PublicKeyAuthorized, ssh.UserCert, "alice", now); err == nil {
		t.Fatal("SECURITY: expired cert was accepted — validity window not enforced")
	}
}

func TestVerify_RejectsWrongPrincipal(t *testing.T) {
	ca := mustGenCA(t)
	_, subjPub := mustGenSubject(t)

	now := time.Now().UTC()
	req := sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "alice",
		Principals:          []string{"alice"},
		ValidBefore:         now.Add(time.Hour),
	}
	certAuth, _, err := sshca.SignUserCertificate(ca.PrivateKeyPEM, req)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// Cert is for "alice"; verifying as "root" must be rejected.
	if _, err := sshca.VerifyCertificate(certAuth, ca.PublicKeyAuthorized, ssh.UserCert, "root", now); err == nil {
		t.Fatal("SECURITY: cert accepted for a principal it was not issued for")
	}
}

func TestVerify_RejectsWrongCertType(t *testing.T) {
	ca := mustGenCA(t)
	_, subjPub := mustGenSubject(t)

	now := time.Now().UTC()
	req := sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "alice",
		Principals:          []string{"alice"},
		ValidBefore:         now.Add(time.Hour),
	}
	// A USER cert...
	certAuth, _, err := sshca.SignUserCertificate(ca.PrivateKeyPEM, req)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// ...must not be accepted as a HOST cert.
	if _, err := sshca.VerifyCertificate(certAuth, ca.PublicKeyAuthorized, ssh.HostCert, "alice", now); err == nil {
		t.Fatal("SECURITY: user cert accepted as host cert — cert-type confusion")
	}
}

func TestVerify_RejectsTamperedCert(t *testing.T) {
	ca := mustGenCA(t)
	_, subjPub := mustGenSubject(t)

	now := time.Now().UTC()
	req := sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "alice",
		Principals:          []string{"alice"},
		ValidBefore:         now.Add(time.Hour),
	}
	certAuth, _, err := sshca.SignUserCertificate(ca.PrivateKeyPEM, req)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// Flip a byte in the base64 body of the authorized_keys line.
	fields := strings.Fields(certAuth)
	if len(fields) < 2 {
		t.Fatalf("unexpected cert format: %q", certAuth)
	}
	b := []byte(fields[1])
	// Mutate a middle character to a different valid base64 char.
	i := len(b) / 2
	if b[i] == 'A' {
		b[i] = 'B'
	} else {
		b[i] = 'A'
	}
	tampered := fields[0] + " " + string(b)
	if _, err := sshca.VerifyCertificate(tampered, ca.PublicKeyAuthorized, ssh.UserCert, "alice", now); err == nil {
		t.Fatal("SECURITY: tampered cert accepted — signature not verified")
	}
}

// TestSign_RejectsGarbageSubjectKey: signing must fail on an unparseable
// subject public key rather than producing a bogus cert.
func TestSign_RejectsGarbageSubjectKey(t *testing.T) {
	ca := mustGenCA(t)
	req := sshca.SignRequest{
		PublicKeyAuthorized: "not-a-real-ssh-key",
		KeyID:               "x",
		Principals:          []string{"x"},
		ValidBefore:         time.Now().Add(time.Hour),
	}
	if _, _, err := sshca.SignUserCertificate(ca.PrivateKeyPEM, req); err == nil {
		t.Fatal("expected error signing with a garbage subject key")
	}
}

// TestSign_DefaultsShortValidityWindow: when ValidBefore is zero the issuer
// applies a safe short-lived default (never an unbounded/forever cert).
func TestSign_DefaultsShortValidityWindow(t *testing.T) {
	ca := mustGenCA(t)
	_, subjPub := mustGenSubject(t)

	now := time.Now().UTC()
	certAuth, _, err := sshca.SignUserCertificate(ca.PrivateKeyPEM, sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "alice",
		Principals:          []string{"alice"},
		// ValidAfter / ValidBefore intentionally zero.
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	cert, err := sshca.VerifyCertificate(certAuth, ca.PublicKeyAuthorized, ssh.UserCert, "alice", now)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	span := time.Unix(int64(cert.ValidBefore), 0).Sub(time.Unix(int64(cert.ValidAfter), 0))
	if span <= 0 || span > 24*time.Hour {
		t.Errorf("default validity span = %s, want a bounded short-lived window (<=24h)", span)
	}
}
