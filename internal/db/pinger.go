package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DbPinger struct {
	pool        *pgxpool.Pool
	pingTimeout time.Duration
}

func NewDbPinger(pool *pgxpool.Pool, timeout time.Duration) *DbPinger {
	return &DbPinger{
		pool:        pool,
		pingTimeout: timeout,
	}
}

func (p *DbPinger) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, p.pingTimeout)
	defer cancel()

	if err := p.pool.Ping(pingCtx); err != nil {
		return err
	}

	return nil

}
