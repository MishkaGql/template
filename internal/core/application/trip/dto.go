package trip

import (
	"time"

	"github.com/google/uuid"
)

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

type PositionInput struct {
	Coords     Coordinates
	RecordedAt time.Time
}
