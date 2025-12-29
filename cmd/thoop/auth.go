package main

import (
	"fmt"
	"os"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/garrettladley/thoop/internal/repository"
	"github.com/garrettladley/thoop/internal/xslog"
	"github.com/garrettladley/thoop/internal/xsync"
	"github.com/spf13/cobra"
)

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
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

			// start backfill in background after successful auth
			logger := xslog.NewLoggerFromEnv(os.Stderr)
			oauthCfg := oauth.NewConfig(cfg.Whoop)
			tokenSource := oauth.NewDBTokenSource(oauthCfg, querier)
			client := whoop.New(tokenSource, whoop.WithProxyURL(cfg.ProxyURL+"/api/whoop"))
			repo := repository.New(querier)
			syncSvc := xsync.NewService(client, repo, logger)

			fmt.Println("Starting background data sync...")
			if err := syncSvc.StartBackfill(ctx); err != nil {
				logger.WarnContext(ctx, "failed to start backfill", xslog.Error(err))
			}

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
		Long:  "Deletes the locally stored WHOOP authentication token from the database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cfg, err := config.Read()
			if err != nil {
				return err
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

			oauthCfg := oauth.NewConfig(cfg.Whoop)
			tokenSource := oauth.NewDBTokenSource(oauthCfg, querier)

			client := whoop.New(tokenSource, whoop.WithProxyURL(cfg.ProxyURL+"/api/whoop"))
			_ = client.User.RevokeAccess(ctx) // best effort - token may already be invalid

			if err := querier.DeleteToken(ctx); err != nil {
				return fmt.Errorf("failed to delete token: %w", err)
			}

			fmt.Println("Authentication token removed successfully.")

			return nil
		},
	}
}
