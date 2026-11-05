package trip

import (
	"time"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func NewCoordinates(latitude, longitude float64) (Coordinates, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return Coordinates{}, errs.NewValidationServiceError()
	}

	return Coordinates{
		Latitude:  latitude,
		Longitude: longitude,
	}, nil
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
		return nil, errs.NewValidationServiceError()
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, errs.NewInternalServiceError(err)
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
		return errs.NewServiceError(CodeTripAlreadyCompleted, "Operation is not allowed for a completed trip")
	}

	t.Status = StatusCompleted

	finishedAt := time.Now()
	t.FinishedAt = &finishedAt

	return nil
}
