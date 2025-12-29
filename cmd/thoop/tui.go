package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/garrettladley/thoop/internal/client/sse"
	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/garrettladley/thoop/internal/repository"
	"github.com/garrettladley/thoop/internal/session"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/tui"
	"github.com/garrettladley/thoop/internal/xslog"
	"github.com/garrettladley/thoop/internal/xsync"
)

func tuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive TUI",
		Long:  "Opens the full-screen terminal UI for viewing WHOOP data.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Read()
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			if _, err := paths.EnsureDir(); err != nil {
				return err
			}

			if _, err := paths.EnsureLogsDir(); err != nil {
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
			defer func() { _ = sqlDB.Close() }()

			sessionID := session.NewID()
			logPath, err := paths.LogFile(sessionID)
			if err != nil {
				return err
			}
			logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // logPath is from trusted paths package
			if err != nil {
				return fmt.Errorf("failed to open log file: %w", err)
			}
			defer func() { _ = logFile.Close() }()

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			baseLogger := xslog.NewTextLoggerFromEnv(logFile)
			logger := baseLogger.With(xslog.SessionID(sessionID))

			oauthCfg := oauth.NewConfig(cfg.Whoop)
			tokenSource := oauth.NewDBTokenSource(oauthCfg, querier)
			authFlow := oauth.NewServerFlowWithURL(cfg.ProxyURL, querier)

			var apiKey string
			if apiKeyPtr, err := querier.GetAPIKey(ctx); err == nil && apiKeyPtr != nil {
				apiKey = *apiKeyPtr
			}

			client := whoop.New(tokenSource,
				whoop.WithProxyURL(cfg.ProxyURL+"/api/whoop"),
				whoop.WithSessionID(sessionID),
				whoop.WithAPIKey(apiKey),
				whoop.WithLogger(logger),
			)
			logger.InfoContext(ctx, "starting thoop", xslog.Version())

			repo := repository.New(querier)
			syncSvc := xsync.NewService(client, repo, logger)
			dataFetcher := xsync.NewFetcher(client, repo, logger)

			sseClient := sse.NewClient(cfg.ProxyURL, tokenSource, sessionID, apiKey, logger)
			notifProcessor := xsync.NewNotificationProcessor(client, repo, logger)
			notifChan := make(chan storage.Notification, 10)

			if hasToken, _ := tokenSource.HasToken(ctx); hasToken {
				if err := syncSvc.RefreshCurrent(ctx); err != nil {
					logger.WarnContext(ctx, "failed to refresh current data", xslog.Error(err))
				}

				if complete, err := syncSvc.IsBackfillComplete(ctx); err == nil && !complete {
					if err := syncSvc.StartBackfill(ctx); err != nil {
						logger.WarnContext(ctx, "failed to start backfill", xslog.Error(err))
					}
				}
			}

			deps := tui.Deps{
				Ctx:              ctx,
				Cancel:           cancel,
				Logger:           logger,
				TokenChecker:     tokenSource,
				TokenSource:      tokenSource,
				AuthFlow:         authFlow,
				WhoopClient:      client,
				Repository:       repo,
				SyncService:      syncSvc,
				DataFetcher:      dataFetcher,
				SSEClient:        sseClient,
				NotifProcessor:   notifProcessor,
				NotificationChan: notifChan,
			}
			model := tui.New(deps)

			p := tea.NewProgram(&model)

			if _, err := p.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}

			return nil
		},
	}
}
