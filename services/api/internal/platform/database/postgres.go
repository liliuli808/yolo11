package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig carries optional pool tuning values. Zero values fall back to
// sensible defaults.
type PoolConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	HealthCheckPeriod time.Duration
}

// NewPool creates a new PostgreSQL connection pool from a connection string.
// The context controls the initial connection timeout.
func NewPool(ctx context.Context, connString string, cfg PoolConfig) (*pgxpool.Pool, error) {
	if connString == "" {
		return nil, fmt.Errorf("database connection string is empty")
	}

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse database connection string: %w", err)
	}

	// Sensible defaults for a production-ready pool. Explicit cfg values take
	// precedence; otherwise we keep the driver's default when one is set.
	if cfg.MaxConns > 0 {
		config.MaxConns = cfg.MaxConns
	} else if config.MaxConns == 0 {
		config.MaxConns = 25
	}
	if cfg.MinConns > 0 {
		config.MinConns = cfg.MinConns
	} else if config.MinConns == 0 {
		config.MinConns = 5
	}
	if cfg.MaxConnLifetime > 0 {
		config.MaxConnLifetime = cfg.MaxConnLifetime
	} else if config.MaxConnLifetime == 0 {
		config.MaxConnLifetime = time.Hour
	}
	if cfg.HealthCheckPeriod > 0 {
		config.HealthCheckPeriod = cfg.HealthCheckPeriod
	} else if config.HealthCheckPeriod == 0 {
		config.HealthCheckPeriod = 5 * time.Minute
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// Close gracefully closes the connection pool.
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
