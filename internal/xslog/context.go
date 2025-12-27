package xslog

import (
	"context"
	"log/slog"
)

type loggerKey struct{}

// WithLogger stores an enriched logger in context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext retrieves logger from context, or returns slog.Default().
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}

// WithAttrs adds attributes to the context logger.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	logger := FromContext(ctx)
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	return WithLogger(ctx, logger.With(args...))
}
