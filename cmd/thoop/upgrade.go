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
			latest, err := github.NewClient().GetLatestRelease(cmd.Context(), "garrettladley", "thoop")
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w", err)
			}

			currentVersion := version.Get()
			if !version.IsNewer(currentVersion, latest.TagName) {
				fmt.Printf("thoop is up to date (%s)\n", currentVersion)
				return nil
			}

			return upgrade(cmd.Context(), currentVersion, latest.TagName)
		},
	}
}

func upgrade(ctx context.Context, currentVersion, latestVersion string) error {
	fmt.Printf("Updating thoop %s → %s\n", currentVersion, latestVersion)

	if version.IsHomebrew() {
		return brewUpgrade(ctx)
	}
	return goInstallUpgrade(ctx)
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
	fmt.Println("Updating brew...")
	update := exec.CommandContext(ctx, "brew", "update")
	update.Stdout = os.Stdout
	update.Stderr = os.Stderr
	if err := update.Run(); err != nil {
		return fmt.Errorf("brew update failed: %w", err)
	}

	fmt.Println("Updating thoop...")
	cmd := exec.CommandContext(ctx, "brew", "upgrade", "--cask", "thoop")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}
	return nil
}
