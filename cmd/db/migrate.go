package main

import (
	"fmt"

	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := paths.EnsureDir(); err != nil {
				return err
			}

			dbPath, err := paths.DB()
			if err != nil {
				return err
			}

			sqlDB, _, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer func() {
				_ = sqlDB.Close()
			}()

			fmt.Println("Migrations applied successfully")
			return nil
		},
	}
}
