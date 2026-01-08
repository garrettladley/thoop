package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/garrettladley/thoop/internal/migrations/sqlite"
	sqlitec "github.com/garrettladley/thoop/internal/sqlc/sqlite"
	_ "modernc.org/sqlite"
)

const driverName = "sqlite"

// Open opens a connection to the SQLite database and returns a querier.
// It automatically applies any pending migrations.
// The caller is responsible for closing the returned *sql.DB.
func Open(ctx context.Context, dbPath string) (*sql.DB, sqlitec.Querier, error) {
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := sqlite.Apply(ctx, db); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("applying migrations: %w", err)
	}

	querier := sqlitec.New(db)
	return db, querier, nil
}
