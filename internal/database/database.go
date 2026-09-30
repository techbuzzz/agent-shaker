// Package database owns the Postgres connection pool and exposes a small,
// consumer-side Querier interface that handlers depend on (so tests can
// substitute fakes without spinning up a real pool).
//
// The pool itself is *sql.DB driven by the pgx/v5/stdlib shim, which gives
// us pgx's speed and feature set while keeping the familiar database/sql
// API that the existing handlers consume. New code can opt into the native
// pgx API by requesting the *pgxpool.Pool via NewPool().
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// pgx/v5/stdlib registers the "pgx" driver with database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is a thin wrapper around *sql.DB (pgx-backed). Methods on *sql.DB
// remain accessible via embedding.
type DB struct {
	*sql.DB
}

// Querier is the slice of database/sql the application layer depends on.
// Defining it here, in the consumer side, keeps handlers decoupled from
// concrete driver packages and makes them trivially stubbable in tests.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	PingContext(ctx context.Context) error
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// WithTx runs fn inside a transaction. The transaction is committed if fn
// returns nil, otherwise rolled back. The error from rollback is joined with
// the original error when both fail.
func WithTx(ctx context.Context, q Querier, opts *sql.TxOptions, fn func(*sql.Tx) error) error {
	tx, err := q.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			return fmt.Errorf("%w (rollback also failed: %v)", err, rbErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// NewDB opens a connection pool driven by pgx/v5/stdlib. Pool size and
// lifetime are tuned for a single backend process; operators can override
// them via DATABASE_URL query parameters (pool_max_conns, pool_min_conns,
// pool_max_conn_lifetime, etc. — pgxpool recognises them).
func NewDB(ctx context.Context, databaseURL string) (*DB, error) {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Apply defaults. Operators can override via the DSN (pgx supports
	// pool_max_conns etc. in the URL query string).
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{DB: sqlDB}, nil
}

// NewPool opens a native pgxpool.Pool for code that wants the pgx-native API
// (typed pgx.Rows, pgconn.CommandTag, etc.). The pool uses the same DSN
// semantics as NewDB; both pools may co-exist during a gradual migration.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	if cfg.MaxConns < 1 {
		cfg.MaxConns = 25
	}
	if cfg.MinConns < 1 {
		cfg.MinConns = 5
	}
	if cfg.MaxConnLifetime == 0 {
		cfg.MaxConnLifetime = time.Hour
	}
	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = 10 * time.Minute
	}
	if cfg.HealthCheckPeriod == 0 {
		cfg.HealthCheckPeriod = 30 * time.Second
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
