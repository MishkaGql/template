package problem

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

func TestBuildProblem_CodeInvalidRequest_ValidationError(t *testing.T) {
	absErr, ok := errs.AsAbstractError(errs.NewValidationServiceError())
	require.True(t, ok)

	got := BuildProblem(t.Context(), absErr)

	assert.Equal(t, "invalid_request", got.Code)
	assert.Equal(t, int32(http.StatusBadRequest), got.Status)
	assert.Equal(t, "Invalid request", got.Title)
	assert.Equal(t, "https://tripgo.example/problems/invalid-request", got.Type)
	require.NotNil(t, got.Detail)
	assert.Equal(t, "Validation", *got.Detail)
}

func TestBuildProblem_CodeInternalError_InternalError(t *testing.T) {
	absErr, ok := errs.AsAbstractError(errs.NewInternalServiceError(errors.New("boom")))
	require.True(t, ok)

	got := BuildProblem(t.Context(), absErr)

	assert.Equal(t, "internal_error", got.Code)
	assert.Equal(t, int32(http.StatusInternalServerError), got.Status)
	assert.Equal(t, "Internal Server Error", got.Title)
	require.NotNil(t, got.Detail)
	assert.Equal(t, "Internal service error", *got.Detail)
}

func TestBuildProblem_CodeNotFound_TripNotFound(t *testing.T) {
	absErr, ok := errs.AsAbstractError(errs.NewServiceError(trip.CodeTripNotFound, "Trip was not found"))
	require.True(t, ok)

	got := BuildProblem(t.Context(), absErr)

	assert.Equal(t, "trip_not_found", got.Code)
	assert.Equal(t, int32(http.StatusNotFound), got.Status)
	assert.Equal(t, "Trip not found", got.Title)
	assert.Equal(t, "https://tripgo.example/problems/trip-not-found", got.Type)
	require.NotNil(t, got.Detail)
	assert.Equal(t, "Trip was not found", *got.Detail)
}

func TestBuildProblem_CodeConflict_CustomBusinessCode(t *testing.T) {
	tests := []struct {
		name       string
		code       errs.ErrorCode
		wantTitle  string
		wantStatus int32
		wantType   string
	}{
		{
			name:       "driver_busy",
			code:       trip.CodeDriverBusy,
			wantTitle:  "Driver busy",
			wantStatus: http.StatusConflict,
			wantType:   "https://tripgo.example/problems/driver-busy",
		},
		{
			name:       "trip_completed",
			code:       trip.CodeTripAlreadyCompleted,
			wantTitle:  "Trip completed",
			wantStatus: http.StatusConflict,
			wantType:   "https://tripgo.example/problems/trip-completed",
		},
		{
			name:       "idempotency_conflict",
			code:       trip.CodeIdempotencyConflict,
			wantTitle:  "Idempotency conflict",
			wantStatus: http.StatusConflict,
			wantType:   "https://tripgo.example/problems/idempotency-conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absErr, ok := errs.AsAbstractError(errs.NewServiceError(tt.code, "Conflict"))
			require.True(t, ok)

			got := BuildProblem(t.Context(), absErr)

			assert.Equal(t, string(tt.code), got.Code)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, tt.wantTitle, got.Title)
			assert.Equal(t, tt.wantType, got.Type)
		})
	}
}

func TestBuildProblem_CodeInternalError_UnknownCode(t *testing.T) {
	absErr, ok := errs.AsAbstractError(errs.NewServiceError("unknown_code", "Boom"))
	require.True(t, ok)

	got := BuildProblem(t.Context(), absErr)

	assert.Equal(t, int32(http.StatusInternalServerError), got.Status)
	assert.Equal(t, "Internal Server Error", got.Title)
	assert.Equal(t, "https://tripgo.example/problems/unknown-code", got.Type)
}

func TestBuildProblem_No_TraceIdAsInstance(t *testing.T) {
	absErr, ok := errs.AsAbstractError(errs.NewValidationServiceError())
	require.True(t, ok)

	got := BuildProblem(withSpanContext(t), absErr)

	require.NotNil(t, got.Instance)
	assert.Contains(t, *got.Instance, "https://tripgo.example/problems/instances/")
}

func TestBuildProblem_No_InstanceWithoutSpan(t *testing.T) {
	absErr, ok := errs.AsAbstractError(errs.NewValidationServiceError())
	require.True(t, ok)

	got := BuildProblem(context.Background(), absErr)

	assert.Nil(t, got.Instance)
}

func TestSend_CodeNotFound_TripNotFound(t *testing.T) {
	rec := httptest.NewRecorder()

	Send(context.Background(), testLogger(), rec, errs.NewServiceError(trip.CodeTripNotFound, "Trip was not found"))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
}

func TestSend_CodeInternalError_UnknownError(t *testing.T) {
	rec := httptest.NewRecorder()

	Send(context.Background(), testLogger(), rec, errors.New("insert into trips failed: password=secret"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "password=secret")
}

func TestSend_No_InternalDetailsGoToLog(t *testing.T) {
	collector := &recordCollector{}
	rec := httptest.NewRecorder()

	Send(context.Background(), slog.New(collector), rec,
		errs.NewInternalServiceError(errors.New("insert into trips failed: password=secret")))

	require.Len(t, collector.records, 1)
	assert.Equal(t, slog.LevelError, collector.records[0].Level)
	assert.NotContains(t, rec.Body.String(), "password=secret")
}

func TestSend_No_ClientErrorNotLogged(t *testing.T) {
	collector := &recordCollector{}

	Send(context.Background(), slog.New(collector), httptest.NewRecorder(),
		errs.NewServiceError(trip.CodeTripNotFound, "Trip was not found"))

	assert.Empty(t, collector.records)
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func withSpanContext(t *testing.T) context.Context {
	t.Helper()

	traceID, err := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
	require.NoError(t, err)

	spanID, err := trace.SpanIDFromHex("575c2d1c090e38eb")
	require.NoError(t, err)

	// IsValid() требует и trace, и span id: без второго контекст считается невалидным.
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})

	return trace.ContextWithSpanContext(context.Background(), spanCtx)
}

type recordCollector struct {
	records []slog.Record
}

func (c *recordCollector) Enabled(context.Context, slog.Level) bool {
	return true
}

func (c *recordCollector) Handle(_ context.Context, record slog.Record) error {
	c.records = append(c.records, record)

	return nil
}

func (c *recordCollector) WithAttrs([]slog.Attr) slog.Handler {
	return c
}

func (c *recordCollector) WithGroup(string) slog.Handler {
	return c
}
