package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"github.com/garrettladley/thoop"
	"github.com/garrettladley/thoop/internal/client/github"
	"github.com/garrettladley/thoop/internal/version"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "thoop",
		Short: "WHOOP data in your terminal",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := updateVersionIfNecessary(cmd.Context()); err != nil {
				return err
			}
			return runTUI(cmd, args)
		},
	}

	rootCmd.AddCommand(upgradeCmd())
	rootCmd.AddCommand(logoutCmd())
	rootCmd.AddCommand(purgeCmd())
	addDevCommands(rootCmd)

	if err := fang.Execute(context.Background(), rootCmd,
		fang.WithVersion(thoop.Version),
		fang.WithColorSchemeFunc(fang.AnsiColorScheme),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	); err != nil {
		os.Exit(1)
	}
}

func updateVersionIfNecessary(ctx context.Context) error {
	latest, err := github.NewClient().GetLatestThoopRelease(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	var (
		currentVersion = thoop.Version
		latestVersion  = latest.TagName
	)

	if !version.IsNewer(currentVersion, latestVersion) {
		return nil
	}

	verr := version.CheckCompatibilityBetween(currentVersion, latestVersion)
	if verr == nil {
		return nil
	}

	fmt.Printf("Update available: %s → %s\n", currentVersion, latestVersion)
	fmt.Print("Would you like to install it? [y/N]: ")

	if !confirm(ctx, os.Stdin) {
		fmt.Println("Exiting...")
		os.Exit(0)
		return nil
	}

	if err := upgrade(ctx, currentVersion, latest.TagName); err != nil {
		return err
	}

	fmt.Println("\nRun `thoop` again to use the new version.")
	os.Exit(0)
	return nil
}
