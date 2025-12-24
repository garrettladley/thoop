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

	"github.com/garrettladley/thoop/internal/env"
	"github.com/garrettladley/thoop/internal/proxy"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xslog"
)

const (
	keyPort = "port"
	keyEnv  = "env"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()
	if err := run(ctx, logger); err != nil {
		logger.ErrorContext(ctx, "fatal error", xslog.Error(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := proxy.ReadConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	backend, err := initBackend(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize storage backend: %w", err)
	}
	defer func() {
		if err := backend.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close backend", xslog.Error(err))
		}
	}()

	handler := proxy.NewHandler(cfg, backend)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/start", handler.HandleAuthStart)
	mux.HandleFunc("GET /auth/callback", handler.HandleAuthCallback)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := backend.Ping(r.Context()); err != nil {
			logger.ErrorContext(r.Context(), "health check failed", xslog.Error(err))
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("unhealthy"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	wrapped := proxy.Chain(mux,
		proxy.Recovery(logger),
		proxy.RequestID,
		proxy.Logging(logger),
		proxy.RateLimitWithBackend(backend, logger),
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

func initBackend(ctx context.Context, cfg proxy.Config, logger *slog.Logger) (storage.Backend, error) {
	switch cfg.Env {
	case env.Production:
		if cfg.Redis.URL == "" {
			return nil, errors.New("REDIS_URL is required in production")
		}
		logger.InfoContext(ctx, "using Redis backend")
		return storage.NewRedisBackend(cfg.Redis, int(cfg.RateLimit.Limit))
	case env.Development:
		logger.InfoContext(ctx, "using in-memory backend (local development)")
		return storage.NewMemoryBackend(cfg.RateLimit.Limit, cfg.RateLimit.Burst), nil
	default:
		logger.InfoContext(ctx, "using in-memory backend (unknown environment)", slog.String(keyEnv, string(cfg.Env)))
		return storage.NewMemoryBackend(cfg.RateLimit.Limit, cfg.RateLimit.Burst), nil
	}
}
