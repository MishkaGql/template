package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"

	"github.com/MishkaGql/template/internal/config"
)

func NewPool(ctx context.Context, cfg config.DBConfig, mp metric.MeterProvider) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnLifetimeJitter = cfg.Jitter
	poolCfg.PingTimeout = cfg.PingTimeout
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	poolCfg.ConnConfig.Tracer = newCollectionTracer(mp)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	return pool, nil
}
