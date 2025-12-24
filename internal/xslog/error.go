package xslog

import "log/slog"

func Error(err error) slog.Attr {
	const errorKey = "error"
	return slog.String(errorKey, err.Error())
}
