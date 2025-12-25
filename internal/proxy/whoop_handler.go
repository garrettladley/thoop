package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xslog"
	go_json "github.com/goccy/go-json"
)

const whoopAPIURL = "https://api.prod.whoop.com/developer"

type WhoopHandler struct {
	config       Config
	whoopLimiter storage.WhoopRateLimiter
	logger       *slog.Logger
}

func NewWhoopHandler(cfg Config, whoopLimiter storage.WhoopRateLimiter, logger *slog.Logger) *WhoopHandler {
	return &WhoopHandler{
		config:       cfg,
		whoopLimiter: whoopLimiter,
		logger:       logger,
	}
}

func (h *WhoopHandler) HandleWhoopProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	whoopUserID, ok := GetWhoopUserID(ctx)
	if !ok || whoopUserID == "" {
		h.logger.WarnContext(ctx, "missing user key in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	state, err := h.whoopLimiter.CheckAndIncrement(ctx, whoopUserID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to check rate limit",
			xslog.Error(err),
			slog.String(keyWhoopUserID, whoopUserID))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	logAttrs := []any{
		slog.String("whoop_user_id", whoopUserID),
		slog.Bool("allowed", state.Allowed),
		slog.Int("minute_remaining", state.MinuteRemaining),
		slog.Int("day_remaining", state.DayRemaining),
	}
	if state.Reason != nil {
		logAttrs = append(logAttrs, slog.String("reason", string(*state.Reason)))
	}
	h.logger.InfoContext(ctx, "rate limit check", logAttrs...)

	if !state.Allowed {
		var (
			retryAfter int
			message    string
		)

		switch *state.Reason {
		case storage.WhoopRateLimitReasonPerUserMinute:
			retryAfter = int(time.Until(state.MinuteReset).Seconds())
			message = fmt.Sprintf("Per-user rate limit exceeded (%d requests/minute)", h.config.WhoopRateLimit.PerUserMinuteLimit)
		case storage.WhoopRateLimitReasonPerUserDay:
			retryAfter = int(time.Until(state.DayReset).Seconds())
			message = fmt.Sprintf("Per-user rate limit exceeded (%d requests/day)", h.config.WhoopRateLimit.PerUserDayLimit)
		case storage.WhoopRateLimitReasonGlobalMinute:
			retryAfter = int(time.Until(state.MinuteReset).Seconds())
			message = "Global rate limit exceeded (app quota exhausted for this minute)"
		case storage.WhoopRateLimitReasonGlobalDay:
			retryAfter = int(time.Until(state.DayReset).Seconds())
			message = "Global rate limit exceeded (app quota exhausted for today)"
		default:
			retryAfter = 60
			message = "Rate limit exceeded"
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		if state.Reason != nil && *state.Reason != "" {
			w.Header().Set("X-RateLimit-Reason", string(*state.Reason))
		}
		w.WriteHeader(http.StatusTooManyRequests)

		if err := go_json.NewEncoder(w).Encode(map[string]any{
			"error":       "rate_limit_exceeded",
			"message":     message,
			"reason":      state.Reason,
			"retry_after": retryAfter,
		}); err != nil {
			h.logger.ErrorContext(ctx, "failed to encode rate limit response",
				xslog.Error(err),
				slog.String(keyPath, r.URL.Path))
		}
		return
	}

	originalPath := r.URL.Path
	if !strings.HasPrefix(originalPath, "/api/whoop/") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	whoopPath := strings.TrimPrefix(originalPath, "/api/whoop")

	whoopURL, err := url.Parse(whoopAPIURL + whoopPath)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to parse WHOOP URL",
			xslog.Error(err),
			slog.String(keyPath, whoopPath))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	whoopURL.RawQuery = r.URL.RawQuery

	proxyReq, err := http.NewRequestWithContext(ctx, r.Method, whoopURL.String(), r.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to create proxy request",
			xslog.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	for name, values := range r.Header {
		if name == "Connection" || name == "Keep-Alive" || name == "Proxy-Authenticate" ||
			name == "Proxy-Authorization" || name == "Te" || name == "Trailer" ||
			name == "Transfer-Encoding" || name == "Upgrade" || name == "X-Forwarded-For" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to proxy request to WHOOP API",
			xslog.Error(err),
			slog.String(keyURL, whoopURL.String()))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if err := h.whoopLimiter.UpdateFromHeaders(ctx, resp.Header); err != nil {
		h.logger.WarnContext(ctx, "failed to update rate limit from headers",
			xslog.Error(err))
	}

	h.logger.InfoContext(ctx, "proxied request to WHOOP API",
		slog.String(keyMethod, r.Method),
		slog.String(keyPath, whoopPath),
		slog.Int(keyStatus, resp.StatusCode),
		slog.String(keyWhoopUserID, whoopUserID))

	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		h.logger.ErrorContext(ctx, "failed to copy response body", xslog.Error(err))
	}
}
