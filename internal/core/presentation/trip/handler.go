package handler

import (
	"net/http"
	"time"
	"uuid"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/application/trip"
	"github.com/MishkaGql/template/internal/db"
	"github.com/MishkaGql/template/internal/errs"
	"github.com/MishkaGql/template/internal/presentation/http/trip/converter"
)


type TripHandler struct {
	service  *trip.Service
	dbPinger *db.DbPinger
}


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
	var request api.TripData
	
	// Parse request
	err := decodeJSON(r, &request)
	if err != nil {
		SendError(w, errs.InternalServiceError)
		return
	}
	
	// Buisness logic
	output, err := h.service.Create(r.Context(), converter.ToInput(request))
	if err != nil {
		SendError(w, err)
		return
	}

	// Send response
	err = SendResponse(w, converter.ToResponse(output), http.StatusCreated)
	if err != nil {
		SendError(w, errs.InternalServiceError)
		return
	}
}

// (GET /api/v1/trips/{tripId})
func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	
	// Buisness logic
	response, err := h.service.Get(r.Context(), uuid.UUID(tripId))
	if err != nil {
		SendError(w, err)
		return
	}
	
	// Send response
	err = SendResponse(w, response, http.StatusOK)
	if err != nil {
		SendError(w, errs.InternalServiceError)
		return
	}
}

// (POST /api/v1/trips/{tripId}/finish)
func (h *TripHandler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	
	// Buisness logic
	response, err := h.service.Finish(r.Context(), tripId)
	if err != nil {
		SendError(w, err)
		return
	}
	
	// Send response
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
		responseError = SendResponse(w, health, http.StatusOK)
	}

	if responseError != nil {
		SendError(w,errs.InternalServiceError)
	}
}
