package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/garrettladley/thoop/internal/proxy"
	"github.com/garrettladley/thoop/internal/xslog"
)

const (
	keyPort  = "port"
	keyError = "error"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()
	if err := run(ctx, logger); err != nil {
		logger.ErrorContext(ctx, "fatal error", slog.Any(keyError, err))
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := proxy.ReadConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	handler := proxy.NewHandler(cfg)
	defer handler.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/start", handler.HandleAuthStart)
	mux.HandleFunc("GET /auth/callback", handler.HandleAuthCallback)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	wrapped := proxy.Chain(mux,
		proxy.Recovery(logger),
		proxy.RequestID,
		proxy.Logging(logger),
		proxy.RateLimit(cfg.RateLimit, cfg.RateBurst),
		proxy.SecurityHeaders,
	)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           wrapped,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.InfoContext(ctx, "starting proxy server", slog.String(keyPort, cfg.Port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "server error", xslog.Error(err))
		}
	}()

	<-done
	logger.InfoContext(ctx, "shutting down server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	logger.InfoContext(ctx, "server stopped")
	return nil
}
