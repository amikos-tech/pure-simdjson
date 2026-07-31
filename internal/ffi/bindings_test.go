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

var phase12RequiredSymbols = []string{
	"pure_simdjson_element_at_pointer",
	"pure_simdjson_element_at_path",
	"pure_simdjson_element_at_path_wildcard",
	"pure_simdjson_value_views_free",
	"pure_simdjson_array_at",
	"pure_simdjson_array_len",
	"pure_simdjson_object_size",
	"pure_simdjson_minify",
	"pure_simdjson_validate_utf8",
}

var abi13RequiredSymbols = []string{
	"pure_simdjson_get_abi_version",
	"pure_simdjson_set_implementation",
	"pure_simdjson_lock_implementation_selection",
	"pure_simdjson_get_implementation_name_len",
	"pure_simdjson_copy_implementation_name",
	"pure_simdjson_parser_new",
	"pure_simdjson_parser_new_configured",
	"pure_simdjson_parser_free",
	"pure_simdjson_parser_parse",
	"pure_simdjson_parser_get_last_error_len",
	"pure_simdjson_parser_copy_last_error",
	"pure_simdjson_parser_get_last_error_offset",
	"pure_simdjson_parser_get_last_error_has_offset",
	"pure_simdjson_doc_free",
	"pure_simdjson_doc_root",
	"pure_simdjson_element_type",
	"pure_simdjson_element_get_int64",
	"pure_simdjson_element_get_uint64",
	"pure_simdjson_element_get_float64",
	"pure_simdjson_element_get_string",
	"pure_simdjson_element_get_bigint",
	"pure_simdjson_bytes_free",
	"pure_simdjson_element_get_bool",
	"pure_simdjson_element_is_null",
	"pure_simdjson_array_iter_new",
	"pure_simdjson_array_iter_next",
	"pure_simdjson_object_iter_new",
	"pure_simdjson_object_iter_next",
	"pure_simdjson_object_get_field",
	"pure_simdjson_element_at_pointer",
	"pure_simdjson_element_at_path",
	"pure_simdjson_element_at_path_wildcard",
	"pure_simdjson_value_views_free",
	"pure_simdjson_array_at",
	"pure_simdjson_array_len",
	"pure_simdjson_object_size",
	"pure_simdjson_minify",
	"pure_simdjson_validate_utf8",
}

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
				*out = 0x00010003
				return int32(OK)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ProbeABI() error = %v", err)
	}
	if actual != 0x00010003 {
		t.Fatalf("ProbeABI() = 0x%08x, want 0x00010003", actual)
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

func TestBindRequiresEveryPhase12Symbol(t *testing.T) {
	for _, missing := range phase12RequiredSymbols {
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

func TestBindLooksUpCompleteABI13Surface(t *testing.T) {
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

	if len(lookups) < len(abi13RequiredSymbols) {
		t.Fatalf("Bind() lookups = %v, want at least %d required symbols", lookups, len(abi13RequiredSymbols))
	}
	if got := lookups[:len(abi13RequiredSymbols)]; !reflect.DeepEqual(got, abi13RequiredSymbols) {
		t.Fatalf("Bind() required lookups = %v, want %v", got, abi13RequiredSymbols)
	}
	for _, name := range phase12RequiredSymbols {
		if !containsString(lookups, name) {
			t.Errorf("Bind() lookups = %v, missing Phase 12 symbol %q", lookups, name)
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

func TestElementNavigationMarshalsPathBytes(t *testing.T) {
	t.Run("pointer", func(t *testing.T) {
		view := &ValueView{Doc: 11}
		want := ValueView{Doc: 11, State0: 7}
		b := &Bindings{
			elementAtPointer: func(gotView *ValueView, ptr *byte, length uintptr, out *ValueView) int32 {
				if gotView != view {
					t.Fatalf("view = %p, want %p", gotView, view)
				}
				if got := string(unsafeBytes(ptr, length)); got != "/a~1b" {
					t.Fatalf("pointer = %q, want %q", got, "/a~1b")
				}
				*out = want
				return int32(OK)
			},
		}

		got, rc := b.ElementAtPointer(view, "/a~1b")
		if rc != int32(OK) || got != want {
			t.Fatalf("ElementAtPointer() = (%+v, %d), want (%+v, %d)", got, rc, want, OK)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		view := &ValueView{Doc: 12}
		want := ValueView{Doc: 12, State0: 1}
		b := &Bindings{
			elementAtPath: func(gotView *ValueView, ptr *byte, length uintptr, out *ValueView) int32 {
				if gotView != view {
					t.Fatalf("view = %p, want %p", gotView, view)
				}
				if ptr != nil || length != 0 {
					t.Fatalf("empty path = (%p, %d), want (nil, 0)", ptr, length)
				}
				*out = want
				return int32(OK)
			},
		}

		got, rc := b.ElementAtPath(view, "")
		if rc != int32(OK) || got != want {
			t.Fatalf("ElementAtPath() = (%+v, %d), want (%+v, %d)", got, rc, want, OK)
		}
	})
}

func TestElementAtPathWildcardCopiesAndFreesExactlyOnce(t *testing.T) {
	nativeViews := []ValueView{
		{Doc: 21, State0: 1, KindHint: uint32(ValueKindString)},
		{Doc: 21, State0: 2, KindHint: uint32(ValueKindInt64)},
	}
	firstView := nativeViews[0]
	freeCalls := 0
	b := &Bindings{
		elementAtPathWildcard: func(_ *ValueView, path *byte, pathLen uintptr, out **ValueView, count *uintptr) int32 {
			if got := string(unsafeBytes(path, pathLen)); got != ".items[*]" {
				t.Fatalf("path = %q, want %q", got, ".items[*]")
			}
			*out = unsafe.SliceData(nativeViews)
			*count = uintptr(len(nativeViews))
			return int32(OK)
		},
		valueViewsFree: func(ptr *ValueView, count uintptr) int32 {
			freeCalls++
			if ptr != unsafe.SliceData(nativeViews) || count != uintptr(len(nativeViews)) {
				t.Fatalf("ValueViewsFree() = (%p, %d), want (%p, %d)", ptr, count, unsafe.SliceData(nativeViews), len(nativeViews))
			}
			nativeViews[0] = ValueView{}
			return int32(OK)
		},
	}

	views, rc := b.ElementAtPathWildcard(&ValueView{}, ".items[*]")
	if rc != int32(OK) {
		t.Fatalf("ElementAtPathWildcard() rc = %d, want %d", rc, OK)
	}
	if freeCalls != 1 {
		t.Fatalf("ValueViewsFree() calls = %d, want 1", freeCalls)
	}
	if len(views) != 2 || views[0] != firstView {
		t.Fatalf("ElementAtPathWildcard() views = %+v, want copied views beginning with %+v", views, firstView)
	}
}

func TestElementAtPathWildcardValidatesPointerCountPairs(t *testing.T) {
	t.Run("nil zero returns non-nil empty", func(t *testing.T) {
		b := &Bindings{
			elementAtPathWildcard: func(_ *ValueView, _ *byte, _ uintptr, out **ValueView, count *uintptr) int32 {
				*out = nil
				*count = 0
				return int32(OK)
			},
			valueViewsFree: func(*ValueView, uintptr) int32 {
				t.Fatal("ValueViewsFree() called for (nil, 0)")
				return int32(OK)
			},
		}

		views, rc := b.ElementAtPathWildcard(&ValueView{}, ".items[*]")
		if rc != int32(OK) {
			t.Fatalf("ElementAtPathWildcard() rc = %d, want %d", rc, OK)
		}
		if views == nil || len(views) != 0 {
			t.Fatalf("ElementAtPathWildcard() views = %#v, want non-nil empty slice", views)
		}
	})

	t.Run("nil nonzero fails before slicing", func(t *testing.T) {
		b := &Bindings{
			elementAtPathWildcard: func(_ *ValueView, _ *byte, _ uintptr, out **ValueView, count *uintptr) int32 {
				*out = nil
				*count = 2
				return int32(OK)
			},
			valueViewsFree: func(*ValueView, uintptr) int32 {
				t.Fatal("ValueViewsFree() called for (nil, 2)")
				return int32(OK)
			},
		}

		views, rc := b.ElementAtPathWildcard(&ValueView{}, ".items[*]")
		if rc != int32(ErrInternal) || views != nil {
			t.Fatalf("ElementAtPathWildcard() = (%#v, %d), want (nil, %d)", views, rc, ErrInternal)
		}
	})

	t.Run("nonnull zero frees and fails before slicing", func(t *testing.T) {
		t.Setenv("PURE_SIMDJSON_WARN_LEAKS", "1")
		valueViewsFreeFailureWarningCount.Store(0)

		view := ValueView{Doc: 31}
		freeCalls := 0
		b := &Bindings{
			elementAtPathWildcard: func(_ *ValueView, _ *byte, _ uintptr, out **ValueView, count *uintptr) int32 {
				*out = &view
				*count = 0
				return int32(OK)
			},
			valueViewsFree: func(ptr *ValueView, count uintptr) int32 {
				freeCalls++
				if ptr != &view || count != 0 {
					t.Fatalf("ValueViewsFree() = (%p, %d), want (%p, 0)", ptr, count, &view)
				}
				return int32(ErrInvalidArg)
			},
		}

		stderr := captureStderr(t, func() {
			views, rc := b.ElementAtPathWildcard(&ValueView{}, ".items[*]")
			if rc != int32(ErrInternal) || views != nil {
				t.Fatalf("ElementAtPathWildcard() = (%#v, %d), want (nil, %d)", views, rc, ErrInternal)
			}
		})
		if freeCalls != 1 {
			t.Fatalf("ValueViewsFree() calls = %d, want 1", freeCalls)
		}
		if !strings.Contains(stderr, "purejson leak: value_views_free rc=") {
			t.Fatalf("stderr = %q, want value_views_free warning", stderr)
		}
	})
}

func TestElementAtPathWildcardWarnsOnViewArrayFreeFailure(t *testing.T) {
	t.Setenv("PURE_SIMDJSON_WARN_LEAKS", "1")
	valueViewsFreeFailureWarningCount.Store(0)

	nativeViews := []ValueView{{Doc: 41, State0: 1}}
	freeCalls := 0
	b := &Bindings{
		elementAtPathWildcard: func(_ *ValueView, _ *byte, _ uintptr, out **ValueView, count *uintptr) int32 {
			*out = unsafe.SliceData(nativeViews)
			*count = uintptr(len(nativeViews))
			return int32(OK)
		},
		valueViewsFree: func(*ValueView, uintptr) int32 {
			freeCalls++
			return int32(ErrInternal)
		},
	}

	stderr := captureStderr(t, func() {
		views, rc := b.ElementAtPathWildcard(&ValueView{}, ".items[*]")
		if rc != int32(OK) || len(views) != 1 {
			t.Fatalf("ElementAtPathWildcard() = (%+v, %d), want one copied view and OK", views, rc)
		}
	})
	if freeCalls != 1 {
		t.Fatalf("ValueViewsFree() calls = %d, want 1", freeCalls)
	}
	if !strings.Contains(stderr, "purejson leak: value_views_free rc=") {
		t.Fatalf("stderr = %q, want value_views_free warning", stderr)
	}
	if strings.Contains(stderr, "purejson leak: bytes_free rc=") {
		t.Fatalf("stderr = %q, must not use bytes_free warning", stderr)
	}
}

func TestArrayAndObjectWrappersPreserveNativeValues(t *testing.T) {
	view := &ValueView{Doc: 51}
	wantView := ValueView{Doc: 51, State0: 9}
	b := &Bindings{
		arrayAt: func(gotView *ValueView, index uint64, out *ValueView) int32 {
			if gotView != view || index != uint64(1)<<40 {
				t.Fatalf("arrayAt(view, index) = (%p, %d), want (%p, %d)", gotView, index, view, uint64(1)<<40)
			}
			*out = wantView
			return int32(OK)
		},
		arrayLen: func(gotView *ValueView, out *uint64) int32 {
			if gotView != view {
				t.Fatalf("arrayLen view = %p, want %p", gotView, view)
			}
			*out = uint64(1) << 40
			return int32(OK)
		},
		objectSize: func(gotView *ValueView, out *uint64) int32 {
			if gotView != view {
				t.Fatalf("objectSize view = %p, want %p", gotView, view)
			}
			*out = 17
			return int32(OK)
		},
	}

	gotView, rc := b.ArrayAt(view, uint64(1)<<40)
	if rc != int32(OK) || gotView != wantView {
		t.Fatalf("ArrayAt() = (%+v, %d), want (%+v, %d)", gotView, rc, wantView, OK)
	}
	length, rc := b.ArrayLen(view)
	if rc != int32(OK) || length != uint64(1)<<40 {
		t.Fatalf("ArrayLen() = (%d, %d), want (%d, %d)", length, rc, uint64(1)<<40, OK)
	}
	size, rc := b.ObjectSize(view)
	if rc != int32(OK) || size != 17 {
		t.Fatalf("ObjectSize() = (%d, %d), want (17, %d)", size, rc, OK)
	}
}

func TestMinifyPreservesSourceDestinationOrderAndCapacity(t *testing.T) {
	t.Run("non-empty", func(t *testing.T) {
		src := []byte("{ \"a\" : 1 }")
		dst := make([]byte, len(src)+8)
		b := &Bindings{
			minify: func(srcPtr *byte, srcLen uintptr, dstPtr *byte, dstCap uintptr, outWritten *uintptr) int32 {
				if srcPtr != unsafe.SliceData(src) || srcLen != uintptr(len(src)) {
					t.Fatalf("source = (%p, %d), want (%p, %d)", srcPtr, srcLen, unsafe.SliceData(src), len(src))
				}
				if dstPtr != unsafe.SliceData(dst) || dstCap != uintptr(len(dst)) {
					t.Fatalf("destination = (%p, %d), want (%p, %d)", dstPtr, dstCap, unsafe.SliceData(dst), len(dst))
				}
				*outWritten = 7
				return int32(OK)
			},
		}

		written, rc := b.Minify(dst, src)
		if rc != int32(OK) || written != 7 {
			t.Fatalf("Minify() = (%d, %d), want (7, %d)", written, rc, OK)
		}
	})

	t.Run("empty", func(t *testing.T) {
		b := &Bindings{
			minify: func(srcPtr *byte, srcLen uintptr, dstPtr *byte, dstCap uintptr, outWritten *uintptr) int32 {
				if srcPtr != nil || srcLen != 0 || dstPtr != nil || dstCap != 0 {
					t.Fatalf("empty Minify native args = (%p, %d, %p, %d), want nil/zero pairs", srcPtr, srcLen, dstPtr, dstCap)
				}
				*outWritten = 0
				return int32(OK)
			},
		}

		written, rc := b.Minify(nil, nil)
		if rc != int32(OK) || written != 0 {
			t.Fatalf("Minify(nil, nil) = (%d, %d), want (0, %d)", written, rc, OK)
		}
	})
}

func TestValidateUTF8ReturnsValiditySeparatelyFromStatus(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		data := []byte("valid")
		b := &Bindings{
			validateUTF8: func(ptr *byte, length uintptr, out *byte) int32 {
				if ptr != unsafe.SliceData(data) || length != uintptr(len(data)) {
					t.Fatalf("data = (%p, %d), want (%p, %d)", ptr, length, unsafe.SliceData(data), len(data))
				}
				*out = 1
				return int32(OK)
			},
		}

		valid, rc := b.ValidateUTF8(data)
		if rc != int32(OK) || !valid {
			t.Fatalf("ValidateUTF8() = (%t, %d), want (true, %d)", valid, rc, OK)
		}
	})

	t.Run("empty invalid result with successful status", func(t *testing.T) {
		b := &Bindings{
			validateUTF8: func(ptr *byte, length uintptr, out *byte) int32 {
				if ptr != nil || length != 0 {
					t.Fatalf("empty data = (%p, %d), want (nil, 0)", ptr, length)
				}
				*out = 0
				return int32(OK)
			},
		}

		valid, rc := b.ValidateUTF8(nil)
		if rc != int32(OK) || valid {
			t.Fatalf("ValidateUTF8(nil) = (%t, %d), want (false, %d)", valid, rc, OK)
		}
	})
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
