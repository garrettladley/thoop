package xhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryTransport_RespectsRetryAfter(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.Header().Set("Retry-After", "0") // immediate retry for test speed
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRetryTransport_ClampsRetryAfter(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	var waitStart atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count == 1 {
			waitStart.Store(time.Now().UnixNano())
			w.Header().Set("Retry-After", "3600") // 1 hour, should be clamped
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// Verify the wait was clamped
		elapsed := time.Since(time.Unix(0, waitStart.Load()))
		if elapsed > 200*time.Millisecond {
			t.Errorf("wait was too long: %v (expected clamped to ~100ms)", elapsed)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxAttempts:   3,
		MaxRetryAfter: 100 * time.Millisecond, // clamp to 100ms
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}
}

func TestRetryTransport_ExponentialBackoff(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	var ts1, ts2, ts3 atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		now := time.Now().UnixNano()
		switch count {
		case 1:
			ts1.Store(now)
		case 2:
			ts2.Store(now)
		case 3:
			ts3.Store(now)
		}
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
		BackoffFactor:  2.0,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}

	// First backoff should be ~50ms
	firstBackoff := time.Duration(ts2.Load() - ts1.Load())
	if firstBackoff < 40*time.Millisecond || firstBackoff > 100*time.Millisecond {
		t.Errorf("first backoff %v not in expected range [40ms, 100ms]", firstBackoff)
	}

	// Second backoff should be ~100ms (50ms * 2.0)
	secondBackoff := time.Duration(ts3.Load() - ts2.Load())
	if secondBackoff < 80*time.Millisecond || secondBackoff > 200*time.Millisecond {
		t.Errorf("second backoff %v not in expected range [80ms, 200ms]", secondBackoff)
	}
}

func TestRetryTransport_MaxAttemptsExhausted(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRetryTransport_ContextCancellation(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("Retry-After", "10") // 10 seconds
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
		MaxAttempts:    5,
		MaxRetryAfter:  10 * time.Second,
		InitialBackoff: 10 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := transport.RoundTrip(req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt before cancellation, got %d", attempts.Load())
	}
}

func TestRetryTransport_NonRetryableStatus(t *testing.T) {
	t.Parallel()

	testCases := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
	}

	for _, statusCode := range testCases {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()

			var attempts atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				w.WriteHeader(statusCode)
			}))
			defer srv.Close()

			transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
				MaxAttempts:    3,
				InitialBackoff: 10 * time.Millisecond,
			})

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != statusCode {
				t.Errorf("expected status %d, got %d", statusCode, resp.StatusCode)
			}
			if attempts.Load() != 1 {
				t.Errorf("expected 1 attempt (no retry), got %d", attempts.Load())
			}
		})
	}
}

func TestRetryTransport_NetworkError(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	// Create a transport that fails the first 2 times with a network error
	failingTransport := &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			count := attempts.Add(1)
			if count < 3 {
				return nil, errors.New("connection refused")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	transport := NewRetryTransport(failingTransport, RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRetryTransport_RetryableStatusCodes(t *testing.T) {
	t.Parallel()

	retryableCodes := []int{
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout,      // 504
	}

	for _, statusCode := range retryableCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()

			var attempts atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count := attempts.Add(1)
				if count < 2 {
					w.WriteHeader(statusCode)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			transport := NewRetryTransport(http.DefaultTransport, RetryConfig{
				MaxAttempts:    3,
				InitialBackoff: 10 * time.Millisecond,
			})

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			if attempts.Load() != 2 {
				t.Errorf("expected 2 attempts, got %d", attempts.Load())
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		header        string
		maxRetryAfter time.Duration
		want          time.Duration
	}{
		{
			name:   "empty header",
			header: "",
			want:   0,
		},
		{
			name:   "seconds",
			header: "5",
			want:   5 * time.Second,
		},
		{
			name:          "seconds clamped",
			header:        "3600",
			maxRetryAfter: 30 * time.Second,
			want:          30 * time.Second,
		},
		{
			name:   "invalid",
			header: "invalid",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &http.Response{
				Header: http.Header{},
			}
			if tt.header != "" {
				resp.Header.Set("Retry-After", tt.header)
			}

			got := ParseRetryAfter(resp, tt.maxRetryAfter)
			if got != tt.want {
				t.Errorf("ParseRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockTransport struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}
