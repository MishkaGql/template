package errs

type ErrorCode string

func (code ErrorCode) String() string {
	return string(code)
}

const (
	InvalidRequest ErrorCode = "invalid_request"
	InternalError  ErrorCode = "internal_error"
)
