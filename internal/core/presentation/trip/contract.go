package trip

import (
	"context"

	"github.com/google/uuid"

	applicationTrip "github.com/MishkaGql/template/internal/core/application/trip"
	"github.com/MishkaGql/template/internal/core/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=trip

type (
	Pinger interface {
		Ping(ctx context.Context) error
	}

	Service interface {
		Create(ctx context.Context, input applicationTrip.Input) (*trip.Trip, error)
		Get(ctx context.Context, id uuid.UUID) (*trip.Trip, error)
		Finish(ctx context.Context, id uuid.UUID) (*trip.Trip, error)

		CreateTripPosition(ctx context.Context, id uuid.UUID, input applicationTrip.PositionInput) (*trip.Position, error)
		ListTripPositions(ctx context.Context, id uuid.UUID) ([]trip.Position, error)
	}
)
