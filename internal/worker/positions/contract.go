package positions

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Trip struct {
	ID       uuid.UUID `db:"id"`
	DriverID uuid.UUID `db:"driver_id"`
}

type Repository interface {
	FindStale(context.Context, time.Duration) ([]Trip, error)
}

type Gateway interface {
	RequestPosition(ctx context.Context, tripID, driverID uuid.UUID) error
}

type Ticker interface {
	Ticks() <-chan time.Time
	Stop()
}

type ticker struct{ *time.Ticker }

func (t ticker) Ticks() <-chan time.Time { return t.C }
