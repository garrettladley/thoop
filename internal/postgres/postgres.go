package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	pgsqlc "github.com/garrettladley/thoop/internal/postgres/sqlc"
)

type Config struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

func DefaultConfig(url string) Config {
	return Config{
		URL:             url,
		MaxConns:        20,
		MinConns:        2,
		MaxConnLifetime: 15 * time.Minute,
		MaxConnIdleTime: 10 * time.Minute,
		ConnectTimeout:  60 * time.Second,
	}
}

type DB interface {
	pgsqlc.Querier
	Pool() *pgxpool.Pool
	Close()
}

type db struct {
	pool *pgxpool.Pool
	*pgsqlc.Queries
}

var _ DB = (*db)(nil)

func New(ctx context.Context, cfg Config) (DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	return &db{
		pool:    pool,
		Queries: pgsqlc.New(pool),
	}, nil
}

func (d *db) Pool() *pgxpool.Pool {
	return d.pool
}

func (d *db) Close() {
	d.pool.Close()
}
