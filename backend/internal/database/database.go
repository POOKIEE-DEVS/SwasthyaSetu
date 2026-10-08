// Package database opens Postgres (production) or SQLite (local development
// and tests) behind one small API, and creates the tables at startup.
//
// The database is never required to start: chat and patient calls don't
// use it, and the emergency path must not depend on it. If it can't be
// reached, Open still returns a DB; Ready reports false, and EnsureReady
// retries table creation at most every 30 seconds from the requests that
// need it.
//
// Queries are written once, with Postgres placeholders ($1, $2, ...).
// For SQLite they are rewritten to ?1, ?2, ..., which bind by number too.
package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// RetryInterval is how often a request may retry an unreachable database,
// so requests don't each wait on a connection attempt.
const RetryInterval = 30 * time.Second

// ErrUnavailable means the database could not be reached or set up.
var ErrUnavailable = errors.New("database unavailable")

// Querier runs queries; both *DB and *Tx are one.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// DB is the application's database handle.
type DB struct {
	kind    string // "postgres" or "sqlite"
	pool    *sql.DB
	openErr error
	log     *slog.Logger

	initMu      sync.Mutex
	ready       atomic.Bool
	lastAttempt atomic.Int64 // unix nanoseconds
}

// Open prepares a handle for a DATABASE_URL. It does not connect, and never
// fails: a URL that can't be used is reported by Init.
func Open(url string, log *slog.Logger) *DB {
	url = strings.TrimSpace(url)
	d := &DB{kind: "sqlite", log: log}
	if isPostgres(url) {
		d.kind = "postgres"
		d.pool, d.openErr = openPostgres(url)
	} else {
		d.pool, d.openErr = openSQLite(url)
	}
	if d.openErr != nil {
		// Every query then fails with the reason, like an unreachable server.
		d.pool = sql.OpenDB(failingConnector{d.openErr})
	}
	return d
}

func isPostgres(url string) bool {
	return strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://")
}

func openPostgres(url string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		// The parse error can quote the URL, password included.
		return nil, errors.New("DATABASE_URL is not a valid Postgres connection string")
	}
	// One round trip per query and no named prepared statements, so a
	// connection pooler in transaction mode (Neon's "-pooler" host,
	// PgBouncer) is safe.
	cfg.DefaultQueryExecMode = pgx.QueryExecModeExec
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	// Hosted Postgres closes idle connections. The driver pings a
	// connection idle for over a second before reusing it, and these limits
	// retire connections before the server does.
	pool := stdlib.OpenDB(*cfg)
	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(5 * time.Minute)
	pool.SetConnMaxIdleTime(time.Minute)
	return pool, nil
}

// openSQLite takes SQLAlchemy-style URLs, as the Python backend did:
// sqlite:///relative.db, sqlite:////absolute.db, and sqlite:// or
// sqlite:///:memory: for an in-memory database.
func openSQLite(url string) (*sql.DB, error) {
	if !strings.HasPrefix(url, "sqlite:") {
		return nil, errors.New(`DATABASE_URL must start with "postgres://", "postgresql://" or "sqlite:///"`)
	}
	path := strings.TrimPrefix(strings.TrimPrefix(url, "sqlite:"), "//")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		path = ":memory:"
	}
	// _time_format=sqlite stores times as "2006-01-02 15:04:05.999999999-07:00",
	// which reads back as time.Time and sorts correctly as text.
	dsn := path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_time_format=sqlite"
	pool, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// One connection: SQLite allows one writer at a time anyway, and an
	// in-memory database exists only inside its connection.
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	pool.SetConnMaxLifetime(0)
	pool.SetConnMaxIdleTime(0)
	return pool, nil
}

// Kind is "postgres" or "sqlite".
func (d *DB) Kind() string { return d.kind }

// Ready reports whether the tables exist and the last setup succeeded.
func (d *DB) Ready() bool { return d.ready.Load() }

// SetReady overrides the ready flag (tests simulate an outage with it).
func (d *DB) SetReady(ready bool) { d.ready.Store(ready) }

// Init creates any missing tables. It returns an error if the database
// can't be reached.
func (d *DB) Init(ctx context.Context) error {
	d.initMu.Lock()
	defer d.initMu.Unlock()
	d.lastAttempt.Store(time.Now().UnixNano())
	if d.ready.Load() {
		return nil
	}
	if d.openErr != nil {
		return d.openErr
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for _, statement := range schema(d.kind) {
		if _, err := d.pool.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("create tables: %w", err)
		}
	}
	d.ready.Store(true)
	return nil
}

// TryInit is Init that logs a failure instead of returning it.
func (d *DB) TryInit(ctx context.Context) bool {
	if err := d.Init(ctx); err != nil {
		d.log.Error("database unavailable: sign-in, verification and chat history are off "+
			"until it is reachable (check DATABASE_URL)", "error", err)
		return false
	}
	return true
}

// EnsureReady returns nil when the database is set up. Otherwise it retries
// the setup, at most once per RetryInterval, and returns ErrUnavailable if
// that fails.
func (d *DB) EnsureReady(ctx context.Context) error {
	if d.ready.Load() {
		return nil
	}
	last := time.Unix(0, d.lastAttempt.Load())
	if time.Since(last) > RetryInterval && d.TryInit(ctx) {
		return nil
	}
	return ErrUnavailable
}

// Close closes the connection pool.
func (d *DB) Close() error { return d.pool.Close() }

// ExecContext runs a statement.
func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.pool.ExecContext(ctx, rebind(d.kind, query), args...)
}

// QueryContext runs a query returning rows.
func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.pool.QueryContext(ctx, rebind(d.kind, query), args...)
}

// QueryRowContext runs a query returning at most one row.
func (d *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.pool.QueryRowContext(ctx, rebind(d.kind, query), args...)
}

// InTx runs fn in a transaction, committing if it returns nil and rolling
// back otherwise.
func (d *DB) InTx(ctx context.Context, fn func(tx *Tx) error) error {
	sqlTx, err := d.pool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	tx := &Tx{tx: sqlTx, kind: d.kind}
	if err := fn(tx); err != nil {
		_ = sqlTx.Rollback()
		return err
	}
	return sqlTx.Commit()
}

// Tx is a transaction. It runs the same queries as DB.
type Tx struct {
	tx   *sql.Tx
	kind string
}

// ExecContext runs a statement in the transaction.
func (t *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, rebind(t.kind, query), args...)
}

// QueryContext runs a query in the transaction.
func (t *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, rebind(t.kind, query), args...)
}

// QueryRowContext runs a single-row query in the transaction.
func (t *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, rebind(t.kind, query), args...)
}

// rebind turns $1, $2, ... into ?1, ?2, ... for SQLite. The queries in this
// codebase never contain a literal "$".
func rebind(kind, query string) string {
	if kind != "sqlite" {
		return query
	}
	return strings.ReplaceAll(query, "$", "?")
}

// failingConnector stands in for a database whose URL can't be used.
type failingConnector struct{ err error }

func (c failingConnector) Connect(context.Context) (driver.Conn, error) { return nil, c.err }
func (c failingConnector) Driver() driver.Driver                        { return failingDriver(c) }

type failingDriver struct{ err error }

func (d failingDriver) Open(string) (driver.Conn, error) { return nil, d.err }
