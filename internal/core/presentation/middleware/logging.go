package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/MishkaGql/template/internal/core/sharedkernel/tracelog"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Logging(serviceName string, logger *slog.Logger, routeOf RouteFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			level := slog.LevelInfo
			if rec.status >= http.StatusInternalServerError {
				level = slog.LevelError
			}

			markSpanError(r, rec.status)

			tracelog.WithTrace(r.Context(), logger).Log(r.Context(), level, "request handled",
				slog.String("service", serviceName),
				slog.String("method", r.Method),
				slog.String("route", routeOf(r)),
				slog.Int("status", rec.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		})
	}
}

func markSpanError(r *http.Request, status int) {
	if status < http.StatusBadRequest {
		return
	}

	span := trace.SpanFromContext(r.Context())
	if !span.IsRecording() {
		return
	}

	errorType := "http_" + strconv.Itoa(status)

	span.SetStatus(codes.Error, errorType)
	span.SetAttributes(attribute.String("error.type", errorType))
}
