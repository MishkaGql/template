package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func QueryOne[T any](
	ctx context.Context,
	querier Querier,
	sql string,
	args pgx.QueryRewriter,
	timeout time.Duration,
) (T, error) {
	var zero T

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	row, err := querier.Query(timeoutCtx, sql, args)
	if err != nil {
		return zero, err
	}

	return pgx.CollectOneRow(row, pgx.RowToStructByName[T])
}
