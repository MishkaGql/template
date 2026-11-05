package errstest

import (
	"errors"
	"testing"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

func AssertCode(t *testing.T, err error, want errs.ErrorCode) {
	t.Helper()

	if err == nil {
		t.Fatalf("code: got nil, want: code %q", want)
	}

	var absErr errs.AbstractError
	if !errors.As(err, &absErr) {
		t.Fatalf("err type: got: %T, want: %T ", err, absErr)
	}

	if absErr.Code() != want {
		t.Errorf("code: got %q, want %q", absErr.Code(), want)
	}
}
