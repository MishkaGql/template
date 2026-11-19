package trip

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MishkaGql/template/internal/core/infrastructure/db"
	"github.com/MishkaGql/template/internal/worker/positions"
)

func (r *Repository) FindStale(ctx context.Context, age time.Duration) ([]positions.Trip, error) {
	const query = `SELECT id, driver_id FROM trips
        WHERE status = 'active'
          AND COALESCE(last_position_at, started_at) < now() - @age::interval
        ORDER BY COALESCE(last_position_at, started_at), id`
	return db.QueryMore[positions.Trip](ctx, r.pool, query, pgx.NamedArgs{"age": age}, r.queryTimeout)
}
