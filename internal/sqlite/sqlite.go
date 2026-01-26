package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/garrettladley/thoop/internal/migrations/sqlite"
	litesqlc "github.com/garrettladley/thoop/internal/sqlite/sqlc"
	_ "modernc.org/sqlite"
)

const driverName = "sqlite"

type Config struct {
	Path string
}

func DefaultConfig(path string) Config {
	return Config{Path: path}
}

type DB interface {
	litesqlc.Querier
	Close() error
}

type db struct {
	conn *sql.DB
	*litesqlc.Queries
}

var _ DB = (*db)(nil)

func New(ctx context.Context, cfg Config) (DB, error) {
	conn, err := sql.Open(driverName, cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	if err := sqlite.Apply(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return &db{
		conn:    conn,
		Queries: litesqlc.New(conn),
	}, nil
}

func (d *db) Close() error {
	if err := d.conn.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	return nil
}
