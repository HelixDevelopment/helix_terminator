package migrations

import (
	"context"
	"os"
	"strings"
	"testing"
)

// --- Unit tests: pure DSN-rewriting logic. No infra required. ---

func TestToPGX5DSN_RewritesSchemeAndPinsMigrationsTable(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{name: "postgres scheme", in: "postgres://user:pass@localhost:5432/db?sslmode=disable"},
		{name: "postgresql scheme", in: "postgresql://user:pass@localhost:5432/db"},
		{name: "already pgx5", in: "pgx5://user:pass@localhost:5432/db"},
		{name: "unsupported scheme", in: "mysql://user:pass@localhost:3306/db", wantErr: true},
		{name: "not a URL", in: "://not a url", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := toPGX5DSN(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("toPGX5DSN(%q) = %q, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("toPGX5DSN(%q) unexpected error: %v", tc.in, err)
			}
			if !strings.HasPrefix(got, "pgx5://") {
				t.Errorf("toPGX5DSN(%q) = %q, want pgx5:// scheme", tc.in, got)
			}
			if !strings.Contains(got, "x-migrations-table="+migrationsTable) {
				t.Errorf("toPGX5DSN(%q) = %q, want x-migrations-table=%s", tc.in, got, migrationsTable)
			}
		})
	}
}

func TestToPGX5DSN_PreservesExplicitMigrationsTable(t *testing.T) {
	got, err := toPGX5DSN("postgres://u:p@host/db?x-migrations-table=custom_table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "x-migrations-table=custom_table") {
		t.Errorf("toPGX5DSN did not preserve caller-supplied x-migrations-table, got %q", got)
	}
	if strings.Contains(got, migrationsTable) {
		t.Errorf("toPGX5DSN overwrote caller-supplied x-migrations-table with the default, got %q", got)
	}
}

func TestToPGX5DSN_ScopesSearchPathToServiceSchema(t *testing.T) {
	got, err := toPGX5DSN("postgres://u:p@host/db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "search_path="+Schema) {
		t.Errorf("toPGX5DSN(%q) = %q, want search_path=%s (schema-per-service)", "postgres://u:p@host/db", got, Schema)
	}
}

func TestConnectionURL_ScopesSearchPathAndPreservesScheme(t *testing.T) {
	got, err := ConnectionURL("postgres://u:p@host/db?sslmode=disable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "postgres://") {
		t.Errorf("ConnectionURL(%q) = %q, want unchanged postgres:// scheme", "postgres://u:p@host/db", got)
	}
	if !strings.Contains(got, "search_path="+Schema) {
		t.Errorf("ConnectionURL(%q) = %q, want search_path=%s", "postgres://u:p@host/db", got, Schema)
	}
	if !strings.Contains(got, "sslmode=disable") {
		t.Errorf("ConnectionURL(%q) = %q, dropped an existing query parameter", "postgres://u:p@host/db", got)
	}
}

func TestConnectionURL_PreservesExplicitSearchPath(t *testing.T) {
	got, err := ConnectionURL("postgres://u:p@host/db?search_path=custom_schema")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "search_path=custom_schema") {
		t.Errorf("ConnectionURL did not preserve caller-supplied search_path, got %q", got)
	}
}

func TestConnectionURL_IdempotentOnAlreadyScopedURL(t *testing.T) {
	first, err := ConnectionURL("postgres://u:p@host/db")
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	second, err := ConnectionURL(first)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if first != second {
		t.Errorf("ConnectionURL not idempotent: first=%q second=%q", first, second)
	}
}

func TestEnsureSchema_EmptyDatabaseURL(t *testing.T) {
	if err := EnsureSchema(context.Background(), ""); err == nil {
		t.Fatal("EnsureSchema(ctx, \"\") = nil error, want error for empty DATABASE_URL")
	}
}

func TestRun_EmptyDatabaseURL(t *testing.T) {
	if _, err := Run("", nil); err == nil {
		t.Fatal("Run(\"\", nil) = nil error, want error for empty DATABASE_URL")
	}
}

// --- Real-infra integration test. ---
//
// Gated on TEST_DATABASE_URL per §11.4.27 (mocks/fakes are forbidden outside
// unit tests - every other test type MUST exercise the real system). When
// the env var is unset (the common case for a plain `go test ./...` with no
// database available) this test SKIPs with an honest reason rather than
// faking a PASS, per §11.4.3 per-environment-topology dispatch.
//
// This service has THREE forward migrations (001_init, 002_add_delivery_
// fields, 003_add_slack_channel), so a successful Run() MUST leave the
// schema at version 3 (the highest embedded version), not merely a
// non-zero version - a schema stuck at version 1 or 2 (002/003 silently
// skipped/failed) would be a §11.4.108 SOURCE-declares-it/RUNTIME-does-
// not-have-it defect.
//
// To run for real:
//
//	podman run -d --rm --name migrate-notification-pg -e POSTGRES_PASSWORD=postgres \
//	  -p 15543:5432 docker.io/library/postgres:17
//	TEST_DATABASE_URL="postgres://postgres:postgres@localhost:15543/postgres?sslmode=disable" \
//	  GOMAXPROCS=2 go test -p 2 -run TestRun_Integration -v ./migrations/...
func TestRun_Integration(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("SKIP: TEST_DATABASE_URL not set - no real PostgreSQL instance available for this run (topology_unsupported per §11.4.69)")
	}

	logger := &testLogger{t: t}

	const wantVersion = 3 // highest embedded migration version (001_init + 002_add_delivery_fields + 003_add_slack_channel)

	version, err := Run(dbURL, logger)
	if err != nil {
		t.Fatalf("Run() first invocation failed: %v", err)
	}
	if version != wantVersion {
		t.Fatalf("Run() first invocation left schema at version %d, want the highest embedded version %d (multi-file migration path)", version, wantVersion)
	}

	// Idempotency: a second invocation against the now-migrated database
	// MUST succeed as a no-op, not error and not re-apply anything.
	version2, err := Run(dbURL, logger)
	if err != nil {
		t.Fatalf("Run() second invocation (idempotency check) failed: %v", err)
	}
	if version2 != version {
		t.Fatalf("Run() second invocation left schema at version %d, want unchanged version %d", version2, version)
	}
}

type testLogger struct{ t *testing.T }

func (l *testLogger) Printf(format string, v ...interface{}) {
	l.t.Logf(format, v...)
}
