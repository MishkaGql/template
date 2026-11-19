package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/contracts/openapi"
	"github.com/MishkaGql/template/internal/core/presentation/middleware"
	"github.com/MishkaGql/template/internal/core/presentation/problem"
	"github.com/MishkaGql/template/internal/core/presentation/trip"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
	"github.com/MishkaGql/template/internal/observability"
)

func NewRouter(h *trip.Handler, logger *slog.Logger, metrics middleware.ActiveRequests) (http.Handler, error) {
	spec, err := openapi.Load()
	if err != nil {
		return nil, fmt.Errorf("load push-service.openapi.yaml spec: %w", err)
	}
	spec.Servers = nil

	validator := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		ErrorHandlerWithOpts: func(
			ctx context.Context,
			_ error,
			w http.ResponseWriter,
			_ *http.Request,
			_ nethttpmiddleware.ErrorHandlerOpts,
		) {
			problem.Send(ctx, logger, w, errs.NewValidation())
		},
	})

	mux := chi.NewRouter()
	mux.Use(validator)

	handler := api.HandlerWithOptions(h, api.ChiServerOptions{
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			problem.Send(r.Context(), logger, w, errs.NewValidation())
		},
	})

	routeOf := func(r *http.Request) string {
		if r.Pattern != "" {
			return r.Pattern
		}

		rctx := chi.NewRouteContext()
		if !mux.Match(rctx, r.Method, r.URL.Path) {
			return ""
		}

		return rctx.RoutePattern()
	}

	return middleware.Recover(logger)(
		middleware.Tracing(observability.ServiceName, routeOf)(
			middleware.Logging(observability.ServiceName, logger, routeOf)(
				middleware.Metrics(metrics, routeOf)(handler),
			),
		),
	), nil
}
