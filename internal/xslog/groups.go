package xslog

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/garrettladley/thoop/internal/xcontext"
)

const (
	groupRequest  = "request"
	groupResponse = "response"
	groupError    = "error"
	groupUser     = "user"
)

const (
	keyID         = "id"
	keyHost       = "host"
	keyUserAgent  = "user_agent"
	keyProto      = "proto"
	keyQuery      = "query"
	keyStatusText = "status_text"
	keyDurationMS = "duration_ms"
	keyMessage    = "message"
	keyType       = "type"
	keyValue      = "value"
)

// RequestGroup returns all request metadata as a group.
func RequestGroup(r *http.Request) slog.Attr {
	attrs := []any{
		RequestMethod(r),
		RequestPath(r),
		RequestIP(r),
		slog.String(keyHost, r.Host),
		slog.String(keyUserAgent, r.UserAgent()),
		slog.String(keyProto, r.Proto),
	}
	if id, ok := xcontext.GetRequestID(r.Context()); ok {
		attrs = append(attrs, slog.String(keyID, id))
	}
	if r.URL.RawQuery != "" {
		attrs = append(attrs, slog.String(keyQuery, r.URL.RawQuery))
	}
	return slog.Group(groupRequest, attrs...)
}

// RequestGroupMinimal returns essential request fields only.
func RequestGroupMinimal(r *http.Request) slog.Attr {
	attrs := []any{
		RequestMethod(r),
		RequestPath(r),
		RequestIP(r),
	}
	if id, ok := xcontext.GetRequestID(r.Context()); ok {
		attrs = append(attrs, slog.String(keyID, id))
	}
	return slog.Group(groupRequest, attrs...)
}

// ResponseGroup returns response metadata as a group.
func ResponseGroup(status int, duration time.Duration) slog.Attr {
	return slog.Group(groupResponse,
		HTTPStatus(status),
		slog.String(keyStatusText, http.StatusText(status)),
		Duration(duration),
		slog.Int64(keyDurationMS, duration.Milliseconds()),
	)
}

// ErrorGroup returns error info as a group.
func ErrorGroup(err error) slog.Attr {
	if err == nil {
		return slog.Group(groupError)
	}
	return slog.Group(groupError,
		slog.String(keyMessage, err.Error()),
		slog.String(keyType, fmt.Sprintf("%T", err)),
	)
}

// ErrorGroupWithStack returns error + stack trace (for panics).
func ErrorGroupWithStack(err any) slog.Attr {
	return slog.Group(groupError,
		slog.Any(keyValue, err),
		slog.String(keyType, fmt.Sprintf("%T", err)),
		Stack(),
	)
}

// UserGroup returns user context as a group.
func UserGroup(userID string) slog.Attr {
	return slog.Group(groupUser,
		slog.String(keyID, userID),
	)
}
