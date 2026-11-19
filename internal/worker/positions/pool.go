package positions

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/MishkaGql/template/internal/config"
	"github.com/MishkaGql/template/internal/core/sharedkernel/tracelog"
)

type job struct {
	trip Trip
	scan trace.SpanContext
}

type Pool struct {
	cfg                config.PositionConfig
	repo               Repository
	push               Gateway
	logger             *slog.Logger
	metrics            instruments
	queue              chan *job
	mu                 sync.Mutex
	pending            map[uuid.UUID]*job
	receivedDuringScan map[uuid.UUID]bool
	active             atomic.Int64

	stopScan    context.CancelFunc
	stopWorkers context.CancelFunc
	done        chan struct{}
}

func New(cfg config.PositionConfig, repo Repository, push Gateway, logger *slog.Logger, meter metric.Meter) (*Pool, error) {
	p := &Pool{
		cfg:     cfg,
		repo:    repo,
		push:    push,
		logger:  logger,
		queue:   make(chan *job, cfg.QueueSize),
		pending: make(map[uuid.UUID]*job),
		done:    make(chan struct{}),
	}
	if err := p.registerMetrics(meter); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Pool) Start(ctx context.Context) {
	p.start(ctx, ticker{time.NewTicker(p.cfg.CheckInterval)})
}

func (p *Pool) start(ctx context.Context, ticks Ticker) {
	scanCtx, stopScan := context.WithCancel(ctx)
	workerCtx, stopWorkers := context.WithCancel(context.WithoutCancel(ctx))
	p.stopScan, p.stopWorkers = stopScan, stopWorkers

	var workers sync.WaitGroup
	for range p.cfg.Workers {
		workers.Go(func() {
			for j := range p.queue {
				p.process(workerCtx, j)
			}
		})
	}
	go func() {
		defer close(p.done)
		for scanCtx.Err() == nil {
			select {
			case <-scanCtx.Done():
			case <-ticks.Ticks():
				p.scan(scanCtx)
			}
		}
		ticks.Stop()
		close(p.queue)
		workers.Wait()
		stopWorkers()
		stopScan()
	}()
}

func (p *Pool) Stop(ctx context.Context) error {
	p.stopScan()
	select {
	case <-p.done:
		return ctx.Err()
	case <-ctx.Done():
		p.stopWorkers()
		<-p.done
		return ctx.Err()
	}
}

func (p *Pool) Cancel(id uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pending, id)
	if p.receivedDuringScan != nil {
		p.receivedDuringScan[id] = true
	}
}

func (p *Pool) scan(ctx context.Context) {
	ctx, span := otel.Tracer("positions").Start(ctx, "positions.scan")
	defer span.End()

	p.mu.Lock()
	p.receivedDuringScan = make(map[uuid.UUID]bool)
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.receivedDuringScan = nil
		p.mu.Unlock()
	}()

	trips, err := p.repo.FindStale(ctx, p.cfg.StaleAfter)
	if err != nil {
		if ctx.Err() == nil {
			tracelog.WithTrace(ctx, p.logger).Error("find stale trips", "error", err)
		}
		return
	}
	for _, t := range trips {
		if ctx.Err() != nil {
			return
		}
		result := p.enqueue(t, span.SpanContext())
		if result == "" {
			continue
		}
		p.metrics.checks.Add(ctx, 1, resultLabel(result))
		if result == "dropped" {
			p.metrics.requests.Add(ctx, 1, resultLabel(result))
			tracelog.WithTrace(ctx, p.logger).Warn("position queue full", "trip_id", t.ID)
		}
	}
}

func (p *Pool) enqueue(t Trip, scan trace.SpanContext) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.receivedDuringScan[t.ID] {
		return ""
	}
	if p.pending[t.ID] != nil {
		return "duplicate"
	}
	j := &job{trip: t, scan: scan}
	select {
	case p.queue <- j:
		p.pending[t.ID] = j
		return "queued"
	default:
		return "dropped"
	}
}

func (p *Pool) process(ctx context.Context, j *job) {
	id := j.trip.ID
	p.mu.Lock()
	current := p.pending[id] == j
	p.mu.Unlock()
	if !current {
		return
	}
	defer func() {
		p.mu.Lock()
		if p.pending[id] == j {
			delete(p.pending, id)
		}
		p.mu.Unlock()
	}()
	if ctx.Err() != nil {
		p.metrics.requests.Add(ctx, 1, resultLabel("dropped"))
		return
	}
	ctx, span := otel.Tracer("positions").Start(ctx, "positions.request", trace.WithNewRoot(), trace.WithLinks(trace.Link{SpanContext: j.scan}))
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, p.cfg.RequestTimeout)
	defer cancel()

	p.active.Add(1)
	defer p.active.Add(-1)
	start := time.Now()
	err := p.push.RequestPosition(ctx, id, j.trip.DriverID)
	duration := time.Since(start).Seconds()
	result := "success"
	if err != nil {
		result = "error"
		span.RecordError(err)
		tracelog.WithTrace(ctx, p.logger).Warn("position request failed", "trip_id", id, "error", err)
	}
	p.metrics.requests.Add(ctx, 1, resultLabel(result))
	p.metrics.duration.Record(ctx, duration, resultLabel(result))
}
