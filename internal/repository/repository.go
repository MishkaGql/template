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
	"github.com/MishkaGql/template/internal/db"
	"github.com/MishkaGql/template/internal/errs"
	"github.com/MishkaGql/template/internal/repository/entity"
	"github.com/MishkaGql/template/internal/repository/mapper"
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

func (r *TripRepository) SaveTripStatusHistory(
	ctx context.Context,
	id api.TripId,
	from *api.TripStatus,
	to api.TripStatus,
	reason *string,
) error {
	querier := db.ExtractQuerier(ctx, r.pool)
	
	sql := `
		INSERT INTO trip_status_history 
		(trip_id, from_status, to_status, reason) 
		VALUES 
		(@trip_id, @from_status, @to_status, @reason)
	`
	args := pgx.NamedArgs{
		"trip_id":     id,
		"from_status": from,
		"to_status":   to,
		"reason":      reason,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	_, err := querier.Exec(timeoutCtx, sql, args)
	if err != nil {
		return errs.NewInternalServiceError(err)
	}

	return nil
}

func (r *TripRepository) Create(ctx context.Context, trip api.Trip) (api.Trip, error) {
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
		RETURNING *
  	`
	args := pgx.StructArgs(mapper.ToEntity(trip))

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	row, err := querier.Query(timeoutCtx, sql, args)
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	tripEntity, err := pgx.CollectOneRow(row, pgx.RowToStructByName[entity.TripEntity])
	if err != nil {
		activeTripExistCode := "23505"
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == activeTripExistCode {
			return api.Trip{}, errs.NewServiceError(errs.DriverBusy, "Driver already has an active trip")
		}
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	return mapper.ToApi(tripEntity), nil
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

	return mapper.ToApi(tripEntity), nil
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
		WHERE id = @trip_id
		RETURNING *
	`
	args = pgx.NamedArgs{"status": api.Completed, "finished_at": finishedAt, "trip_id": tripId}

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
