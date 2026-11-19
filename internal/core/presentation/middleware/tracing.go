package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func Tracing(serviceName string, routeOf RouteFunc) func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware(serviceName,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + routeOf(r)
		}),
	)
}
