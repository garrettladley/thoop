package main

import (
	"context"
	"os"
	"syscall"

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

	if err := fang.Execute(context.Background(), rootCmd, fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)); err != nil {
		os.Exit(1)
	}
}
