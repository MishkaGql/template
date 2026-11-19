package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"

	"github.com/MishkaGql/template/internal/config"
	applicationTrip "github.com/MishkaGql/template/internal/core/application/trip"
	"github.com/MishkaGql/template/internal/core/infrastructure/db"
	"github.com/MishkaGql/template/internal/core/infrastructure/gateway/push"
	infrastructureTrip "github.com/MishkaGql/template/internal/core/infrastructure/trip"
	presentationTrip "github.com/MishkaGql/template/internal/core/presentation/trip"
	"github.com/MishkaGql/template/internal/observability"
	"github.com/MishkaGql/template/internal/worker/positions"
)

type Container struct {
	Pool      *pgxpool.Pool
	Pinger    *db.Pinger
	Positions *positions.Pool
	Router    http.Handler

	otelProvider *observability.Provider
	push         *push.Client
}

func New(ctx context.Context, cfg *config.Config) (_ *Container, err error) {
	// ИНФРАСТРУКТУРНЫЕ ЗАВИСИМОСТИ

	otelProvider, err := observability.New(ctx, observability.ServiceName)
	if err != nil {
		return nil, err
	}

	c := &Container{otelProvider: otelProvider}
	defer func() {
		if err != nil {
			closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
			defer cancel()
			err = errors.Join(err, c.Close(closeCtx))
		}
	}()

	meterProvider := otelProvider.MeterProvider()
	metrics, err := observability.NewMetrics(meterProvider)
	if err != nil {
		return nil, err
	}

	c.Pool, c.Pinger, err = openDB(ctx, cfg.DB, meterProvider)
	if err != nil {
		return nil, err
	}

	c.push, err = push.New(cfg.Positions.PushURL, cfg.Positions.RequestTimeout)
	if err != nil {
		return nil, err
	}

	// АРХИТЕКТУРНЫЕ ЗАВИСИМОСТИ
	logger := observability.NewLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	repo := infrastructureTrip.NewRepository(c.Pool, cfg.DB.QueryTimeout)
	tx := db.NewPgxTxManager(c.Pool)
	service := applicationTrip.NewService(repo, tx, logger, metrics)

	c.Positions, err = positions.New(
		cfg.Positions,
		repo,
		c.push,
		logger.With("service", observability.ServiceName),
		meterProvider.Meter("positions"),
	)
	if err != nil {
		return nil, err
	}

	handler := presentationTrip.NewHandler(service, c.Pinger, logger, c.Positions.Cancel)

	c.Router, err = NewRouter(handler, logger, metrics)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func openDB(
	ctx context.Context,
	cfg config.DBConfig,
	meterProvider metric.MeterProvider,
) (*pgxpool.Pool, *db.Pinger, error) {
	pool, err := db.NewPool(ctx, cfg, meterProvider)
	if err != nil {
		return nil, nil, err
	}

	pinger := db.NewPinger(pool, cfg.PingTimeout)
	if err := pinger.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, pinger, nil
}

func (c *Container) Close(ctx context.Context) error {
	if c.push != nil {
		c.push.Close()
	}

	if c.Pool != nil {
		c.Pool.Close()
	}

	return c.otelProvider.Shutdown(ctx)
}
