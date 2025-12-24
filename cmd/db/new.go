package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newMigrationCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			entries, err := os.ReadDir("migrations")
			if err != nil {
				return fmt.Errorf("failed to read migrations directory: %w", err)
			}

			var nextNum int
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".sql") {
					parts := strings.Split(entry.Name(), "_")
					if len(parts) > 0 {
						var num int
						if _, err := fmt.Sscanf(parts[0], "%d", &num); err == nil {
							if num > nextNum {
								nextNum = num
							}
						}
					}
				}
			}
			nextNum++

			filename := filepath.Join("migrations", fmt.Sprintf("%06d_%s.sql", nextNum, name))

			if _, err := os.Stat(filename); err == nil {
				return fmt.Errorf("migration file already exists: %s", filename)
			}

			content := fmt.Sprintf("-- Migration: %s\n\n", name)
			if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
				return fmt.Errorf("failed to create migration file: %w", err)
			}

			fmt.Printf("Created migration: %s\n", filename)
			return nil
		},
	}
}
