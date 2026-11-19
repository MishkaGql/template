package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestError(t *testing.T) {
	cause := errors.New("connection refused")
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "EmptyMsgAndCause",
			err:  New("test_code", "", nil),
		},
		{
			name: "MsgOnly",
			err:  New("test_code", "trips missing", nil),
			want: "trips missing",
		},
		{
			name: "CauseOnly",
			err:  New("test_code", "", cause),
			want: "connection refused",
		},
		{
			name: "MsgAndCause",
			err:  New("test_code", "trips missing", cause),
			want: "trips missing: connection refused",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	t.Run("EmptyMessage", func(t *testing.T) {
		t.Parallel()
		err := New("test_code", "", errors.New("connection refused"))

		if got := err.Message(); got != nil {
			t.Errorf("Message() = %q, want nil", *got)
		}
	})

	t.Run("MessagePresent", func(t *testing.T) {
		t.Parallel()
		want := "trips missing"
		err := New("test_code", want, errors.New("connection refused"))

		if got := err.Message(); *got != want {
			t.Errorf("Message() = %q, want %q", *got, want)
		}
	})
}

func TestAsError(t *testing.T) {
	t.Run("WrappedError", func(t *testing.T) {
		t.Parallel()

		want := New("test_code", "trips missing", nil)
		err := fmt.Errorf("get trips: %w", want)
		got, ok := AsError(err)

		if !ok {
			t.Fatal("AsError() ok = false, want true")
		}

		if got != want {
			t.Errorf("AsError() = %p, want original error %p", got, want)
		}
	})

	t.Run("OtherError", func(t *testing.T) {
		t.Parallel()

		err := errors.New("connection refused")
		got, ok := AsError(err)

		if ok || got != nil {
			t.Errorf("AsError() = (%v, %t), want (nil, false)", got, ok)
		}
	})
}
