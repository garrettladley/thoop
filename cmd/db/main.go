package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
	}
	rootCmd.AddCommand(newMigrationCmd())
	rootCmd.AddCommand(migrateCmd())
	rootCmd.AddCommand(tokenCmd())

	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}
