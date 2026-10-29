package trip

import (
	"context"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/google/uuid"
)

type (

	TxManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	TripRepository interface {
		Save(ctx context.Context, trip trip.Trip) (*trip.Trip, error) 
		Get(ctx context.Context,  tripId uuid.UUID) (*trip.Trip, error) 
	}
)