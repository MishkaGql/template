package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracing_No_SpanNamedByRouteTemplate(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111/finish", nil)

	Tracing("trip-service", func(*http.Request) string { return "/api/v1/trips/{tripId}/finish" })(handler).ServeHTTP(rec, req)

	require.NoError(t, provider.ForceFlush(context.Background()))

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, "POST /api/v1/trips/{tripId}/finish", spans[0].Name())
	assert.NotContains(t, spans[0].Name(), "1f0a9c62")
}

func TestMetrics_No_ActiveRequestCounter(t *testing.T) {
	fake := &activeRequestsRecorder{}

	var during int64

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		during = fake.current()
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111", nil)

	Metrics(fake, func(*http.Request) string { return "/api/v1/trips/{tripId}" })(handler).ServeHTTP(rec, req)

	assert.Equal(t, int64(1), during, "во время запроса счётчик должен быть 1")
	assert.Equal(t, int64(0), fake.current(), "после ответа счётчик должен вернуться к 0")
	assert.Equal(t, "/api/v1/trips/{tripId}", fake.route, "в атрибут http.route должен попасть шаблон")
	assert.Equal(t, http.MethodGet, fake.method)
}

type activeRequestsRecorder struct {
	value  int64
	method string
	route  string
}

func (r *activeRequestsRecorder) AddActiveRequest(_ context.Context, delta int64, method, route string) {
	r.value += delta
	r.method = method
	r.route = route
}

func (r *activeRequestsRecorder) current() int64 {
	return r.value
}
