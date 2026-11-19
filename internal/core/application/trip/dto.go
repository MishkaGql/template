package trip

import (
	"time"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/internal/core/domain/trip"
)

type Input struct {
	UserID   uuid.UUID
	DriverID uuid.UUID
	Price    int64

	StartPoint trip.Coordinates
	EndPoint   trip.Coordinates
}

type PositionInput struct {
	Coords     trip.Coordinates
	RecordedAt time.Time
}
