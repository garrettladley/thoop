package whoop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/garrettladley/thoop/internal/version"
	go_json "github.com/goccy/go-json"
	"golang.org/x/oauth2"
)

type Client struct {
	User     UserService
	Cycle    CycleService
	Recovery RecoveryService
	Sleep    SleepService
	Workout  WorkoutService

	baseURL     string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
}

func New(tokenSource oauth2.TokenSource, opts ...Option) *Client {
	const baseURL = "https://api.prod.whoop.com/developer"
	c := &Client{
		baseURL:     baseURL,
		httpClient:  http.DefaultClient,
		tokenSource: tokenSource,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.User = &userService{client: c}
	c.Cycle = &cycleService{client: c}
	c.Recovery = &recoveryService{client: c}
	c.Sleep = &sleepService{client: c}
	c.Workout = &workoutService{client: c}

	return c
}

type Option func(*Client)

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

func (c *Client) do(ctx context.Context, method string, path string, query url.Values, result any) error {
	token, err := c.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("getting token: %w", err)
	}

	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set(version.Header, version.Get())

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
