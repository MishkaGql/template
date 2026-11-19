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
	args := pgx.StructArgs(toTripEntity(t))

	querier := db.ExtractQuerier(ctx, r.pool)
	entity, err := db.QueryOne[TripEntity](ctx, querier, sql, args, r.queryTimeout)
	if err != nil {
		const uniqueViolation = "23505"
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok &&
			pgErr.Code == uniqueViolation && pgErr.ConstraintName == "trips_driver_id_status_idx" {
			return nil, errs.New(trip.CodeDriverBusy, "Driver already has an active trips", err)
		}
		return nil, errs.NewInternal(err)
	}

	return toTripDomain(entity), nil
}

func (r *Repository) Update(ctx context.Context, t *trip.Trip) (*trip.Trip, error) {
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
	    finished_at     = @finished_at,
	    updated_at      = now()
	WHERE id = @id
	RETURNING *`
	args := pgx.StructArgs(toTripEntity(t))

	querier := db.ExtractQuerier(ctx, r.pool)
	entity, err := db.QueryOne[TripEntity](ctx, querier, sql, args, r.queryTimeout)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.New(trip.CodeTripNotFound, "Trip was not found", err)
		}
		return nil, errs.NewInternal(err)
	}

	return toTripDomain(entity), nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	return r.get(ctx, id, false)
}

func (r *Repository) GetForUpdate(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	if _, ok := db.ExtractTx(ctx); !ok {
		return nil, errs.NewInternal(nil)
	}

	return r.get(ctx, id, true)
}

func (r *Repository) get(ctx context.Context, id uuid.UUID, forUpdate bool) (*trip.Trip, error) {
	sql := `SELECT * FROM trips WHERE id = @id`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	args := pgx.NamedArgs{"id": id}

	querier := db.ExtractQuerier(ctx, r.pool)
	entity, err := db.QueryOne[TripEntity](ctx, querier, sql, args, r.queryTimeout)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.New(trip.CodeTripNotFound, "Trip was not found", err)
		}
		return nil, errs.NewInternal(err)
	}

	return toTripDomain(entity), nil
}

func (r *Repository) SaveStatusHistory(ctx context.Context, h *trip.StatusHistory) error {
	sql := `
	INSERT INTO trip_status_history (trip_id, from_status, to_status)
	VALUES (@trip_id, @from_status, @to_status);
	`
	args := pgx.StructArgs(toStatusHistoryEntity(h))

	timeoutCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	querier := db.ExtractQuerier(ctx, r.pool)
	_, err := querier.Exec(timeoutCtx, sql, args)
	if err != nil {
		return errs.NewInternal(err)
	}

	return nil
}

func (r *Repository) CreateTripPosition(ctx context.Context, position *trip.Position) (*trip.Position, error) {
	sql := `
	WITH updated AS (
        UPDATE trips
        SET last_position_at = GREATEST(last_position_at, @recorded_at), updated_at = now()
        WHERE id = @trip_id
        RETURNING id
    )
    INSERT INTO trip_positions (trip_id, latitude, longitude, recorded_at)
    SELECT id, @latitude, @longitude, @recorded_at FROM updated
	RETURNING *`
	args := pgx.StructArgs(toPositionEntity(position))

	querier := db.ExtractQuerier(ctx, r.pool)
	entity, err := db.QueryOne[PositionEntity](ctx, querier, sql, args, r.queryTimeout)
	if err != nil {
		return nil, errs.NewInternal(err)
	}

	return toPositionDomain(entity), nil
}

func (r *Repository) ListTripPositions(ctx context.Context, id uuid.UUID) ([]trip.Position, error) {
	sql := `
		SELECT * FROM trip_positions
		WHERE trip_id = @trip_id
		ORDER BY recorded_at ASC, id ASC
	`
	args := pgx.NamedArgs{"trip_id": id}

	querier := db.ExtractQuerier(ctx, r.pool)
	entities, err := db.QueryMore[PositionEntity](ctx, querier, sql, args, r.queryTimeout)
	if err != nil {
		return nil, errs.NewInternal(err)
	}

	positions := make([]trip.Position, 0, len(entities))
	for _, entity := range entities {
		positions = append(positions, *toPositionDomain(entity))
	}

	return positions, nil
}
