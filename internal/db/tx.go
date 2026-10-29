package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MishkaGql/template/internal/errs"
)

type ctxKey string
const txKey ctxKey = "tx"

type PgxTxManager struct {
	pool *pgxpool.Pool
}

func NewPgxTxManager(pool *pgxpool.Pool) *PgxTxManager {
	return &PgxTxManager{
		pool: pool,
	}
}

// fn() должна возвращать бизнес ошибки, поэтому ошибку в ней не оборачиваем.
// Метод оборачивает только Begin(), Rollback(), Commit().
func (m *PgxTxManager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	// Если вызываем Do() внутри fn() (вложенная транзакция) - не начинаем новую транзакцию.
	// При rollback - откатятся все вложенные.
	_, ok := GetTx(ctx)
	if ok {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return errs.NewInternalServiceError(err)
	}

	defer func() {
		// Почему новый контекст: https://github.com/jackc/pgx/issues/1623
		rollbackErr := tx.Rollback(context.WithoutCancel(ctx))
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			if err == nil {
				err = errs.NewInternalServiceError(rollbackErr)
				return
			}

			if causer, ok := err.(errs.Causer); ok {
				causer.SetCause(rollbackErr)
			} else {
				// Мб лучше панику кинуть или вернуть интернал и залогировать прям здесь
				err = errors.Join(err, rollbackErr)
			}
		}
	}()

	txContext := context.WithValue(ctx, txKey, tx)

	err = fn(txContext)
	
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return errs.NewInternalServiceError(err)
	}

	return nil
}

func GetTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey).(pgx.Tx);
	if ok {
		return tx, true
	}
	return nil, false
}
