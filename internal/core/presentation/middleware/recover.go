package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/MishkaGql/template/internal/core/presentation/problem"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
	"github.com/MishkaGql/template/internal/core/sharedkernel/tracelog"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}

				tracelog.WithTrace(r.Context(), logger).Error("panic recovered",
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
				)

				problem.Send(r.Context(), logger, w, errs.NewInternalServiceError(nil))
			}()

			next.ServeHTTP(w, r)
		})
	}
}
