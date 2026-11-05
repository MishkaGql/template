package errs

import (
	"errors"
)

type AbstractError interface {
	error
	Message() *string
	Code() ErrorCode
}

func AsAbstractError(err error) (AbstractError, bool) {
	if absErr, ok := errors.AsType[AbstractError](err); ok {
		return absErr, true
	}

	return nil, false
}
