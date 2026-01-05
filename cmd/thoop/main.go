package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

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
	addDevCommands(rootCmd)

	if err := fang.Execute(context.Background(), rootCmd,
		fang.WithVersion(version.Get()),
		fang.WithColorSchemeFunc(fang.AnsiColorScheme),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	); err != nil {
		os.Exit(1)
	}
}

func updateVersionIfNecessary(ctx context.Context) error {
	latest, err := github.NewClient().GetLatestThoopRelease(ctx)
	if err != nil {
		return err
	}

	var (
		currentVersion = version.Get()
		latestVersion  = latest.TagName
	)

	verr := version.CheckCompatibilityBetween(currentVersion, latestVersion)
	if verr == nil {
		return nil
	}

	fmt.Printf("Updated required: %s\n", verr.Error())
	fmt.Print("Would you like to install it? [y/N]: ")

	if !confirm(bufio.NewReader(os.Stdin)) {
		fmt.Print("Exiting...\n")
		os.Exit(0)
		return nil
	}

	return upgrade(ctx, currentVersion, latest.TagName)
}
