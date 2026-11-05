package trip

import (
	"context"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/internal/core/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=trip

type (
	TxManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	Repository interface {
		Create(ctx context.Context, t *trip.Trip) (*trip.Trip, error)
		Update(ctx context.Context, t *trip.Trip) (*trip.Trip, error)

		Get(ctx context.Context, id uuid.UUID) (*trip.Trip, error)
		GetForUpdate(ctx context.Context, id uuid.UUID) (*trip.Trip, error)

		SaveStatusHistory(ctx context.Context, h *trip.StatusHistory) error
	}
)
