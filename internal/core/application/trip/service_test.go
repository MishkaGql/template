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

func TestCreate_No_CorrectData(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(activeTrip(), nil)
	repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any()).Return(nil)

	svc := NewService(repo, txPassthrough(ctrl), testLogger(), testMetrics())

	out, err := svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatalf("не ожидали ошибку: %v", err)
	}

	if out.Status != Status(trip.StatusActive) {
		t.Errorf("статус: got %q, want %q", out.Status, trip.StatusActive)
	}
}

func TestCreate_CodeDriverBusy_BusyDriver(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil, errs.NewServiceError(trip.CodeDriverBusy, "Driver already has an active trip"))
	repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any()).Times(0)

	svc := NewService(repo, txPassthrough(ctrl), testLogger(), testMetrics())

	_, err := svc.Create(context.Background(), validInput())

	errstest.AssertCode(t, err, trip.CodeDriverBusy)
}

func TestCreate_CodeInternalError_HistorySaveFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(activeTrip(), nil)
	repo.EXPECT().
		SaveStatusHistory(gomock.Any(), gomock.Any()).
		Return(errs.NewInternalServiceError(errors.New("insert into trip_status_history failed")))

	svc := NewService(repo, txPassthrough(ctrl), testLogger(), testMetrics())

	_, err := svc.Create(context.Background(), validInput())

	errstest.AssertCode(t, err, errs.InternalError)
}

func TestCreate_CodeInvalidRequest_IncorrectCoordinates(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)
	tx := NewMockTxManager(ctrl)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	input := validInput()
	input.StartPoint.Latitude = 91

	svc := NewService(repo, tx, testLogger(), testMetrics())

	_, err := svc.Create(context.Background(), input)

	errstest.AssertCode(t, err, errs.InvalidRequest)
}

func TestCreate_CodeInvalidRequest_EmptyUserId(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)
	tx := NewMockTxManager(ctrl)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	input := validInput()
	input.UserID = uuid.Nil

	svc := NewService(repo, tx, testLogger(), testMetrics())

	_, err := svc.Create(context.Background(), input)

	errstest.AssertCode(t, err, errs.InvalidRequest)
}

func TestGet_No_CorrectId(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().Get(gomock.Any(), testTripID).Return(activeTrip(), nil)

	svc := NewService(repo, NewMockTxManager(ctrl), testLogger(), testMetrics())

	out, err := svc.Get(context.Background(), testTripID)
	if err != nil {
		t.Fatalf("не ожидали ошибку: %v", err)
	}

	if out.ID != testTripID {
		t.Errorf("id: got %v, want %v", out.ID, testTripID)
	}
}

func TestGet_CodeTripNotFound_UnknownId(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().
		Get(gomock.Any(), testTripID).
		Return(nil, errs.NewServiceError(trip.CodeTripNotFound, "Trip was not found"))

	svc := NewService(repo, NewMockTxManager(ctrl), testLogger(), testMetrics())

	_, err := svc.Get(context.Background(), testTripID)

	errstest.AssertCode(t, err, trip.CodeTripNotFound)
}

func TestFinish_No_CorrectId(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().GetForUpdate(gomock.Any(), testTripID).Return(activeTrip(), nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(completedTrip(), nil)
	repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any()).Return(nil)

	svc := NewService(repo, txPassthrough(ctrl), testLogger(), testMetrics())

	out, err := svc.Finish(context.Background(), testTripID)
	if err != nil {
		t.Fatalf("не ожидали ошибку: %v", err)
	}

	if out.Status != Status(trip.StatusCompleted) {
		t.Errorf("статус: got %q, want %q", out.Status, trip.StatusCompleted)
	}
}

func TestFinish_CodeTripAlreadyCompleted_CompletedTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().GetForUpdate(gomock.Any(), testTripID).Return(completedTrip(), nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
	repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any()).Times(0)

	svc := NewService(repo, txPassthrough(ctrl), testLogger(), testMetrics())

	_, err := svc.Finish(context.Background(), testTripID)

	errstest.AssertCode(t, err, trip.CodeTripAlreadyCompleted)
}

func TestFinish_CodeTripNotFound_UnknownId(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().
		GetForUpdate(gomock.Any(), testTripID).
		Return(nil, errs.NewServiceError(trip.CodeTripNotFound, "Trip was not found"))
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	svc := NewService(repo, txPassthrough(ctrl), testLogger(), testMetrics())

	_, err := svc.Finish(context.Background(), testTripID)

	errstest.AssertCode(t, err, trip.CodeTripNotFound)
}

func TestFinish_CodeInternalError_UpdateFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)

	repo.EXPECT().GetForUpdate(gomock.Any(), testTripID).Return(activeTrip(), nil)
	repo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil, errs.NewInternalServiceError(errors.New("update trips failed")))
	repo.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any()).Times(0)

	svc := NewService(repo, txPassthrough(ctrl), testLogger(), testMetrics())

	_, err := svc.Finish(context.Background(), testTripID)

	errstest.AssertCode(t, err, errs.InternalError)
}

func validInput() Input {
	return Input{
		UserID:   testUserID,
		DriverID: testDriverID,
		Price:    1450,
		StartPoint: Coordinates{
			Latitude:  59.9398,
			Longitude: 30.3146,
		},
		EndPoint: Coordinates{
			Latitude:  59.9290,
			Longitude: 30.3626,
		},
	}
}

func activeTrip() *trip.Trip {
	return &trip.Trip{
		ID:       testTripID,
		UserID:   testUserID,
		DriverID: testDriverID,
		Price:    1450,
		Status:   trip.StatusActive,
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

func completedTrip() *trip.Trip {
	t := activeTrip()

	t.Status = trip.StatusCompleted

	return t
}

func txPassthrough(ctrl *gomock.Controller) *MockTxManager {
	tx := NewMockTxManager(ctrl)
	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	return tx
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func testMetrics() BusinessMetrics {
	return noopMetrics{}
}

type noopMetrics struct{}

func (noopMetrics) TripCreated(context.Context)   {}
func (noopMetrics) TripCompleted(context.Context) {}
