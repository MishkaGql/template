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

var definitionByCode = map[errs.ErrorCode]definition{
	errs.InvalidRequest:           {title: "Invalid request", status: http.StatusBadRequest},
	errs.InternalError:            {title: "Internal Server Error", status: http.StatusInternalServerError},
	trip.CodeDriverBusy:           {title: "Driver busy", status: http.StatusConflict},
	trip.CodeTripNotFound:         {title: "Trip not found", status: http.StatusNotFound},
	trip.CodeTripAlreadyCompleted: {title: "Trip completed", status: http.StatusConflict},
	trip.CodeIdempotencyConflict:  {title: "Idempotency conflict", status: http.StatusConflict},
}

func Send(ctx context.Context, logger *slog.Logger, w http.ResponseWriter, err error) {
	absErr, ok := errs.AsAbstractError(err)
	if !ok {
		absErr = errs.NewInternalServiceError(err)
	}

	problem := BuildProblem(ctx, absErr)
	logInternal(ctx, logger, err, absErr)

	httpsender.SendResponse(w, httpsender.MIMEProblemJSON, int(problem.Status), problem)
}

func BuildProblem(ctx context.Context, err errs.AbstractError) api.Problem {
	code := err.Code()
	def := definitionOf(code)

	return api.Problem{
		Title:    def.title,
		Detail:   err.Message(),
		Code:     string(code),
		Type:     baseURL + strings.ReplaceAll(string(code), "_", "-"),
		Status:   def.status,
		Instance: instanceOf(ctx),
	}
}

func definitionOf(code errs.ErrorCode) definition {
	def, ok := definitionByCode[code]
	if !ok {
		def = definitionByCode[errs.InternalError]
	}

	return def
}

func instanceOf(ctx context.Context) *string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}

	instance := baseURL + "instances/" + spanCtx.TraceID().String()

	return &instance
}

func logInternal(ctx context.Context, logger *slog.Logger, cause error, err errs.AbstractError) {
	if logger == nil || definitionOf(err.Code()).status < http.StatusInternalServerError {
		return
	}

	tracelog.WithTrace(ctx, logger).Error("internal error", "error", cause)
}
