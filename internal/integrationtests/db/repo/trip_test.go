package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/infrastructure/db"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs/errstest"
)

var timeComparison = cmpopts.EquateApproxTime(time.Microsecond)

func TestCreate_Success(t *testing.T) {
	repo := mustNewRepository(t)
	want := mustNewTrip(t, uuid.New())

	got, err := repo.Create(t.Context(), want)
	if err != nil {
		t.Fatalf("Create(): unexpected error: %v", err)
	}

	if diff := cmp.Diff(want, got, timeComparison); diff != "" {
		t.Errorf("Get() mismatch (-want +got):\n%s", diff)
	}
}

func TestCreate_DriverBusy_ActiveTripExists(t *testing.T) {
	repo := mustNewRepository(t)
	first := mustNewTrip(t, uuid.New())
	second := mustNewTrip(t, first.DriverID)

	if _, err := repo.Create(t.Context(), first); err != nil {
		t.Fatalf("Create(first): unexpected error: %v", err)
	}

	_, err := repo.Create(t.Context(), second)
	errstest.AssertCode(t, err, trip.CodeDriverBusy)

	_, err = repo.Get(t.Context(), second.ID)
	errstest.AssertCode(t, err, trip.CodeTripNotFound)
}

func TestCreate_InternalError_DuplicateTripID(t *testing.T) {
	repo := mustNewRepository(t)
	first := mustNewTrip(t, uuid.New())
	second := mustNewTrip(t, uuid.New())
	second.ID = first.ID

	if _, err := repo.Create(t.Context(), first); err != nil {
		t.Fatalf("Create(first): unexpected error: %v", err)
	}

	_, err := repo.Create(t.Context(), second)
	errstest.AssertCode(t, err, errs.InternalError)
}

func TestGet_TripNotFound_UnknownTripID(t *testing.T) {
	repo := mustNewRepository(t)
	id := uuid.New()

	_, err := repo.Get(t.Context(), id)

	errstest.AssertCode(t, err, trip.CodeTripNotFound)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("Get(%s) cause = %v, want pgx.ErrNoRows", id, err)
	}
}

func TestUpdate_TripNotFound_UnknownTripID(t *testing.T) {
	repo := mustNewRepository(t)
	tr := mustNewTrip(t, uuid.New())

	_, err := repo.Update(t.Context(), tr)

	errstest.AssertCode(t, err, trip.CodeTripNotFound)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("Update(%s) cause = %v, want pgx.ErrNoRows", tr.ID, err)
	}
}

func TestGetForUpdate_InternalError_MissingTransaction(t *testing.T) {
	repo := mustNewRepository(t)

	_, err := repo.GetForUpdate(t.Context(), uuid.New())

	errstest.AssertCode(t, err, errs.InternalError)
}

func TestTransaction_InternalError_HistorySaveFailed(t *testing.T) {
	repo := mustNewRepository(t)
	tx := db.NewPgxTxManager(pool)
	tr := mustNewTrip(t, uuid.New())

	err := tx.Do(t.Context(), func(ctx context.Context) error {
		if _, err := repo.Create(ctx, tr); err != nil {
			return err
		}

		return repo.SaveStatusHistory(ctx, &trip.StatusHistory{
			TripID:   tr.ID,
			ToStatus: "unknown",
		})
	})
	errstest.AssertCode(t, err, errs.InternalError)

	_, err = repo.Get(t.Context(), tr.ID)
	errstest.AssertCode(t, err, trip.CodeTripNotFound)

	if got := mustCountHistory(t, tr.ID); got != 0 {
		t.Errorf("Do() history count = %d, want 0 after rollback", got)
	}
}

func TestTransaction_Success_RollbackAfterPanic(t *testing.T) {
	repo := mustNewRepository(t)
	tr := mustNewTrip(t, uuid.New())
	tx := db.NewPgxTxManager(pool)

	func() {
		defer func() {
			if got := recover(); got != "abort transaction" {
				t.Errorf("Do() panic = %v, want abort transaction", got)
			}
		}()

		if err := tx.Do(t.Context(), func(ctx context.Context) error {
			if _, err := repo.Create(ctx, tr); err != nil {
				return err
			}

			panic("abort transaction")
		}); err != nil {
			t.Fatalf("Do() before panic: unexpected error: %v", err)
		}
	}()

	_, err := repo.Get(t.Context(), tr.ID)

	errstest.AssertCode(t, err, trip.CodeTripNotFound)
}

func TestSaveStatusHistory_Success(t *testing.T) {
	repo := mustNewRepository(t)
	tr := mustNewTrip(t, uuid.New())

	if _, err := repo.Create(t.Context(), tr); err != nil {
		t.Fatalf("Create(): unexpected error: %v", err)
	}

	history := &trip.StatusHistory{
		TripID:   tr.ID,
		ToStatus: trip.StatusActive,
	}

	if err := repo.SaveStatusHistory(t.Context(), history); err != nil {
		t.Fatalf("SaveStatusHistory(): unexpected error: %v", err)
	}

	if got := mustCountHistory(t, tr.ID); got != 1 {
		t.Errorf("SaveStatusHistory() count = %d, want 1", got)
	}
}
