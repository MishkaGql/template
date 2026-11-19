package trip

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs/errstest"
)

var (
	testUserID   = uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c")
	testDriverID = uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5")
)

func TestNew_Success(t *testing.T) {
	start, end := validCoordinates()
	got := mustNewTrip(t, start, end)

	want := &Trip{
		UserID:     testUserID,
		DriverID:   testDriverID,
		Price:      1450,
		Status:     StatusActive,
		StartPoint: start,
		EndPoint:   end,
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Trip{}, "ID", "StartedAt")); diff != "" {
		t.Errorf("New() mismatch (-want +got):\n%s", diff)
	}
	if got.ID == uuid.Nil || got.StartedAt.IsZero() {
		t.Errorf("New() missing ID or start time: id=%v started_at=%v", got.ID, got.StartedAt)
	}
}

func TestNew_InvalidRequest(t *testing.T) {
	start, end := validCoordinates()
	tests := []struct {
		name             string
		userID, driverID uuid.UUID
		price            int64
	}{
		{"empty user ID", uuid.Nil, testDriverID, 1450},
		{"empty driver ID", testUserID, uuid.Nil, 1450},
		{"negative price", testUserID, testDriverID, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.userID, tt.driverID, tt.price, start, end)
			errstest.AssertCode(t, err, errs.InvalidRequest)
		})
	}
}

func TestNewCoordinates_Success(t *testing.T) {
	want := Coordinates{Latitude: 59.9398, Longitude: 30.3146}
	got, err := NewCoordinates(want.Latitude, want.Longitude)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("NewCoordinates() = %+v, want %+v", got, want)
	}
}

func TestNewCoordinates_InvalidRequest(t *testing.T) {
	tests := []struct {
		name        string
		coordinates Coordinates
	}{
		{"latitude above 90", Coordinates{Latitude: 90.1}},
		{"latitude below -90", Coordinates{Latitude: -90.1}},
		{"longitude above 180", Coordinates{Longitude: 180.1}},
		{"longitude below -180", Coordinates{Longitude: -180.1}},
		{"NaN latitude", Coordinates{Latitude: math.NaN()}},
		{"NaN longitude", Coordinates{Longitude: math.NaN()}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewCoordinates(tt.coordinates.Latitude, tt.coordinates.Longitude)
			errstest.AssertCode(t, err, errs.InvalidRequest)
		})
	}
}

func TestComplete(t *testing.T) {
	start, end := validCoordinates()
	tr := mustNewTrip(t, start, end)

	if err := tr.Complete(); err != nil {
		t.Fatalf("Complete(): unexpected error: %v", err)
	}
	if tr.Status != StatusCompleted || tr.FinishedAt == nil {
		t.Fatalf("Complete() status=%q finished_at=%v, want a completed trip with a finish time", tr.Status, tr.FinishedAt)
	}

	err := tr.Complete()
	errstest.AssertCode(t, err, CodeTripAlreadyCompleted)
}

func validCoordinates() (Coordinates, Coordinates) {
	return Coordinates{Latitude: 59.9398, Longitude: 30.3146},
		Coordinates{Latitude: 59.9290, Longitude: 30.3626}
}

func mustNewTrip(t *testing.T, start, end Coordinates) *Trip {
	t.Helper()

	tr, err := New(testUserID, testDriverID, 1450, start, end)
	if err != nil {
		t.Fatalf("New(%s, %s, %d, %v, %v): unexpected error: %v", testUserID, testDriverID, 1450, start, end, err)
	}

	return tr
}
