package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

var (
	ErrRateLimited   = errors.New("rate limit exceeded")
	ErrInvalidPath   = errors.New("invalid proxy path")
	ErrUpstreamError = errors.New("upstream error")
)

type RateLimitInfo struct {
	RetryAfter time.Duration
	Reason     string
	Message    string
}

type ProxyRequest struct {
	Method  string
	Path    string
	Query   string
	Headers http.Header
	Body    io.ReadCloser
	UserID  int64
}

type ProxyResponse struct {
	StatusCode int
	Headers    http.Header
	Body       io.ReadCloser
}

type Service interface {
	// CheckRateLimit checks if the user can make a request.
	// Returns nil if allowed.
	// Returns ErrRateLimited with RateLimitInfo if rate limited.
	CheckRateLimit(ctx context.Context, userID int64) (*RateLimitInfo, error)

	// ProxyRequest forwards a request to the WHOOP API.
	// Returns the upstream response.
	// Returns ErrInvalidPath if the path is malformed.
	// Returns ErrUpstreamError if the upstream request fails.
	ProxyRequest(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error)
}
