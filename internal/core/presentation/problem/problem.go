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
	problem := buildProblem(ctx, err)
	if problem.Status == http.StatusInternalServerError {
		tracelog.WithTrace(ctx, logger).ErrorContext(ctx, "request failed", "error", err)
	}
	httpsender.SendResponse(w, httpsender.MIMEProblemJSON, int(problem.Status), problem)
}

func buildProblem(ctx context.Context, err error) api.Problem {
	publicErr, ok := errs.AsError(err)
	if !ok || publicErr == nil {
		publicErr = errs.NewInternal(nil)
	}

	def, ok := definitionByCode[publicErr.Code()]
	if !ok {
		publicErr = errs.NewInternal(nil)
		def = definitionByCode[errs.InternalError]
	}

	return api.Problem{
		Title:    def.title,
		Detail:   publicErr.Message(),
		Code:     string(publicErr.Code()),
		Type:     baseURL + strings.ReplaceAll(string(publicErr.Code()), "_", "-"),
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
