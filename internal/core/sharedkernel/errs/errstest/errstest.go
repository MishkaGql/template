package errstest

import (
	"testing"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

func AssertCode(t *testing.T, err error, want errs.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want *errs.Error with code %q", want)
	}

	publicErr, ok := errs.AsError(err)
	if !ok {
		t.Fatalf("error type = %T, want *errs.Error in error chain", err)
	}

	if got := publicErr.Code(); got != want {
		t.Errorf("code = %q, want %q", got, want)
	}
}
