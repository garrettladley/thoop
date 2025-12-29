package proxy

import (
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

	userID, ok := xcontext.GetWhoopUserID(ctx)
	if !ok {
		xhttp.Error(w, http.StatusUnauthorized)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

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

	if err := writeSSEEvent(w, flusher, "connected", map[string]string{
		"user_id": userID,
		"time":    time.Now().Format(time.RFC3339),
	}); err != nil {
		logger.ErrorContext(ctx, "failed to send connected event", xslog.Error(err))
		return
	}

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "SSE connection closed", xslog.UserID(userID))
			return

		case notification, ok := <-notifCh:
			if !ok {
				logger.InfoContext(ctx, "notification channel closed", xslog.UserID(userID))
				return
			}

			if err := writeSSEEvent(w, flusher, "notification", notification); err != nil {
				logger.ErrorContext(ctx, "failed to send notification event", xslog.Error(err))
				return
			}

		case t := <-heartbeat.C:
			if err := writeSSEEvent(w, flusher, "heartbeat", map[string]string{
				"time": t.Format(time.RFC3339),
			}); err != nil {
				logger.ErrorContext(ctx, "failed to send heartbeat", xslog.Error(err))
				return
			}
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data any) error {
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
