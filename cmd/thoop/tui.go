package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/garrettladley/thoop/internal/cache"
	"github.com/garrettladley/thoop/internal/client/sse"
	"github.com/garrettladley/thoop/internal/client/whoop"
	"github.com/garrettladley/thoop/internal/config"
	"github.com/garrettladley/thoop/internal/keyring"
	"github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/paths"
	"github.com/garrettladley/thoop/internal/repository"
	"github.com/garrettladley/thoop/internal/session"
	"github.com/garrettladley/thoop/internal/sqlite"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/tui"
	"github.com/garrettladley/thoop/internal/xslog"
	"github.com/garrettladley/thoop/internal/xsync"
)

func runTUI(cmd *cobra.Command, _ []string) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	if _, err := paths.EnsureDir(); err != nil {
		return fmt.Errorf("failed to ensure directory: %w", err)
	}

	if _, err := paths.EnsureLogsDir(); err != nil {
		return fmt.Errorf("failed to ensure logs directory: %w", err)
	}

	dbPath, err := paths.DB()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	db, err := sqlite.New(ctx, sqlite.DefaultConfig(dbPath))
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	kr := keyring.NewOSKeyring()
	if !kr.Available() {
		fmt.Fprintln(os.Stderr, "Error: OS keyring is not available.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "On Linux, thoop requires a secret service for secure credential storage.")
		fmt.Fprintln(os.Stderr, "Please ensure one of the following is installed and running:")
		fmt.Fprintln(os.Stderr, "  - gnome-keyring (GNOME)")
		fmt.Fprintln(os.Stderr, "  - kwallet (KDE)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "For headless servers, consider running with D-Bus session:")
		fmt.Fprintln(os.Stderr, "  dbus-run-session -- thoop")
		return fmt.Errorf("keyring unavailable")
	}

	sessionID := session.NewID()
	logPath, err := paths.LogFile(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get log file path: %w", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // logPath is from trusted paths package
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	baseLogger := xslog.NewTextLoggerFromEnv(logFile)
	logger := baseLogger.With(xslog.SessionID(sessionID))
	slog.SetDefault(logger)

	tokenSource := oauth.NewProxyTokenSource(config.ServerURL, db, kr)
	authFlow := oauth.NewServerFlow(config.ServerURL, db, kr)

	apiKey, _ := kr.Get(keyring.KeyAPIKey)

	client := whoop.New(tokenSource,
		whoop.WithProxyURL(config.ServerURL+"/api/whoop"),
		whoop.WithSessionID(sessionID),
		whoop.WithAPIKey(apiKey),
		whoop.WithLogger(logger),
	)
	logger.InfoContext(ctx, "starting thoop", xslog.Version())

	repo := repository.New(db)
	cacheSvc := cache.NewService(client, repo)

	sseClient := sse.NewClient(config.ServerURL, tokenSource, sessionID, apiKey, logger)
	notifProcessor := xsync.NewNotificationProcessor(client, repo, logger)
	notifChan := make(chan storage.Notification, 10)

	deps := tui.Deps{
		Ctx:              ctx,
		Cancel:           cancel,
		Logger:           logger,
		TokenChecker:     tokenSource,
		TokenSource:      tokenSource,
		AuthFlow:         authFlow,
		CacheService:     cacheSvc,
		SSEClient:        sseClient,
		NotifProcessor:   notifProcessor,
		NotificationChan: notifChan,
	}
	model := tui.New(deps)

	p := tea.NewProgram(&model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return fmt.Errorf("running tui: %w", err)
	}

	return nil
}
