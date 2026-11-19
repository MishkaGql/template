package errs

func New(code Code, msg string, cause error) *Error {
	return &Error{
		code:    code,
		message: msg,
		cause:   cause,
	}
}

func NewInternal(cause error) *Error {
	return &Error{
		message: "Internal server error",
		code:    InternalError,
		cause:   cause,
	}
}

func NewValidation() *Error {
	return &Error{
		message: "Request validation failed",
		code:    InvalidRequest,
	}
}
