package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MishkaGql/template/internal/config"
	applicationTrip "github.com/MishkaGql/template/internal/core/application/trip"
	"github.com/MishkaGql/template/internal/core/infrastructure/db"
	infrastructureTrip "github.com/MishkaGql/template/internal/core/infrastructure/trip"
	presentationTrip "github.com/MishkaGql/template/internal/core/presentation/trip"
	"github.com/MishkaGql/template/internal/gateway/push"
	"github.com/MishkaGql/template/internal/observability"
	"github.com/MishkaGql/template/internal/worker/positions"
)

type Container struct {
	Pool   *pgxpool.Pool
	Pinger *db.Pinger
	Router http.Handler

	otelProvider *observability.Provider
	Positions    *positions.Pool
	push         *push.Client
}

func New(ctx context.Context, cfg *config.Config) (_ *Container, err error) {
	logger := observability.NewLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	otelProvider, err := observability.New(ctx, observability.ServiceName)
	if err != nil {
		return nil, err
	}

	c := &Container{
		otelProvider: otelProvider,
	}
	defer func() {
		if err != nil {
			closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
			defer cancel()
			err = errors.Join(err, c.Close(closeCtx))
		}
	}()

	metrics, err := observability.NewMetrics(otelProvider.MeterProvider())
	if err != nil {
		return nil, err
	}

	pool, err := db.NewPool(ctx, cfg.DB, otelProvider.MeterProvider())
	if err != nil {
		return nil, err
	}
	c.Pool = pool

	pinger := db.NewPinger(pool, cfg.DB.PingTimeout)
	if err := pinger.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	tx := db.NewPgxTxManager(pool)

	repo := infrastructureTrip.NewRepository(pool, cfg.DB.QueryTimeout)
	service := applicationTrip.NewService(repo, tx, logger, metrics)
	client, err := push.New(cfg.Positions.PushURL, cfg.Positions.RequestTimeout)
	if err != nil {
		return nil, err
	}
	c.push = client
	c.Positions, err = positions.New(cfg.Positions, repo, client, logger.With("service", observability.ServiceName), otelProvider.MeterProvider().Meter("positions"))
	if err != nil {
		return nil, err
	}
	handler := presentationTrip.NewHandler(service, pinger, logger, c.Positions.Cancel)
	c.Router, err = NewRouter(handler, logger, metrics)
	if err != nil {
		return nil, err
	}
	c.Pinger = pinger
	return c, nil
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
