package httpsender

import (
	"encoding/json"
	"net/http"
)

type MIME string

const (
	MIMEJSON        MIME = "application/json"
	MIMEProblemJSON MIME = "application/problem+json"
)

func SendResponse(w http.ResponseWriter, mime MIME, status int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	writeJSON(w, mime, status, body)
}

func writeJSON(w http.ResponseWriter, mime MIME, status int, body []byte) {
	w.Header().Set("Content-Type", string(mime))
	w.WriteHeader(status)

	_, _ = w.Write(body)
}
