package httpsender

import (
	"encoding/json"
	"net/http"
)

func ParseRequest(r *http.Request, data any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	return d.Decode(data)
}
