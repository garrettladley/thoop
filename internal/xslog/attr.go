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
