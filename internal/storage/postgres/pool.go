package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"styleagent/internal/core/errors"
)

// Pool represents a database connection pool with "get usage release" pattern.
type Pool struct {
	cfg         *Config
	db          *sql.DB
	mu          sync.RWMutex
	openCount   int
	idleCount   int
	waitCount   int
	waitDuration time.Duration
}

// NewPool creates a new database connection pool.
func NewPool(cfg *Config) (*Pool, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Pool{
		cfg:       cfg,
		db:        db,
		openCount: 0,
		idleCount: 0,
	}, nil
}

// Get acquires a connection from the pool.
func (p *Pool) Get(ctx context.Context) (*sql.Conn, error) {
	start := time.Now()

	conn, err := p.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	p.mu.Lock()
	p.openCount++
	p.idleCount++
	elapsed := time.Since(start)
	p.waitDuration += elapsed
	if elapsed > time.Second {
		p.waitCount++
	}
	p.mu.Unlock()

	return conn, nil
}

// Release returns a connection to the pool.
func (p *Pool) Release(conn *sql.Conn) {
	if conn == nil {
		return
	}

	conn.Close()

	p.mu.Lock()
	p.openCount--
	p.idleCount--
	p.mu.Unlock()
}

// WithConnection executes a function with a connection from the pool.
// This is the recommended pattern: get usage release.
func (p *Pool) WithConnection(ctx context.Context, fn func(*sql.Conn) error) error {
	conn, err := p.Get(ctx)
	if err != nil {
		return err
	}
	defer p.Release(conn)

	return fn(conn)
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	return p.db.Close()
}

// Stats returns connection pool statistics.
func (p *Pool) Stats() *PoolStats {
	stats := p.db.Stats()

	p.mu.RLock()
	defer p.mu.RUnlock()

	return &PoolStats{
		OpenConnections:  stats.OpenConnections,
		InUseConnections: stats.InUse,
		IdleConnections:  stats.Idle,
		WaitCount:       stats.WaitCount + int64(p.waitCount),
		WaitDuration:    stats.WaitDuration + p.waitDuration,
		MaxOpenConns:    p.cfg.MaxOpenConns,
	}
}

// PoolStats holds pool statistics.
type PoolStats struct {
	OpenConnections  int
	InUseConnections int
	IdleConnections  int
	WaitCount       int64
	WaitDuration    time.Duration
	MaxOpenConns    int
}

// IsHealthy checks if the pool is healthy.
func (p *Pool) IsHealthy() bool {
	stats := p.Stats()
	return stats.OpenConnections > 0 && stats.OpenConnections < stats.MaxOpenConns
}

// Ping pings the database to check connectivity.
func (p *Pool) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Exec executes a query without returning rows.
func (p *Pool) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var result sql.Result
	var err error

	p.WithConnection(ctx, func(conn *sql.Conn) error {
		result, err = conn.ExecContext(ctx, query, args...)
		return err
	})

	return result, err
}

// Query executes a query and returns rows.
func (p *Pool) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	p.WithConnection(ctx, func(conn *sql.Conn) error {
		rows, err = conn.QueryContext(ctx, query, args...)
		return err
	})

	return rows, err
}

// QueryRow executes a query and returns a single row.
func (p *Pool) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	var row *sql.Row

	p.WithConnection(ctx, func(conn *sql.Conn) error {
		row = conn.QueryRowContext(ctx, query, args...)
		return nil
	})

	return row
}

// Begin starts a new transaction.
func (p *Pool) Begin(ctx context.Context) (*sql.Tx, error) {
	return p.db.BeginTx(ctx, nil)
}

// NOTE: This module uses the standard library's database/sql package
// which already implements a connection pool. The Pool wrapper provides
// additional statistics and the "get usage release" pattern.
var _ = errors.ErrDBConnectionFailed
