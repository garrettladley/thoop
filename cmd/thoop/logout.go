package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/garrettladley/thoop/internal/keyring"
	"github.com/garrettladley/thoop/internal/paths"
)

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear all stored credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var errs []error

			// delete SQLite database
			dbPath, err := paths.DB()
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to get db path: %w", err))
			} else if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
				errs = append(errs, fmt.Errorf("failed to delete database: %w", err))
			}

			// delete keyring credentials
			kr := keyring.NewOSKeyring()
			if err := kr.DeleteAll(); err != nil {
				errs = append(errs, fmt.Errorf("failed to delete keyring credentials: %w", err))
			}

			if len(errs) > 0 {
				for _, e := range errs {
					fmt.Fprintf(os.Stderr, "Warning: %v\n", e)
				}
				return fmt.Errorf("logout completed with errors")
			}

			fmt.Println("Successfully logged out")
			return nil
		},
	}
}
