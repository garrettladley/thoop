package sse

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp"
	go_json "github.com/goccy/go-json"
	"golang.org/x/oauth2"
)

type PollResponse struct {
	Notifications []storage.Notification `json:"notifications"`
	ServerTime    time.Time              `json:"server_time"`
}

type PollClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPollClient(baseURL string, tokenSource oauth2.TokenSource, sessionID string) *PollClient {
	transport := &pollTransport{
		base:        xhttp.NewTransport(),
		tokenSource: tokenSource,
		sessionID:   sessionID,
	}
	return &PollClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}
}

func (c *PollClient) Poll(ctx context.Context, since time.Time) (*PollResponse, error) {
	u, err := url.Parse(c.baseURL + "/api/notifications")
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	q := u.Query()
	if !since.IsZero() {
		q.Set("since", since.Format(time.RFC3339))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result PollResponse
	if err := go_json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

type pollTransport struct {
	base        http.RoundTripper
	tokenSource oauth2.TokenSource
	sessionID   string
}

var _ http.RoundTripper = (*pollTransport)(nil)

func (t *pollTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	if t.sessionID != "" {
		xhttp.SetRequestHeaderSessionID(req, t.sessionID)
	}

	return t.base.RoundTrip(req)
}
