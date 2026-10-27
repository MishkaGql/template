package internal

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/repository"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	repository 	*repository.TripRepository
	tx         	TxManager
}

func NewService(r *repository.TripRepository, tx TxManager) *Service {
	return &Service{
		repository: r,
		tx:         tx,
	}
}


func (svc Service) Create(ctx context.Context, tripData api.TripData, startedAt time.Time) (api.Trip, error) {

	trip := api.Trip{
		DriverId: tripData.DriverId,
		EndPoint: api.Coordinates{
			Latitude:  tripData.EndPoint.Latitude,
			Longitude: tripData.EndPoint.Longitude,
		},
		FinishedAt:     nil,
		Id:             uuid.New(),
		LastPositionAt: nil,
		Price:          tripData.Price,
		StartPoint: api.Coordinates{
			Latitude:  tripData.StartPoint.Latitude,
			Longitude: tripData.StartPoint.Longitude,
		},
		StartedAt: startedAt,
		Status:    api.Active,
		UserId:    tripData.UserId,
	}

	var result api.Trip

	err := svc.tx.Do(ctx, func(ctx context.Context) error {
		savedTrip, err := svc.repository.Create(ctx, trip)
		if err != nil {
			return err
		}

		err = svc.repository.SaveTripStatusHistory(ctx, savedTrip.Id, nil, api.Active, nil)
		if err != nil {
			return err
		}

		result = savedTrip
		return nil
	})
	
	if err != nil {
		return api.Trip{}, err
	}

	return result, nil
}
func (svc *Service) Get(ctx context.Context, id api.TripId) (api.Trip, error) {

	result, err := svc.repository.Get(ctx, uuid.UUID(id))
	if err != nil {
		return api.Trip{}, err
	}

	return result, nil
}

func (svc *Service) Finish(ctx context.Context, finishedAt time.Time, id api.TripId) (api.Trip, error) {

	var result api.Trip
	err := svc.tx.Do(ctx, func(ctx context.Context) error {
		trip, err := svc.repository.Finish(ctx, uuid.UUID(id), finishedAt)
		if err != nil {
			return err
		}

		reason := "trip successfully finished"
		fromStatus := api.Active
		err = svc.repository.SaveTripStatusHistory(
			ctx,
			trip.Id,
			&fromStatus,
			api.Completed,
			&reason,
		)
		if err != nil {
			return err
		}

		result = trip
		return nil

	})

	if err != nil {
		return api.Trip{}, err
	}

	return result, nil
}
