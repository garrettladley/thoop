package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/keyring"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
)

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and clear all stored credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runLogout(cmd.Context())
		},
	}
}

func runLogout(ctx context.Context) error {
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

	// go directly to WHOOP, not through proxy
	client := whoop.New(tokenSource)
	_ = client.User.RevokeAccess(ctx) // best effort - token may already be invalid

	// delete from keyring
	if err := kr.DeleteAll(); err != nil {
		fmt.Printf("Warning: failed to clear keyring: %v\n", err)
	}

	// delete token metadata from SQLite
	if err := querier.DeleteTokenMetadata(ctx); err != nil {
		return fmt.Errorf("failed to delete token metadata: %w", err)
	}

	fmt.Println("Successfully logged out")
	return nil
}
