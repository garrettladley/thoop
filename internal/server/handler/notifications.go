package handler

import (
	"net/http"
	"time"

	"github.com/garrettladley/thoop/internal/apperr"
	"github.com/garrettladley/thoop/internal/service/notification"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xslog"
)

type Notifications struct {
	service notification.Service
}

func NewNotifications(service notification.Service) *Notifications {
	return &Notifications{service: service}
}

// HandlePoll handles GET /api/notifications requests.
func (h *Notifications) HandlePoll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	userID, ok := xcontext.GetWhoopUserID(ctx)
	if !ok {
		apperr.WriteError(w, apperr.Unauthorized("unauthorized", "missing user context"))
		return
	}

	var since time.Time
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			apperr.WriteError(w, apperr.BadRequest("invalid_request", "invalid since parameter, expected RFC3339 format"))
			return
		}
	}

	result, err := h.service.GetNotifications(ctx, userID, since)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch notifications",
			xslog.Error(err),
			xslog.UserID(userID),
		)
		apperr.WriteError(w, apperr.Internal("internal_error", "failed to fetch notifications", err))
		return
	}

	logger.DebugContext(ctx, "fetched notifications",
		xslog.UserID(userID),
		xslog.SinceTime(since),
		xslog.Count(len(result.Notifications)),
	)

	apperr.WriteOK(w, result)
}
