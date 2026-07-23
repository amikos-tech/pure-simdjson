package purejson

import (
	"errors"
	"strings"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

func TestErrorHasOffset(t *testing.T) {
	testCases := []struct {
		name       string
		err        *Error
		wantOffset uint64
		wantKnown  bool
		wantText   string
	}{
		{
			name:       "known nonzero",
			err:        &Error{code: int32(ffi.ErrInvalidJSON), offset: 3, hasOffset: true, err: ErrInvalidJSON},
			wantOffset: 3,
			wantKnown:  true,
			wantText:   "offset=3",
		},
		{
			name:       "known zero",
			err:        &Error{code: int32(ffi.ErrInvalidJSON), offset: 0, hasOffset: true, err: ErrInvalidJSON},
			wantOffset: 0,
			wantKnown:  true,
			wantText:   "offset=0",
		},
		{
			name:       "unknown",
			err:        &Error{code: int32(ffi.ErrInvalidJSON), err: ErrInvalidJSON},
			wantOffset: 0,
			wantKnown:  false,
		},
		{
			name:       "nil",
			err:        nil,
			wantOffset: 0,
			wantKnown:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Offset(); got != tc.wantOffset {
				t.Fatalf("Offset() = %d, want %d", got, tc.wantOffset)
			}
			if got := tc.err.HasOffset(); got != tc.wantKnown {
				t.Fatalf("HasOffset() = %t, want %t", got, tc.wantKnown)
			}
			if tc.err == nil {
				return
			}
			if tc.wantText != "" && !strings.Contains(tc.err.Error(), tc.wantText) {
				t.Fatalf("Error() = %q, want %q", tc.err.Error(), tc.wantText)
			}
			if !tc.wantKnown && strings.Contains(tc.err.Error(), "offset=") {
				t.Fatalf("Error() = %q, want no offset clause", tc.err.Error())
			}
		})
	}
}

func TestWrapStatusInternalCodesMapToErrInternal(t *testing.T) {
	testCases := []struct {
		name string
		code int32
	}{
		{name: "internal", code: int32(ffi.ErrInternal)},
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

func TestWrapStatusMapsPanicAndCPPExceptionSeparately(t *testing.T) {
	testCases := []struct {
		name string
		code int32
		want error
	}{
		{name: "panic", code: int32(ffi.ErrPanic), want: ErrPanic},
		{name: "cpp-exception", code: int32(ffi.ErrCPPException), want: ErrCPPException},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := wrapStatus(tc.code)
			if !errors.Is(err, tc.want) {
				t.Fatalf("wrapStatus(%d) error = %v, want %v", tc.code, err, tc.want)
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

func TestWrapStatusMapsNotImplementedSeparately(t *testing.T) {
	err := wrapStatus(int32(ffi.ErrNotImplemented))
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("wrapStatus(%d) error = %v, want ErrNotImplemented", ffi.ErrNotImplemented, err)
	}

	var nativeErr *Error
	if !errors.As(err, &nativeErr) {
		t.Fatalf("wrapStatus(%d) error = %v, want *Error", ffi.ErrNotImplemented, err)
	}
	if nativeErr.Code() != int32(ffi.ErrNotImplemented) {
		t.Fatalf("wrapStatus(%d) native code = %d, want %d", ffi.ErrNotImplemented, nativeErr.Code(), ffi.ErrNotImplemented)
	}
}

func TestWrapStatusMapsDepthLimitSeparately(t *testing.T) {
	err := wrapStatus(int32(ffi.ErrDepthLimit))
	if !errors.Is(err, ErrDepthLimitExceeded) {
		t.Fatalf("wrapStatus(%d) error = %v, want ErrDepthLimitExceeded", ffi.ErrDepthLimit, err)
	}

	var nativeErr *Error
	if !errors.As(err, &nativeErr) {
		t.Fatalf("wrapStatus(%d) error = %v, want *Error", ffi.ErrDepthLimit, err)
	}
	if nativeErr.Code() != int32(ffi.ErrDepthLimit) {
		t.Fatalf("wrapStatus(%d) native code = %d, want %d", ffi.ErrDepthLimit, nativeErr.Code(), ffi.ErrDepthLimit)
	}
}

func TestSentinelMapping(t *testing.T) {
	testCases := []struct {
		name string
		code ffi.ErrorCode
		want error
	}{
		{name: "capacity limit", code: ffi.ErrCapacityLimit, want: ErrCapacityLimitExceeded},
		{name: "depth limit", code: ffi.ErrDepthLimit, want: ErrDepthLimitExceeded},
		{name: "kernel locked", code: ffi.ErrKernelLocked, want: ErrKernelLocked},
		{name: "invalid argument remains internal", code: ffi.ErrInvalidArg, want: ErrInternal},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := wrapStatus(int32(tc.code))
			if !errors.Is(err, tc.want) {
				t.Fatalf("wrapStatus(%d) error = %v, want %v", tc.code, err, tc.want)
			}
		})
	}
}
