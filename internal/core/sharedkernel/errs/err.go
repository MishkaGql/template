package errs

import (
	"errors"
	"log/slog"
)

type Code string

const (
	InvalidRequest Code = "invalid_request"
	InternalError  Code = "internal_error"
)

type Error struct {
	message string
	code    Code
	cause   error
}

func (e *Error) Error() string {
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

func (e *Error) Unwrap() error {
	return e.cause
}

func (e *Error) Message() *string {
	if e.message == "" {
		return nil
	}

	return &e.message
}

func (e *Error) Code() Code {
	return e.code
}

func (e *Error) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("msg", e.message),
		slog.String("code", string(e.code)),
	}
	if e.cause != nil {
		attrs = append(attrs, slog.Any("cause", e.cause))
	}

	return slog.GroupValue(attrs...)
}

func AsError(err error) (*Error, bool) {
	if absErr, ok := errors.AsType[*Error](err); ok {
		return absErr, true
	}

	return nil, false
}
