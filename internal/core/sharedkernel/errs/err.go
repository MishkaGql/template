package errs

import (
	"log/slog"
)

type ServiceError struct {
	message string
	code    ErrorCode
	cause   error
}

var _ AbstractError = (*ServiceError)(nil)

func NewServiceError(code ErrorCode, msg string) *ServiceError {
	return &ServiceError{
		code:    code,
		message: msg,
	}
}

func NewServiceErrorWithCause(code ErrorCode, msg string, cause error) *ServiceError {
	return &ServiceError{
		code:    code,
		message: msg,
		cause:   cause,
	}
}

func (e *ServiceError) Message() *string {
	if e.message == "" {
		return nil
	}

	return &e.message
}

func (e *ServiceError) Code() ErrorCode {
	return e.code
}

func (e *ServiceError) Cause() error {
	return e.cause
}

func (e *ServiceError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("msg", e.message),
		slog.String("code", e.code.String()),
	)
}

func (e *ServiceError) Error() string {
	switch {
	case e.message == "":
		if e.cause == nil {
			return ""
		}
		return e.cause.Error()
	case e.cause == nil:
		return e.message
	default:
		return e.message + ": " + e.cause.Error()
	}
}

func (e *ServiceError) Unwrap() error {
	return e.cause
}

func NewInternalServiceError(cause error) *ServiceError {
	return &ServiceError{
		message: "Internal service error",
		code:    InternalError,
		cause:   cause,
	}
}

func NewValidationServiceError() *ServiceError {
	return &ServiceError{
		message: "Validation",
		code:    InvalidRequest,
	}
}
