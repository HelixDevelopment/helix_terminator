// Package migrations embeds org-service's SQL schema migrations and
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
// defaulting to "helixterminator"). golang-migrate tracks applied
// versions in a bookkeeping table (default name "schema_migrations").
// Because the database is shared, every service MUST use its own,
// distinctly-named bookkeeping table (via the x-migrations-table DSN
// option) so services never clobber each other's migration version
// state.
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the "pgx5" database driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.up.sql *.down.sql
var fs embed.FS

// migrationsTable is this service's dedicated golang-migrate bookkeeping
// table name. It MUST be unique across every service sharing the
// "helixterminator" database.
const migrationsTable = "org_service_schema_migrations"

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

// toPGX5DSN rewrites a standard "postgres://"/"postgresql://" DSN into the
// "pgx5://" scheme golang-migrate's database/pgx/v5 driver registers
// itself under, and pins this service's dedicated migrations bookkeeping
// table via the x-migrations-table query option.
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
	u.RawQuery = q.Encode()

	return u.String(), nil
}
