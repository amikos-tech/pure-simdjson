package purejson

import (
	"errors"
	"fmt"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

var (
	// ErrInvalidHandle reports that a parser, document, or element handle was not valid.
	ErrInvalidHandle = errors.New("invalid handle")
	// ErrClosed reports use of a parser or document after Close succeeded.
	ErrClosed = errors.New("closed")
	// ErrParserBusy reports that a parser still owns a live document.
	ErrParserBusy = errors.New("parser busy")
	// ErrNumberOutOfRange reports that a numeric conversion overflowed the target type.
	ErrNumberOutOfRange = errors.New("number out of range")
	// ErrPrecisionLoss reports that a numeric conversion would lose precision.
	ErrPrecisionLoss = errors.New("precision loss")
	// ErrCPUUnsupported reports that the loaded native library cannot run on the current CPU.
	ErrCPUUnsupported = errors.New("cpu unsupported")
	// ErrABIVersionMismatch reports that the Go wrapper and native library expose different ABI versions.
	ErrABIVersionMismatch = errors.New("abi version mismatch")
	// ErrInvalidJSON reports invalid JSON input.
	ErrInvalidJSON = errors.New("invalid json")
	// ErrElementNotFound reports lookup of a missing element.
	ErrElementNotFound = errors.New("element not found")
	// ErrWrongType reports an accessor call on the wrong JSON value kind.
	ErrWrongType = errors.New("wrong type")
	// ErrInternal reports native panics, internal failures, and unknown status codes.
	ErrInternal = errors.New("internal error")
)

var errLoadLibrary = errors.New("load library")

// Error carries native status details while still participating in Go's
// sentinel-error matching via Unwrap. Status details are exposed through
// accessor methods so callers cannot mutate them after construction.
type Error struct {
	code    int32
	offset  uint64
	message string
	err     error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	label := "purejson error"
	if e.err != nil {
		label = e.err.Error()
	}

	switch {
	case e.code != 0 && e.message != "" && hasOffset(e.offset):
		return fmt.Sprintf("%s (code=%d, offset=%d): %s", label, e.code, e.offset, e.message)
	case e.code != 0 && e.message != "":
		return fmt.Sprintf("%s (code=%d): %s", label, e.code, e.message)
	case e.code != 0 && hasOffset(e.offset):
		return fmt.Sprintf("%s (code=%d, offset=%d)", label, e.code, e.offset)
	case e.code != 0:
		return fmt.Sprintf("%s (code=%d)", label, e.code)
	case e.message != "":
		return fmt.Sprintf("%s: %s", label, e.message)
	default:
		return label
	}
}

// Code returns the native status code returned by the FFI call.
func (e *Error) Code() int32 {
	if e == nil {
		return 0
	}
	return e.code
}

// Offset returns the reported byte offset for parse errors. Zero means unknown.
func (e *Error) Offset() uint64 {
	if e == nil {
		return 0
	}
	return e.offset
}

// Message returns the native error message, when available.
func (e *Error) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

type nativeDetails struct {
	message string
	offset  uint64
}

func wrapParserStatus(bindings *ffi.Bindings, parser ffi.ParserHandle, code int32) error {
	if code == int32(ffi.OK) {
		return nil
	}

	details := nativeDetails{}
	if bindings != nil {
		if message, rc := bindings.ParserLastError(parser); rc == int32(ffi.OK) && message != "" {
			details.message = message
		}
		if offset, rc := bindings.ParserLastErrorOffset(parser); rc == int32(ffi.OK) {
			details.offset = offset
		}
	}

	return newError(code, details, sentinelForStatus(code))
}

func wrapStatus(code int32) error {
	if code == int32(ffi.OK) {
		return nil
	}
	return newError(code, nativeDetails{}, sentinelForStatus(code))
}

func wrapABIMismatch(expected, actual uint32, libraryPath string) error {
	return newError(int32(ffi.ErrABIMismatch), nativeDetails{
		message: fmt.Sprintf("expected ABI 0x%08x, got 0x%08x from %s", expected, actual, libraryPath),
		offset:  ffi.LastErrorOffsetUnknown,
	}, ErrABIVersionMismatch)
}

func wrapLoadFailure(message string, err error) error {
	loadErr := errLoadLibrary
	if err != nil {
		loadErr = fmt.Errorf("%w: %v", errLoadLibrary, err)
	}
	return newError(0, nativeDetails{
		message: message,
		offset:  ffi.LastErrorOffsetUnknown,
	}, loadErr)
}

func newError(code int32, details nativeDetails, err error) error {
	if code == int32(ffi.OK) && err == nil && details.message == "" {
		return nil
	}

	if details.message == "" && !hasOffset(details.offset) && err != nil && code == 0 {
		return err
	}

	return &Error{
		code:    code,
		offset:  normalizeOffset(details.offset),
		message: details.message,
		err:     err,
	}
}

func sentinelForStatus(code int32) error {
	switch ffi.ErrorCode(code) {
	case ffi.ErrInvalidHandle:
		return ErrInvalidHandle
	case ffi.ErrParserBusy:
		return ErrParserBusy
	case ffi.ErrWrongType:
		return ErrWrongType
	case ffi.ErrElementNotFound:
		return ErrElementNotFound
	case ffi.ErrInvalidJSON:
		return ErrInvalidJSON
	case ffi.ErrNumberOutOfRange:
		return ErrNumberOutOfRange
	case ffi.ErrPrecisionLoss:
		return ErrPrecisionLoss
	case ffi.ErrCPUUnsupported:
		return ErrCPUUnsupported
	case ffi.ErrABIMismatch:
		return ErrABIVersionMismatch
	default:
		return ErrInternal
	}
}

func normalizeOffset(offset uint64) uint64 {
	if !hasOffset(offset) {
		return 0
	}
	return offset
}

func hasOffset(offset uint64) bool {
	return offset != 0 && offset != ffi.LastErrorOffsetUnknown
}
