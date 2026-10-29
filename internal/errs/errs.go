package errs

import (
	"errors"
	"log/slog"
	"strings"
	"github.com/MishkaGql/template/api"
)

// не уверен насколько он нужен
type Causer interface {
	SetCause(err error)
}

type ServiceError struct {
	Message string
	Code    ErrorCode

	// Сюда идут только не бизнес ошибки (бдшная, сетевая и тд..). Или идут) Подумать надо
	// Может иметь вложенные ошибки.
	Cause   error
}

func NewServiceError(code ErrorCode, msg string) ServiceError {
	return ServiceError{
		Code:    code,
		Message: msg,
	}
}

// Не должен логировать cause,
// ибо он может содержать в себе много вложенных ошибок
func (e ServiceError) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("msg", e.Message),
		slog.String("code", e.Code.String()),
	)
}

func (e ServiceError) SetCause(err error) {
	if e.Cause == nil {
		e.Cause = err
	} else {
		e.Cause = errors.Join(err, e.Cause)
	}
}

func (error ServiceError) Error() string {
	return error.Message
}

var baseUrl = "https://tripgo.example/problems/"

func ToProblem(err ServiceError) api.Problem {
	codeDef := codeDefinitionByCode[err.Code]
	typeUrl := baseUrl + strings.Replace(err.Code.String(), "_", "-", 0)

	return api.Problem{
		Title:  codeDef.Title,
		Status: int32(codeDef.Status),
		Type:   typeUrl,
		Code:   err.Code.String(),
		Detail: &err.Message,
	}
}

func NewInternalServiceError(cause error) ServiceError {
	err := InternalServiceError
	err.Cause = cause
	return err
}

var (
	InternalServiceError = ServiceError{
		Message: "Internal server error",
		Code:    InternalError,
	}

	ValidationServiceError = ServiceError{
		Message: "Request validation failed",
		Code:    InvalidRequest,
	}
)
