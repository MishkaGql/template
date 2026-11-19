package httpsender

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

type MIME string

const (
	MIMEJSON        MIME = "application/json"
	MIMEProblemJSON MIME = "application/problem+json"
)

const internalErrorJSON = `{
	"type":"https://tripgo.example/problems/internal-error",
	"title":"Internal Server Error",
	"status":500,
	"code":"` + string(errs.InternalError) + `",
	"detail":"Internal server error"
}`

func ParseRequest(r *http.Request, data any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	if err := d.Decode(data); err != nil {
		return err
	}

	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return errs.NewValidation()
	}

	return nil
}

func SendResponse(w http.ResponseWriter, mime MIME, status int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		w.Header().Set("Content-Type", string(MIMEProblemJSON))
		w.WriteHeader(http.StatusInternalServerError)

		_, _ = w.Write([]byte(internalErrorJSON))
		return
	}

	w.Header().Set("Content-Type", string(mime))
	w.WriteHeader(status)

	_, _ = w.Write(body)
}
