package purejson

import (
	"errors"
	"fmt"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

var (
	ErrInvalidHandle      = errors.New("invalid handle")
	ErrClosed             = errors.New("closed")
	ErrParserBusy         = errors.New("parser busy")
	ErrNumberOutOfRange   = errors.New("number out of range")
	ErrPrecisionLoss      = errors.New("precision loss")
	ErrCPUUnsupported     = errors.New("cpu unsupported")
	ErrABIVersionMismatch = errors.New("abi version mismatch")
	ErrInvalidJSON        = errors.New("invalid json")
	ErrElementNotFound    = errors.New("element not found")
	ErrWrongType          = errors.New("wrong type")
)

var errLoadLibrary = errors.New("load library")

type Error struct {
	Code    int32
	Offset  uint64
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	label := "purejson error"
	if e.Err != nil {
		label = e.Err.Error()
	}

	switch {
	case e.Code != 0 && e.Message != "" && hasOffset(e.Offset):
		return fmt.Sprintf("%s (code=%d, offset=%d): %s", label, e.Code, e.Offset, e.Message)
	case e.Code != 0 && e.Message != "":
		return fmt.Sprintf("%s (code=%d): %s", label, e.Code, e.Message)
	case e.Code != 0 && hasOffset(e.Offset):
		return fmt.Sprintf("%s (code=%d, offset=%d)", label, e.Code, e.Offset)
	case e.Code != 0:
		return fmt.Sprintf("%s (code=%d)", label, e.Code)
	case e.Message != "":
		return fmt.Sprintf("%s: %s", label, e.Message)
	default:
		return label
	}
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
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
		Code:    code,
		Offset:  normalizeOffset(details.offset),
		Message: details.message,
		Err:     err,
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
		return nil
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
