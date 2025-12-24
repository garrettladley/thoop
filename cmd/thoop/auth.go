package main

import (
	"fmt"

	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/spf13/cobra"
)

func authCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with WHOOP",
		Long:  "Opens browser to authenticate with WHOOP and stores the token locally.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cfg, err := config.Read()
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			if _, err := paths.EnsureDir(); err != nil {
				return err
			}

			dbPath, err := paths.DB()
			if err != nil {
				return err
			}

			sqlDB, querier, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				_ = sqlDB.Close()
			}()

			flow := oauth.NewProxyFlowWithURL(cfg.ProxyURL, querier)

			token, err := flow.Run(ctx)
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			fmt.Printf("Authentication successful!\n")
			fmt.Printf("Token expires: %s\n", token.Expiry.Format("2006-01-02 15:04:05"))

			return nil
		},
	}
}
