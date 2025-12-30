package whoop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/garrettladley/thoop/internal/xhttp"
	go_json "github.com/goccy/go-json"
	"golang.org/x/oauth2"
)

type Client struct {
	User     UserService
	Cycle    CycleService
	Recovery RecoveryService
	Sleep    SleepService
	Workout  WorkoutService

	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

func New(tokenSource oauth2.TokenSource, opts ...Option) *Client {
	const baseURL = "https://api.prod.whoop.com/developer"

	cfg := &clientConfig{
		baseURL:     baseURL,
		tokenSource: tokenSource,
		logger:      slog.Default(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	transport := &whoopTransport{
		base:         xhttp.NewTransport(),
		tokenSource:  cfg.tokenSource,
		sessionID:    cfg.sessionID,
		apiKey:       cfg.apiKey,
		isUsingProxy: cfg.isUsingProxy,
	}

	c := &Client{
		baseURL:    cfg.baseURL,
		httpClient: &http.Client{Transport: transport, Timeout: cfg.timeout},
		logger:     cfg.logger,
	}

	c.User = &userService{client: c}
	c.Cycle = &cycleService{client: c}
	c.Recovery = &recoveryService{client: c}
	c.Sleep = &sleepService{client: c}
	c.Workout = &workoutService{client: c}

	return c
}

type clientConfig struct {
	baseURL      string
	isUsingProxy bool
	tokenSource  oauth2.TokenSource
	logger       *slog.Logger
	sessionID    string
	apiKey       string
	timeout      time.Duration
}

type Option func(*clientConfig)

func WithProxyURL(baseURL string) Option {
	return func(cfg *clientConfig) {
		cfg.baseURL = baseURL
		cfg.isUsingProxy = true
	}
}

func WithSessionID(sessionID string) Option {
	return func(cfg *clientConfig) { cfg.sessionID = sessionID }
}

func WithLogger(logger *slog.Logger) Option {
	return func(cfg *clientConfig) { cfg.logger = logger }
}

func WithAPIKey(apiKey string) Option {
	return func(cfg *clientConfig) { cfg.apiKey = apiKey }
}

func WithTimeout(d time.Duration) Option {
	return func(cfg *clientConfig) { cfg.timeout = d }
}

func (c *Client) do(ctx context.Context, method string, path string, query url.Values, result any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}
		if err := go_json.NewDecoder(bytes.NewReader(body)).Decode(result); err != nil {
			return fmt.Errorf("decoding response: %w\nbody: %s", err, string(body))
		}
	}

	return nil
}

type whoopTransport struct {
	base         http.RoundTripper
	tokenSource  oauth2.TokenSource
	sessionID    string
	apiKey       string
	isUsingProxy bool
}

var _ http.RoundTripper = (*whoopTransport)(nil)

func (t *whoopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	if t.isUsingProxy {
		if t.sessionID != "" {
			xhttp.SetRequestHeaderSessionID(req, t.sessionID)
		}
		if t.apiKey != "" {
			req.Header.Set(xhttp.XAPIKey, t.apiKey)
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("round trip: %w", err)
	}
	return resp, nil
}
