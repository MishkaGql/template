package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/MishkaGql/template/internal/errs"
)

func decodeJSON(r *http.Request, data any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(data)
	if err != nil {
		return err
	}

	return nil
}

func SendResponse(w http.ResponseWriter, data any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return err
	}
	return nil
}

func SendError(w http.ResponseWriter, err error) {
	var serviceError errs.ServiceError

	if !errors.As(err, &serviceError) {
		panic(fmt.Sprintf("Unexpected error type %T", err))
	}
	problem := errs.ToProblem(serviceError)
	
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(int(problem.Status))

	encodeErr := json.NewEncoder(w).Encode(problem)
	if encodeErr != nil {
		slog.Info("Грешный ты")
	}
}
