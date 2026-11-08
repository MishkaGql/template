package repository

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
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

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

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

	query, args, err := psql.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(id, from, to, reason).
		ToSql()
	if err != nil {
		return errs.NewInternalServiceError(err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	_, err = querier.Exec(timeoutCtx, query, args...)
	if err != nil {
		return errs.NewInternalServiceError(err)
	}

	return nil
}

func (r *TripRepository) Create(ctx context.Context, trip api.Trip) (api.Trip, error) {
	querier := db.ExtractQuerier(ctx, r.pool)

	query, args, err := psql.Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at",
		).
		Values(
			trip.Id, trip.UserId, trip.DriverId,
			trip.StartPoint.Latitude, trip.StartPoint.Longitude,
			trip.EndPoint.Latitude, trip.EndPoint.Longitude,
			trip.Price, trip.Status, trip.StartedAt,
		).
		Suffix("RETURNING *").
		ToSql()
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	row, err := querier.Query(timeoutCtx, query, args...)
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

	query, args, err := psql.Select("*").
		From("trips").
		Where(sq.Eq{"id": tripId}).
		ToSql()
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	row, err := querier.Query(timeoutCtx, query, args...)
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

	query, args, err := psql.Select("status").
		From("trips").
		Where(sq.Eq{"id": tripId}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	row, err := querier.Query(timeoutCtx, query, args...)
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

	query, args, err = psql.Update("trips").
		Set("status", api.Completed).
		Set("finished_at", finishedAt).
		Where(sq.Eq{"id": tripId}).
		Suffix("RETURNING *").
		ToSql()
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	row, err = querier.Query(timeoutCtx, query, args...)
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	tripEntity, err := pgx.CollectOneRow(row, pgx.RowToStructByName[entity.TripEntity])
	if err != nil {
		return api.Trip{}, errs.NewInternalServiceError(err)
	}

	return mapper.ToApi(tripEntity), nil
}
