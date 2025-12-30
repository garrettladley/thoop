package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

const migrationsDir = "sql"

//go:embed sql/*.sql
var migrationsFS embed.FS

func Apply(ctx context.Context, db *sql.DB) error {
	if err := createHistoryTable(ctx, db); err != nil {
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
		applied, err := isMigrationApplied(ctx, db, filename)
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
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", filename, err)
			}
		}

		if err := recordMigration(ctx, db, filename); err != nil {
			return err
		}
	}

	return nil
}

func createHistoryTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS migrations_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("creating migrations history table: %w", err)
	}
	return nil
}

func isMigrationApplied(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM migrations_history WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking if migration applied: %w", err)
	}
	return count > 0, nil
}

func recordMigration(ctx context.Context, db *sql.DB, name string) error {
	_, err := db.ExecContext(ctx, "INSERT INTO migrations_history (name) VALUES (?)", name)
	if err != nil {
		return fmt.Errorf("recording migration: %w", err)
	}
	return nil
}
