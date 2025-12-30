package sse

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
	go_json "github.com/goccy/go-json"
	"golang.org/x/oauth2"
)

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	backoffFactor  = 2
)

type Event struct {
	Type string
	Data []byte
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	transport  *sseTransport
	poller     *PollClient
	logger     *slog.Logger
}

func NewClient(baseURL string, tokenSource oauth2.TokenSource, sessionID string, apiKey string, logger *slog.Logger) *Client {
	transport := &sseTransport{
		base:        xhttp.NewTransport(),
		tokenSource: tokenSource,
		sessionID:   sessionID,
		apiKey:      apiKey,
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Transport: transport},
		transport:  transport,
		poller:     NewPollClient(baseURL, tokenSource, sessionID),
		logger:     logger,
	}
}

type NotificationHandler func(notification storage.Notification)

// Connect establishes an SSE connection and calls the handler for each notification.
// It automatically reconnects with exponential backoff on disconnection.
// Returns when the context is cancelled.
func (c *Client) Connect(ctx context.Context, handler NotificationHandler) error {
	backoff := initialBackoff

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		err := c.connectOnce(ctx, handler)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return fmt.Errorf("context cancelled: %w", ctxErr)
			}

			c.logger.WarnContext(ctx, "SSE connection failed, reconnecting",
				xslog.Error(err),
				xslog.Backoff(backoff),
			)

			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-timer.C:
			}

			// increase backoff for next attempt
			backoff *= backoffFactor
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		} else {
			// connection closed cleanly, reset backoff
			backoff = initialBackoff
		}
	}
}

// connectOnce establishes a single SSE connection and processes events until disconnection.
// On each connect, it first polls for any missed unacked notifications before streaming.
func (c *Client) connectOnce(ctx context.Context, handler NotificationHandler) error {
	if err := c.catchUp(ctx, handler); err != nil {
		return fmt.Errorf("catching up: %w", err)
	}

	url := c.baseURL + "/api/notifications/stream"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	c.logger.DebugContext(ctx, "SSE connection established")

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent Event

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// empty line signals end of event
			if currentEvent.Type != "" && len(currentEvent.Data) > 0 {
				c.handleEvent(ctx, currentEvent, handler)
			}
			currentEvent = Event{}
			continue
		}

		if eventType, found := strings.CutPrefix(line, "event:"); found {
			currentEvent.Type = strings.TrimSpace(eventType)
		} else if data, found := strings.CutPrefix(line, "data:"); found {
			currentEvent.Data = []byte(strings.TrimSpace(data))
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stream: %w", err)
	}

	return nil
}

// catchUp polls for all unacked notifications and delivers them to the handler.
// Uses a buffered channel to avoid loading all notifications into memory at once.
func (c *Client) catchUp(ctx context.Context, handler NotificationHandler) error {
	ch := make(chan storage.Notification, defaultPollLimit)

	var pollErr error
	var total int

	go func() {
		total, pollErr = c.poller.PollAll(ctx, ch)
	}()

	for n := range ch {
		handler(n)
	}

	if pollErr != nil {
		return pollErr
	}

	if total > 0 {
		c.logger.DebugContext(ctx, "caught up on missed notifications",
			xslog.Count(total),
		)
	}

	return nil
}

func (c *Client) handleEvent(ctx context.Context, event Event, handler NotificationHandler) {
	switch event.Type {
	case "notification":
		var notification storage.Notification
		if err := go_json.Unmarshal(event.Data, &notification); err != nil {
			c.logger.WarnContext(ctx, "failed to parse notification",
				xslog.Error(err),
				xslog.Data(string(event.Data)),
			)
			return
		}
		handler(notification)

	case "heartbeat":
		c.logger.DebugContext(ctx, "received heartbeat")

	case "connected":
		c.logger.DebugContext(ctx, "received connected event", xslog.Data(string(event.Data)))

	case "shutdown":
		c.logger.InfoContext(ctx, "server is shutting down, will reconnect")

	default:
		c.logger.DebugContext(ctx, "received unknown event type", xslog.Type(event.Type))
	}
}

type sseTransport struct {
	base        http.RoundTripper
	tokenSource oauth2.TokenSource
	sessionID   string
	apiKey      string
}

var _ http.RoundTripper = (*sseTransport)(nil)

func (t *sseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	if t.sessionID != "" {
		xhttp.SetRequestHeaderSessionID(req, t.sessionID)
	}
	if t.apiKey != "" {
		req.Header.Set(xhttp.XAPIKey, t.apiKey)
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("round trip: %w", err)
	}
	return resp, nil
}

func (c *Client) SetAPIKey(apiKey string) {
	c.transport.apiKey = apiKey
}
