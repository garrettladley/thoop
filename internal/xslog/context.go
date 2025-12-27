package xslog

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}

func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	logger := FromContext(ctx)
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	return WithLogger(ctx, logger.With(args...))
}
