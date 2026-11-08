package errs

import (
	"net/http"
)

// Может кидаться на всех слоях.
// Через него другие сервисы, вызывающие методы друг друга, могут понять, что пошло не так.
type ErrorCode string

const (
	InvalidRequest       ErrorCode = "invalid_request"
	InternalError        ErrorCode = "internal_error"
	DriverBusy           ErrorCode = "driver_busy"
	TripNotFound         ErrorCode = "trip_not_found"
	TripAlreadyCompleted ErrorCode = "trip_completed"
	IdempotencyConflict  ErrorCode = "idempotency_conflict"
)

func (code ErrorCode) String() string {
	return string(code)
}

// На каждый ErrorCode есть свой codeDefinition.
// codeDefinition статичен, определяется в handler слое через мапу.
type codeDefinition struct {
	Title  string
	Status int
}

func NewCodeDefinition(title string, status int) codeDefinition {
	return codeDefinition{
		Title:  title,
		Status: status,
	}
}

var codeDefinitionByCode = map[ErrorCode]codeDefinition{
	InvalidRequest:       NewCodeDefinition("Invalid request", http.StatusBadRequest),
	InternalError:        NewCodeDefinition("Internal Server Error", http.StatusInternalServerError),
	DriverBusy:           NewCodeDefinition("Driver busy", http.StatusConflict),
	TripNotFound:         NewCodeDefinition("Trip not found", http.StatusNotFound),
	TripAlreadyCompleted: NewCodeDefinition("Trip completed", http.StatusConflict),
	IdempotencyConflict:  NewCodeDefinition("Idempotency conflict", http.StatusConflict),
}
