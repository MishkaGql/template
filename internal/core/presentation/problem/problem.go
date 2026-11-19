package problem

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/presentation/httpsender"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
	"github.com/MishkaGql/template/internal/core/sharedkernel/tracelog"
)

const baseURL = "https://tripgo.example/problems/"

type definition struct {
	title  string
	status int32
}

var definitionByCode = map[errs.Code]definition{
	errs.InvalidRequest:           {title: "Invalid request", status: http.StatusBadRequest},
	errs.InternalError:            {title: "Internal Server Error", status: http.StatusInternalServerError},
	trip.CodeDriverBusy:           {title: "Driver busy", status: http.StatusConflict},
	trip.CodeTripNotFound:         {title: "Trip not found", status: http.StatusNotFound},
	trip.CodeTripAlreadyCompleted: {title: "Trip completed", status: http.StatusConflict},
	trip.CodeIdempotencyConflict:  {title: "Idempotency conflict", status: http.StatusConflict},
}

func Send(ctx context.Context, logger *slog.Logger, w http.ResponseWriter, err error) {
	publicErr, ok := errs.AsError(err)

	if !ok {
		publicErr = errs.NewInternal(nil)
	}

	problem := buildProblem(ctx, publicErr)
	if problem.Status == http.StatusInternalServerError {
		tracelog.WithTrace(ctx, logger).ErrorContext(ctx, "request failed", "error", err)
	}
	httpsender.SendResponse(w, httpsender.MIMEProblemJSON, int(problem.Status), problem)
}

func buildProblem(ctx context.Context, publicErr *errs.Error) api.Problem {
	_, ok := definitionByCode[publicErr.Code()]

	if !ok {
		publicErr = errs.NewInternal(nil)
	}

	return newProblem(ctx, publicErr, definitionByCode[publicErr.Code()])
}

func newProblem(ctx context.Context, err *errs.Error, def definition) api.Problem {
	return api.Problem{
		Title:    def.title,
		Detail:   err.Message(),
		Code:     string(err.Code()),
		Type:     baseURL + strings.ReplaceAll(string(err.Code()), "_", "-"),
		Status:   def.status,
		Instance: instanceOf(ctx),
	}
}

func instanceOf(ctx context.Context) *string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}

	instance := baseURL + "instances/" + spanCtx.TraceID().String()

	return &instance
}
