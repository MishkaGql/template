package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNewMetrics_No_BusinessCounters(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	metrics, err := NewMetrics(provider)
	require.NoError(t, err)

	metrics.TripCreated(context.Background())
	metrics.TripCreated(context.Background())
	metrics.TripCompleted(context.Background())

	assert.Equal(t, int64(2), counterValue(t, reader, "tripgo.trips.created"))
	assert.Equal(t, int64(1), counterValue(t, reader, "tripgo.trips.completed"))
}

func TestAddActiveRequest_No_RouteTemplate(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	metrics, err := NewMetrics(provider)
	require.NoError(t, err)

	metrics.AddActiveRequest(context.Background(), 1, "GET", "/api/v1/trips/{tripId}")
	metrics.AddActiveRequest(context.Background(), -1, "GET", "/api/v1/trips/{tripId}")

	assert.Equal(t, int64(0), counterValue(t, reader, "http.server.active_requests"))
}

func counterValue(t *testing.T, reader *sdkmetric.ManualReader, name string) int64 {
	t.Helper()

	var data metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &data))

	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != name {
				continue
			}

			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "метрика %s должна быть целочисленным счётчиком", name)

			var total int64
			for _, point := range sum.DataPoints {
				total += point.Value
			}

			return total
		}
	}

	t.Fatalf("метрика %s не найдена", name)

	return 0
}
