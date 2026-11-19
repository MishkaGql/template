package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const instrumentationName = "github.com/MishkaGql/template"

type Metrics struct {
	positionsSaved metric.Int64Counter
	tripsCreated   metric.Int64Counter
	tripsCompleted metric.Int64Counter
	activeRequests metric.Int64UpDownCounter
}

func NewMetrics(provider metric.MeterProvider) (*Metrics, error) {
	meter := provider.Meter(instrumentationName)

	created, err := meter.Int64Counter(
		"tripgo.trips.created",
		metric.WithDescription("Созданные поездки"),
	)
	if err != nil {
		return nil, err
	}

	completed, err := meter.Int64Counter(
		"tripgo.trips.completed",
		metric.WithDescription("Завершённые поездки"),
	)
	if err != nil {
		return nil, err
	}

	active, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Запросы в обработке прямо сейчас"),
	)
	if err != nil {
		return nil, err
	}

	saved, err := meter.Int64Counter("tripgo.positions.saved")
	if err != nil {
		return nil, err
	}
	return &Metrics{
		positionsSaved: saved,
		tripsCreated:   created,
		tripsCompleted: completed,
		activeRequests: active,
	}, nil
}

func (m *Metrics) PositionSaved(ctx context.Context) { m.positionsSaved.Add(ctx, 1) }

func (m *Metrics) TripCreated(ctx context.Context) {
	m.tripsCreated.Add(ctx, 1)
}

func (m *Metrics) TripCompleted(ctx context.Context) {
	m.tripsCompleted.Add(ctx, 1)
}

func (m *Metrics) AddActiveRequest(ctx context.Context, delta int64, method, route string) {
	m.activeRequests.Add(ctx, delta, metric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
	))
}
