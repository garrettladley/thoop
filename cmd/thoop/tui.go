package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/db"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/garrettladley/thoop/internal/tui"
	"github.com/garrettladley/thoop/internal/xslog"
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

			dbPath, err := paths.DB()
			if err != nil {
				return err
			}

			sqlDB, querier, err := db.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() { _ = sqlDB.Close() }()

			oauthCfg := oauth.NewConfig(cfg.Whoop)
			tokenSource := oauth.NewDBTokenSource(oauthCfg, querier)
			authFlow := oauth.NewProxyFlowWithURL(cfg.ProxyURL, querier)

			client := whoop.New(tokenSource, whoop.WithBaseURL(cfg.ProxyURL+"/api/whoop"))

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			logger := xslog.NewLoggerFromEnv(os.Stderr)
			logger.InfoContext(ctx, "starting thoop", xslog.Version())

			deps := tui.Deps{
				Ctx:          ctx,
				Cancel:       cancel,
				Logger:       logger,
				TokenChecker: tokenSource,
				TokenSource:  tokenSource,
				AuthFlow:     authFlow,
				WhoopClient:  client,
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
