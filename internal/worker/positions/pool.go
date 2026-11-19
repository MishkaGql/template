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

const (
	resultSkipped   = ""
	resultQueued    = "queued"
	resultDuplicate = "duplicate"
	resultDropped   = "dropped"
	resultSuccess   = "success"
	resultError     = "error"
)

type job struct {
	trip Trip
	scan trace.SpanContext
}

type Pool struct {
	cfg     config.PositionConfig
	repo    Repository
	push    Gateway
	logger  *slog.Logger
	metrics instruments

	queue  chan *job
	active atomic.Int64

	mu                 sync.Mutex
	pending            map[uuid.UUID]*job
	receivedDuringScan map[uuid.UUID]bool

	stopScan    context.CancelFunc
	stopWorkers context.CancelFunc
	done        chan struct{}
}

func New(
	cfg config.PositionConfig,
	repo Repository,
	push Gateway,
	logger *slog.Logger,
	meter metric.Meter,
) (*Pool, error) {
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
			for task := range p.queue {
				p.process(workerCtx, task)
			}
		})
	}

	go func() {
		p.runScanner(scanCtx, ticks)
		close(p.queue)
		workers.Wait()

		stopWorkers()
		stopScan()
		close(p.done)
	}()
}

func (p *Pool) runScanner(ctx context.Context, ticks Ticker) {
	defer ticks.Stop()

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-ticks.Ticks():
			p.scan(ctx)
		}
	}
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

	for _, trip := range trips {
		if ctx.Err() != nil {
			return
		}

		result := p.enqueue(trip, span.SpanContext())
		if result == resultSkipped {
			continue
		}

		p.metrics.checks.Add(ctx, 1, resultLabel(result))

		if result == resultDropped {
			p.metrics.requests.Add(ctx, 1, resultLabel(result))
			tracelog.WithTrace(ctx, p.logger).Warn("position queue full", "trip_id", trip.ID)
		}
	}
}

func (p *Pool) enqueue(trip Trip, scan trace.SpanContext) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.receivedDuringScan[trip.ID] {
		return resultSkipped
	}

	if p.pending[trip.ID] != nil {
		return resultDuplicate
	}

	task := &job{trip: trip, scan: scan}

	select {
	case p.queue <- task:
		p.pending[trip.ID] = task
		return resultQueued
	default:
		return resultDropped
	}
}

func (p *Pool) process(ctx context.Context, task *job) {
	id := task.trip.ID

	p.mu.Lock()
	pending := p.pending[id] == task
	p.mu.Unlock()

	if !pending {
		return
	}

	defer func() {
		p.mu.Lock()
		if p.pending[id] == task {
			delete(p.pending, id)
		}
		p.mu.Unlock()
	}()

	if ctx.Err() != nil {
		p.metrics.requests.Add(ctx, 1, resultLabel(resultDropped))
		return
	}

	p.sendRequest(ctx, task)
}

func (p *Pool) sendRequest(ctx context.Context, task *job) {
	ctx, span := otel.Tracer("positions").Start(ctx, "positions.request",
		trace.WithNewRoot(),
		trace.WithLinks(trace.Link{SpanContext: task.scan}),
	)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, p.cfg.RequestTimeout)
	defer cancel()

	p.active.Add(1)
	defer p.active.Add(-1)

	start := time.Now()
	err := p.push.RequestPosition(ctx, task.trip.ID, task.trip.DriverID)
	duration := time.Since(start).Seconds()

	result := resultSuccess
	if err != nil {
		result = resultError
		span.RecordError(err)
		tracelog.WithTrace(ctx, p.logger).Warn("position request failed", "trip_id", task.trip.ID, "error", err)
	}

	p.metrics.requests.Add(ctx, 1, resultLabel(result))
	p.metrics.duration.Record(ctx, duration, resultLabel(result))
}
