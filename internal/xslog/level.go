package xslog

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type Level string

var _ fmt.Stringer = (*Level)(nil)

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

const EnvKey = "LOG_LEVEL"

const Default = LevelInfo

func Parse(s string) (Level, error) {
	switch Level(strings.ToLower(s)) {
	case LevelDebug:
		return LevelDebug, nil
	case LevelInfo:
		return LevelInfo, nil
	case LevelWarn:
		return LevelWarn, nil
	case LevelError:
		return LevelError, nil
	default:
		return "", fmt.Errorf("invalid log level: %q (valid: debug, info, warn, error)", s)
	}
}

func FromEnv() Level {
	s := os.Getenv(EnvKey)
	if s == "" {
		return Default
	}
	level, err := Parse(s)
	if err != nil {
		return Default
	}
	return level
}

func (l Level) ToSlog() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l Level) String() string {
	return string(l)
}

func NewLogger(w io.Writer, level Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level.ToSlog(),
	}))
}

func NewLoggerFromEnv(w io.Writer) *slog.Logger {
	return NewLogger(w, FromEnv())
}
