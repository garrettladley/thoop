package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
	go_json "github.com/goccy/go-json"
)

const (
	sseHeartbeatInterval = 30 * time.Second
	sseWriteTimeout      = 45 * time.Second
)

type SSEHandler struct {
	notificationStore storage.NotificationStore
}

func NewSSEHandler(notificationStore storage.NotificationStore) *SSEHandler {
	return &SSEHandler{
		notificationStore: notificationStore,
	}
}

func (h *SSEHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	logger.DebugContext(ctx, "SSE HandleStream called")

	userID, ok := xcontext.GetWhoopUserID(ctx)
	if !ok {
		logger.WarnContext(ctx, "SSE: no user ID in context")
		xhttp.Error(w, http.StatusUnauthorized)
		return
	}

	logger.DebugContext(ctx, "SSE: got user ID", xslog.UserID(userID))

	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.WarnContext(ctx, "SSE: flusher not supported")
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	logger.DebugContext(ctx, "SSE: subscribing to notifications")

	notifCh, unsubscribe, err := h.notificationStore.Subscribe(ctx, userID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to subscribe to notifications",
			xslog.Error(err),
			xslog.UserID(userID),
		)
		http.Error(w, "failed to subscribe", http.StatusInternalServerError)
		return
	}
	defer unsubscribe()

	logger.InfoContext(ctx, "SSE connection established", xslog.UserID(userID))

	rc := http.NewResponseController(w)

	if err := writeSSEEvent(rc, w, flusher, "connected", map[string]any{
		"user_id": userID,
		"time":    time.Now().Format(time.RFC3339),
	}); err != nil {
		logger.ErrorContext(ctx, "failed to send connected event", xslog.Error(err))
		return
	}

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		// check if server is shutting down
		if ctx.Err() != nil && xcontext.IsShutdownInProgress(ctx) {
			logger.InfoContext(ctx, "SSE graceful shutdown initiated", xslog.UserID(userID))

			// best effort: send shutdown event to client
			_ = writeSSEEvent(rc, w, flusher, "shutdown", map[string]string{
				"reason": "server-restart",
				"time":   time.Now().Format(time.RFC3339),
			})

			return
		}

		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "SSE connection closed by client", xslog.UserID(userID))
			return

		case notification, ok := <-notifCh:
			if !ok {
				logger.InfoContext(ctx, "notification channel closed", xslog.UserID(userID))
				return
			}

			if err := writeSSEEvent(rc, w, flusher, "notification", notification); err != nil {
				logger.ErrorContext(ctx, "failed to send notification event", xslog.Error(err))
				return
			}

		case t := <-heartbeat.C:
			if err := writeSSEEvent(rc, w, flusher, "heartbeat", map[string]string{
				"time": t.Format(time.RFC3339),
			}); err != nil {
				logger.ErrorContext(ctx, "failed to send heartbeat", xslog.Error(err))
				return
			}
		}
	}
}

func writeSSEEvent(rc *http.ResponseController, w http.ResponseWriter, flusher http.Flusher, event string, data any) error {
	// extend write deadline before each write (ignore if not supported)
	if err := rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	jsonData, err := go_json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	if _, err := fmt.Fprintf(w, "data: %s\n\n", jsonData); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	flusher.Flush()
	return nil
}
