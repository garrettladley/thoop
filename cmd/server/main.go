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

	xredis "github.com/garrettladley/thoop/internal/redis"
	"github.com/garrettladley/thoop/internal/server"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp/middleware"
	"github.com/garrettladley/thoop/internal/xslog"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

const (
	keyPort          = "port"
	keyPerUserMinute = "per_user_minute"
	keyPerUserDay    = "per_user_day"
	keyGlobalMinute  = "global_minute"
	keyGlobalDay     = "global_day"
)

func main() {
	_ = godotenv.Load()

	logger := xslog.NewLoggerFromEnv(os.Stdout)
	slog.SetDefault(logger)

	ctx := context.Background()
	if err := run(ctx, logger); err != nil {
		logger.ErrorContext(ctx, "fatal error", xslog.Error(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := server.ReadConfig()
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

	whoopLimiter := initWhoopLimiter(ctx, cfg, redisClient, logger)
	tokenCache := initTokenCache(ctx, redisClient, logger)
	tokenValidator := server.NewTokenValidator(tokenCache, whoopLimiter)
	notificationStore := initNotificationStore(ctx, redisClient, logger)

	handler := server.NewHandler(cfg, backend)
	whoopHandler := server.NewWhoopHandler(cfg, whoopLimiter)
	webhookHandler := server.NewWebhookHandler(cfg.Whoop.ClientSecret, notificationStore)
	sseHandler := server.NewSSEHandler(notificationStore)
	notificationsHandler := server.NewNotificationsHandler(notificationStore)

	mux := http.NewServeMux()

	// Unauthenticated routes - protected by global IP rate limiter
	unauthedMux := http.NewServeMux()
	unauthedMux.HandleFunc("GET /auth/start", handler.HandleAuthStart)
	unauthedMux.HandleFunc("GET /auth/callback", handler.HandleAuthCallback)
	unauthedMux.HandleFunc("POST /webhooks/whoop", webhookHandler.HandleWebhook)
	unauthedMux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	unauthedWrapped := middleware.Chain(unauthedMux,
		server.RateLimitWithBackend(backend),
	)
	mux.Handle("/auth/", unauthedWrapped)
	mux.Handle("/webhooks/", unauthedWrapped)
	mux.Handle("/health", unauthedWrapped)

	// Authenticated routes - protected by per-user WHOOP rate limiter
	whoopMux := http.NewServeMux()
	whoopMux.HandleFunc("/api/whoop/", whoopHandler.HandleWhoopProxy)
	whoopWrapped := middleware.Chain(whoopMux,
		middleware.VersionCheck,
		server.WhoopAuth(tokenValidator),
	)
	mux.Handle("/api/whoop/", whoopWrapped)

	notificationsMux := http.NewServeMux()
	notificationsMux.HandleFunc("GET /api/notifications", notificationsHandler.HandlePoll)
	notificationsMux.HandleFunc("GET /api/notifications/stream", sseHandler.HandleStream)
	notificationsWrapped := middleware.Chain(notificationsMux,
		server.WhoopAuth(tokenValidator),
	)
	mux.Handle("/api/notifications", notificationsWrapped)
	mux.Handle("/api/notifications/stream", notificationsWrapped)

	wrapped := middleware.Chain(mux,
		middleware.Recovery,
		middleware.Logging,
		middleware.Logger(logger),
		middleware.RequestID(),
		middleware.ClientSessionID,
		middleware.SecurityHeaders,
	)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           wrapped,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // disabled for SSE; use SetWriteDeadline per-request
		IdleTimeout:       60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.InfoContext(ctx, "starting server",
			xslog.Version(),
			slog.String(keyPort, cfg.Port))
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

func initBackend(ctx context.Context, cfg server.Config, redisClient *redis.Client, logger *slog.Logger) (storage.Backend, error) {
	logger.InfoContext(ctx, "initializing Redis backend")
	return storage.NewRedisBackend(storage.RedisConfig{Client: redisClient}, int(cfg.RateLimit.Limit))
}

func initWhoopLimiter(ctx context.Context, cfg server.Config, redisClient *redis.Client, logger *slog.Logger) storage.WhoopRateLimiter {
	whoopCfg := storage.WhoopRateLimiterConfig{
		PerUserMinuteLimit: cfg.WhoopRateLimit.PerUserMinuteLimit,
		PerUserDayLimit:    cfg.WhoopRateLimit.PerUserDayLimit,
		GlobalMinuteLimit:  cfg.WhoopRateLimit.GlobalMinuteLimit,
		GlobalDayLimit:     cfg.WhoopRateLimit.GlobalDayLimit,
	}

	logger.InfoContext(ctx, "initializing WHOOP rate limiter",
		slog.Int(keyPerUserMinute, whoopCfg.PerUserMinuteLimit),
		slog.Int(keyPerUserDay, whoopCfg.PerUserDayLimit),
		slog.Int(keyGlobalMinute, whoopCfg.GlobalMinuteLimit),
		slog.Int(keyGlobalDay, whoopCfg.GlobalDayLimit))

	return storage.NewWhoopRedisLimiter(storage.RedisConfig{Client: redisClient}, whoopCfg)
}

func initTokenCache(ctx context.Context, redisClient *redis.Client, logger *slog.Logger) storage.TokenCache {
	logger.InfoContext(ctx, "initializing token cache")
	return storage.NewRedisTokenCache(storage.RedisConfig{Client: redisClient})
}

func initNotificationStore(ctx context.Context, redisClient *redis.Client, logger *slog.Logger) storage.NotificationStore {
	logger.InfoContext(ctx, "initializing notification store")
	return storage.NewRedisNotificationStore(redisClient)
}
