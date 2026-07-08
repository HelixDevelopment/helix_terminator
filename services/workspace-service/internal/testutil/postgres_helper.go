// Package testutil — postgres_helper.go provides a self-contained
// PostgreSQL test-container launcher for stress and chaos test suites
// that need a real database.
package testutil

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/helixdevelopment/workspace-service/migrations"
)

// pgTestLogger adapts *testing.T to the migrations.Logger interface.
type pgTestLogger struct{ t *testing.T }

func (l *pgTestLogger) Printf(format string, v ...interface{}) {
	l.t.Logf("[migrations] "+format, v...)
}

// StartTestPostgres boots a real, disposable PostgreSQL 17.2 container
// via rootless podman, applies workspace-service migrations, and returns
// a schema-scoped DATABASE_URL ready for pgxpool.New. The container is
// torn down via t.Cleanup.
//
// Returns ("", false) if podman is not available — callers should
// t.Skip with an honest reason rather than faking success.
func StartTestPostgres(t *testing.T) (dbURL string, available bool) {
	t.Helper()

	if _, err := exec.LookPath("podman"); err != nil {
		return "", false
	}

	port, err := freeTestPort()
	if err != nil {
		t.Fatalf("failed to allocate free port for test postgres: %v", err)
	}

	name := fmt.Sprintf("workspace-stress-%d", time.Now().UnixNano())

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

	if !waitForReady(name, 30*time.Second) {
		t.Fatalf("PostgreSQL container %s did not become ready in time", name)
	}

	rawURL := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/postgres?sslmode=disable", port)

	// Retry migrations to absorb the postgres image's known
	// temp-server-then-restart startup race.
	for attempt := 1; attempt <= 5; attempt++ {
		version, runErr := migrations.Run(rawURL, &pgTestLogger{t: t})
		if runErr == nil {
			t.Logf("migrations applied to %s at schema version %d (attempt %d)", name, version, attempt)
			poolURL, perr := migrations.ConnectionURL(rawURL)
			if perr != nil {
				t.Fatalf("migrations.ConnectionURL failed: %v", perr)
			}
			return poolURL, true
		}
		t.Logf("migrations.Run attempt %d/5 failed: %v", attempt, runErr)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	t.Fatalf("failed to apply migrations after 5 attempts")
	return "", false
}

func waitForReady(containerName string, timeout time.Duration) bool {
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

func freeTestPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
