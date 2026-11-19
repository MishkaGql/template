package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

type ctxKey struct{}

const rollbackTimeout = 5 * time.Second

type PgxTxManager struct {
	db *pgxpool.Pool
}

func NewPgxTxManager(pool *pgxpool.Pool) *PgxTxManager {
	return &PgxTxManager{
		db: pool,
	}
}

func (m *PgxTxManager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	_, ok := ExtractTx(ctx)
	if ok {
		return fn(ctx)
	}

	tx, err := m.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return errs.NewInternal(err)
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
		defer cancel()

		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errs.NewInternal(errors.Join(err, rollbackErr))
		}
	}()

	txContext := context.WithValue(ctx, ctxKey{}, tx)
	if err = fn(txContext); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return errs.NewInternal(err)
	}

	return nil
}

func ExtractTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(ctxKey{}).(pgx.Tx)
	return tx, ok
}
