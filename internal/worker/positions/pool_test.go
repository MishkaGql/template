package positions

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"

	"github.com/MishkaGql/template/internal/config"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestScan_DuplicateDroppedAndCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 1)
		inFlight := newTestTrip()
		queued := newTestTrip()
		overflow := newTestTrip()

		release := make(chan struct{})
		f.push.EXPECT().RequestPosition(gomock.Any(), inFlight.ID, inFlight.DriverID).
			DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error {
				<-release
				return nil
			})

		f.scanTrips(inFlight)
		f.scanTrips(inFlight, queued, overflow)
		f.scanTrips(queued)

		require.Equal(t, map[string]int64{"queued": 2, "duplicate": 2, "dropped": 1}, f.counters(t, "tripgo.stale_checks"))

		f.pool.Cancel(queued.ID)
		close(release)
		require.NoError(t, f.pool.Stop(t.Context()))
		require.Equal(t, map[string]int64{"success": 1, "dropped": 1}, f.counters(t, "tripgo.position_requests"))
	})
}

func TestStop_DrainsQueueWithFixedWorkerCount(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 2, 8)
		trips := make([]Trip, 8)
		release := make(chan struct{})
		for i := range trips {
			trips[i] = newTestTrip()
			f.push.EXPECT().RequestPosition(gomock.Any(), trips[i].ID, trips[i].DriverID).
				DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error {
					<-release
					return nil
				})
		}

		f.scanTrips(trips...)

		require.Equal(t, int64(2), f.pool.active.Load())
		require.Len(t, f.pool.queue, 6)

		done := make(chan error, 1)
		go func() { done <- f.pool.Stop(t.Context()) }()
		synctest.Wait()
		close(release)
		require.NoError(t, <-done)
		require.Equal(t, map[string]int64{"success": 8}, f.counters(t, "tripgo.position_requests"))
	})
}

func TestWorker_ContinuesAfterPushError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 2)
		first := newTestTrip()
		next := newTestTrip()

		gomock.InOrder(
			f.push.EXPECT().RequestPosition(gomock.Any(), first.ID, first.DriverID).Return(errors.New("push unavailable")),
			f.push.EXPECT().RequestPosition(gomock.Any(), next.ID, next.DriverID).Return(nil),
		)

		f.scanTrips(first, next)
		require.NoError(t, f.pool.Stop(t.Context()))
		require.Equal(t, map[string]int64{"error": 1, "success": 1}, f.counters(t, "tripgo.position_requests"))
	})
}

func TestStop_DeadlineCancelsPush(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 1)
		trip := newTestTrip()

		f.push.EXPECT().RequestPosition(gomock.Any(), trip.ID, trip.DriverID).
			DoAndReturn(func(ctx context.Context, _, _ uuid.UUID) error {
				<-ctx.Done()
				return ctx.Err()
			})

		f.scanTrips(trip)

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		require.ErrorIs(t, f.pool.Stop(ctx), context.DeadlineExceeded)
	})
}

func TestScan_CancelDuringQuery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 1)
		trip := newTestTrip()

		release := make(chan struct{})
		f.repo.EXPECT().FindStale(gomock.Any(), time.Minute).DoAndReturn(func(context.Context, time.Duration) ([]Trip, error) {
			<-release
			return []Trip{trip}, nil
		})

		f.tick()
		f.pool.Cancel(trip.ID)
		close(release)
		synctest.Wait()
		require.NoError(t, f.pool.Stop(t.Context()))
	})
}

func TestScan_StopsOnParentCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 2, 1)
		f.cancel()
		<-f.pool.done
	})
}

type fixture struct {
	pool   *Pool
	repo   *MockRepository
	push   *MockGateway
	ticks  chan time.Time
	reader *sdkmetric.ManualReader
	cancel context.CancelFunc
}

func startPool(t *testing.T, workers, size int) fixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)
	push := NewMockGateway(ctrl)
	ticker := NewMockTicker(ctrl)

	reader := sdkmetric.NewManualReader()
	meter := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test")

	cfg := config.PositionConfig{
		Workers:        workers,
		QueueSize:      size,
		StaleAfter:     time.Minute,
		RequestTimeout: time.Hour,
	}
	p, err := New(cfg, repo, push, slog.New(slog.DiscardHandler), meter)
	require.NoError(t, err)

	ticks := make(chan time.Time)
	ticker.EXPECT().Ticks().Return((<-chan time.Time)(ticks)).AnyTimes()
	ticker.EXPECT().Stop()

	ctx, cancel := context.WithCancel(t.Context())
	p.start(ctx, ticker)

	return fixture{pool: p, repo: repo, push: push, ticks: ticks, reader: reader, cancel: cancel}
}

func (f fixture) scanTrips(trips ...Trip) {
	f.repo.EXPECT().FindStale(gomock.Any(), time.Minute).Return(trips, nil)
	f.tick()
}

func (f fixture) tick() {
	f.ticks <- time.Now()
	synctest.Wait()
}

func newTestTrip() Trip {
	return Trip{ID: uuid.New(), DriverID: uuid.New()}
}

func (f fixture) counters(t *testing.T, name string) map[string]int64 {
	t.Helper()
	var data metricdata.ResourceMetrics
	require.NoError(t, f.reader.Collect(t.Context(), &data))
	counts := make(map[string]int64)
	for _, s := range data.ScopeMetrics {
		for _, m := range s.Metrics {
			if m.Name != name {
				continue
			}
			for _, pt := range m.Data.(metricdata.Sum[int64]).DataPoints {
				result, _ := pt.Attributes.Value("result")
				counts[result.AsString()] = pt.Value
			}
		}
	}
	return counts
}
