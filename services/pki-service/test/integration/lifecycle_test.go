//go:build integration

// Package integration_test drives the REAL certificate lifecycle of
// pki-service — issue -> chain-verify -> revoke -> re-verify-rejected —
// through the service's real HTTP handlers, backed by a REAL rootless
// Podman-hosted PostgreSQL 17.2 instance with the project's real
// migrations applied. No mocks, no fakes, no in-memory stand-ins
// (Constitution §11.4.27 — every test type beyond unit tests MUST
// interact with the real, fully implemented system).
//
// Run with:
//
//	GOWORK=off GOMAXPROCS=2 go test -tags integration -p 2 ./test/integration/...
//
// Requires rootless `podman` on PATH (§11.4.161). SKIPs with a reason
// (§11.4.3) when podman is unavailable — never silently PASSes.
package integration_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/helixdevelopment/pki-service/internal/model"
	"github.com/helixdevelopment/pki-service/internal/server"
)

// pgContainer wraps a real rootless-Podman-hosted Postgres instance used
// as the pki-service's real backing store for this test.
type pgContainer struct {
	name string
	dsn  string
	pool *pgxpool.Pool
}

const podmanBin = "podman"

// startPostgresContainer boots a real postgres:17.2 container (rootless
// Podman, plain default userns — §11.4.161: no :z / --userns=keep-id /
// label=disable), waits for it to accept connections, and applies the
// project's real migrations/*.sql. SKIPs (never fake-PASSes) if podman
// is unavailable, per §11.4.3.
func startPostgresContainer(t *testing.T) *pgContainer {
	t.Helper()

	if _, err := exec.LookPath(podmanBin); err != nil {
		t.Skipf("SKIP (topology_unsupported, §11.4.3): rootless podman not found on PATH — real-persistence cert-lifecycle integration test requires it: %v", err)
	}

	name := fmt.Sprintf("pki-svc-it-%d", time.Now().UnixNano())
	runCmd := exec.Command(podmanBin, "run", "-d", "--rm",
		"--name", name,
		"-e", "POSTGRES_PASSWORD=pkiintegrationtest",
		"-e", "POSTGRES_DB=pki_integration",
		"-p", "127.0.0.1::5432",
		"docker.io/library/postgres:17.2",
	)
	out, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("FAILED: podman run postgres:17.2: %v\n%s", err, out)
	}

	ok := false
	defer func() {
		if !ok {
			_ = exec.Command(podmanBin, "stop", "-t", "5", name).Run()
		}
	}()

	port := waitForMappedPort(t, name, "5432/tcp", 30*time.Second)
	dsn := fmt.Sprintf("postgres://postgres:pkiintegrationtest@127.0.0.1:%s/pki_integration?sslmode=disable", port)

	pool := waitForPoolReady(t, dsn, 30*time.Second)
	applyMigrations(t, pool)

	ok = true
	return &pgContainer{name: name, dsn: dsn, pool: pool}
}

// Stop tears down the container and closes the pool. Called via defer
// on every exit path (§11.4.14 — every test MUST leave the target in a
// quiescent state).
func (c *pgContainer) Stop(t *testing.T) {
	t.Helper()
	if c.pool != nil {
		c.pool.Close()
	}
	out, err := exec.Command(podmanBin, "stop", "-t", "5", c.name).CombinedOutput()
	if err != nil {
		t.Logf("warning: failed to stop integration postgres container %s: %v\n%s", c.name, err, out)
	}
}

func waitForMappedPort(t *testing.T, name, portSpec string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command(podmanBin, "port", name, portSpec).Output()
		if err == nil {
			line := strings.TrimSpace(string(out))
			if line != "" {
				parts := strings.Split(line, ":")
				if len(parts) == 2 && parts[1] != "" {
					return parts[1]
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("FAILED: timed out after %s waiting for podman port mapping of %s %s", timeout, name, portSpec)
	return ""
}

func waitForPoolReady(t *testing.T, dsn string, timeout time.Duration) *pgxpool.Pool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pool, err := pgxpool.New(context.Background(), dsn)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			pingErr := pool.Ping(pingCtx)
			cancel()
			if pingErr == nil {
				return pool
			}
			lastErr = pingErr
			pool.Close()
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("FAILED: postgres never became ready within %s: %v", timeout, lastErr)
	return nil
}

// applyMigrations applies every real migrations/*.sql file in lexical
// order against the real database — the same schema the production
// service depends on.
func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("FAILED: could not determine test file path to locate migrations directory")
	}
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")

	matches, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		t.Fatalf("FAILED: globbing migrations dir %s: %v", migrationsDir, err)
	}
	if len(matches) == 0 {
		t.Fatalf("FAILED: no *.sql migration files found under %s", migrationsDir)
	}
	sort.Strings(matches)

	for _, path := range matches {
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("FAILED: reading migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(sqlBytes)); err != nil {
			t.Fatalf("FAILED: applying real migration %s against real postgres: %v", path, err)
		}
	}
}

// TestCertificateLifecycle_RealPersistence drives the FULL certificate
// lifecycle through pki-service's real service layer (real Gin router,
// real handlers, real repository) against REAL Postgres persistence:
//
//	issue a cert -> verify chain succeeds -> revoke ->
//	verify a subsequent verification of the revoked cert is REJECTED
//	(while a non-revoked sibling cert still verifies).
//
// Every assertion reflects REAL stored/verified state — no mocked
// returns anywhere in this test (§11.4.27).
func TestCertificateLifecycle_RealPersistence(t *testing.T) {
	pg := startPostgresContainer(t)
	defer pg.Stop(t)

	t.Setenv("DATABASE_URL", pg.dsn)
	t.Setenv("PKI_ENCRYPTION_KEY", "integration-test-encryption-key-32B!!")

	srv, err := server.New(nil)
	if err != nil {
		t.Fatalf("FAILED: server.New against real DB: %v", err)
	}

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()
	client := ts.Client()

	// --- 1. Issue a CA (real handler -> real crypto -> real INSERT) ---
	orgID := uuid.New()
	caBody, _ := json.Marshal(map[string]interface{}{
		"org_id":        orgID.String(),
		"name":          "Integration Test Root CA",
		"description":   "queue#4 real-persistence cert-lifecycle integration test",
		"validity_days": 3650,
	})
	caRespBody := doJSON(t, client, ts.URL, http.MethodPost, "/api/v1/pki/ca", caBody, http.StatusCreated)
	var ca model.CAResponse
	mustUnmarshal(t, caRespBody, &ca)
	if ca.ID == uuid.Nil || ca.CACertPEM == "" {
		t.Fatalf("FAILED: CA create response missing id/cert_pem: %+v", ca)
	}
	t.Logf("PASS: CA created with real persisted id=%s", ca.ID)

	// --- 2. Issue a certificate under that CA (real handler -> real chain sign -> real INSERT) ---
	certBody, _ := json.Marshal(map[string]interface{}{
		"name":          "integration-test-leaf",
		"subject":       "CN=integration.test,O=Helix,C=US",
		"validity_days": 90,
	})
	certRespBody := doJSON(t, client, ts.URL, http.MethodPost,
		fmt.Sprintf("/api/v1/pki/ca/%s/certs", ca.ID), certBody, http.StatusCreated)
	var cert model.CertResponse
	mustUnmarshal(t, certRespBody, &cert)
	if cert.ID == uuid.Nil || cert.CertPEM == "" {
		t.Fatalf("FAILED: certificate create response missing id/cert_pem: %+v", cert)
	}
	if cert.Status != model.StatusActive {
		t.Fatalf("FAILED: freshly issued certificate status = %q, want %q", cert.Status, model.StatusActive)
	}
	t.Logf("PASS: certificate issued with real persisted id=%s status=%s", cert.ID, cert.Status)

	// --- 3. Verify chain succeeds: real x509 signature-chain verification
	//        against the real CA cert returned by the real service. ---
	if err := verifyChain(cert.CertPEM, ca.CACertPEM); err != nil {
		t.Fatalf("FAILED: chain verification of freshly issued certificate failed: %v", err)
	}
	t.Logf("PASS: chain verification (real x509.Certificate.Verify) succeeded for cert %s", cert.ID)

	// Combined live verify (real chain-crypto + real GET-fetched revocation
	// state) MUST accept an active certificate.
	if err := verifyCertLive(t, client, ts.URL, cert.ID.String()); err != nil {
		t.Fatalf("FAILED: combined live verify of active cert unexpectedly rejected: %v", err)
	}
	t.Logf("PASS: combined live verify accepted active cert %s", cert.ID)

	// --- 4. Issue a SECOND, sibling certificate on the same CA — the
	//        non-revoked control used below to prove revocation isn't
	//        globally blanket-rejecting every cert. ---
	cert2Body, _ := json.Marshal(map[string]interface{}{
		"name":          "integration-test-leaf-control",
		"subject":       "CN=integration-control.test,O=Helix,C=US",
		"validity_days": 90,
	})
	cert2RespBody := doJSON(t, client, ts.URL, http.MethodPost,
		fmt.Sprintf("/api/v1/pki/ca/%s/certs", ca.ID), cert2Body, http.StatusCreated)
	var cert2 model.CertResponse
	mustUnmarshal(t, cert2RespBody, &cert2)
	if cert2.ID == uuid.Nil {
		t.Fatalf("FAILED: control certificate create response missing id: %+v", cert2)
	}

	// --- 5. Revoke the FIRST certificate (real handler -> real UPDATE). ---
	revokeBody, _ := json.Marshal(map[string]string{"reason": "queue4-integration-test-revocation"})
	revokeReq, err := http.NewRequest(http.MethodPost,
		ts.URL+fmt.Sprintf("/api/v1/pki/certs/%s/revoke", cert.ID), bytes.NewReader(revokeBody))
	if err != nil {
		t.Fatalf("FAILED: building revoke request: %v", err)
	}
	revokeReq.Header.Set("Content-Type", "application/json")
	revokeResp, err := client.Do(revokeReq)
	if err != nil {
		t.Fatalf("FAILED: revoke request: %v", err)
	}
	defer revokeResp.Body.Close()
	if revokeResp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(revokeResp.Body)
		t.Fatalf("FAILED: revoke response = %d, want %d; body=%s", revokeResp.StatusCode, http.StatusNoContent, b)
	}
	t.Logf("PASS: revoke request accepted for cert %s", cert.ID)

	// --- 6. Real DB row inspection: assert the REAL persisted row
	//        reflects the revocation, independent of the API layer. ---
	var dbStatus string
	var dbRevokedAt *time.Time
	var dbReason string
	row := pg.pool.QueryRow(context.Background(),
		`SELECT status, revoked_at, revocation_reason FROM certificates WHERE id = $1`, cert.ID)
	if err := row.Scan(&dbStatus, &dbRevokedAt, &dbReason); err != nil {
		t.Fatalf("FAILED: reading real DB row for revoked cert %s: %v", cert.ID, err)
	}
	if dbStatus != "revoked" {
		t.Fatalf("FAILED: real DB row status = %q, want %q", dbStatus, "revoked")
	}
	if dbRevokedAt == nil {
		t.Fatalf("FAILED: real DB row revoked_at is NULL after revoke — revocation did not persist")
	}
	if dbReason != "queue4-integration-test-revocation" {
		t.Fatalf("FAILED: real DB row revocation_reason = %q, want %q", dbReason, "queue4-integration-test-revocation")
	}
	t.Logf("PASS: real DB row for cert %s reflects revoked state — status=%s revoked_at=%s reason=%q",
		cert.ID, dbStatus, dbRevokedAt.Format(time.RFC3339), dbReason)

	// --- 7. THE anti-bluff assertion: a SUBSEQUENT verification of the
	//        revoked certificate is REJECTED, driven purely by the real
	//        persisted revocation state fetched live from the service. ---
	err = verifyCertLive(t, client, ts.URL, cert.ID.String())
	if err == nil {
		t.Fatalf("FAILED (anti-bluff violation, §11.4.123): live verification of REVOKED certificate %s did NOT reject it", cert.ID)
	}
	if !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("FAILED: verification of revoked cert %s rejected for the wrong reason: %v", cert.ID, err)
	}
	t.Logf("PASS: subsequent verification of revoked cert %s correctly REJECTED: %v", cert.ID, err)

	// --- 8. Control: the NON-revoked sibling certificate still verifies. ---
	if err := verifyCertLive(t, client, ts.URL, cert2.ID.String()); err != nil {
		t.Fatalf("FAILED: non-revoked control cert %s unexpectedly rejected: %v", cert2.ID, err)
	}
	t.Logf("PASS: non-revoked control cert %s still verifies successfully", cert2.ID)

	// --- 9. The revoked cert's signature chain remains cryptographically
	//        intact (revocation is a state fact, not a signature fact) —
	//        proves step 7's rejection came from the real persisted
	//        revocation state, not from a broken/mutated certificate. ---
	if err := verifyChain(cert.CertPEM, ca.CACertPEM); err != nil {
		t.Fatalf("FAILED: revoked cert's signature chain should remain cryptographically valid: %v", err)
	}
	t.Logf("PASS: revoked cert %s signature chain remains cryptographically valid (rejection is state-driven, not signature-driven)", cert.ID)
}

// verifyChain performs a REAL x509 signature-chain verification of
// certPEM against caCertPEM as the sole trust root — this is the
// cryptographic half of "verify chain succeeds" pki-service leaves to
// its consumers (the service itself has no dedicated /verify endpoint).
func verifyChain(certPEM, caCertPEM string) error {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return fmt.Errorf("could not decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("could not parse certificate: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(caCertPEM)) {
		return fmt.Errorf("could not add CA certificate to trust pool")
	}

	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	return err
}

// verifyCertLive performs the FULL real-world verification a consumer
// of pki-service must perform: fetch the certificate AND its issuing CA
// live from the real service (backed by real persistence), cryptographically
// verify the chain, and reject if the real persisted status is not
// "active" — most notably "revoked". This combines the crypto half
// (verifyChain) with the real revocation-state half that pki-service's
// repository/handler layer owns (internal/repository's status column).
func verifyCertLive(t *testing.T, client *http.Client, baseURL, certID string) error {
	t.Helper()

	certRespBody := doJSON(t, client, baseURL, http.MethodGet, "/api/v1/pki/certs/"+certID, nil, http.StatusOK)
	var got model.CertResponse
	mustUnmarshal(t, certRespBody, &got)

	caRespBody := doJSON(t, client, baseURL, http.MethodGet, "/api/v1/pki/ca/"+got.CAID.String(), nil, http.StatusOK)
	var ca model.CAResponse
	mustUnmarshal(t, caRespBody, &ca)

	if err := verifyChain(got.CertPEM, ca.CACertPEM); err != nil {
		return fmt.Errorf("chain verification failed: %w", err)
	}

	if got.Status == model.StatusRevoked {
		reason := "<nil>"
		if got.RevocationReason != nil {
			reason = *got.RevocationReason
		}
		return fmt.Errorf("certificate %s is revoked (reason=%q, revoked_at=%v)", certID, reason, got.RevokedAt)
	}
	if got.Status != model.StatusActive {
		return fmt.Errorf("certificate %s has non-active status %q", certID, got.Status)
	}
	return nil
}

func doJSON(t *testing.T, client *http.Client, baseURL, method, path string, body []byte, wantStatus int) []byte {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatalf("FAILED: building request %s %s: %v", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("FAILED: request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("FAILED: reading response body for %s %s: %v", method, path, err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("FAILED: %s %s = %d, want %d; body=%s", method, path, resp.StatusCode, wantStatus, respBody)
	}
	return respBody
}

func mustUnmarshal(t *testing.T, data []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("FAILED: unmarshal response %s: %v", data, err)
	}
}
