package logging

import (
	"context"
	"log/slog"

	"github.com/go-logr/logr"
)

// ToContext adds logger to context.
func ToContext(ctx context.Context, l *slog.Logger) context.Context {
	return logr.NewContextWithSlogLogger(ctx, l)
}

// FromContext returns logger from context.
func FromContext(ctx context.Context) *slog.Logger {
	return logr.FromContextAsSlogLogger(ctx)
}
