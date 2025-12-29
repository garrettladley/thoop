package xslog

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/garrettladley/thoop/internal/version"
	"github.com/garrettladley/thoop/internal/xhttp"
)

const (
	keyError = "error"
)

func Error(err error) slog.Attr {
	return slog.String(keyError, err.Error())
}

func ErrorAny(err any) slog.Attr {
	return slog.Any(keyError, err)
}

func RequestID(requestID string) slog.Attr {
	const requestIDKey = "request_id"
	return slog.String(requestIDKey, requestID)
}

func Stack() slog.Attr {
	const stackKey = "stack"
	return slog.String(stackKey, string(debug.Stack()))
}

func HTTPStatus(status int) slog.Attr {
	const statusKey = "status"
	return slog.Int(statusKey, status)
}

func Duration(duration time.Duration) slog.Attr {
	const durationKey = "duration"
	return slog.Duration(durationKey, duration)
}

func RequestMethod(r *http.Request) slog.Attr {
	const methodKey = "method"
	return slog.String(methodKey, r.Method)
}

func RequestPath(r *http.Request) slog.Attr {
	const pathKey = "path"
	return slog.String(pathKey, r.URL.Path)
}

func IP(ip string) slog.Attr {
	const ipKey = "ip"
	return slog.String(ipKey, ip)
}

func RequestIP(r *http.Request) slog.Attr {
	return IP(xhttp.GetRequestIP(r))
}

func Version() slog.Attr {
	const versionKey = "version"
	return slog.String(versionKey, version.Get())
}

func ClientVersion(clientVersion string) slog.Attr {
	const clientVersionKey = "client_version"
	return slog.String(clientVersionKey, clientVersion)
}

func ProxyVersion(proxyVersion string) slog.Attr {
	const proxyVersionKey = "proxy_version"
	return slog.String(proxyVersionKey, proxyVersion)
}

func MinVersion(minVersion string) slog.Attr {
	const minVersionKey = "min_version"
	return slog.String(minVersionKey, minVersion)
}

func CycleID(id int64) slog.Attr {
	const cycleIDKey = "cycle_id"
	return slog.Int64(cycleIDKey, id)
}

func Count(count int) slog.Attr {
	const countKey = "count"
	return slog.Int(countKey, count)
}

func Start(t time.Time) slog.Attr {
	const startKey = "start"
	return slog.Time(startKey, t)
}

func End(t time.Time) slog.Attr {
	const endKey = "end"
	return slog.Time(endKey, t)
}

func SessionID(id string) slog.Attr {
	const sessionIDKey = "session_id"
	return slog.String(sessionIDKey, id)
}

func UserID(id string) slog.Attr {
	const userIDKey = "user_id"
	return slog.String(userIDKey, id)
}

func EntityType(entityType string) slog.Attr {
	const entityTypeKey = "entity_type"
	return slog.String(entityTypeKey, entityType)
}

func EntityID(id string) slog.Attr {
	const entityIDKey = "entity_id"
	return slog.String(entityIDKey, id)
}

func Action(action string) slog.Attr {
	const actionKey = "action"
	return slog.String(actionKey, action)
}

func Timestamp(ts string) slog.Attr {
	const timestampKey = "timestamp"
	return slog.String(timestampKey, ts)
}

func Since(since string) slog.Attr {
	const sinceKey = "since"
	return slog.String(sinceKey, since)
}

func SinceTime(t time.Time) slog.Attr {
	const sinceKey = "since"
	return slog.Time(sinceKey, t)
}

func Data(data string) slog.Attr {
	const dataKey = "data"
	return slog.String(dataKey, data)
}

func Backoff(d time.Duration) slog.Attr {
	const backoffKey = "backoff"
	return slog.Duration(backoffKey, d)
}

func Type(t string) slog.Attr {
	const typeKey = "type"
	return slog.String(typeKey, t)
}
