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

const (
	keyAllowed         = "allowed"
	keyMinuteRemaining = "minute_remaining"
	keyDayRemaining    = "day_remaining"
	keyReason          = "reason"
	keyURL             = "url"
)

type WhoopHandler struct {
	config       Config
	whoopLimiter storage.WhoopRateLimiter
}

func NewWhoopHandler(cfg Config, whoopLimiter storage.WhoopRateLimiter) *WhoopHandler {
	return &WhoopHandler{
		config:       cfg,
		whoopLimiter: whoopLimiter,
	}
}

func (h *WhoopHandler) HandleWhoopProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	whoopUserID, ok := GetWhoopUserID(ctx)
	if !ok || whoopUserID == "" {
		logger.WarnContext(ctx, "missing user key in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	state, err := h.whoopLimiter.CheckAndIncrement(ctx, whoopUserID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check rate limit",
			xslog.ErrorGroup(err),
			xslog.UserGroup(whoopUserID))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	logAttrs := []any{
		xslog.UserGroup(whoopUserID),
		slog.Bool(keyAllowed, state.Allowed),
		slog.Int(keyMinuteRemaining, state.MinuteRemaining),
		slog.Int(keyDayRemaining, state.DayRemaining),
	}
	if state.Reason != nil {
		logAttrs = append(logAttrs, slog.String(keyReason, string(*state.Reason)))
	}
	logger.InfoContext(ctx, "rate limit check", logAttrs...)

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
			w.Header().Set("X-Ratelimit-Reason", string(*state.Reason))
		}
		w.WriteHeader(http.StatusTooManyRequests)

		if err := go_json.NewEncoder(w).Encode(map[string]any{
			"error":       "rate_limit_exceeded",
			"message":     message,
			"reason":      state.Reason,
			"retry_after": retryAfter,
		}); err != nil {
			logger.ErrorContext(ctx, "failed to encode rate limit response",
				xslog.ErrorGroup(err),
				xslog.RequestPath(r))
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
		logger.ErrorContext(ctx, "failed to parse WHOOP URL",
			xslog.ErrorGroup(err),
			xslog.RequestPath(r))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	whoopURL.RawQuery = r.URL.RawQuery

	proxyReq, err := http.NewRequestWithContext(ctx, r.Method, whoopURL.String(), r.Body)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create proxy request",
			xslog.ErrorGroup(err))
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
		logger.ErrorContext(ctx, "failed to proxy request to WHOOP API",
			xslog.ErrorGroup(err),
			slog.String(keyURL, whoopURL.String()))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if err := h.whoopLimiter.UpdateFromHeaders(ctx, resp.Header); err != nil {
		logger.WarnContext(ctx, "failed to update rate limit from headers",
			xslog.ErrorGroup(err))
	}

	logger.InfoContext(ctx, "proxied request to WHOOP API",
		xslog.RequestMethod(r),
		xslog.RequestPath(r),
		xslog.HTTPStatus(resp.StatusCode),
		xslog.UserGroup(whoopUserID))

	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.ErrorContext(ctx, "failed to copy response body", xslog.ErrorGroup(err))
	}
}
