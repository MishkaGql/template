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

func TestNew(t *testing.T) {
	start, end := validCoordinates()

	tests := []struct {
		name     string
		userID   uuid.UUID
		driverID uuid.UUID
		price    int64
		wantErr  bool
	}{
		{
			name:     "Success",
			userID:   testUserID,
			driverID: testDriverID,
			price:    1450,
		},
		{
			name:     "InvalidRequest_EmptyUserID",
			userID:   uuid.Nil,
			driverID: testDriverID,
			price:    1450,

			wantErr: true,
		},
		{
			name:     "InvalidRequest_EmptyDriverID",
			userID:   testUserID,
			driverID: uuid.Nil,
			price:    1450,

			wantErr: true,
		},
		{
			name:     "InvalidRequest_NegativePrice",
			userID:   testUserID,
			driverID: testDriverID,
			price:    -1,

			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.userID, tt.driverID, tt.price, start, end)

			if tt.wantErr {
				errstest.AssertCode(t, err, errs.InvalidRequest)
				return
			}

			if err != nil {
				t.Fatalf("New(%s, %s, %d, %v, %v): unexpected error: %v", tt.userID, tt.driverID, tt.price, start, end, err)
			}

			want := &Trip{
				UserID:     tt.userID,
				DriverID:   tt.driverID,
				Price:      tt.price,
				Status:     StatusActive,
				StartPoint: start,
				EndPoint:   end,
			}

			if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(Trip{}, "ID", "StartedAt")); diff != "" {
				t.Errorf("New(%s, %s, %d, %v, %v) mismatch (-want +got):\n%s", tt.userID, tt.driverID, tt.price, start, end, diff)
			}

			if got.ID == uuid.Nil {
				t.Errorf("New() ID = %s, want non-zero UUID", got.ID)
			}

			if got.StartedAt.IsZero() {
				t.Errorf("New() StartedAt = %v, want non-zero time", got.StartedAt)
			}
		})
	}
}

func TestNewCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		wantErr   bool
	}{
		{
			name:      "Success",
			latitude:  59.9398,
			longitude: 30.3146,
		},
		{
			name:      "InvalidRequest_LatitudeAboveMax",
			latitude:  90.1,
			longitude: 0,

			wantErr: true,
		},
		{
			name:      "InvalidRequest_LatitudeBelowMin",
			latitude:  -90.1,
			longitude: 0,

			wantErr: true,
		},
		{
			name:      "InvalidRequest_LongitudeAboveMax",
			latitude:  0,
			longitude: 180.1,

			wantErr: true,
		},
		{
			name:      "InvalidRequest_LongitudeBelowMin",
			latitude:  0,
			longitude: -180.1,

			wantErr: true,
		},
		{
			name:     "InvalidRequest_NaNLatitude",
			latitude: math.NaN(),

			wantErr: true,
		},
		{
			name:      "InvalidRequest_NaNLongitude",
			longitude: math.NaN(),

			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewCoordinates(tt.latitude, tt.longitude)

			if tt.wantErr {
				errstest.AssertCode(t, err, errs.InvalidRequest)
				return
			}

			if err != nil {
				t.Fatalf("NewCoordinates(%v, %v): unexpected error: %v", tt.latitude, tt.longitude, err)
			}

			want := Coordinates{
				Latitude:  tt.latitude,
				Longitude: tt.longitude,
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("NewCoordinates(%v, %v) mismatch (-want +got):\n%s", tt.latitude, tt.longitude, diff)
			}
		})
	}
}

func TestComplete(t *testing.T) {
	start, end := validCoordinates()
	tr := mustNewTrip(t, start, end)

	if err := tr.Complete(); err != nil {
		t.Fatalf("Complete(): unexpected error: %v", err)
	}

	if tr.FinishedAt == nil {
		t.Fatal("Complete() FinishedAt = nil, want completion time")
	}

	finishedTr := mustNewTrip(t, start, end)
	finishedTr.Status = StatusCompleted

	err := finishedTr.Complete()

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
