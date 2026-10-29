package trip

import (
	"context"

	"github.com/MishkaGql/template/internal/core/application/trip/converter"
	"github.com/MishkaGql/template/internal/core/application/trip/dto"
	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/google/uuid"
)

type Service struct {
	repo 		TripRepository
	tx         	TxManager
}

func NewService(repo TripRepository, tx TxManager) *Service {
	return &Service{
		repo: repo,
		tx:   tx,
	}
}

func (svc Service) Create(ctx context.Context, input dto.TripInput) (dto.TripOutput, error) {
	t := trip.New(
		input.UserId, 
		input.DriverId, 
		input.Price,
		trip.Coordinates(input.StartPoint),
		trip.Coordinates(input.EndPoint),
	)
	
	var saved *trip.Trip
	err := svc.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		
		saved, err = svc.repo.Save(ctx, *t)
		if err != nil {
			return err
		}

		// ....

		return nil

	})
	
	if err != nil {
		return dto.TripOutput{}, err
	}

	return converter.ToOutput(*saved), nil
}

func (svc *Service) Get(ctx context.Context, id uuid.UUID) (dto.TripOutput, error) {

	t, err := svc.repo.Get(ctx, uuid.UUID(id))
	if err != nil {
		return dto.TripOutput{}, err
	}

	return converter.ToOutput(*t), nil
}

func (svc *Service) Finish(ctx context.Context, id uuid.UUID) (dto.TripOutput, error) {
	
	t, err := svc.repo.Get(ctx, uuid.UUID(id))
	if err != nil {
		return dto.TripOutput{}, err
	}
	
	t.Complete()
	var saved *trip.Trip
	
	err = svc.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		saved, err = svc.repo.Save(ctx, *t)
		if err != nil {
			return err
		}
		
		//...

		
		return nil

	})

	if err != nil {
		return dto.TripOutput{}, err
	}

	return converter.ToOutput(*saved), nil
}
