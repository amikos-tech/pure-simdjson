package purejson

import (
	"errors"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

func TestWrapStatusInternalCodesMapToErrInternal(t *testing.T) {
	testCases := []struct {
		name string
		code int32
	}{
		{name: "internal", code: int32(ffi.ErrInternal)},
		{name: "panic", code: int32(ffi.ErrPanic)},
		{name: "cpp-exception", code: int32(ffi.ErrCPPException)},
		{name: "unknown", code: 12345},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := wrapStatus(tc.code)
			if !errors.Is(err, ErrInternal) {
				t.Fatalf("wrapStatus(%d) error = %v, want ErrInternal", tc.code, err)
			}

			var nativeErr *Error
			if !errors.As(err, &nativeErr) {
				t.Fatalf("wrapStatus(%d) error = %v, want *Error", tc.code, err)
			}
			if nativeErr.Code() != tc.code {
				t.Fatalf("wrapStatus(%d) native code = %d, want %d", tc.code, nativeErr.Code(), tc.code)
			}
		})
	}
}
