package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/garrettladley/thoop/internal/migrations/postgres"
	"github.com/garrettladley/thoop/internal/oauth"
	xredis "github.com/garrettladley/thoop/internal/redis"
	"github.com/garrettladley/thoop/internal/server"
	"github.com/garrettladley/thoop/internal/server/handler"
	servermw "github.com/garrettladley/thoop/internal/server/middleware"
	"github.com/garrettladley/thoop/internal/service/auth"
	"github.com/garrettladley/thoop/internal/service/notification"
	"github.com/garrettladley/thoop/internal/service/proxy"
	"github.com/garrettladley/thoop/internal/service/token"
	"github.com/garrettladley/thoop/internal/service/user"
	"github.com/garrettladley/thoop/internal/service/webhook"
	pgc "github.com/garrettladley/thoop/internal/sqlc/postgres"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp/middleware"
	"github.com/garrettladley/thoop/internal/xslog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

const (
	keyPort          = "port"
	keyPerUserMinute = "per_user_minute"
	keyPerUserDay    = "per_user_day"
	keyGlobalMinute  = "global_minute"
	keyGlobalDay     = "global_day"
	keyGracePeriod   = "grace_period"

	sseShutdownGracePeriod = 2 * time.Second
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

	pool, err := initPostgres(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize postgres: %w", err)
	}
	defer pool.Close()

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
	notificationStore := initNotificationStore(ctx, pool, redisClient, logger)

	// Database layer
	queries := pgc.New(pool)

	// Services
	userService := user.NewPostgresService(queries)
	tokenService := token.NewValidator(tokenCache, whoopLimiter)
	notificationService := notification.NewStore(notificationStore)
	webhookService := webhook.NewProcessor(cfg.Whoop.ClientSecret, notificationStore)
	proxyService := proxy.NewProxy(whoopLimiter, proxy.RateLimitConfig{
		PerUserMinuteLimit: cfg.WhoopRateLimit.PerUserMinuteLimit,
		PerUserDayLimit:    cfg.WhoopRateLimit.PerUserDayLimit,
		GlobalMinuteLimit:  cfg.WhoopRateLimit.GlobalMinuteLimit,
		GlobalDayLimit:     cfg.WhoopRateLimit.GlobalDayLimit,
	})
	authService := auth.NewOAuth(
		oauth.NewConfig(cfg),
		backend, // StateStore
		userService,
		whoopLimiter,
	)

	// Handlers
	authHandler := handler.NewAuth(authService)
	webhookHandler := handler.NewWebhook(webhookService)
	proxyHandler := handler.NewProxy(proxyService)
	notificationsHandler := handler.NewNotifications(notificationService)
	sseHandler := handler.NewSSE(notificationService)

	mux := http.NewServeMux()

	// Unauthenticated routes - protected by global IP rate limiter
	unauthedMux := http.NewServeMux()
	unauthedMux.HandleFunc("GET /auth/start", authHandler.HandleAuthStart)
	unauthedMux.HandleFunc("GET /auth/callback", authHandler.HandleAuthCallback)
	unauthedMux.HandleFunc("POST /webhooks/whoop", webhookHandler.HandleWebhook)
	unauthedMux.HandleFunc("GET /health", handler.HandleHealth)
	unauthedWrapped := middleware.Chain(unauthedMux,
		servermw.RateLimitWithBackend(backend),
	)
	mux.Handle("/auth/", unauthedWrapped)
	mux.Handle("/webhooks/", unauthedWrapped)
	mux.Handle("/health", unauthedWrapped)

	// Authenticated routes - protected by API key + WHOOP token + rate limiter
	whoopMux := http.NewServeMux()
	whoopMux.HandleFunc("/api/whoop/", proxyHandler.HandleWhoopProxy)
	whoopWrapped := middleware.Chain(whoopMux,
		middleware.VersionCheck,
		servermw.APIKeyAuth(userService),
		servermw.BearerAuth(tokenService),
	)
	mux.Handle("/api/whoop/", whoopWrapped)

	notificationsMux := http.NewServeMux()
	notificationsMux.HandleFunc("GET /api/notifications", notificationsHandler.HandlePoll)
	notificationsMux.HandleFunc("POST /api/notifications/ack", notificationsHandler.HandleAcknowledge)
	notificationsMux.HandleFunc("GET /api/notifications/stream", sseHandler.HandleStream)
	notificationsWrapped := middleware.Chain(notificationsMux,
		servermw.APIKeyAuth(userService),
		servermw.BearerAuth(tokenService),
	)
	mux.Handle("/api/notifications", notificationsWrapped)
	mux.Handle("/api/notifications/ack", notificationsWrapped)
	mux.Handle("/api/notifications/stream", notificationsWrapped)

	wrapped := middleware.Chain(mux,
		middleware.Recovery,
		middleware.Logging,
		middleware.Logger(logger),
		middleware.ShutdownContext,
		middleware.RequestID(),
		middleware.ClientSessionID,
		middleware.SecurityHeaders,
	)

	shutdownCoordinator := server.NewShutdownCoordinator(sseShutdownGracePeriod)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           wrapped,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // disabled for SSE; use SetWriteDeadline per-request
		IdleTimeout:       60 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return shutdownCoordinator.BaseContext()
		},
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.InfoContext(ctx, "starting server",
			xslog.Version(),
			slog.String(keyPort, cfg.Port))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "server error", xslog.Error(err))
		}
	}()

	<-done
	logger.InfoContext(ctx, "shutdown signal received, initiating graceful shutdown")

	// cancel base context and wait grace period for SSE connections to close
	shutdownCoordinator.InitiateShutdown()
	logger.InfoContext(ctx, "SSE grace period complete, shutting down server",
		slog.Duration(keyGracePeriod, sseShutdownGracePeriod))

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
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

func initNotificationStore(ctx context.Context, pool *pgxpool.Pool, redisClient *redis.Client, logger *slog.Logger) storage.NotificationStore {
	logger.InfoContext(ctx, "initializing notification store (PostgreSQL + Redis pub/sub)")
	return storage.NewHybridNotificationStore(pool, redisClient)
}

func initPostgres(ctx context.Context, cfg server.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	logger.InfoContext(ctx, "initializing PostgreSQL")

	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if err := postgres.Apply(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return pool, nil
}
