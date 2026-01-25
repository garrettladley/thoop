package xhttp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// RetryConfig configures the retry transport behavior.
type RetryConfig struct {
	MaxAttempts    uint          // Maximum number of retry attempts (default: 3)
	MaxRetryAfter  time.Duration // Maximum Retry-After to respect (default: 30s)
	InitialBackoff time.Duration // Initial backoff duration (default: 1s)
	MaxBackoff     time.Duration // Maximum backoff duration (default: 10s)
	BackoffFactor  float64       // Backoff multiplier (default: 2.0)
}

// DefaultRetryConfig returns sensible defaults for retry behavior.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:    3,
		MaxRetryAfter:  30 * time.Second,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
	}
}

// retryTransport wraps an http.RoundTripper with retry logic.
type retryTransport struct {
	base   http.RoundTripper
	config RetryConfig
}

var _ http.RoundTripper = (*retryTransport)(nil)

// NewRetryTransport creates a new retry transport wrapping the given transport.
func NewRetryTransport(base http.RoundTripper, config RetryConfig) http.RoundTripper {
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}
	if config.MaxRetryAfter == 0 {
		config.MaxRetryAfter = 30 * time.Second
	}
	if config.InitialBackoff == 0 {
		config.InitialBackoff = 1 * time.Second
	}
	if config.MaxBackoff == 0 {
		config.MaxBackoff = 10 * time.Second
	}
	if config.BackoffFactor == 0 {
		config.BackoffFactor = 2.0
	}

	return &retryTransport{
		base:   base,
		config: config,
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp    *http.Response
		err     error
		backoff = t.config.InitialBackoff
	)

	for attempt := uint(0); attempt < t.config.MaxAttempts; attempt++ {
		resp, err = t.base.RoundTrip(req)
		// check for network errors (retryable)
		if err != nil {
			if attempt+1 >= t.config.MaxAttempts {
				return nil, fmt.Errorf("round trip: %w", err)
			}

			if waitErr := t.wait(req.Context(), backoff); waitErr != nil {
				return nil, fmt.Errorf("waiting for retry: %w", waitErr)
			}

			backoff = t.nextBackoff(backoff)
			continue
		}

		// check for retryable status codes
		if !t.isRetryable(resp.StatusCode) {
			return resp, nil
		}

		// this is the last attempt, return the response as-is
		if attempt+1 >= t.config.MaxAttempts {
			return resp, nil
		}

		waitDuration := ParseRetryAfter(resp, t.config.MaxRetryAfter)
		if waitDuration == 0 {
			waitDuration = backoff
		}

		// drain and close body to reuse connection
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if waitErr := t.wait(req.Context(), waitDuration); waitErr != nil {
			return nil, fmt.Errorf("waiting for retry: %w", waitErr)
		}

		backoff = t.nextBackoff(backoff)
	}

	return resp, nil
}

// isRetryable returns true for status codes that should trigger a retry.
func (t *retryTransport) isRetryable(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return true
	default:
		return statusCode >= 500 && statusCode < 600
	}
}

// nextBackoff calculates the next backoff duration with exponential growth.
func (t *retryTransport) nextBackoff(current time.Duration) time.Duration {
	next := time.Duration(float64(current) * t.config.BackoffFactor)
	if next > t.config.MaxBackoff {
		return t.config.MaxBackoff
	}
	return next
}

// wait blocks for the specified duration, respecting context cancellation.
func (t *retryTransport) wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

// ParseRetryAfter parses the Retry-After header from a response.
// Returns 0 if the header is missing or invalid.
// This is exported for use by SSE clients that handle their own retry logic.
func ParseRetryAfter(resp *http.Response, maxRetryAfter time.Duration) time.Duration {
	header := resp.Header.Get("Retry-After")
	if header == "" {
		return 0
	}

	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		duration := time.Duration(seconds) * time.Second
		if maxRetryAfter > 0 && duration > maxRetryAfter {
			return maxRetryAfter
		}
		return duration
	}

	if date, err := http.ParseTime(header); err == nil {
		duration := time.Until(date)
		if duration <= 0 {
			return 0
		}
		if maxRetryAfter > 0 && duration > maxRetryAfter {
			return maxRetryAfter
		}
		return duration
	}

	return 0
}
