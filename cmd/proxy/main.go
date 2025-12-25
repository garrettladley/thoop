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
	xredis "github.com/garrettladley/thoop/internal/redis"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xslog"
	"github.com/redis/go-redis/v9"
)

const (
	keyPort          = "port"
	keyEnv           = "env"
	keyPerUserMinute = "per_user_minute"
	keyPerUserDay    = "per_user_day"
	keyGlobalMinute  = "global_minute"
	keyGlobalDay     = "global_day"
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

	redisClient, err := xredis.New(ctx, xredis.Config{URL: cfg.Redis.URL})
	if err != nil {
		return fmt.Errorf("failed to initialize redis client: %w", err)
	}

	backend, err := initBackend(ctx, cfg, redisClient, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize storage backend: %w", err)
	}
	defer func() {
		if err := backend.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close backend", xslog.Error(err))
		}
	}()

	whoopLimiter, err := initWhoopLimiter(ctx, cfg, redisClient, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize WHOOP rate limiter: %w", err)
	}

	tokenCache := initTokenCache(ctx, cfg, redisClient, logger)
	tokenValidator := proxy.NewTokenValidator(tokenCache, logger)

	handler := proxy.NewHandler(cfg, backend)
	whoopHandler := proxy.NewWhoopHandler(cfg, whoopLimiter, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/start", handler.HandleAuthStart)
	mux.HandleFunc("GET /auth/callback", handler.HandleAuthCallback)

	whoopMux := http.NewServeMux()
	whoopMux.HandleFunc("/api/whoop/", whoopHandler.HandleWhoopProxy)
	whoopWrapped := proxy.Chain(whoopMux,
		proxy.WhoopAuth(tokenValidator, logger),
	)
	mux.Handle("/api/whoop/", whoopWrapped)

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

func initBackend(ctx context.Context, cfg proxy.Config, redisClient *redis.Client, logger *slog.Logger) (storage.Backend, error) {
	switch cfg.Env {
	case env.Production:
		logger.InfoContext(ctx, "using Redis backend")
		return storage.NewRedisBackend(storage.RedisConfig{Client: redisClient}, int(cfg.RateLimit.Limit))
	case env.Development:
		logger.InfoContext(ctx, "using in-memory backend (local development)")
		return storage.NewMemoryBackend(cfg.RateLimit.Limit, cfg.RateLimit.Burst), nil
	default:
		logger.InfoContext(ctx, "using in-memory backend (unknown environment)", slog.String(keyEnv, string(cfg.Env)))
		return storage.NewMemoryBackend(cfg.RateLimit.Limit, cfg.RateLimit.Burst), nil
	}
}

func initWhoopLimiter(ctx context.Context, cfg proxy.Config, redisClient *redis.Client, logger *slog.Logger) (storage.WhoopRateLimiter, error) {
	whoopCfg := storage.WhoopRateLimiterConfig{
		PerUserMinuteLimit: cfg.WhoopRateLimit.PerUserMinuteLimit,
		PerUserDayLimit:    cfg.WhoopRateLimit.PerUserDayLimit,
		GlobalMinuteLimit:  cfg.WhoopRateLimit.GlobalMinuteLimit,
		GlobalDayLimit:     cfg.WhoopRateLimit.GlobalDayLimit,
	}

	switch cfg.Env {
	case env.Production:
		logger.InfoContext(ctx, "using Redis WHOOP rate limiter",
			slog.Int(keyPerUserMinute, whoopCfg.PerUserMinuteLimit),
			slog.Int(keyPerUserDay, whoopCfg.PerUserDayLimit),
			slog.Int(keyGlobalMinute, whoopCfg.GlobalMinuteLimit),
			slog.Int(keyGlobalDay, whoopCfg.GlobalDayLimit))

		return storage.NewWhoopRedisLimiter(storage.RedisConfig{Client: redisClient}, whoopCfg), nil
	case env.Development:
		logger.InfoContext(ctx, "using in-memory WHOOP rate limiter",
			slog.Int(keyPerUserMinute, whoopCfg.PerUserMinuteLimit),
			slog.Int(keyPerUserDay, whoopCfg.PerUserDayLimit),
			slog.Int(keyGlobalMinute, whoopCfg.GlobalMinuteLimit),
			slog.Int(keyGlobalDay, whoopCfg.GlobalDayLimit))
		return storage.NewWhoopMemoryLimiter(whoopCfg), nil
	default:
		logger.InfoContext(ctx, "using in-memory WHOOP rate limiter (unknown environment)", slog.String(keyEnv, string(cfg.Env)))
		return storage.NewWhoopMemoryLimiter(whoopCfg), nil
	}
}

func initTokenCache(ctx context.Context, cfg proxy.Config, redisClient *redis.Client, logger *slog.Logger) storage.TokenCache {
	switch cfg.Env {
	case env.Production:
		logger.InfoContext(ctx, "using Redis token cache")
		return storage.NewRedisTokenCache(storage.RedisConfig{Client: redisClient})
	case env.Development:
		logger.InfoContext(ctx, "using in-memory token cache")
		return storage.NewMemoryTokenCache(time.Minute)
	default:
		logger.InfoContext(ctx, "using in-memory token cache (unknown environment)", slog.String(keyEnv, string(cfg.Env)))
		return storage.NewMemoryTokenCache(time.Minute)
	}
}
