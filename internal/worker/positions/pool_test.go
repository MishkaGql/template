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

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

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
	repo, push, ticker := NewMockRepository(ctrl), NewMockGateway(ctrl), NewMockTicker(ctrl)
	reader := sdkmetric.NewManualReader()
	meter := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test")
	cfg := config.PositionConfig{Workers: workers, QueueSize: size, StaleAfter: time.Minute, RequestTimeout: time.Hour}
	p, err := New(cfg, repo, push, slog.New(slog.DiscardHandler), meter)
	require.NoError(t, err)
	ticks := make(chan time.Time)
	ticker.EXPECT().Ticks().Return((<-chan time.Time)(ticks)).AnyTimes()
	ticker.EXPECT().Stop()
	ctx, cancel := context.WithCancel(t.Context())
	p.start(ctx, ticker)
	return fixture{p, repo, push, ticks, reader, cancel}
}

func (f fixture) tick() { f.ticks <- time.Now(); synctest.Wait() }

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

func TestScan_DuplicateDroppedAndCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 1)
		a, b, c := Trip{uuid.New(), uuid.New()}, Trip{uuid.New(), uuid.New()}, Trip{uuid.New(), uuid.New()}
		release := make(chan struct{})
		f.push.EXPECT().RequestPosition(gomock.Any(), a.ID, a.DriverID).DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error { <-release; return nil })
		gomock.InOrder(
			f.repo.EXPECT().FindStale(gomock.Any(), time.Minute).Return([]Trip{a}, nil),
			f.repo.EXPECT().FindStale(gomock.Any(), time.Minute).Return([]Trip{a, b, c}, nil),
			f.repo.EXPECT().FindStale(gomock.Any(), time.Minute).Return([]Trip{b}, nil),
		)
		f.tick()
		f.tick()
		f.tick()
		require.Equal(t, map[string]int64{"queued": 2, "duplicate": 2, "dropped": 1}, f.counters(t, "tripgo.stale_checks"))
		f.pool.Cancel(b.ID)
		close(release)
		require.NoError(t, f.pool.Stop(t.Context()))
		require.Equal(t, map[string]int64{"success": 1, "dropped": 1}, f.counters(t, "tripgo.position_requests"))
	})
}

func TestStop_DrainsWithFixedConcurrencyAfterPushError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 2, 8)
		trips := make([]Trip, 8)
		release := make(chan struct{})
		for i := range trips {
			trips[i] = Trip{uuid.New(), uuid.New()}
			f.push.EXPECT().RequestPosition(gomock.Any(), trips[i].ID, trips[i].DriverID).DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error {
				<-release
				if i == 0 {
					return errors.New("push unavailable")
				}
				return nil
			})
		}
		f.repo.EXPECT().FindStale(gomock.Any(), time.Minute).Return(trips, nil)
		f.tick()
		require.Equal(t, int64(2), f.pool.active.Load())
		require.Len(t, f.pool.queue, 6)
		done := make(chan error, 1)
		go func() { done <- f.pool.Stop(t.Context()) }()
		synctest.Wait()
		close(release)
		require.NoError(t, <-done)
		require.Equal(t, map[string]int64{"success": 7, "error": 1}, f.counters(t, "tripgo.position_requests"))
	})
}

func TestStop_DeadlineCancelsPush(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 1)
		a := Trip{uuid.New(), uuid.New()}
		f.repo.EXPECT().FindStale(gomock.Any(), time.Minute).Return([]Trip{a}, nil)
		f.push.EXPECT().RequestPosition(gomock.Any(), a.ID, a.DriverID).DoAndReturn(func(ctx context.Context, _, _ uuid.UUID) error {
			<-ctx.Done()
			return ctx.Err()
		})
		f.tick()
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		require.ErrorIs(t, f.pool.Stop(ctx), context.DeadlineExceeded)
	})
}

func TestScan_CancelDuringQuery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := startPool(t, 1, 1)
		a := Trip{uuid.New(), uuid.New()}
		release := make(chan struct{})
		f.repo.EXPECT().FindStale(gomock.Any(), time.Minute).DoAndReturn(func(context.Context, time.Duration) ([]Trip, error) { <-release; return []Trip{a}, nil })
		f.tick()
		f.pool.Cancel(a.ID)
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
