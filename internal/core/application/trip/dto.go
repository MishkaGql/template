package trip

import (
	"time"

	"github.com/google/uuid"
)

type Status string

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Input struct {
	UserID   uuid.UUID
	DriverID uuid.UUID
	Price    int64

	StartPoint Coordinates
	EndPoint   Coordinates
}

type Output struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	DriverID uuid.UUID
	Price    int64
	Status   Status

	StartPoint Coordinates
	EndPoint   Coordinates

	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}
