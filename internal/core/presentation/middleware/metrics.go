package middleware

import (
	"context"
	"net/http"
)

type ActiveRequests interface {
	AddActiveRequest(ctx context.Context, delta int64, method, route string)
}

func Metrics(metrics ActiveRequests, routeOf RouteFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := routeOf(r)

			metrics.AddActiveRequest(r.Context(), 1, r.Method, route)
			defer metrics.AddActiveRequest(r.Context(), -1, r.Method, route)

			next.ServeHTTP(w, r)
		})
	}
}
