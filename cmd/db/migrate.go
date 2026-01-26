package main

import (
	"fmt"

	"github.com/garrettladley/thoop/internal/paths"
	"github.com/garrettladley/thoop/internal/sqlite"
	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := paths.EnsureDir(); err != nil {
				return fmt.Errorf("failed to ensure directory: %w", err)
			}

			dbPath, err := paths.DB()
			if err != nil {
				return fmt.Errorf("failed to get database path: %w", err)
			}

			db, err := sqlite.New(cmd.Context(), sqlite.DefaultConfig(dbPath))
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = db.Close()
			}()

			fmt.Println("Migrations applied successfully")
			return nil
		},
	}
}
