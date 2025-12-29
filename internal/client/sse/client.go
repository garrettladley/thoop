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
	"github.com/garrettladley/thoop/internal/version"
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
	baseURL     string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
	sessionID   string
	logger      *slog.Logger
}

func NewClient(baseURL string, tokenSource oauth2.TokenSource, sessionID string, logger *slog.Logger) *Client {
	return &Client{
		baseURL:     baseURL,
		httpClient:  &http.Client{Timeout: 0}, // no timeout for SSE
		tokenSource: tokenSource,
		sessionID:   sessionID,
		logger:      logger,
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
			return ctx.Err()
		default:
		}

		err := c.connectOnce(ctx, handler)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			c.logger.WarnContext(ctx, "SSE connection failed, reconnecting",
				xslog.Error(err),
				xslog.Backoff(backoff),
			)

			// wait before reconnecting using timer to avoid memory leak
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
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
func (c *Client) connectOnce(ctx context.Context, handler NotificationHandler) error {
	token, err := c.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("getting token: %w", err)
	}

	url := c.baseURL + "/api/notifications/stream"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set(version.Header, version.Get())

	if c.sessionID != "" {
		xhttp.SetRequestHeaderSessionID(req, c.sessionID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	c.logger.InfoContext(ctx, "SSE connection established")

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

	default:
		c.logger.DebugContext(ctx, "received unknown event type", xslog.Type(event.Type))
	}
}
