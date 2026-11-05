package errs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testCode      = ErrorCode("test_code")
	testMessage   = "trip was not found"
	testCause     = "pgconnect: connection refused"
	testSeparator = ": "
)

func TestError_PrintNothing_NotAnyMsg(t *testing.T) {
	err := &ServiceError{code: testCode}

	assert.Empty(t, err.Error())
	assert.Nil(t, err.Message())
}

func TestError_PrintOneMsg_NotCause(t *testing.T) {
	err := NewServiceError(testCode, testMessage)

	assert.Equal(t, testMessage, err.Error())
	assert.Equal(t, testMessage, *err.Message())
}

func TestError_PrintTwoMsg_OneCause(t *testing.T) {
	cause := errors.New(testCause)

	err := NewServiceErrorWithCause(testCode, testMessage, cause)

	assert.Equal(t, testMessage+testSeparator+testCause, err.Error())
	assert.Equal(t, testMessage, *err.Message())
	assert.ErrorIs(t, err, cause)
}

func TestError_PrintOneCauseMsg_ErrMessageEmpty(t *testing.T) {
	err := NewServiceErrorWithCause(testCode, "", errors.New(testCause))

	assert.Equal(t, testCause, err.Error())
	assert.Nil(t, err.Message())
}
