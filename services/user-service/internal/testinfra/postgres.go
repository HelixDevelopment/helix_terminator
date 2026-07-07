//go:build integration

// Package testinfra provides shared real-infrastructure test helpers for
// user-service's integration test suites (build-tagged "integration").
// It boots a real, rootless-podman PostgreSQL 17.2 container, applies
// user-service's real golang-migrate schema against it via
// migrations.Run, and hands back a ready-to-use DATABASE_URL.
//
// Mirrors auth-service's internal/testinfra/postgres.go exactly (same
// rootless-podman container lifecycle, same startup-race handling,
// same schema-scoped connection URL); user-service previously had no
// such helper package, so real-database integration tests here had to
// author their own boot logic inline. Per Constitution §11.4.27
// (mocks/stubs/fakes are permitted only in unit tests) every test
// importing this package exercises a real PostgreSQL instance and the
// real migration runner - never an in-memory or mocked database.
package testinfra

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/helixdevelopment/user-service/migrations"
)

// pgLogger adapts *testing.T to the migrations.Logger interface so
// migration progress is captured in the test's own output (visible
// with `go test -v`, folded into failure output otherwise).
type pgLogger struct{ t *testing.T }

func (l *pgLogger) Printf(format string, v ...interface{}) {
	l.t.Logf("[migrations] "+format, v...)
}

// StartPostgres boots a real, disposable PostgreSQL 17.2 container via
// rootless podman on a freshly-allocated free port, waits for it to
// accept connections, applies every pending user-service migration
// against it via the real migrations.Run runner (the same runner
// server.New invokes at process startup), and returns a ready-to-use
// DATABASE_URL scoped to user-service's dedicated schema
// (migrations.Schema) via migrations.ConnectionURL - the same
// schema-scoped URL server.New builds for its own steady-state pool,
// so callers that connect directly (pgxpool.New(ctx, dbURL)) see the
// same "users" table server.New's repository sees, not the shared
// database's "public" schema (schema-per-service, GAP-01).
//
// The container is torn down automatically via t.Cleanup. When podman
// is not available on PATH, the call SKIPs with an honest reason
// (topology_unsupported per §11.4.69) rather than faking success.
func StartPostgres(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("SKIP: podman not found on PATH - no real container runtime available for this run (topology_unsupported per §11.4.69)")
	}

	port, err := freePort()
	if err != nil {
		t.Fatalf("failed to allocate a free port for the test PostgreSQL container: %v", err)
	}

	name := fmt.Sprintf("user-svc-it-%d", time.Now().UnixNano())

	runArgs := []string{
		"run", "-d", "--rm",
		"--name", name,
		"-e", "POSTGRES_PASSWORD=postgres",
		"-e", "POSTGRES_USER=postgres",
		"-e", "POSTGRES_DB=postgres",
		"-p", fmt.Sprintf("127.0.0.1:%d:5432", port),
		"docker.io/library/postgres:17.2",
	}

	runCtx, runCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer runCancel()
	if out, err := exec.CommandContext(runCtx, "podman", runArgs...).CombinedOutput(); err != nil {
		t.Fatalf("podman run failed: %v\n%s", err, out)
	}

	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer stopCancel()
		if out, err := exec.CommandContext(stopCtx, "podman", "stop", "-t", "5", name).CombinedOutput(); err != nil {
			t.Logf("warning: failed to stop test postgres container %s: %v\n%s", name, err, out)
		}
	})

	if !waitForPostgresReady(name, 30*time.Second) {
		t.Fatalf("PostgreSQL container %s did not become ready to accept connections in time", name)
	}

	dbURL := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/postgres?sslmode=disable", port)

	// The official postgres Docker image's entrypoint starts the
	// PostgreSQL server process TWICE on first boot: once as a
	// temporary, localhost-only server used purely to run initdb-time
	// initialization, which it then STOPS, before starting the real,
	// network-reachable server. `pg_isready` (waitForPostgresReady,
	// above) can observe the temporary server as "ready" in that
	// narrow window, moments before it is torn down - a real
	// connection attempt landing right then sees ECONNRESET, not a
	// clean refusal. Retry migrations.Run a few times with backoff to
	// absorb that one-time startup race deterministically rather than
	// racing it with a single attempt.
	var runErr error
	for attempt := 1; attempt <= 5; attempt++ {
		var version uint
		version, runErr = migrations.Run(dbURL, &pgLogger{t: t})
		if runErr == nil {
			t.Logf("real user-service migrations applied to %s at schema version %d (attempt %d)", name, version, attempt)
			poolURL, perr := migrations.ConnectionURL(dbURL)
			if perr != nil {
				t.Fatalf("migrations.ConnectionURL failed: %v", perr)
			}
			return poolURL
		}
		t.Logf("migrations.Run attempt %d/5 against %s failed (likely the postgres image's known temp-server-then-restart startup race): %v", attempt, name, runErr)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	t.Fatalf("failed to apply real user-service migrations against the test database after 5 attempts: %v", runErr)
	return ""
}

// waitForPostgresReady polls `podman exec <name> pg_isready` until the
// server reports it is accepting connections, THEN additionally waits
// for the official postgres image's startup log to show the server
// becoming ready TWICE (temporary init-only server, then the real
// server it restarts into) - the documented postgres Docker image
// startup sequence. Relying on pg_isready alone can observe the first
// (temporary) server as ready.
func waitForPostgresReady(containerName string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		out, err := exec.CommandContext(ctx, "podman", "exec", containerName, "pg_isready", "-U", "postgres").CombinedOutput()
		cancel()
		if err == nil && strings.Contains(string(out), "accepting connections") {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if time.Now().After(deadline) {
		return false
	}

	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		out, err := exec.CommandContext(ctx, "podman", "logs", containerName).CombinedOutput()
		cancel()
		if err == nil && strings.Count(string(out), "database system is ready to accept connections") >= 2 {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// freePort asks the OS for an ephemeral TCP port, then releases it
// immediately so podman can bind it. Small TOCTOU race window is
// acceptable for disposable, sequential test-container use.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
