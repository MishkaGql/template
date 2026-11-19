package trip

import (
	"time"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

const (
	CodeDriverBusy           errs.Code = "driver_busy"
	CodeTripNotFound         errs.Code = "trip_not_found"
	CodeTripAlreadyCompleted errs.Code = "trip_completed"
	CodeIdempotencyConflict  errs.Code = "idempotency_conflict"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type StatusHistory struct {
	TripID     uuid.UUID
	FromStatus *Status
	ToStatus   Status
}

type Trip struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	DriverID uuid.UUID
	Price    int64
	Status   Status

	StartPoint Coordinates
	EndPoint   Coordinates

	LastPositionAt *time.Time

	StartedAt  time.Time
	FinishedAt *time.Time
}

func New(
	userID uuid.UUID,
	driverID uuid.UUID,
	price int64,
	startPoint Coordinates,
	endPoint Coordinates,
) (*Trip, error) {
	if userID == uuid.Nil || driverID == uuid.Nil || price < 0 {
		return nil, errs.NewValidation()
	}

	if err := startPoint.validate(); err != nil {
		return nil, err
	}

	if err := endPoint.validate(); err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, errs.NewInternal(err)
	}

	return &Trip{
		ID:       id,
		UserID:   userID,
		DriverID: driverID,
		Price:    price,
		Status:   StatusActive,

		StartPoint: startPoint,
		EndPoint:   endPoint,

		StartedAt: time.Now(),
	}, nil
}

func (t *Trip) Complete() error {
	if t.Status == StatusCompleted {
		return errs.New(CodeTripAlreadyCompleted, "Operation is not allowed for a completed trips", nil)
	}

	t.Status = StatusCompleted

	finishedAt := time.Now()
	t.FinishedAt = &finishedAt

	return nil
}
