// Package migrations embeds workspace-service's SQL schema migrations and
// applies them to PostgreSQL at process startup using golang-migrate
// (github.com/golang-migrate/migrate/v4) with the pgx/v5 database driver
// and the iofs (embed.FS) source driver.
//
// Run is idempotent: invoking it against a database that is already at
// the latest embedded schema version is a no-op (migrate.ErrNoChange),
// never an error. This closes GAP-02 (queue item #2): migrations were
// previously assumed pre-applied and never actually run by the service
// binaries.
//
// All 25 helix_terminator services share a single PostgreSQL database
// (see infrastructure/docker/compose/docker-compose.yml, POSTGRES_DB
// defaulting to "helixterminator"). Two independent collision surfaces
// follow from that:
//
//  1. golang-migrate's own bookkeeping table (default name
//     "schema_migrations") - closed by giving every service its own,
//     distinctly-named table via the x-migrations-table DSN option.
//  2. Application object names - multiple services each declare their
//     own tables (e.g. "workspaces"), which collide the instant a
//     second service's migrator runs against the shared database
//     ("relation already exists").
//
// (2) is closed by schema-per-service (operator decision, GAP-01
// remediation): every service migrates into its own dedicated
// PostgreSQL schema (see the Schema constant below) instead of the
// shared "public" schema, selected via the standard PostgreSQL
// "search_path" connection parameter.
//
// workspace-service has TWO independent startup sites that each open
// their own pgxpool.Pool against DATABASE_URL: cmd/workspace-service/
// main.go (its pool is currently unused after construction - dead
// code, kept as-is here since removing it is out of this change's
// scope) and internal/server/server.go's New() (this is the pool that
// actually backs the running service's repository). Both call sites
// invoke Run before opening their own pool and both build their pool
// DSN via ConnectionURL, so regardless of call order or which site
// executes first, neither pool ever queries the schema before Run has
// applied every pending migration (Run's idempotency makes calling it
// from both sites safe - the second call is a no-op).
package migrations

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the "pgx5" database driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
)

//go:embed *.up.sql *.down.sql
var fs embed.FS

// Schema is the dedicated PostgreSQL schema this service's migrations
// and steady-state repository queries operate in. It MUST be unique
// across every service sharing the "helixterminator" database (see the
// package doc comment above for the naming convention).
const Schema = "workspace_service"

// migrationsTable is this service's dedicated golang-migrate bookkeeping
// table name. It MUST be unique across every service sharing the
// "helixterminator" database. Schema isolation (Schema, above) already
// makes this unique per-service on its own (each service's bookkeeping
// table now lives inside its own schema), but the service-qualified
// name is kept as defense-in-depth against a future schema-isolation
// regression.
const migrationsTable = "workspace_service_schema_migrations"

// Logger is the minimal logging interface Run needs. *log.Logger and the
// service-local Logger interfaces (server.Logger) both satisfy it.
type Logger interface {
	Printf(format string, v ...interface{})
}

// Run applies every pending embedded migration to the PostgreSQL database
// identified by databaseURL (a "postgres://" or "postgresql://" DSN, the
// same value read from the DATABASE_URL environment variable). It returns
// the schema version left in place after running (0 if no migrations
// exist yet) and a non-nil error on any real failure - including when the
// schema is left in a "dirty" (partially applied) state, which requires
// manual operator intervention and MUST NOT be silently served against.
//
// Run opens its own short-lived database/sql connection for the duration
// of the migration and closes it before returning; it does not share a
// connection pool with the service's steady-state repository.
func Run(databaseURL string, logger Logger) (version uint, err error) {
	if databaseURL == "" {
		return 0, errors.New("migrations: DATABASE_URL is empty")
	}

	if serr := EnsureSchema(context.Background(), databaseURL); serr != nil {
		return 0, serr
	}

	src, err := iofs.New(fs, ".")
	if err != nil {
		return 0, fmt.Errorf("migrations: load embedded source: %w", err)
	}

	dsn, err := toPGX5DSN(databaseURL)
	if err != nil {
		return 0, fmt.Errorf("migrations: invalid DATABASE_URL: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return 0, fmt.Errorf("migrations: init migrator: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if err == nil {
			switch {
			case srcErr != nil:
				err = fmt.Errorf("migrations: close source: %w", srcErr)
			case dbErr != nil:
				err = fmt.Errorf("migrations: close database: %w", dbErr)
			}
		}
	}()

	upErr := m.Up()
	if upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
		return 0, fmt.Errorf("migrations: apply: %w", upErr)
	}

	v, dirty, verr := m.Version()
	if verr != nil {
		if errors.Is(verr, migrate.ErrNilVersion) {
			if logger != nil {
				logger.Printf("migrations: no embedded migrations found - nothing to apply")
			}
			return 0, nil
		}
		return 0, fmt.Errorf("migrations: read schema version: %w", verr)
	}
	if dirty {
		return v, fmt.Errorf("migrations: schema at version %d is dirty (a previous migration failed partway) - manual operator intervention required", v)
	}

	if logger != nil {
		if errors.Is(upErr, migrate.ErrNoChange) {
			logger.Printf("migrations: schema already up to date at version %d (no-op)", v)
		} else {
			logger.Printf("migrations: applied pending migrations - schema now at version %d", v)
		}
	}
	return v, nil
}

// EnsureSchema creates this service's dedicated PostgreSQL schema
// (Schema, above) if it does not already exist. It is idempotent and
// safe to call on every process startup. Schema creation is done over
// a short-lived, unpooled native pgx connection against the RAW
// databaseURL (untouched postgres:// / postgresql:// scheme, no
// search_path applied yet) - CREATE SCHEMA does not depend on
// search_path, and the schema must exist BEFORE any connection sets
// search_path to it (an unqualified statement issued against a
// search_path whose only entry does not yet exist fails with "no
// schema has been selected to create in").
//
// Run calls EnsureSchema automatically before applying migrations. It
// is exported so callers can provision the schema independently of
// running a migration (e.g. an operator smoke-check).
func EnsureSchema(ctx context.Context, databaseURL string) error {
	if databaseURL == "" {
		return errors.New("migrations: DATABASE_URL is empty")
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("migrations: connect to create schema %q: %w", Schema, err)
	}
	defer conn.Close(ctx)

	stmt := "CREATE SCHEMA IF NOT EXISTS " + pgx.Identifier{Schema}.Sanitize()
	if _, err := conn.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("migrations: create schema %q: %w", Schema, err)
	}
	return nil
}

// ConnectionURL rewrites databaseURL so that every connection opened
// against it defaults to this service's dedicated schema (Schema,
// above), via the standard PostgreSQL "search_path" connection
// parameter. It does NOT rewrite the URL scheme - pgxpool.New (and
// database/sql via the pgx stdlib driver) both parse "postgres://" /
// "postgresql://" DSNs natively.
//
// Verified against github.com/jackc/pgx/v5/pgconn's ParseConfig
// (v5.10.0, pgconn/config.go): any DSN query parameter pgconn does not
// itself recognise as a libpq connection setting (host, port,
// sslmode, ... - "search_path" is not among them) is placed into
// RuntimeParams and sent to PostgreSQL as a startup parameter on every
// new physical connection - equivalent to `SET search_path` run
// immediately after connecting, for every connection in a pgxpool
// pool, not just the first. golang-migrate's pgx/v5 driver opens its
// connections the same way (database/sql via the pgx stdlib driver,
// which shares pgconn's DSN parsing), so toPGX5DSN below applies the
// identical mechanism for the migration connection.
//
// Callers (server/main startup code, and any test helper that boots a
// real database) MUST build their steady-state pgxpool.New(...) DSN
// via ConnectionURL(databaseURL), never the raw databaseURL directly,
// so the migrator and the running service always agree on which
// schema they operate against.
func ConnectionURL(databaseURL string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("migrations: invalid DATABASE_URL: %w", err)
	}
	q := u.Query()
	applySearchPath(q)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// applySearchPath sets the search_path query parameter to this
// service's Schema, unless the caller already supplied an explicit
// override (mirrors the x-migrations-table override behaviour below).
// Idempotent: applying it twice (e.g. Run() called again against a
// databaseURL that already has search_path=Schema from a prior
// ConnectionURL/toPGX5DSN pass) leaves the value unchanged.
func applySearchPath(q url.Values) {
	if q.Get("search_path") == "" {
		q.Set("search_path", Schema)
	}
}

// toPGX5DSN rewrites a standard "postgres://"/"postgresql://" DSN into the
// "pgx5://" scheme golang-migrate's database/pgx/v5 driver registers
// itself under, pins this service's dedicated migrations bookkeeping
// table via the x-migrations-table query option, and scopes every
// connection the migrator opens to this service's dedicated schema via
// search_path (applySearchPath) so migrations land in Schema, not
// "public".
func toPGX5DSN(databaseURL string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "postgres", "postgresql", "pgx", "pgx5":
		u.Scheme = "pgx5"
	default:
		return "", fmt.Errorf("unsupported scheme %q (expected postgres:// or postgresql://)", u.Scheme)
	}

	q := u.Query()
	if q.Get("x-migrations-table") == "" {
		q.Set("x-migrations-table", migrationsTable)
	}
	applySearchPath(q)
	u.RawQuery = q.Encode()

	return u.String(), nil
}
