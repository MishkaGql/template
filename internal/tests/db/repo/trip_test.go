//go:build integration

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/infrastructure/db"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs/errstest"
)

const (
	startLatitude   float64 = 59.9398
	startLongtitude float64 = 30.3146

	endLatitude   float64 = 59.9290
	endLongtitude float64 = 30.3626

	price int64 = 1450
)

var cmpOpts = cmpopts.EquateApproxTime(time.Second)

func TestTripRepository(t *testing.T) {
	suite.Run(t, new(BaseSuite))
}

func (s *BaseSuite) TestCreate_No_CorrectData() {
	ctx := s.Context()
	tr := newTrip(s.T(), uuid.New())

	saved, err := s.TripRepo.Create(ctx, tr)
	require.NoError(s.T(), err)

	got, err := s.TripRepo.Get(ctx, tr.ID)
	require.NoError(s.T(), err)

	if diff := cmp.Diff(tr, saved, cmpOpts); diff != "" {
		s.T().Fatalf("mismatch domain and saved (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(tr, got, cmpOpts); diff != "" {
		s.T().Fatalf("mismatch domain and get (-want +got):\n%s", diff)
	}
}

func (s *BaseSuite) TestCreate_CodeDriverBusy_AlreadyCreated() {
	ctx := s.Context()
	driverID := uuid.New()

	first := newTrip(s.T(), driverID)
	second := newTrip(s.T(), driverID)

	_, err := s.TripRepo.Create(ctx, first)
	require.NoError(s.T(), err)

	_, err = s.TripRepo.Create(ctx, second)
	errstest.AssertCode(s.T(), err, trip.CodeDriverBusy)

	_, err = s.TripRepo.Get(ctx, second.ID)
	errstest.AssertCode(s.T(), err, trip.CodeTripNotFound)

	gotFirst, err := s.TripRepo.Get(ctx, first.ID)
	require.NoError(s.T(), err)

	if diff := cmp.Diff(first, gotFirst, cmpOpts); diff != "" {
		s.T().Fatalf("mismatch first trip (-want +got):\n%s", diff)
	}
}

func (s *BaseSuite) TestGet_CodeTripNotFound_UnknownId() {
	_, err := s.TripRepo.Get(s.Context(), uuid.New())
	errstest.AssertCode(s.T(), err, trip.CodeTripNotFound)
}

func (s *BaseSuite) TestUpdate_CodeTripNotFound_UnknownId() {
	_, err := s.TripRepo.Update(s.Context(), newTrip(s.T(), uuid.New()))
	errstest.AssertCode(s.T(), err, trip.CodeTripNotFound)
}

func (s *BaseSuite) TestFinish_No_CorrectId() {
	ctx := s.Context()
	tr := newTrip(s.T(), uuid.New())

	_, err := s.TripRepo.Create(ctx, tr)
	require.NoError(s.T(), err)

	txManager := db.NewPgxTxManager(s.Pool)
	var saved *trip.Trip

	err = txManager.Do(ctx, func(ctx context.Context) error {
		current, getErr := s.TripRepo.GetForUpdate(ctx, tr.ID)
		if getErr != nil {
			return getErr
		}

		from := current.Status
		if completeErr := current.Complete(); completeErr != nil {
			return completeErr
		}

		var updateErr error
		saved, updateErr = s.TripRepo.Update(ctx, current)
		if updateErr != nil {
			return updateErr
		}

		return s.TripRepo.SaveStatusHistory(ctx, trip.NewStatusHistory(saved.ID, &from, trip.StatusCompleted))
	})
	require.NoError(s.T(), err)

	got, err := s.TripRepo.Get(ctx, tr.ID)
	require.NoError(s.T(), err)

	require.Equal(s.T(), trip.StatusCompleted, got.Status)
	require.NotNil(s.T(), got.FinishedAt)

	if diff := cmp.Diff(saved, got, cmpOpts); diff != "" {
		s.T().Fatalf("mismatch saved and got (-want +got):\n%s", diff)
	}

	assert.Equal(s.T(), 1, countHistory(s.T(), s.Pool, tr.ID))
}

func (s *BaseSuite) TestTransactionRollback_CodeInternalError_InvalidStatus() {
	ctx := s.Context()
	tr := newTrip(s.T(), uuid.New())

	txManager := db.NewPgxTxManager(s.Pool)

	err := txManager.Do(ctx, func(ctx context.Context) error {
		if _, createErr := s.TripRepo.Create(ctx, tr); createErr != nil {
			return createErr
		}
		const unknownStatus = "unluck"
		return s.TripRepo.SaveStatusHistory(ctx, trip.NewStatusHistory(tr.ID, nil, trip.Status(unknownStatus)))
	})
	errstest.AssertCode(s.T(), err, errs.InternalError)

	_, err = s.TripRepo.Get(ctx, tr.ID)
	errstest.AssertCode(s.T(), err, trip.CodeTripNotFound)

	assert.Equal(s.T(), 0, countHistory(s.T(), s.Pool, tr.ID), "INSERT в trip_status_history должен откатиться")
}

func (s *BaseSuite) TestSaveStatusHistory_No_CorrectTrip() {
	ctx := s.Context()
	tr := newTrip(s.T(), uuid.New())

	_, err := s.TripRepo.Create(ctx, tr)
	require.NoError(s.T(), err)

	err = s.TripRepo.SaveStatusHistory(ctx, trip.NewStatusHistory(tr.ID, nil, trip.StatusActive))
	require.NoError(s.T(), err)

	assert.Equal(s.T(), 1, countHistory(s.T(), s.Pool, tr.ID))
}

func newTrip(t require.TestingT, driverID uuid.UUID) *trip.Trip {
	start, err := trip.NewCoordinates(startLatitude, startLongtitude)
	require.NoError(t, err)

	end, err := trip.NewCoordinates(endLatitude, endLongtitude)
	require.NoError(t, err)

	tr, err := trip.New(uuid.New(), driverID, price, start, end)
	require.NoError(t, err)

	return tr
}

// countHistory оставляем, так как в репозитории может отсутствовать публичный метод для получения истории статусов.
func countHistory(t require.TestingT, pool *pgxpool.Pool, tripID uuid.UUID) int {
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM trip_status_history WHERE trip_id = $1`, tripID).Scan(&count)
	require.NoError(t, err)

	return count
}
