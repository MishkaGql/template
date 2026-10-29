package db

// Общий интерфейс для работы с с pgx.Tx и pgxpool.Pool.
// Причина? Методы репы могут быть вызваны как через pool, так и через tx
// и что бы репа не знала как ее вызывают - приводим pool и tx к единому интерфейсу

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func ExtractQuerier(ctx context.Context, pool *pgxpool.Pool) Querier {
	tx, ok := GetTx(ctx)
	if ok {
		return tx
	}
	return pool
}
