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

func TestCreateTrip_No_CorrectData(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().Create(gomock.Any(), expectedInput()).Return(handlerOutput(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validBody))

	handler.CreateTrip(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusCreated {
		t.Fatalf("статус: got %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestCreateTrip_No_LocationWithTripId(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().Create(gomock.Any(), expectedInput()).Return(handlerOutput(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validBody))

	handler.CreateTrip(rec, req, api.CreateTripParams{})

	location := rec.Header().Get("Location")
	if !strings.HasSuffix(location, "/"+handlerTripID.String()) {
		t.Errorf("Location: got %q, want суффикс %q", location, handlerTripID.String())
	}
}

func TestCreateTrip_CodeInvalidRequest_IncorrectBody(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(`{"price": "дорого"}`))

	handler.CreateTrip(rec, req, api.CreateTripParams{})

	assertProblem(t, rec, http.StatusBadRequest, errs.InvalidRequest)
}

func TestCreateTrip_CodeDriverBusy_BusyDriver(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().
		Create(gomock.Any(), expectedInput()).
		Return(applicationTrip.Output{}, errs.NewServiceError(trip.CodeDriverBusy, "Driver already has an active trip"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validBody))

	handler.CreateTrip(rec, req, api.CreateTripParams{})

	assertProblem(t, rec, http.StatusConflict, trip.CodeDriverBusy)
}

func TestCreateTrip_CodeInternalError_DetailNotExposed(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().
		Create(gomock.Any(), expectedInput()).
		Return(applicationTrip.Output{}, errs.NewInternalServiceError(errors.New("insert failed: password=secret")))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(validBody))

	handler.CreateTrip(rec, req, api.CreateTripParams{})

	if strings.Contains(rec.Body.String(), "password=secret") {
		t.Errorf("тело ответа раскрывает детали внутренней ошибки: %s", rec.Body.String())
	}
}

func TestGetTrip_No_CorrectId(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().Get(gomock.Any(), handlerTripID).Return(handlerOutput(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+handlerTripID.String(), nil)

	handler.GetTrip(rec, req, handlerTripID)

	if rec.Code != http.StatusOK {
		t.Fatalf("статус: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGetTrip_CodeTripNotFound_UnknownId(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().
		Get(gomock.Any(), handlerTripID).
		Return(applicationTrip.Output{}, errs.NewServiceError(trip.CodeTripNotFound, "Trip was not found"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+handlerTripID.String(), nil)

	handler.GetTrip(rec, req, handlerTripID)

	assertProblem(t, rec, http.StatusNotFound, trip.CodeTripNotFound)
}

func TestFinishTrip_No_CorrectId(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().Finish(gomock.Any(), handlerTripID).Return(handlerFinishedOutput(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+handlerTripID.String()+"/finish", nil)

	handler.FinishTrip(rec, req, handlerTripID)

	if rec.Code != http.StatusOK {
		t.Fatalf("статус: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestFinishTrip_CodeTripCompleted_AlreadyFinished(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().
		Finish(gomock.Any(), handlerTripID).
		Return(applicationTrip.Output{}, errs.NewServiceError(trip.CodeTripAlreadyCompleted, "Operation is not allowed"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+handlerTripID.String()+"/finish", nil)

	handler.FinishTrip(rec, req, handlerTripID)

	assertProblem(t, rec, http.StatusConflict, trip.CodeTripAlreadyCompleted)
}

func TestFinishTrip_CodeTripNotFound_UnknownId(t *testing.T) {
	handler, service := newTestHandler(t)

	service.EXPECT().
		Finish(gomock.Any(), handlerTripID).
		Return(applicationTrip.Output{}, errs.NewServiceError(trip.CodeTripNotFound, "Trip was not found"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+handlerTripID.String()+"/finish", nil)

	handler.FinishTrip(rec, req, handlerTripID)

	assertProblem(t, rec, http.StatusNotFound, trip.CodeTripNotFound)
}

func newTestHandler(t *testing.T) (*Handler, *MockService) {
	t.Helper()

	ctrl := gomock.NewController(t)

	service := NewMockService(ctrl)

	return NewHandler(service, NewMockPinger(ctrl), slog.New(slog.DiscardHandler)), service
}

func expectedInput() applicationTrip.Input {
	return applicationTrip.Input{
		UserID:   handlerUserID,
		DriverID: handlerDriverID,
		Price:    1450,
		StartPoint: applicationTrip.Coordinates{
			Latitude:  59.9398,
			Longitude: 30.3146,
		},
		EndPoint: applicationTrip.Coordinates{
			Latitude:  59.9290,
			Longitude: 30.3626,
		},
	}
}

func handlerOutput() applicationTrip.Output {
	return applicationTrip.Output{
		ID:         handlerTripID,
		UserID:     handlerUserID,
		DriverID:   handlerDriverID,
		Price:      1450,
		Status:     applicationTrip.Status(trip.StatusActive),
		StartPoint: expectedInput().StartPoint,
		EndPoint:   expectedInput().EndPoint,
	}
}

func handlerFinishedOutput() applicationTrip.Output {
	out := handlerOutput()
	out.Status = applicationTrip.Status(trip.StatusCompleted)

	return out
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
		t.Fatalf("тело не является JSON: %v", err)
	}
}

func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode errs.ErrorCode) {
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
