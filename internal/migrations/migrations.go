package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const migrationsDir = "migrations"

func Apply(db *sql.DB) error {
	if err := createHistoryTable(db); err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var upFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}

	sort.Strings(upFiles)

	for _, filename := range upFiles {
		applied, err := isMigrationApplied(db, filename)
		if err != nil {
			return err
		}

		if applied {
			continue
		}

		// filename comes from os.ReadDir which only returns base names, safe from path traversal
		content, err := os.ReadFile(filepath.Join(migrationsDir, filename)) //nolint:gosec
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		statements := strings.SplitSeq(string(content), ";")
		for stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w", filename, err)
			}
		}

		if err := recordMigration(db, filename); err != nil {
			return err
		}

		fmt.Printf("Applied migration: %s\n", filename)
	}

	return nil
}

func createHistoryTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func isMigrationApplied(db *sql.DB, name string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM migrations_history WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func recordMigration(db *sql.DB, name string) error {
	_, err := db.Exec("INSERT INTO migrations_history (name) VALUES (?)", name)
	return err
}
