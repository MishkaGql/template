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
	PositionSaved(ctx context.Context)
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

func (s *Service) Create(ctx context.Context, input Input) (*trip.Trip, error) {
	t, err := trip.New(
		input.UserID,
		input.DriverID,
		input.Price,
		trip.Coordinates(input.StartPoint),
		trip.Coordinates(input.EndPoint),
	)
	if err != nil {
		return nil, err
	}

	var saved *trip.Trip
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		saved, err = s.repo.Create(ctx, t)
		if err != nil {
			return err
		}

		history := &trip.StatusHistory{TripID: saved.ID, ToStatus: saved.Status}
		return s.repo.SaveStatusHistory(ctx, history)
	})
	if err != nil {
		return nil, err
	}

	tracelog.WithTrace(ctx, s.logger).Info("trips created",
		slog.String("trip_id", saved.ID.String()),
		slog.String("driver_id", saved.DriverID.String()),
	)
	s.metrics.TripCreated(ctx)

	return saved, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Finish(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	var saved *trip.Trip

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		t, err := s.repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}

		from := t.Status
		if err := t.Complete(); err != nil {
			return err
		}

		saved, err = s.repo.Update(ctx, t)
		if err != nil {
			return err
		}

		history := &trip.StatusHistory{TripID: saved.ID, FromStatus: &from, ToStatus: saved.Status}
		return s.repo.SaveStatusHistory(ctx, history)
	})
	if err != nil {
		return nil, err
	}

	tracelog.WithTrace(ctx, s.logger).Info("trips completed",
		slog.String("trip_id", saved.ID.String()),
	)
	s.metrics.TripCompleted(ctx)

	return saved, nil
}

func (s *Service) CreateTripPosition(ctx context.Context, id uuid.UUID, input PositionInput) (*trip.Position, error) {
	var saved *trip.Position

	err := s.tx.Do(ctx, func(ctx context.Context) error {
		t, err := s.repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}

		position, err := t.AddPosition(trip.Coordinates(input.Coords), input.RecordedAt)
		if err != nil {
			return err
		}

		saved, err = s.repo.CreateTripPosition(ctx, position)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.metrics.PositionSaved(ctx)
	return saved, nil
}

func (s *Service) ListTripPositions(ctx context.Context, id uuid.UUID) ([]trip.Position, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, err
	}

	pos, err := s.repo.ListTripPositions(ctx, id)
	return pos, err
}
