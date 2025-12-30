package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	go_json "github.com/goccy/go-json"

	"github.com/garrettladley/thoop/internal/xhttp"
)

const (
	defaultBaseURL = "https://api.github.com"
	defaultTimeout = 10 * time.Second
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type Option func(*Client)

func WithHTTPClient(c *http.Client) Option {
	return func(client *Client) { client.httpClient = c }
}

func WithBaseURL(url string) Option {
	return func(client *Client) { client.baseURL = url }
}

func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: xhttp.NewHTTPClient(xhttp.WithTimeout(defaultTimeout)),
		baseURL:    defaultBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func (c *Client) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var release Release
	if err := go_json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}
