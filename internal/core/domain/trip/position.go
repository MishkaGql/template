package trip

import (
	"time"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

type Position struct {
	ID          int64
	TripID      uuid.UUID
	Coordinates Coordinates
	RecordedAt  time.Time
}

func (t *Trip) AddPosition(coordinates Coordinates, recordedAt time.Time) (*Position, error) {
	if t.Status != StatusActive {
		return nil, errs.New(CodeTripAlreadyCompleted, "Operation is not allowed for a completed trips", nil)
	}

	if recordedAt.After(time.Now().Add(time.Minute)) {
		return nil, errs.NewValidation()
	}

	if err := coordinates.validate(); err != nil {
		return nil, err
	}

	return &Position{
		TripID:      t.ID,
		Coordinates: coordinates,
		RecordedAt:  recordedAt,
	}, nil
}
