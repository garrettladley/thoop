package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/garrettladley/thoop/internal/client/github"
	"github.com/garrettladley/thoop/internal/version"
)

func upgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Check for updates and install if available",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			currentVersion := version.Get()

			client := github.NewClient()
			latest, err := client.GetLatestRelease(ctx, "garrettladley", "thoop")
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w", err)
			}

			if !version.IsNewer(currentVersion, latest.TagName) {
				fmt.Printf("thoop is up to date (%s)\n", currentVersion)
				return nil
			}

			fmt.Printf("Updating thoop %s → %s\n", currentVersion, latest.TagName)

			if version.IsHomebrew() {
				return brewUpgrade(ctx)
			}

			return goInstallUpgrade(ctx)
		},
	}
}

func goInstallUpgrade(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "install", "github.com/garrettladley/thoop/cmd/thoop@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}
	fmt.Println("Successfully updated!")
	return nil
}

func brewUpgrade(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "brew", "upgrade", "thoop")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}
	return nil
}
