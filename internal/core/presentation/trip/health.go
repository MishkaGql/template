package trip

import (
	"net/http"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/core/presentation/httpsender"
)

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
