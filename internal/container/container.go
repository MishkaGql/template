package container

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"

	"github.com/MishkaGql/template/internal/config"
	applicationTrip "github.com/MishkaGql/template/internal/core/application/trip"
	"github.com/MishkaGql/template/internal/core/infrastructure/db"
	infrastructureTrip "github.com/MishkaGql/template/internal/core/infrastructure/trip"
	presentationTrip "github.com/MishkaGql/template/internal/core/presentation/trip"
	"github.com/MishkaGql/template/internal/observability"
)

type Container struct {
	Pool   *pgxpool.Pool
	Pinger *db.Pinger
	Router http.Handler
}

func New(
	ctx context.Context,
	cfg *config.Config,
	logger *slog.Logger,
	metrics *observability.Metrics,
	mp metric.MeterProvider,
) (*Container, error) {
	pool, err := db.NewPool(ctx, cfg.DB, mp)
	if err != nil {
		return nil, err
	}

	pinger := db.NewPinger(pool, cfg.DB.PingTimeout)

	tx := db.NewPgxTxManager(pool)

	repo := infrastructureTrip.NewRepository(pool, cfg.DB.QueryTimeout)
	service := applicationTrip.NewService(repo, tx, logger, metrics)
	handler := presentationTrip.NewHandler(service, pinger, logger)

	router, err := NewRouter(handler, logger, metrics)
	if err != nil {
		pool.Close()
		return nil, err
	}

	return &Container{
		Pool:   pool,
		Pinger: pinger,
		Router: router,
	}, nil
}

func (c *Container) Close() {
	c.Pool.Close()
}
