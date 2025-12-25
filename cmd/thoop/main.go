package main

import (
	"context"
	"os"
	"syscall"

	"github.com/charmbracelet/fang"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func main() {
	_ = godotenv.Load()

	rootCmd := &cobra.Command{
		Use:   "thoop",
		Short: "WHOOP API client",
	}

	rootCmd.AddCommand(authCmd())
	rootCmd.AddCommand(testCmd())
	rootCmd.AddCommand(tuiCmd())

	if err := fang.Execute(context.Background(), rootCmd, fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)); err != nil {
		os.Exit(1)
	}
}
