package trip

import (
	"context"

	"github.com/google/uuid"

	applicationTrip "github.com/MishkaGql/template/internal/core/application/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=trip

type (
	Pinger interface {
		Ping(ctx context.Context) error
	}

	Service interface {
		Create(ctx context.Context, input applicationTrip.Input) (applicationTrip.Output, error)
		Get(ctx context.Context, id uuid.UUID) (applicationTrip.Output, error)
		Finish(ctx context.Context, id uuid.UUID) (applicationTrip.Output, error)
	}
)
