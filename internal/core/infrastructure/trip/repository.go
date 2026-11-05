package trip

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/infrastructure/db"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

type Repository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *Repository {
	return &Repository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *Repository) Create(ctx context.Context, t *trip.Trip) (*trip.Trip, error) {
	// prepare
	sql := `
	INSERT INTO trips (
	    id, user_id, driver_id,
	    start_latitude, start_longitude,
	    end_latitude, end_longitude,
	    price, status, started_at, finished_at
	) VALUES (
	    @id, @user_id, @driver_id,
	    @start_latitude, @start_longitude,
	    @end_latitude, @end_longitude,
	    @price, @status, @started_at, @finished_at
	)
	RETURNING *`
	args := pgx.StructArgs(MapToTripEntity(t))

	// query
	querier := db.ExtractQuerier(ctx, r.pool)
	entity, err := db.QueryOne[TripEntity](ctx, querier, sql, args, r.queryTimeout)
	// check
	if err != nil {
		const codeAlreadyHasTrip = "23505"
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == codeAlreadyHasTrip {
			return nil, errs.NewServiceError(trip.CodeDriverBusy, "Driver already has an active trip")
		}
		return nil, errs.NewInternalServiceError(err)
	}

	return MapToTripDomain(entity), nil
}

func (r *Repository) Update(ctx context.Context, t *trip.Trip) (*trip.Trip, error) {
	// prepare
	sql := `
	UPDATE trips SET
	    user_id         = @user_id,
	    driver_id       = @driver_id,
	    start_latitude  = @start_latitude,
	    start_longitude = @start_longitude,
	    end_latitude    = @end_latitude,
	    end_longitude   = @end_longitude,
	    price           = @price,
	    status          = @status,
	    started_at      = @started_at,
	    finished_at     = @finished_at
	WHERE id = @id
	RETURNING *`
	args := pgx.StructArgs(MapToTripEntity(t))

	// query
	querier := db.ExtractQuerier(ctx, r.pool)
	entity, err := db.QueryOne[TripEntity](ctx, querier, sql, args, r.queryTimeout)
	// check
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.NewServiceError(trip.CodeTripNotFound, "Trip was not found")
		}
		return nil, errs.NewInternalServiceError(err)
	}

	return MapToTripDomain(entity), nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	return r.get(ctx, id, false)
}

func (r *Repository) GetForUpdate(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	return r.get(ctx, id, true)
}

func (r *Repository) get(ctx context.Context, id uuid.UUID, forUpdate bool) (*trip.Trip, error) {
	// prepare
	sql := `SELECT * FROM trips WHERE id = @id`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	args := pgx.NamedArgs{"id": id}

	// query
	querier := db.ExtractQuerier(ctx, r.pool)
	entity, err := db.QueryOne[TripEntity](ctx, querier, sql, args, r.queryTimeout)
	// check
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.NewServiceError(trip.CodeTripNotFound, "Trip was not found")
		}
		return nil, errs.NewInternalServiceError(err)
	}

	return MapToTripDomain(entity), nil
}

func (r *Repository) SaveStatusHistory(ctx context.Context, h *trip.StatusHistory) error {
	// prepare
	sql := `
	INSERT INTO trip_status_history (trip_id, from_status, to_status)
	VALUES (@trip_id, @from_status, @to_status);
	`
	args := pgx.StructArgs(MapToStatusHistoryEntity(h))

	// query
	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	querier := db.ExtractQuerier(ctx, r.pool)
	_, err := querier.Exec(timeoutCtx, sql, args)
	// check
	if err != nil {
		return errs.NewInternalServiceError(err)
	}

	return nil
}
