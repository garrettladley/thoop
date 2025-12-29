package server

import (
	"net/http"
	"time"

	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
	go_json "github.com/goccy/go-json"
)

type NotificationsHandler struct {
	notificationStore storage.NotificationStore
}

func NewNotificationsHandler(notificationStore storage.NotificationStore) *NotificationsHandler {
	return &NotificationsHandler{
		notificationStore: notificationStore,
	}
}

type NotificationsResponse struct {
	Notifications []storage.Notification `json:"notifications"`
	ServerTime    time.Time              `json:"server_time"`
}

func (h *NotificationsHandler) HandlePoll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	userID, ok := xcontext.GetWhoopUserID(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			logger.WarnContext(ctx, "invalid since parameter",
				xslog.Since(sinceStr),
				xslog.Error(err),
			)
			http.Error(w, "invalid since parameter, expected RFC3339 format", http.StatusBadRequest)
			return
		}
	}

	notifications, err := h.notificationStore.GetSince(ctx, userID, since)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch notifications",
			xslog.Error(err),
			xslog.UserID(userID),
		)
		http.Error(w, "failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	response := NotificationsResponse{
		Notifications: notifications,
		ServerTime:    time.Now(),
	}

	if response.Notifications == nil {
		response.Notifications = []storage.Notification{}
	}

	xhttp.SetHeaderContentTypeApplicationJSON(w)
	if err := go_json.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(ctx, "failed to encode response", xslog.Error(err))
		return
	}

	logger.DebugContext(ctx, "fetched notifications",
		xslog.UserID(userID),
		xslog.SinceTime(since),
		xslog.Count(len(notifications)),
	)
}
