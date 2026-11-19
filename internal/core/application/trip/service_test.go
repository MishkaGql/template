package trip

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs/errstest"
)

var (
	testUserID   = uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c")
	testDriverID = uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5")
	testTripID   = uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
)

func TestCreate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := t.Context()
	tx, txCtx := expectTransaction(t, ctrl)

	want := activeTrip()
	repo := NewMockRepository(ctrl)

	gomock.InOrder(
		repo.EXPECT().Create(txCtx, gomock.Any()).Return(want, nil),
		repo.EXPECT().SaveStatusHistory(txCtx, &trip.StatusHistory{
			TripID:   testTripID,
			ToStatus: trip.StatusActive,
		}).Return(nil),
	)

	metrics := &metricsRecorder{}
	svc := NewService(repo, tx, testLogger(), metrics)

	out, err := svc.Create(ctx, validInput())
	if err != nil {
		t.Fatalf("не ожидали ошибку: %v", err)
	}

	if out != want {
		t.Errorf("Create() = %p, want repository trips %p", out, want)
	}

	if metrics.created != 1 {
		t.Errorf("Create() counter = %d, want 1 after successful transaction", metrics.created)
	}
}

func TestCreate_InternalError_CommitFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(activeTrip(), nil)
	repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any()).Return(nil)

	tx := NewMockTxManager(ctrl)
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			if err := fn(ctx); err != nil {
				return err
			}

			return errs.NewInternal(errors.New("commit unavailable"))
		},
	)

	metrics := &metricsRecorder{}
	svc := NewService(repo, tx, testLogger(), metrics)

	_, err := svc.Create(t.Context(), validInput())

	errstest.AssertCode(t, err, errs.InternalError)

	if metrics.created != 0 {
		t.Errorf("Create() counter = %d, want 0 after commit failure", metrics.created)
	}
}

func TestGet_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().Get(gomock.Any(), testTripID).Return(activeTrip(), nil)

	svc := NewService(repo, NewMockTxManager(ctrl), testLogger(), &metricsRecorder{})

	out, err := svc.Get(t.Context(), testTripID)
	if err != nil {
		t.Fatalf("не ожидали ошибку: %v", err)
	}

	if out.ID != testTripID {
		t.Errorf("id: got %v, want %v", out.ID, testTripID)
	}
}

func TestGet_TripNotFound_UnknownTripID(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().
		Get(gomock.Any(), testTripID).
		Return(nil, errs.New(trip.CodeTripNotFound, "Trip was not found", nil))

	svc := NewService(repo, NewMockTxManager(ctrl), testLogger(), &metricsRecorder{})

	_, err := svc.Get(t.Context(), testTripID)

	errstest.AssertCode(t, err, trip.CodeTripNotFound)
}

func TestFinish_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := t.Context()
	tx, txCtx := expectTransaction(t, ctrl)

	current := activeTrip()
	from := trip.StatusActive
	repo := NewMockRepository(ctrl)

	gomock.InOrder(
		repo.EXPECT().GetForUpdate(txCtx, testTripID).Return(current, nil),
		repo.EXPECT().Update(txCtx, current).Return(current, nil),
		repo.EXPECT().SaveStatusHistory(txCtx, &trip.StatusHistory{
			TripID:     testTripID,
			FromStatus: &from,
			ToStatus:   trip.StatusCompleted,
		}).Return(nil),
	)

	metrics := &metricsRecorder{}
	svc := NewService(repo, tx, testLogger(), metrics)

	out, err := svc.Finish(ctx, testTripID)
	if err != nil {
		t.Fatalf("не ожидали ошибку: %v", err)
	}

	if out != current {
		t.Errorf("Finish() = %p, want repository trips %p", out, current)
	}

	if metrics.completed != 1 {
		t.Errorf("Finish() counter = %d, want 1 after successful transaction", metrics.completed)
	}
}

func TestFinish_InternalError_HistorySaveFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().GetForUpdate(gomock.Any(), testTripID).Return(activeTrip(), nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(completedTrip(), nil)
	repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any()).
		Return(errs.NewInternal(errors.New("history unavailable")))

	metrics := &metricsRecorder{}
	tx, _ := expectTransaction(t, ctrl)
	svc := NewService(repo, tx, testLogger(), metrics)

	_, err := svc.Finish(t.Context(), testTripID)

	errstest.AssertCode(t, err, errs.InternalError)

	if metrics.completed != 0 {
		t.Errorf("Finish() counter = %d, want 0 after rollback", metrics.completed)
	}
}

func validInput() Input {
	return Input{
		UserID:   testUserID,
		DriverID: testDriverID,
		Price:    1450,
		StartPoint: trip.Coordinates{
			Latitude:  59.9398,
			Longitude: 30.3146,
		},
		EndPoint: trip.Coordinates{
			Latitude:  59.9290,
			Longitude: 30.3626,
		},
	}
}

func activeTrip() *trip.Trip {
	input := validInput()
	return &trip.Trip{
		ID:         testTripID,
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		Price:      input.Price,
		Status:     trip.StatusActive,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
	}
}

func completedTrip() *trip.Trip {
	t := activeTrip()

	t.Status = trip.StatusCompleted

	return t
}

func expectTransaction(t *testing.T, ctrl *gomock.Controller) (*MockTxManager, context.Context) {
	t.Helper()

	ctx := t.Context()
	txCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	tx := NewMockTxManager(ctrl)
	tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
		return fn(txCtx)
	})
	return tx, txCtx
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

type metricsRecorder struct{ created, completed int }

func (m *metricsRecorder) TripCreated(context.Context)   { m.created++ }
func (m *metricsRecorder) TripCompleted(context.Context) { m.completed++ }
func (*metricsRecorder) PositionSaved(context.Context)   {}
