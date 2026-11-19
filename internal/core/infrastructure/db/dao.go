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

func QueryMore[T any](
	ctx context.Context,
	querier Querier,
	sql string,
	args pgx.QueryRewriter,
	timeout time.Duration,
) ([]T, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rows, err := querier.Query(timeoutCtx, sql, args)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}
