package main

import (
	"fmt"
	"time"

	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/keyring"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/spf13/cobra"
)

func tokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Show the stored OAuth token",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			dbPath, err := paths.DB()
			if err != nil {
				return fmt.Errorf("failed to get database path: %w", err)
			}

			sqlDB, querier, err := db.Open(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = sqlDB.Close()
			}()

			kr := keyring.NewOSKeyring()

			// get metadata from SQLite
			metadata, err := querier.GetTokenMetadata(ctx)
			if err != nil {
				return fmt.Errorf("failed to get token metadata: %w", err)
			}

			// get secrets from keyring
			accessToken, err := kr.Get(keyring.KeyAccessToken)
			if err != nil {
				return fmt.Errorf("failed to get access token from keyring: %w", err)
			}

			refreshToken, _ := kr.Get(keyring.KeyRefreshToken)
			apiKey, _ := kr.Get(keyring.KeyAPIKey)

			fmt.Printf("Access Token:  %s\n", accessToken)
			if refreshToken != "" {
				fmt.Printf("Refresh Token: %s\n", refreshToken)
			}
			if apiKey != "" {
				fmt.Printf("API Key:       %s\n", apiKey)
			}
			fmt.Printf("Token Type:    %s\n", metadata.TokenType)
			fmt.Printf("Expiry:        %s\n", metadata.Expiry.Format(time.RFC3339))

			if metadata.Expiry.Before(time.Now()) {
				fmt.Printf("Status:        EXPIRED\n")
			} else {
				fmt.Printf("Status:        Valid (expires in %s)\n", time.Until(metadata.Expiry).Round(time.Second))
			}

			return nil
		},
	}
}
