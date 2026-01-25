package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/keyring"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
)

func purgeCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete all local data and revoke WHOOP access",
		Long: `Permanently delete all thoop data from your computer and revoke access:
  - Revokes thoop's access to your WHOOP account
  - Deletes OAuth tokens and API keys from your system keyring
  - Deletes cached WHOOP data (cycles, sleep, recovery, workouts)
  - Deletes log files and configuration`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPurge(cmd.Context(), force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runPurge(ctx context.Context, force bool) error {
	dir, err := paths.Dir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	if !force {
		fmt.Println("This will:")
		fmt.Println("  - Revoke thoop's access to your WHOOP account")
		fmt.Println("  - Delete all credentials from your system keyring")
		fmt.Printf("  - Delete all data in %s\n", dir)
		fmt.Print("\nAre you sure? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input != "y" && input != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// revoke access before deleting credentials
	if err := revokeAccess(ctx); err != nil {
		fmt.Printf("Warning: failed to revoke access: %v\n", err)
	}

	// delete from keyring
	kr := keyring.NewOSKeyring()
	if err := kr.DeleteAll(); err != nil {
		fmt.Printf("Warning: failed to clear keyring: %v\n", err)
	}

	// delete the entire thoop directory
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to delete %s: %w", dir, err)
	}

	fmt.Println("All local data has been deleted and WHOOP access revoked.")
	return nil
}

func revokeAccess(ctx context.Context) error {
	dbPath, err := paths.DB()
	if err != nil {
		return err
	}

	sqlDB, querier, err := db.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	kr := keyring.NewOSKeyring()
	tokenSource := oauth.NewProxyTokenSource(config.ServerURL, querier, kr)

	// go directly to WHOOP, not through proxy
	client := whoop.New(tokenSource)

	return client.User.RevokeAccess(ctx)
}
