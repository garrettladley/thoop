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

func RequestGroup(r *http.Request) slog.Attr {
	attrs := []slog.Attr{
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
	return slog.GroupAttrs(groupRequest, attrs...)
}

func ResponseGroup(status int, duration time.Duration) slog.Attr {
	return slog.Group(groupResponse,
		HTTPStatus(status),
		slog.String(keyStatusText, http.StatusText(status)),
		Duration(duration),
		slog.Int64(keyDurationMS, duration.Milliseconds()),
	)
}

func ErrorGroup(err error) slog.Attr {
	if err == nil {
		return slog.Group(groupError)
	}
	return slog.Group(groupError,
		slog.String(keyMessage, err.Error()),
		slog.String(keyType, fmt.Sprintf("%T", err)),
	)
}

func ErrorGroupWithStack(err any) slog.Attr {
	return slog.Group(groupError,
		slog.Any(keyValue, err),
		slog.String(keyType, fmt.Sprintf("%T", err)),
		Stack(),
	)
}

func UserGroup(userID int64) slog.Attr {
	return slog.Group(groupUser,
		slog.Int64(keyID, userID),
	)
}
