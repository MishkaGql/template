package trip

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/sharedkernel/tracelog"
)

type BusinessMetrics interface {
	TripCreated(ctx context.Context)
	TripCompleted(ctx context.Context)
}

type Service struct {
	repo    Repository
	tx      TxManager
	logger  *slog.Logger
	metrics BusinessMetrics
}

func NewService(repo Repository, tx TxManager, logger *slog.Logger, metrics BusinessMetrics) *Service {
	return &Service{
		repo:    repo,
		tx:      tx,
		logger:  logger,
		metrics: metrics,
	}
}

func (svc *Service) Create(ctx context.Context, input Input) (Output, error) {
	startPoint, err := trip.NewCoordinates(input.StartPoint.Latitude, input.StartPoint.Longitude)
	if err != nil {
		return Output{}, err
	}

	endPoint, err := trip.NewCoordinates(input.EndPoint.Latitude, input.EndPoint.Longitude)
	if err != nil {
		return Output{}, err
	}

	t, err := trip.New(
		input.UserID,
		input.DriverID,
		input.Price,
		startPoint,
		endPoint,
	)
	if err != nil {
		return Output{}, err
	}

	var saved *trip.Trip

	err = svc.tx.Do(ctx, func(ctx context.Context) error {
		var err error

		saved, err = svc.repo.Create(ctx, t)
		if err != nil {
			return err
		}

		history := trip.NewStatusHistory(saved.ID, nil, trip.StatusActive)
		return svc.repo.SaveStatusHistory(ctx, history)
	})
	if err != nil {
		return Output{}, err
	}

	tracelog.WithTrace(ctx, svc.logger).Info("trip created",
		slog.String("trip_id", saved.ID.String()),
		slog.String("driver_id", saved.DriverID.String()),
	)
	svc.metrics.TripCreated(ctx)

	return MapToOutput(saved), nil
}

func (svc *Service) Get(ctx context.Context, id uuid.UUID) (Output, error) {
	t, err := svc.repo.Get(ctx, id)
	if err != nil {
		return Output{}, err
	}

	return MapToOutput(t), nil
}

func (svc *Service) Finish(ctx context.Context, id uuid.UUID) (Output, error) {
	var saved *trip.Trip

	err := svc.tx.Do(ctx, func(ctx context.Context) error {
		t, err := svc.repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}

		from := t.Status
		if err := t.Complete(); err != nil {
			return err
		}

		saved, err = svc.repo.Update(ctx, t)
		if err != nil {
			return err
		}

		history := trip.NewStatusHistory(saved.ID, &from, trip.StatusCompleted)
		return svc.repo.SaveStatusHistory(ctx, history)
	})
	if err != nil {
		return Output{}, err
	}

	tracelog.WithTrace(ctx, svc.logger).Info("trip completed",
		slog.String("trip_id", saved.ID.String()),
	)
	svc.metrics.TripCompleted(ctx)

	return MapToOutput(saved), nil
}
