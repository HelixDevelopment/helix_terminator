package sshca_test

// Independent-oracle captured evidence (Constitution §11.4.107(8) metamorphic /
// independent oracle + §11.4.5 / §11.4.69 captured evidence): certificates
// minted by internal/sshca are cross-checked with the SYSTEM `ssh-keygen -L`
// tool. This proves the certificates are well-formed and correctly scoped
// according to OpenSSH itself — a different implementation from the one under
// test — not merely according to our own VerifyCertificate oracle.
//
// SKIP-with-reason (§11.4.3) when ssh-keygen is unavailable on the host.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helixdevelopment/pki-service/internal/sshca"
)

func TestEvidence_SSHKeygenCrossChecksIssuedCerts(t *testing.T) {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("SKIP (§11.4.3): ssh-keygen not on PATH — independent-oracle cross-check unavailable")
	}

	ca, err := sshca.GenerateCA()
	if err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	_, subjPub, err := sshca.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	dir := t.TempDir()

	inspect := func(label, certAuth string) string {
		f := filepath.Join(dir, label+"-cert.pub")
		if err := os.WriteFile(f, []byte(certAuth+"\n"), 0o600); err != nil {
			t.Fatalf("write cert: %v", err)
		}
		out, err := exec.Command(keygen, "-L", "-f", f).CombinedOutput()
		if err != nil {
			t.Fatalf("ssh-keygen -L failed on %s cert: %v\n%s", label, err, out)
		}
		return string(out)
	}

	now := time.Now().UTC()

	// --- USER certificate ---
	userCert, userSerial, err := sshca.SignUserCertificate(ca.PrivateKeyPEM, sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "alice@helixterminator.io",
		Principals:          []string{"alice", "deploy"},
		ValidAfter:          now.Add(-1 * time.Minute),
		ValidBefore:         now.Add(15 * time.Minute),
	})
	if err != nil {
		t.Fatalf("SignUserCertificate: %v", err)
	}
	userOut := inspect("user", userCert)
	t.Logf("=== INDEPENDENT ORACLE: ssh-keygen -L (USER cert, serial %d) ===\n%s", userSerial, userOut)

	for _, want := range []string{
		"user certificate",
		"alice@helixterminator.io", // KeyID
		"alice",                    // principal
		"deploy",                   // principal
		"ssh-ed25519-cert-v01@openssh.com",
	} {
		if !strings.Contains(userOut, want) {
			t.Errorf("ssh-keygen output for user cert missing %q", want)
		}
	}

	// --- HOST certificate ---
	hostCert, hostSerial, err := sshca.SignHostCertificate(ca.PrivateKeyPEM, sshca.SignRequest{
		PublicKeyAuthorized: subjPub,
		KeyID:               "web01.helixterminator.io",
		Principals:          []string{"web01.helixterminator.io"},
		ValidBefore:         now.Add(90 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("SignHostCertificate: %v", err)
	}
	hostOut := inspect("host", hostCert)
	t.Logf("=== INDEPENDENT ORACLE: ssh-keygen -L (HOST cert, serial %d) ===\n%s", hostSerial, hostOut)

	if !strings.Contains(hostOut, "host certificate") {
		t.Errorf("ssh-keygen did not classify the host cert as a host certificate:\n%s", hostOut)
	}
	if !strings.Contains(hostOut, "web01.helixterminator.io") {
		t.Errorf("ssh-keygen output for host cert missing hostname principal")
	}
}
