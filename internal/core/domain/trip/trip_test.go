package trip

import (
	"testing"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs/errstest"
)

var (
	testUserID   = uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c")
	testDriverID = uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5")
)

func TestNewCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		wantErr   bool
	}{
		{name: "No_CorrectData", latitude: 59.9398, longitude: 30.3146},
		{name: "No_LatitudeUpperBoundary", latitude: 90, longitude: 0},
		{name: "No_LongitudeUpperBoundary", latitude: 0, longitude: 180},
		{name: "CodeInvalidRequest_LatitudeAboveMax", latitude: 90.1, longitude: 0, wantErr: true},
		{name: "CodeInvalidRequest_LatitudeBelowMin", latitude: -90.1, longitude: 0, wantErr: true},
		{name: "CodeInvalidRequest_LongitudeAboveMax", latitude: 0, longitude: 180.1, wantErr: true},
		{name: "CodeInvalidRequest_LongitudeBelowMin", latitude: 0, longitude: -180.1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCoordinates(tt.latitude, tt.longitude)

			if tt.wantErr {
				errstest.AssertCode(t, err, errs.InvalidRequest)
				return
			}

			if err != nil {
				t.Fatalf("не ожидали ошибку: %v", err)
			}

			if got.Latitude != tt.latitude || got.Longitude != tt.longitude {
				t.Errorf("координаты не совпали: got %+v", got)
			}
		})
	}
}

func TestNew(t *testing.T) {
	start, end := validCoordinates()

	tests := []struct {
		name     string
		userID   uuid.UUID
		driverID uuid.UUID
		price    int64
		wantErr  bool
	}{
		{name: "No_CorrectData", userID: testUserID, driverID: testDriverID, price: 1450},
		{name: "No_ZeroPrice", userID: testUserID, driverID: testDriverID, price: 0},
		{name: "CodeInvalidRequest_EmptyUserId", userID: uuid.Nil, driverID: testDriverID, price: 1450, wantErr: true},
		{name: "CodeInvalidRequest_EmptyDriverId", userID: testUserID, driverID: uuid.Nil, price: 1450, wantErr: true},
		{name: "CodeInvalidRequest_NegativePrice", userID: testUserID, driverID: testDriverID, price: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.userID, tt.driverID, tt.price, start, end)

			if tt.wantErr {
				errstest.AssertCode(t, err, errs.InvalidRequest)
				return
			}

			if err != nil {
				t.Fatalf("не ожидали ошибку: %v", err)
			}

			if got.ID == uuid.Nil {
				t.Error("id не сгенерирован")
			}
			if got.Status != StatusActive {
				t.Errorf("статус: got %q, want %q", got.Status, StatusActive)
			}
			if got.FinishedAt != nil {
				t.Error("finished_at должен быть nil у активной поездки")
			}
			if got.StartedAt.IsZero() {
				t.Error("started_at не проставлен")
			}
		})
	}
}

func TestComplete(t *testing.T) {
	start, end := validCoordinates()

	tests := []struct {
		name     string
		prepare  func(t *testing.T) *Trip
		wantCode errs.ErrorCode
	}{
		{
			name: "No_ActiveTrip",
			prepare: func(t *testing.T) *Trip {
				return newTrip(t, start, end)
			},
		},
		{
			name: "CodeTripAlreadyCompleted_AlreadyCompletedTrip",
			prepare: func(t *testing.T) *Trip {
				tr := newTrip(t, start, end)
				if err := tr.Complete(); err != nil {
					t.Fatalf("первое завершение: %v", err)
				}

				return tr
			},
			wantCode: CodeTripAlreadyCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := tt.prepare(t)

			err := tr.Complete()

			if tt.wantCode != "" {
				errstest.AssertCode(t, err, tt.wantCode)
				return
			}

			if err != nil {
				t.Fatalf("не ожидали ошибку: %v", err)
			}

			if tr.Status != StatusCompleted {
				t.Errorf("статус: got %q, want %q", tr.Status, StatusCompleted)
			}
			if tr.FinishedAt == nil {
				t.Fatal("finished_at не проставлен")
			}
		})
	}
}

func TestNewStatusHistory(t *testing.T) {
	tripID := uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	from := StatusActive

	tests := []struct {
		name string
		from *Status
		to   Status
	}{
		{name: "No_TripCreated", from: nil, to: StatusActive},
		{name: "No_TripCompleted", from: &from, to: StatusCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := NewStatusHistory(tripID, tt.from, tt.to)

			if history.TripID != tripID {
				t.Errorf("trip_id: got %v, want %v", history.TripID, tripID)
			}
			if history.ToStatus != tt.to {
				t.Errorf("to_status: got %q, want %q", history.ToStatus, tt.to)
			}
		})
	}
}

func validCoordinates() (Coordinates, Coordinates) {
	return Coordinates{Latitude: 59.9398, Longitude: 30.3146},
		Coordinates{Latitude: 59.9290, Longitude: 30.3626}
}

func newTrip(t *testing.T, start, end Coordinates) *Trip {
	t.Helper()

	tr, err := New(testUserID, testDriverID, 1450, start, end)
	if err != nil {
		t.Fatalf("подготовка поездки: %v", err)
	}

	return tr
}
