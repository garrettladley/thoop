package xerrors

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xslog"
	go_json "github.com/goccy/go-json"
)

type errorResponse struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func WriteError(ctx context.Context, w http.ResponseWriter, err error) {
	appErr := As(err)
	if appErr == nil {
		appErr = Internal(WithCause(err))
	}

	logError(ctx, appErr)

	xhttp.SetHeaderContentTypeApplicationJSON(w)

	if appErr.RateLimit != nil {
		if appErr.RateLimit.RetryAfter > 0 {
			xhttp.SetHeaderRetryAfter(w, appErr.RateLimit.RetryAfter)
		}
		if appErr.RateLimit.Reason != "" {
			w.Header().Set(xhttp.XRateLimitReason, appErr.RateLimit.Reason)
		}
	}

	w.WriteHeader(appErr.StatusCode)

	resp := errorResponse{Message: appErr.Message}
	if appErr.Validation != nil {
		resp.Fields = appErr.Validation.Fields
	}

	_ = go_json.NewEncoder(w).Encode(resp)
}

func logError(ctx context.Context, err *Error) {
	logger := xslog.FromContext(ctx)
	attrs := []any{
		xslog.HTTPStatus(err.StatusCode),
		slog.String("message", err.Message),
	}
	if err.Cause != nil {
		attrs = append(attrs, xslog.Error(err.Cause))
	}
	if err.RateLimit != nil {
		attrs = append(attrs, slog.Any("rate_limit", err.RateLimit))
	}
	if err.Validation != nil {
		attrs = append(attrs, slog.Any("validation", err.Validation))
	}

	switch err.StatusCode / 100 {
	case 5:
		logger.ErrorContext(ctx, "server error", attrs...)
	case 4:
		logger.WarnContext(ctx, "client error", attrs...)
	default:
		logger.InfoContext(ctx, "error response", attrs...)
	}
}
