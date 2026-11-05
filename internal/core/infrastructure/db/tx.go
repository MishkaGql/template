package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

type ctxKey struct{}

var txKey ctxKey

type PgxTxManager struct {
	db *pgxpool.Pool
}

func NewPgxTxManager(pool *pgxpool.Pool) *PgxTxManager {
	return &PgxTxManager{
		db: pool,
	}
}

func (m *PgxTxManager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	_, ok := getTx(ctx)
	if ok {
		return fn(ctx)
	}

	tx, err := m.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return errs.NewInternalServiceError(err)
	}

	defer func() {
		rollbackErr := tx.Rollback(context.WithoutCancel(ctx))
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			if err == nil {
				err = errs.NewInternalServiceError(rollbackErr)
				return
			}
			err = errors.Join(err, rollbackErr)
		}
	}()

	txContext := context.WithValue(ctx, txKey, tx)
	if err = fn(txContext); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return errs.NewInternalServiceError(err)
	}

	return nil
}

func getTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	if ok {
		return tx, true
	}
	return nil, false
}
