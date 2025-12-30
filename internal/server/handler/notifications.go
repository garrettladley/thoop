package handler

import (
	"net/http"
	"strconv"

	go_json "github.com/goccy/go-json"

	"github.com/garrettladley/thoop/internal/apperr"
	"github.com/garrettladley/thoop/internal/service/notification"
	"github.com/garrettladley/thoop/internal/xcontext"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
)

type Notifications struct {
	service notification.Service
}

func NewNotifications(service notification.Service) *Notifications {
	return &Notifications{service: service}
}

// HandlePoll handles GET /api/notifications requests.
// Query params: cursor (int64 id, default 0), limit (int32, default 100)
func (h *Notifications) HandlePoll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	userID, ok := xcontext.GetWhoopUserID(ctx)
	if !ok {
		apperr.WriteError(w, apperr.Unauthorized("unauthorized", "missing user context"))
		return
	}

	var cursor int64
	if cursorStr := r.URL.Query().Get("cursor"); cursorStr != "" {
		var err error
		cursor, err = strconv.ParseInt(cursorStr, 10, 64)
		if err != nil || cursor < 0 {
			apperr.WriteError(w, apperr.BadRequest("invalid_request", "invalid cursor parameter (expected non-negative integer)"))
			return
		}
	}

	const defaultLimit = 100
	limit := int32(defaultLimit)
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		l, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil || l <= 0 || l > 1000 {
			apperr.WriteError(w, apperr.BadRequest("invalid_request", "invalid limit parameter (must be 1-1000)"))
			return
		}
		limit = int32(l)
	}

	result, err := h.service.GetUnacked(ctx, userID, cursor, limit)
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
		xslog.Count(len(result.Notifications)),
	)

	xhttp.WriteOK(w, result)
}

type acknowledgeRequest struct {
	TraceIDs []string `json:"trace_ids"`
}

// HandleAcknowledge handles POST /api/notifications/ack requests.
func (h *Notifications) HandleAcknowledge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := xslog.FromContext(ctx)

	userID, ok := xcontext.GetWhoopUserID(ctx)
	if !ok {
		apperr.WriteError(w, apperr.Unauthorized("unauthorized", "missing user context"))
		return
	}

	var req acknowledgeRequest
	if err := go_json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperr.WriteError(w, apperr.BadRequest("invalid_request", "invalid JSON body"))
		return
	}

	if len(req.TraceIDs) == 0 {
		apperr.WriteError(w, apperr.BadRequest("invalid_request", "trace_ids array cannot be empty"))
		return
	}

	if err := h.service.Acknowledge(ctx, userID, req.TraceIDs); err != nil {
		logger.ErrorContext(ctx, "failed to acknowledge notifications",
			xslog.Error(err),
			xslog.UserID(userID),
		)
		apperr.WriteError(w, apperr.Internal("internal_error", "failed to acknowledge notifications", err))
		return
	}

	logger.DebugContext(ctx, "acknowledged notifications",
		xslog.UserID(userID),
		xslog.Count(len(req.TraceIDs)),
	)

	xhttp.WriteOK(w, map[string]string{"status": "ok"})
}
