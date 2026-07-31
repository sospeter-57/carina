package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config holds Postgres connection configuration.
type Config struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(url string) Config {
	return Config{
		URL:             url,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
	}
}

// Pool wraps pgxpool.Pool with additional convenience methods.
type Pool struct {
	*pgxpool.Pool
	config Config
}

// New creates a new connection pool from the given Config.
// Returns an error if the pool cannot be created or the connection cannot be established.
func New(ctx context.Context, cfg Config) (*Pool, error) {
	if cfg.URL == "" {
		return nil, errors.New("postgres: DATABASE_URL is required")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}

	// Ensure MinConns doesn't exceed MaxConns
	if poolCfg.MinConns > poolCfg.MaxConns {
		poolCfg.MinConns = poolCfg.MaxConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	// Verify connection is live
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping failed: %w", err)
	}

	return &Pool{Pool: pool, config: cfg}, nil
}

// Health checks the database connectivity with a timeout.
func (p *Pool) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return p.Ping(ctx)
}

// DB returns a *sql.DB for compatibility with tools that expect the stdlib interface.
// This is primarily used by the migration runner.
func (p *Pool) DB() *sql.DB {
	return p.Pool.Config().ConnConfig.SQLDB()
}

// Close closes the connection pool.
func (p *Pool) Close() {
	p.Pool.Close()
}

// Stats returns connection pool statistics.
func (p *Pool) Stats() *pgxpool.Stat {
	return p.Pool.Stat()
}