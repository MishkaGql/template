package tracelog

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

func WithTrace(ctx context.Context, logger *slog.Logger) *slog.Logger {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return logger
	}

	return logger.With(
		slog.String("trace_id", spanCtx.TraceID().String()),
		slog.String("span_id", spanCtx.SpanID().String()),
	)
}
