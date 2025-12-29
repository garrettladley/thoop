package proxy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
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

	whoopUserID, ok := xcontext.GetWhoopUserID(ctx)
	if !ok || whoopUserID == 0 {
		logger.WarnContext(ctx, "missing user key in context")
		xhttp.Error(w, http.StatusUnauthorized)
		return
	}

	userKey := strconv.FormatInt(whoopUserID, 10)

	state, err := h.whoopLimiter.CheckAndIncrement(ctx, userKey)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check rate limit",
			xslog.ErrorGroup(err),
			xslog.UserGroup(whoopUserID))
		xhttp.Error(w, http.StatusInternalServerError)
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
			retryAfter time.Duration
			message    string
			reason     string
		)

		if state.Reason != nil {
			reason = string(*state.Reason)
		}

		switch *state.Reason {
		case storage.WhoopRateLimitReasonPerUserMinute:
			retryAfter = time.Until(state.MinuteReset)
			message = fmt.Sprintf("Per-user rate limit exceeded (%d requests/minute)", h.config.WhoopRateLimit.PerUserMinuteLimit)
		case storage.WhoopRateLimitReasonPerUserDay:
			retryAfter = time.Until(state.DayReset)
			message = fmt.Sprintf("Per-user rate limit exceeded (%d requests/day)", h.config.WhoopRateLimit.PerUserDayLimit)
		case storage.WhoopRateLimitReasonGlobalMinute:
			retryAfter = time.Until(state.MinuteReset)
			message = "Global rate limit exceeded (app quota exhausted for this minute)"
		case storage.WhoopRateLimitReasonGlobalDay:
			retryAfter = time.Until(state.DayReset)
			message = "Global rate limit exceeded (app quota exhausted for today)"
		default:
			retryAfter = time.Minute
			message = "Rate limit exceeded"
		}

		xhttp.WriteRateLimitError(w, &xhttp.RateLimitError{
			RetryAfter: retryAfter,
			Reason:     reason,
			Message:    message,
		})
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
		xhttp.Error(w, http.StatusInternalServerError)
		return
	}
	whoopURL.RawQuery = r.URL.RawQuery

	proxyReq, err := http.NewRequestWithContext(ctx, r.Method, whoopURL.String(), r.Body)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create proxy request",
			xslog.ErrorGroup(err))
		xhttp.Error(w, http.StatusInternalServerError)
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
		xhttp.Error(w, http.StatusBadGateway)
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
