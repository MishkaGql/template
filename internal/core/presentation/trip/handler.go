package trip

import (
	"log/slog"
	"net/http"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/core/presentation/httpsender"
	"github.com/MishkaGql/template/internal/core/presentation/problem"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

type Handler struct {
	service  Service
	dbPinger Pinger
	logger   *slog.Logger
}

var _ api.ServerInterface = (*Handler)(nil)

func NewHandler(service Service, pinger Pinger, logger *slog.Logger) *Handler {
	return &Handler{
		service:  service,
		dbPinger: pinger,
		logger:   logger,
	}
}

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	var request api.TripData

	if err := httpsender.ParseRequest(r, &request); err != nil {
		problem.Send(r.Context(), h.logger, w, errs.NewValidationServiceError())
		return
	}

	output, err := h.service.Create(r.Context(), MapToInput(request))
	if err != nil {
		problem.Send(r.Context(), h.logger, w, err)
		return
	}

	w.Header().Set("Location", r.URL.Path+"/"+output.ID.String())
	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusCreated, MapToResponse(output))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	output, err := h.service.Get(r.Context(), tripId)
	if err != nil {
		problem.Send(r.Context(), h.logger, w, err)

		return
	}

	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusOK, MapToResponse(output))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	output, err := h.service.Finish(r.Context(), tripId)
	if err != nil {
		problem.Send(r.Context(), h.logger, w, err)

		return
	}

	httpsender.SendResponse(w, httpsender.MIMEJSON, http.StatusOK, MapToResponse(output))
}
