package xslog

import (
	"log/slog"

	"github.com/garrettladley/thoop/internal/version"
)

func Version() slog.Attr {
	const versionKey = "version"
	return slog.String(versionKey, version.Get())
}
