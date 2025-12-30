package sse

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp"
	go_json "github.com/goccy/go-json"
	"golang.org/x/oauth2"
)

type PollResponse struct {
	Notifications []storage.Notification `json:"notifications"`
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

const defaultPollLimit = 100

// Poll fetches unacknowledged notifications using cursor-based pagination.
// Pass cursor=0 for the first page. Returns notifications with id > cursor.
func (c *PollClient) Poll(ctx context.Context, cursor int64, limit int32) (*PollResponse, error) {
	u, err := url.Parse(c.baseURL + "/api/notifications")
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	q := u.Query()
	if cursor > 0 {
		q.Set("cursor", strconv.FormatInt(cursor, 10))
	}
	if limit > 0 {
		q.Set("limit", strconv.FormatInt(int64(limit), 10))
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

func (c *PollClient) Ack(ctx context.Context, traceIDs []string) error {
	if len(traceIDs) == 0 {
		return nil
	}

	body, err := go_json.Marshal(map[string][]string{"trace_ids": traceIDs})
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/notifications/ack", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// PollAll fetches all unacknowledged notifications by paginating until exhausted.
// Sends notifications to the provided channel as they are fetched.
// Returns total count fetched, or error. Closes channel when done or on error.
func (c *PollClient) PollAll(ctx context.Context, ch chan<- storage.Notification) (int, error) {
	defer close(ch)

	var (
		cursor int64
		total  int
	)
	for {
		resp, err := c.Poll(ctx, cursor, defaultPollLimit)
		if err != nil {
			return total, err
		}

		for _, n := range resp.Notifications {
			select {
			case ch <- n:
				total++
			case <-ctx.Done():
				return total, fmt.Errorf("context cancelled: %w", ctx.Err())
			}
		}

		if len(resp.Notifications) < defaultPollLimit {
			break
		}

		cursor = resp.Notifications[len(resp.Notifications)-1].ID
	}

	return total, nil
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

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("round trip: %w", err)
	}
	return resp, nil
}
