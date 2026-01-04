package main

import (
	"context"
	"os"
	"syscall"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"github.com/garrettladley/thoop/internal/version"
)

func main() {

	rootCmd := &cobra.Command{
		Use:     "thoop",
		Short:   "WHOOP data in your terminal",
		Version: version.Get(),
		RunE:    runTUI,
	}

	rootCmd.AddCommand(upgradeCmd())
	addDevCommands(rootCmd)

	if err := fang.Execute(context.Background(), rootCmd, fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)); err != nil {
		os.Exit(1)
	}
}
