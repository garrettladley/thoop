package main

import (
	"fmt"
	"time"

	"github.com/garrettladley/thoop/internal/db"
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

			token, err := querier.GetToken(ctx)
			if err != nil {
				return fmt.Errorf("failed to get token: %w", err)
			}

			fmt.Printf("Access Token:  %s\n", token.AccessToken)
			if token.RefreshToken != nil {
				fmt.Printf("Refresh Token: %s\n", *token.RefreshToken)
			}
			fmt.Printf("Token Type:    %s\n", token.TokenType)
			fmt.Printf("Expiry:        %s\n", token.Expiry.Format(time.RFC3339))

			if token.Expiry.Before(time.Now()) {
				fmt.Printf("Status:        EXPIRED\n")
			} else {
				fmt.Printf("Status:        Valid (expires in %s)\n", time.Until(token.Expiry).Round(time.Second))
			}

			return nil
		},
	}
}
