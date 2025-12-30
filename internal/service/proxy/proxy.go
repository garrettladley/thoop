package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
)

const whoopAPIURL = "https://api.prod.whoop.com/developer"

type RateLimitConfig struct {
	PerUserMinuteLimit int
	PerUserDayLimit    int
	GlobalMinuteLimit  int
	GlobalDayLimit     int
}

type Proxy struct {
	whoopLimiter storage.WhoopRateLimiter
	rateLimitCfg RateLimitConfig
	httpClient   *http.Client
}

var _ Service = (*Proxy)(nil)

func NewProxy(whoopLimiter storage.WhoopRateLimiter, rateLimitCfg RateLimitConfig) *Proxy {
	return &Proxy{
		whoopLimiter: whoopLimiter,
		rateLimitCfg: rateLimitCfg,
		httpClient:   xhttp.NewHTTPClient(xhttp.WithTimeout(30 * time.Second)),
	}
}

func (p *Proxy) CheckRateLimit(ctx context.Context, userID int64) (*RateLimitInfo, error) {
	logger := xslog.FromContext(ctx)
	userKey := strconv.FormatInt(userID, 10)

	state, err := p.whoopLimiter.CheckAndIncrement(ctx, userKey)
	if err != nil {
		return nil, fmt.Errorf("checking rate limit: %w", err)
	}

	if state.Allowed {
		return nil, nil
	}

	var (
		retryAfter time.Duration
		message    string
		reason     string
	)

	if state.Reason != nil {
		reason = string(*state.Reason)

		switch *state.Reason {
		case storage.WhoopRateLimitReasonPerUserMinute:
			retryAfter = time.Until(state.MinuteReset)
			message = fmt.Sprintf("Per-user rate limit exceeded (%d requests/minute)", p.rateLimitCfg.PerUserMinuteLimit)
		case storage.WhoopRateLimitReasonPerUserDay:
			retryAfter = time.Until(state.DayReset)
			message = fmt.Sprintf("Per-user rate limit exceeded (%d requests/day)", p.rateLimitCfg.PerUserDayLimit)
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
	} else {
		retryAfter = time.Minute
		message = "Rate limit exceeded"
	}

	logger.WarnContext(ctx, "rate limit exceeded", xslog.UserID(userID))

	return &RateLimitInfo{
		RetryAfter: retryAfter,
		Reason:     reason,
		Message:    message,
	}, ErrRateLimited
}

func (p *Proxy) ProxyRequest(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	logger := xslog.FromContext(ctx)

	if !strings.HasPrefix(req.Path, "/api/whoop/") {
		return nil, ErrInvalidPath
	}
	whoopPath := strings.TrimPrefix(req.Path, "/api/whoop")

	whoopURL, err := url.Parse(whoopAPIURL + whoopPath)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse URL: %v", ErrUpstreamError, err)
	}
	whoopURL.RawQuery = req.Query

	proxyReq, err := http.NewRequestWithContext(ctx, req.Method, whoopURL.String(), req.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrUpstreamError, err)
	}

	// copy headers, excluding hop-by-hop and internal headers
	for name, values := range req.Headers {
		if isHopByHopHeader(name) || name == xhttp.XAPIKey {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed: %v", ErrUpstreamError, err)
	}

	if updateErr := p.whoopLimiter.UpdateFromHeaders(ctx, resp.Header); updateErr != nil {
		logger.WarnContext(ctx, "failed to update rate limit from headers", xslog.Error(updateErr))
	}

	return &ProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       resp.Body,
	}, nil
}

func isHopByHopHeader(name string) bool {
	switch name {
	case "Connection", "Keep-Alive", "Proxy-Authenticate",
		"Proxy-Authorization", "Te", "Trailer",
		"Transfer-Encoding", "Upgrade", "X-Forwarded-For":
		return true
	}
	return false
}

func CopyResponse(w http.ResponseWriter, resp *ProxyResponse) error {
	defer func() { _ = resp.Body.Close() }()

	for name, values := range resp.Headers {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, err := io.Copy(w, resp.Body)
	return err
}
