package handler

import (
	"net/http"
	"time"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal"
	"github.com/MishkaGql/template/internal/db"
	"github.com/MishkaGql/template/internal/errs"
)

type TripHandler struct {
	service  *internal.Service
	dbPinger *db.DbPinger
}

// не скомпилится если не соответсвует интерфейсу
var _ api.ServerInterface = (*TripHandler)(nil)

func NewHandler(service *internal.Service, pinger *db.DbPinger) *TripHandler {
	return &TripHandler{
		service:  service,
		dbPinger: pinger,
	}
}

// (POST /api/v1/trips)
func (h *TripHandler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	params api.CreateTripParams,
) {
	tripStartedAt := time.Now()
	var trip api.TripData

	err := decodeJSON(r, &trip)
	if err != nil {
		SendError(w, errs.InternalServiceError)
		return
	}
	
	response, err := h.service.Create(r.Context(), trip, tripStartedAt)
	if err != nil {
		SendError(w, err)
		return
	}
	
	err = SendResponse(w, response, http.StatusCreated)
	if err != nil {
		SendError(w, errs.InternalServiceError)
		return
	}
}

// (GET /api/v1/trips/{tripId})
func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {

	response, err := h.service.Get(r.Context(), tripId)
	if err != nil {
		SendError(w, err)
		return
	}

	err = SendResponse(w, response, http.StatusOK)
	if err != nil {
		SendError(w, errs.InternalServiceError)
		return
	}
}

// (POST /api/v1/trips/{tripId}/finish)
func (h *TripHandler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	finishedAt := time.Now()

	response, err := h.service.Finish(r.Context(), finishedAt, tripId)
	if err != nil {
		SendError(w, err)
		return
	}

	err = SendResponse(w, response, http.StatusOK)
	if err != nil {
		SendError(w, errs.InternalServiceError)
		return
	}
}

// (GET /health)
func (tripApi *TripHandler) Health(w http.ResponseWriter, r *http.Request) {
	health := api.HealthResponse{
		Status: "ok",
	}

	err := SendResponse(w, health, http.StatusOK)
	if err != nil {
		SendError(w, errs.InternalServiceError)
	}
}

// (GET /ready)
func (h *TripHandler) Ready(w http.ResponseWriter, r *http.Request) {
	health := api.HealthResponse{}

	err := h.dbPinger.Ping(r.Context())
	var responseError error
	
	if err != nil {
		health.Status = "fail"
		responseError = SendResponse(w, health, http.StatusServiceUnavailable)
	} else {
		health.Status = "ok"
		responseError =SendResponse(w, health, http.StatusOK)
	}

	if responseError != nil {
		SendError(w,errs.InternalServiceError)
	}
}
