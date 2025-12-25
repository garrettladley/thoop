package whoop

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RateLimitInfo contains parsed rate limit information from WHOOP API headers.
// See https://developer.whoop.com/docs/developing/rate-limiting/
type RateLimitInfo struct {
	Limit     int           // Requests allowed in current window
	Remaining int           // Requests remaining in current window
	Reset     time.Duration // Duration until the rate limit resets
}

const (
	// Header keys use canonical form (http.CanonicalHeaderKey)
	limitHeaderKey     = "X-Ratelimit-Limit"
	remainingHeaderKey = "X-Ratelimit-Remaining"
	resetHeaderKey     = "X-Ratelimit-Reset"
)

func ParseRateLimitHeaders(headers http.Header) (*RateLimitInfo, error) {
	var (
		limitStr     = headers.Get(limitHeaderKey)
		remainingStr = headers.Get(remainingHeaderKey)
		resetStr     = headers.Get(resetHeaderKey)
	)

	if limitStr == "" || remainingStr == "" || resetStr == "" {
		return nil, nil
	}

	limit, err := parseRateLimitValue(limitStr)
	if err != nil {
		return nil, err
	}

	remaining, err := parseRateLimitValue(remainingStr)
	if err != nil {
		return nil, err
	}

	resetSeconds, err := strconv.ParseInt(resetStr, 10, 64)
	if err != nil {
		return nil, err
	}

	return &RateLimitInfo{
		Limit:     limit,
		Remaining: remaining,
		Reset:     time.Duration(resetSeconds) * time.Second,
	}, nil
}

// parseRateLimitValue extracts the primary integer value from a rate limit header.
// Handles formats like:
//   - "100" (simple)
//   - "100, 100;window=60, 10000;window=86400" (complex)
func parseRateLimitValue(s string) (int, error) {
	parts := strings.Split(s, ",")
	if len(parts) == 0 {
		return 0, strconv.ErrSyntax
	}

	value := strings.TrimSpace(parts[0])
	if idx := strings.Index(value, ";"); idx != -1 {
		value = value[:idx]
	}

	return strconv.Atoi(value)
}
