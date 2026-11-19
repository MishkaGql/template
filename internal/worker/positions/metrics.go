package positions

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type instruments struct {
	checks   metric.Int64Counter
	requests metric.Int64Counter
	duration metric.Float64Histogram
}

func resultLabel(result string) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String("result", result))
}

func (p *Pool) registerMetrics(meter metric.Meter) error {
	var err error
	p.metrics.checks, err = meter.Int64Counter("tripgo.stale_checks")
	if err != nil {
		return err
	}
	p.metrics.requests, err = meter.Int64Counter("tripgo.position_requests")
	if err != nil {
		return err
	}
	p.metrics.duration, err = meter.Float64Histogram("tripgo.position_requests.duration", metric.WithUnit("s"))
	if err != nil {
		return err
	}
	_, err = meter.Int64ObservableGauge("tripgo.position_requests.queue.size", metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(int64(len(p.queue)))
		return nil
	}))
	if err != nil {
		return err
	}
	_, err = meter.Int64ObservableGauge("tripgo.position_requests.workers.active", metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
		o.Observe(p.active.Load())
		return nil
	}))
	return err
}
