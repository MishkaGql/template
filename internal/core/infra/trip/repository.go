package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/infra/db"
	"github.com/MishkaGql/template/internal/core/infra/trip/entity"
	"github.com/MishkaGql/template/internal/core/infra/trip/transformer"
	"github.com/MishkaGql/template/internal/errs"
)

type TripRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *TripRepository {
	return &TripRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *TripRepository) Save(ctx context.Context, t trip.Trip) (trip.Trip, error) {
	querier := db.ExtractQuerier(ctx, r.pool)

	sql := `
 INSERT INTO trips (
	   	id, user_id, driver_id,
	    start_latitude, start_longitude,
	    end_latitude, end_longitude,
	    price, status, started_at
	) VALUES (
	    @id, @user_id, @driver_id,
	    @start_latitude, @start_longitude,
	    @end_latitude, @end_longitude,
	    @price, @status, @started_at
	)
	ON CONFLICT (id) 
	DO UPDATE SET
	    user_id         = EXCLUDED.user_id,
	    driver_id       = EXCLUDED.driver_id,
	    start_latitude  = EXCLUDED.start_latitude,
	    start_longitude = EXCLUDED.start_longitude,
	    end_latitude    = EXCLUDED.end_longitude,
	    end_longitude   = EXCLUDED.end_longitude,
	    price           = EXCLUDED.price,
	    status          = EXCLUDED.status,
	    started_at      = EXCLUDED.started_at
	WHERE 
	    trips.status 	IS DISTINCT FROM EXCLUDED.status OR
	    trips.price  	IS DISTINCT FROM EXCLUDED.price
	RETURNING *;
  	`
   
	args := pgx.StructArgs(transformer.ToEntity(t))

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	row, err := querier.Query(timeoutCtx, sql, args)
	if err != nil {
		return trip.Trip{}, errs.NewInternalServiceError(err)
	}

	tripEntity, err := pgx.CollectOneRow(row, pgx.RowToStructByName[entity.TripEntity])
	if err != nil {
		activeTripExistCode := "23505"
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == activeTripExistCode {
			return trip.Trip{}, errs.NewServiceError(errs.DriverBusy, "Driver already has an active trip")
		}
		return trip.Trip{}, errs.NewInternalServiceError(err)
	}

	return transformer.ToDomain(tripEntity), nil
}

func (r *TripRepository) Get(ctx context.Context, tripId uuid.UUID) (api.Trip, error) {
	querier := db.ExtractQuerier(ctx, r.pool)

	sql := `SELECT * from trips where id = @id`
	args := pgx.NamedArgs{"id": tripId}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	row, err := querier.Query(timeoutCtx, sql, args)
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	tripEntity, err := pgx.CollectOneRow(row, pgx.RowToStructByName[entity.TripEntity])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Trip{}, errs.NewServiceError(errs.TripNotFound, "Trip was not found")
		}
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	return transformer.ToDomain(tripEntity), nil
}

func (r *TripRepository) Finish(ctx context.Context, tripId uuid.UUID, finishedAt time.Time) (api.Trip, error) {
	querier := db.ExtractQuerier(ctx, r.pool)

	sql := `SELECT status from trips WHERE id = @id FOR UPDATE`
	args := pgx.NamedArgs{"id": tripId}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	row, err := querier.Query(timeoutCtx, sql, args)
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	tripStatus, err := pgx.CollectOneRow(row, pgx.RowTo[api.TripStatus])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Trip{}, errs.NewServiceError(errs.TripNotFound, "Trip was not found")
		}
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	if tripStatus == api.Completed {
		return api.Trip{}, errs.NewServiceError(errs.TripAlreadyCompleted, "Operation is not allowed for a completed trip")
	}

	sql = `
		UPDATE trips
			SET status = @status,
				finished_at = @finished_at
		RETURNING *
	`
	args = pgx.NamedArgs{"status": api.Completed, "finished_at": finishedAt}

	row, err = querier.Query(timeoutCtx, sql, args)
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	tripEntity, err := pgx.CollectOneRow(row, pgx.RowToStructByName[entity.TripEntity])
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}


	return mapper.ToApi(tripEntity), nil
}
