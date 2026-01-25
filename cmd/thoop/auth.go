//go:build dev

package main

import (
	"fmt"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/keyring"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/spf13/cobra"
)

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with WHOOP",
		Long:  "Opens browser to authenticate with WHOOP and stores the token locally.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if _, err := paths.EnsureDir(); err != nil {
				return fmt.Errorf("failed to ensure directory: %w", err)
			}

			dbPath, err := paths.DB()
			if err != nil {
				return fmt.Errorf("failed to get database path: %w", err)
			}

			sqlDB, querier, err := db.Open(ctx, dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = sqlDB.Close()
			}()

			kr := keyring.NewOSKeyring()
			if !kr.Available() {
				return fmt.Errorf("OS keyring is not available")
			}

			flow := oauth.NewServerFlow(config.ServerURL, querier, kr)

			result, err := flow.Run(ctx)
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			fmt.Printf("Authentication successful!\n")
			fmt.Printf("Token expires: %s\n", result.Token.Expiry.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	cmd.AddCommand(purgeCmd())

	return cmd
}

func purgeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "purge",
		Short: "Remove stored authentication token",
		Long:  "Deletes the locally stored WHOOP authentication token from keyring and database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if _, err := paths.EnsureDir(); err != nil {
				return fmt.Errorf("failed to ensure directory: %w", err)
			}

			dbPath, err := paths.DB()
			if err != nil {
				return fmt.Errorf("failed to get database path: %w", err)
			}

			sqlDB, querier, err := db.Open(ctx, dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = sqlDB.Close()
			}()

			kr := keyring.NewOSKeyring()

			tokenSource := oauth.NewProxyTokenSource(config.ServerURL, querier, kr)

			// get API key from keyring for revocation call
			apiKey, _ := kr.Get(keyring.KeyAPIKey)

			client := whoop.New(tokenSource,
				whoop.WithProxyURL(config.ServerURL+"/api/whoop"),
				whoop.WithAPIKey(apiKey),
			)
			_ = client.User.RevokeAccess(ctx) // best effort - token may already be invalid

			// delete from keyring
			if err := kr.DeleteAll(); err != nil {
				fmt.Printf("Warning: failed to clear keyring: %v\n", err)
			}

			// Delete metadata from SQLite
			if err := querier.DeleteTokenMetadata(ctx); err != nil {
				return fmt.Errorf("failed to delete token metadata: %w", err)
			}

			fmt.Println("Authentication token removed successfully.")

			return nil
		},
	}
}
