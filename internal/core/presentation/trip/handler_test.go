package trip

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/MishkaGql/template/api"
	applicationTrip "github.com/MishkaGql/template/internal/core/application/trip"
	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

var (
	handlerUserID   = uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c")
	handlerDriverID = uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5")
	handlerTripID   = uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
)

const validBody = `{
	"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
	"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
	"price": 1450,
	"start_point": {"latitude": 59.9398, "longitude": 30.3146},
	"end_point": {"latitude": 59.9290, "longitude": 30.3626}
}`

func TestCreateTrip_Success(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().Create(gomock.Any(), expectedInput()).Return(handlerOutput(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validBody))

	handler.CreateTrip(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusCreated {
		t.Fatalf("статус: got %d, want %d", rec.Code, http.StatusCreated)
	}

	wantLocation := "/api/v1/trips/" + handlerTripID.String()
	if got := rec.Header().Get("Location"); got != wantLocation {
		t.Errorf("Location = %q, want %q", got, wantLocation)
	}
}

func TestCreateTrip_InvalidRequest_IncorrectBody(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(`{"price": "дорого"}`))

	handler.CreateTrip(rec, req, api.CreateTripParams{})

	assertProblem(t, rec, http.StatusBadRequest, errs.InvalidRequest)
}

func TestCreateTrip_DriverBusy_BusyDriver(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().
		Create(gomock.Any(), expectedInput()).
		Return(nil, errs.New(trip.CodeDriverBusy, "Driver already has an active trips", nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validBody))

	handler.CreateTrip(rec, req, api.CreateTripParams{})

	assertProblem(t, rec, http.StatusConflict, trip.CodeDriverBusy)
}

func TestCreateTrip_InternalError_DetailNotExposed(t *testing.T) {
	cause := errors.New("insert failed: password=secret")
	tests := []struct {
		name string
		err  error
	}{
		{"internal error", errs.NewInternal(cause)},
		{"unmapped code", errs.New("unknown_code", cause.Error(), nil)},
		{"unexpected error", cause},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, service := newTestHandler(t)
			service.EXPECT().Create(gomock.Any(), expectedInput()).Return(nil, tt.err)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validBody))
			handler.CreateTrip(rec, req, api.CreateTripParams{})

			assertProblem(t, rec, http.StatusInternalServerError, errs.InternalError)
			if strings.Contains(rec.Body.String(), "password=secret") {
				t.Errorf("response exposes internal cause: %s", rec.Body)
			}
		})
	}
}

func TestGetTrip_Success(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().Get(gomock.Any(), handlerTripID).Return(handlerOutput(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+handlerTripID.String(), nil)

	handler.GetTrip(rec, req, handlerTripID)

	if rec.Code != http.StatusOK {
		t.Fatalf("статус: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGetTrip_TripNotFound_UnknownTripID(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().
		Get(gomock.Any(), handlerTripID).
		Return(nil, errs.New(trip.CodeTripNotFound, "Trip was not found", nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+handlerTripID.String(), nil)

	handler.GetTrip(rec, req, handlerTripID)

	assertProblem(t, rec, http.StatusNotFound, trip.CodeTripNotFound)
}

func TestFinishTrip_Success(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().Finish(gomock.Any(), handlerTripID).Return(handlerFinishedOutput(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+handlerTripID.String()+"/finish", nil)

	handler.FinishTrip(rec, req, handlerTripID)

	if rec.Code != http.StatusOK {
		t.Fatalf("статус: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestFinishTrip_TripCompleted_AlreadyFinished(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().
		Finish(gomock.Any(), handlerTripID).
		Return(nil, errs.New(trip.CodeTripAlreadyCompleted, "Operation is not allowed", nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+handlerTripID.String()+"/finish", nil)

	handler.FinishTrip(rec, req, handlerTripID)

	assertProblem(t, rec, http.StatusConflict, trip.CodeTripAlreadyCompleted)
}

func TestFinishTrip_TripNotFound_UnknownTripID(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().
		Finish(gomock.Any(), handlerTripID).
		Return(nil, errs.New(trip.CodeTripNotFound, "Trip was not found", nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+handlerTripID.String()+"/finish", nil)

	handler.FinishTrip(rec, req, handlerTripID)

	assertProblem(t, rec, http.StatusNotFound, trip.CodeTripNotFound)
}

func newTestHandler(t *testing.T) (*Handler, *MockService) {
	t.Helper()

	ctrl := gomock.NewController(t)

	service := NewMockService(ctrl)

	return NewHandler(service, NewMockPinger(ctrl), slog.New(slog.DiscardHandler), func(uuid.UUID) {}), service
}

func expectedInput() applicationTrip.Input {
	return applicationTrip.Input{
		UserID:   handlerUserID,
		DriverID: handlerDriverID,
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

func handlerOutput() *trip.Trip {
	input := expectedInput()
	return &trip.Trip{
		ID:         handlerTripID,
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		Price:      input.Price,
		Status:     trip.StatusActive,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
	}
}

func handlerFinishedOutput() *trip.Trip {
	out := handlerOutput()
	out.Status = trip.StatusCompleted

	return out
}

func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode errs.Code) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("статус: got %d, want %d (body: %s)", rec.Code, wantStatus, rec.Body.String())
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("content-type: got %q", ct)
	}

	var body api.Problem
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("тело не является Problem Details: %v", err)
	}

	if body.Code != string(wantCode) {
		t.Errorf("code: got %q, want %q", body.Code, string(wantCode))
	}

	if int(body.Status) != wantStatus {
		t.Errorf("status в теле: got %d, want %d", body.Status, wantStatus)
	}
}
