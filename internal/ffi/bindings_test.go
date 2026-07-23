package ffi

import (
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func TestProbeABIResolvesOnlyABIVersion(t *testing.T) {
	var lookups []string
	actual, err := probeABIWithRegistrar(
		7,
		func(handle uintptr, name string) (uintptr, error) {
			if handle != 7 {
				t.Fatalf("lookup handle = %d, want 7", handle)
			}
			lookups = append(lookups, name)
			return 11, nil
		},
		func(name string, target any, symbol uintptr) error {
			if name != "pure_simdjson_get_abi_version" {
				t.Fatalf("registered name = %q, want ABI getter", name)
			}
			if symbol != 11 {
				t.Fatalf("registered symbol = %d, want 11", symbol)
			}
			getABI, ok := target.(*func(*uint32) int32)
			if !ok {
				t.Fatalf("registered target type = %T, want *func(*uint32) int32", target)
			}
			*getABI = func(out *uint32) int32 {
				*out = 0x00010002
				return int32(OK)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ProbeABI() error = %v", err)
	}
	if actual != 0x00010002 {
		t.Fatalf("ProbeABI() = 0x%08x, want 0x00010002", actual)
	}
	if want := []string{"pure_simdjson_get_abi_version"}; !reflect.DeepEqual(lookups, want) {
		t.Fatalf("ProbeABI() lookups = %v, want %v", lookups, want)
	}
}

func TestProbeABIErrorsNameTheProbeStage(t *testing.T) {
	t.Run("lookup", func(t *testing.T) {
		_, err := probeABIWithRegistrar(
			1,
			func(uintptr, string) (uintptr, error) {
				return 0, errors.New("not found")
			},
			func(string, any, uintptr) error {
				t.Fatal("registrar called after lookup failure")
				return nil
			},
		)
		if err == nil {
			t.Fatal("ProbeABI() error = nil, want lookup error")
		}
		if !strings.Contains(err.Error(), "lookup pure_simdjson_get_abi_version") {
			t.Fatalf("ProbeABI() error = %q, want ABI getter name", err)
		}
	})

	t.Run("call", func(t *testing.T) {
		_, err := probeABIWithRegistrar(
			1,
			func(uintptr, string) (uintptr, error) {
				return 1, nil
			},
			func(_ string, target any, _ uintptr) error {
				getABI := target.(*func(*uint32) int32)
				*getABI = func(*uint32) int32 {
					return int32(ErrInternal)
				}
				return nil
			},
		)
		if err == nil {
			t.Fatal("ProbeABI() error = nil, want call error")
		}
		if !strings.Contains(err.Error(), "call pure_simdjson_get_abi_version") ||
			!strings.Contains(err.Error(), "127") {
			t.Fatalf("ProbeABI() error = %q, want ABI getter and status 127", err)
		}
	})
}

func TestBindRequiresEveryPhase11Symbol(t *testing.T) {
	phase11Symbols := []string{
		"pure_simdjson_parser_new_configured",
		"pure_simdjson_parser_get_last_error_has_offset",
		"pure_simdjson_element_get_bigint",
		"pure_simdjson_set_implementation",
		"pure_simdjson_lock_implementation_selection",
	}

	for _, missing := range phase11Symbols {
		t.Run(missing, func(t *testing.T) {
			var lookups []string
			bindings, err := bindWithRegistrar(
				1,
				func(_ uintptr, name string) (uintptr, error) {
					lookups = append(lookups, name)
					if name == missing {
						return 0, errors.New("symbol not found")
					}
					return 1, nil
				},
				func(string, any, uintptr) error {
					return nil
				},
			)
			if err == nil {
				t.Fatalf("Bind() error = nil with %s missing", missing)
			}
			if bindings != nil {
				t.Fatalf("Bind() bindings = %#v, want nil", bindings)
			}
			if !strings.Contains(err.Error(), missing) {
				t.Fatalf("Bind() error = %q, want missing symbol %q", err, missing)
			}
			if !containsString(lookups, missing) {
				t.Fatalf("Bind() lookups = %v, want %q", lookups, missing)
			}
		})
	}
}

func TestBindLooksUpCompletePhase11Surface(t *testing.T) {
	var lookups []string
	bindings, err := bindWithRegistrar(
		1,
		func(_ uintptr, name string) (uintptr, error) {
			lookups = append(lookups, name)
			return 1, nil
		},
		func(string, any, uintptr) error {
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if bindings == nil {
		t.Fatal("Bind() bindings = nil")
	}

	for _, name := range []string{
		"pure_simdjson_parser_new_configured",
		"pure_simdjson_parser_get_last_error_has_offset",
		"pure_simdjson_element_get_bigint",
		"pure_simdjson_set_implementation",
		"pure_simdjson_lock_implementation_selection",
	} {
		if !containsString(lookups, name) {
			t.Errorf("Bind() lookups = %v, missing %q", lookups, name)
		}
	}
}

func TestParserNewConfiguredPreservesArgumentWidths(t *testing.T) {
	b := &Bindings{
		parserNewConfigured: func(maxCapacity uint64, maxDepth uint32, out *ParserHandle) int32 {
			if maxCapacity != uint64(1)<<40 {
				t.Fatalf("maxCapacity = %d, want %d", maxCapacity, uint64(1)<<40)
			}
			if maxDepth != uint32(2048) {
				t.Fatalf("maxDepth = %d, want 2048", maxDepth)
			}
			*out = ParserHandle(42)
			return int32(OK)
		},
	}

	handle, rc := b.ParserNewConfigured(uint64(1)<<40, uint32(2048))
	if rc != int32(OK) {
		t.Fatalf("ParserNewConfigured() rc = %d, want %d", rc, OK)
	}
	if handle != ParserHandle(42) {
		t.Fatalf("ParserNewConfigured() handle = %d, want 42", handle)
	}
}

func TestParserLastErrorHasOffset(t *testing.T) {
	b := &Bindings{
		parserGetLastErrorHasOffset: func(parser ParserHandle, out *byte) int32 {
			if parser != ParserHandle(21) {
				t.Fatalf("parser = %d, want 21", parser)
			}
			*out = 1
			return int32(OK)
		},
	}

	hasOffset, rc := b.ParserLastErrorHasOffset(ParserHandle(21))
	if rc != int32(OK) {
		t.Fatalf("ParserLastErrorHasOffset() rc = %d, want %d", rc, OK)
	}
	if !hasOffset {
		t.Fatal("ParserLastErrorHasOffset() = false, want true")
	}
}

func TestElementGetBigIntCopiesAndFreesNativeBytes(t *testing.T) {
	payload := []byte("18446744073709551616")
	var freedPtr *byte
	var freedLen uintptr
	b := &Bindings{
		elementGetBigInt: func(_ *ValueView, outPtr **byte, outLen *uintptr) int32 {
			*outPtr = &payload[0]
			*outLen = uintptr(len(payload))
			return int32(OK)
		},
		bytesFree: func(ptr *byte, length uintptr) int32 {
			freedPtr = ptr
			freedLen = length
			return int32(OK)
		},
	}

	value, rc := b.ElementGetBigInt(&ValueView{})
	if rc != int32(OK) {
		t.Fatalf("ElementGetBigInt() rc = %d, want %d", rc, OK)
	}
	payload[0] = '9'
	if value != "18446744073709551616" {
		t.Fatalf("ElementGetBigInt() value = %q, want exact copied spelling", value)
	}
	if freedPtr == nil || freedLen != uintptr(len(payload)) {
		t.Fatalf("BytesFree() = (%p, %d), want non-nil pointer and length %d", freedPtr, freedLen, len(payload))
	}
}

func TestImplementationSelectionWrappers(t *testing.T) {
	var selected string
	var locked bool
	b := &Bindings{
		setImplementation: func(name *byte, length uintptr) int32 {
			if name == nil {
				selected = ""
			} else {
				selected = string(unsafeBytes(name, length))
			}
			return int32(OK)
		},
		lockImplementationSelection: func() int32 {
			locked = true
			return int32(OK)
		},
	}

	if rc := b.SetImplementation("fallback"); rc != int32(OK) {
		t.Fatalf("SetImplementation() rc = %d, want %d", rc, OK)
	}
	if selected != "fallback" {
		t.Fatalf("SetImplementation() selected = %q, want fallback", selected)
	}
	if rc := b.SetImplementation(""); rc != int32(OK) {
		t.Fatalf("SetImplementation(empty) rc = %d, want %d", rc, OK)
	}
	if selected != "" {
		t.Fatalf("SetImplementation(empty) selected = %q, want empty", selected)
	}
	if rc := b.LockImplementationSelection(); rc != int32(OK) {
		t.Fatalf("LockImplementationSelection() rc = %d, want %d", rc, OK)
	}
	if !locked {
		t.Fatal("LockImplementationSelection() did not call native function")
	}
}

func TestElementGetStringWarnsOnBytesFreeFailure(t *testing.T) {
	t.Setenv("PURE_SIMDJSON_WARN_LEAKS", "1")
	bytesFreeFailureWarningCount.Store(0)

	payload := []byte("hello")
	var freed bool
	b := &Bindings{
		elementGetString: func(_ *ValueView, outPtr **byte, outLen *uintptr) int32 {
			*outPtr = &payload[0]
			*outLen = uintptr(len(payload))
			return int32(OK)
		},
		bytesFree: func(_ *byte, _ uintptr) int32 {
			freed = true
			return int32(ErrInternal)
		},
	}

	stderr := captureStderr(t, func() {
		value, rc := b.ElementGetString(&ValueView{})
		if rc != int32(OK) {
			t.Fatalf("ElementGetString() rc = %d, want %d", rc, OK)
		}
		if value != "hello" {
			t.Fatalf("ElementGetString() value = %q, want %q", value, "hello")
		}
	})

	if !freed {
		t.Fatal("BytesFree() was not called")
	}
	if !strings.Contains(stderr, "purejson leak: bytes_free rc=") {
		t.Fatalf("stderr = %q, want bytes_free warning", stderr)
	}
}

func TestElementGetStringWarnsOnFirstBytesFreeFailureWithoutOptIn(t *testing.T) {
	t.Setenv("PURE_SIMDJSON_WARN_LEAKS", "0")
	bytesFreeFailureWarningCount.Store(0)

	payload := []byte("hello")
	b := &Bindings{
		elementGetString: func(_ *ValueView, outPtr **byte, outLen *uintptr) int32 {
			*outPtr = &payload[0]
			*outLen = uintptr(len(payload))
			return int32(OK)
		},
		bytesFree: func(_ *byte, _ uintptr) int32 {
			return int32(ErrInternal)
		},
	}

	stderr := captureStderr(t, func() {
		value, rc := b.ElementGetString(&ValueView{})
		if rc != int32(OK) {
			t.Fatalf("ElementGetString() rc = %d, want %d", rc, OK)
		}
		if value != "hello" {
			t.Fatalf("ElementGetString() value = %q, want %q", value, "hello")
		}
	})

	if !strings.Contains(stderr, "purejson leak: bytes_free rc=") {
		t.Fatalf("stderr = %q, want first bytes_free warning without opt-in", stderr)
	}
}

func TestElementGetStringSkipsBytesFreeForEmptyStrings(t *testing.T) {
	var freed bool
	b := &Bindings{
		elementGetString: func(_ *ValueView, outPtr **byte, outLen *uintptr) int32 {
			*outPtr = nil
			*outLen = 0
			return int32(OK)
		},
		bytesFree: func(_ *byte, _ uintptr) int32 {
			freed = true
			return int32(OK)
		},
	}

	value, rc := b.ElementGetString(&ValueView{})
	if rc != int32(OK) {
		t.Fatalf("ElementGetString() rc = %d, want %d", rc, OK)
	}
	if value != "" {
		t.Fatalf("ElementGetString() value = %q, want empty string", value)
	}
	if freed {
		t.Fatal("BytesFree() was called for an empty string")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func unsafeBytes(ptr *byte, length uintptr) []byte {
	return unsafe.Slice(ptr, length)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	defer r.Close()

	os.Stderr = w
	defer func() {
		os.Stderr = original
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("stderr writer close error = %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stderr) error = %v", err)
	}
	return string(data)
}
