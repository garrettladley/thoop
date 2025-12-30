package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationsDir = "sql"

//go:embed sql/*.sql
var migrationsFS embed.FS

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	if err := createHistoryTable(ctx, pool); err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationsFS, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	upFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		upFiles = append(upFiles, entry.Name())
	}

	sort.Strings(upFiles)

	for _, filename := range upFiles {
		applied, err := isMigrationApplied(ctx, pool, filename)
		if err != nil {
			return err
		}

		if applied {
			continue
		}

		content, err := fs.ReadFile(migrationsFS, migrationsDir+"/"+filename)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		statements := strings.SplitSeq(string(content), ";")
		for stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", filename, err)
			}
		}

		if err := recordMigration(ctx, pool, filename); err != nil {
			return err
		}
	}

	return nil
}

func createHistoryTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS migrations_history (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create history table: %w", err)
	}
	return nil
}

func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM migrations_history WHERE name = $1", name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if migration applied: %w", err)
	}
	return count > 0, nil
}

func recordMigration(ctx context.Context, pool *pgxpool.Pool, name string) error {
	_, err := pool.Exec(ctx, "INSERT INTO migrations_history (name) VALUES ($1)", name)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	return nil
}
