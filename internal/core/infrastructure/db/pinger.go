package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pinger struct {
	db      *pgxpool.Pool
	timeout time.Duration
}

func NewPinger(db *pgxpool.Pool, timeout time.Duration) *Pinger {
	return &Pinger{
		db:      db,
		timeout: timeout,
	}
}

func (p *Pinger) Ping(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	return p.db.Ping(timeoutCtx)
}
