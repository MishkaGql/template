package trip

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/core/presentation/httpsender"
	"github.com/MishkaGql/template/internal/core/presentation/problem"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

type Handler struct {
	cancelPending func(uuid.UUID)
	service       Service
	dbPinger      Pinger
	logger        *slog.Logger
}

var _ api.ServerInterface = (*Handler)(nil)

func NewHandler(service Service, pinger Pinger, logger *slog.Logger, cancelPending func(uuid.UUID)) *Handler {
	return &Handler{
		cancelPending: cancelPending,
		service:       service,
		dbPinger:      pinger,
		logger:        logger,
	}
}

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	var request api.TripData

	if err := httpsender.ParseRequest(r, &request); err != nil {
		problem.Send(r.Context(), h.logger, w, errs.NewValidation())
		return
	}

	output, err := h.service.Create(r.Context(), toInput(request))
	if err != nil {
		problem.Send(r.Context(), h.logger, w, err)
		return
	}

	w.Header().Set("Location", r.URL.Path+"/"+output.ID.String())
	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusCreated, toResponse(output))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	output, err := h.service.Get(r.Context(), tripID)
	if err != nil {
		problem.Send(r.Context(), h.logger, w, err)
		return
	}

	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusOK, toResponse(output))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	output, err := h.service.Finish(r.Context(), tripID)
	if err != nil {
		problem.Send(r.Context(), h.logger, w, err)
		return
	}

	h.cancelPending(tripID)
	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusOK, toResponse(output))
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	health := api.HealthResponse{
		Status: api.Ok,
	}

	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusOK, health)
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	health := api.HealthResponse{}

	if err := h.dbPinger.Ping(r.Context()); err != nil {
		health.Status = api.Unavailable
		httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusServiceUnavailable, health)
		return
	}

	health.Status = api.Ok
	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusOK, health)
}

func (h *Handler) ListTripPositions(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	positions, err := h.service.ListTripPositions(r.Context(), tripID)
	if err != nil {
		problem.Send(r.Context(), h.logger, w, err)
		return
	}

	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusOK, toPositionsResponse(positions))
}

func (h *Handler) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	var request api.PositionData

	if err := httpsender.ParseRequest(r, &request); err != nil {
		problem.Send(r.Context(), h.logger, w, errs.NewValidation())
		return
	}

	position, err := h.service.CreateTripPosition(r.Context(), tripID, toPositionInput(request))
	if err != nil {
		problem.Send(r.Context(), h.logger, w, err)
		return
	}

	h.cancelPending(tripID)
	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusCreated, toPositionResponse(position))
}
