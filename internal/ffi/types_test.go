package ffi

import (
	"testing"
	"unsafe"
)

func TestABINumericContract(t *testing.T) {
	if ABIVersion != 0x00010003 {
		t.Fatalf("ABIVersion = %#08x, want 0x00010003", ABIVersion)
	}
	if LastErrorOffsetUnknown != ^uint64(0) {
		t.Fatalf("LastErrorOffsetUnknown = %#x, want uint64 max", LastErrorOffsetUnknown)
	}

	errorCodes := []struct {
		name string
		got  ErrorCode
		want ErrorCode
	}{
		{"OK", OK, 0},
		{"ErrInvalidArg", ErrInvalidArg, 1},
		{"ErrInvalidHandle", ErrInvalidHandle, 2},
		{"ErrParserBusy", ErrParserBusy, 3},
		{"ErrWrongType", ErrWrongType, 4},
		{"ErrElementNotFound", ErrElementNotFound, 5},
		{"ErrBufferTooSmall", ErrBufferTooSmall, 6},
		{"ErrNotImplemented", ErrNotImplemented, 7},
		{"ErrDepthLimit", ErrDepthLimit, 8},
		{"ErrCapacityLimit", ErrCapacityLimit, 9},
		{"ErrKernelLocked", ErrKernelLocked, 10},
		{"ErrInvalidPath", ErrInvalidPath, 11},
		{"ErrIndexOutOfRange", ErrIndexOutOfRange, 12},
		{"ErrInvalidJSON", ErrInvalidJSON, 32},
		{"ErrNumberOutOfRange", ErrNumberOutOfRange, 33},
		{"ErrPrecisionLoss", ErrPrecisionLoss, 34},
		{"ErrCPUUnsupported", ErrCPUUnsupported, 64},
		{"ErrABIMismatch", ErrABIMismatch, 65},
		{"ErrPanic", ErrPanic, 96},
		{"ErrCPPException", ErrCPPException, 97},
		{"ErrInternal", ErrInternal, 127},
	}
	for _, code := range errorCodes {
		t.Run(code.name, func(t *testing.T) {
			if code.got != code.want {
				t.Fatalf("%s = %d, want %d", code.name, code.got, code.want)
			}
		})
	}

	valueKinds := []struct {
		name string
		got  ValueKind
		want ValueKind
	}{
		{"ValueKindInvalid", ValueKindInvalid, 0},
		{"ValueKindNull", ValueKindNull, 1},
		{"ValueKindBool", ValueKindBool, 2},
		{"ValueKindInt64", ValueKindInt64, 3},
		{"ValueKindUint64", ValueKindUint64, 4},
		{"ValueKindFloat64", ValueKindFloat64, 5},
		{"ValueKindString", ValueKindString, 6},
		{"ValueKindArray", ValueKindArray, 7},
		{"ValueKindObject", ValueKindObject, 8},
		{"ValueKindBigInt", ValueKindBigInt, 9},
	}
	for _, kind := range valueKinds {
		t.Run(kind.name, func(t *testing.T) {
			if kind.got != kind.want {
				t.Fatalf("%s = %d, want %d", kind.name, kind.got, kind.want)
			}
		})
	}
}

func TestValueViewLayout(t *testing.T) {
	var view ValueView

	assertSize(t, "ValueView", unsafe.Sizeof(view), 32)
	assertOffset(t, "ValueView.Doc", unsafe.Offsetof(view.Doc), 0)
	assertOffset(t, "ValueView.State0", unsafe.Offsetof(view.State0), 8)
	assertOffset(t, "ValueView.State1", unsafe.Offsetof(view.State1), 16)
	assertOffset(t, "ValueView.KindHint", unsafe.Offsetof(view.KindHint), 24)
	assertOffset(t, "ValueView.Reserved", unsafe.Offsetof(view.Reserved), 28)
}

func TestIteratorLayouts(t *testing.T) {
	var arrayIter ArrayIter
	var objectIter ObjectIter

	assertSize(t, "ArrayIter", unsafe.Sizeof(arrayIter), 32)
	assertOffset(t, "ArrayIter.Doc", unsafe.Offsetof(arrayIter.Doc), 0)
	assertOffset(t, "ArrayIter.State0", unsafe.Offsetof(arrayIter.State0), 8)
	assertOffset(t, "ArrayIter.State1", unsafe.Offsetof(arrayIter.State1), 16)
	assertOffset(t, "ArrayIter.Index", unsafe.Offsetof(arrayIter.Index), 24)
	assertOffset(t, "ArrayIter.Tag", unsafe.Offsetof(arrayIter.Tag), 28)
	assertOffset(t, "ArrayIter.Reserved", unsafe.Offsetof(arrayIter.Reserved), 30)

	assertSize(t, "ObjectIter", unsafe.Sizeof(objectIter), 32)
	assertOffset(t, "ObjectIter.Doc", unsafe.Offsetof(objectIter.Doc), 0)
	assertOffset(t, "ObjectIter.State0", unsafe.Offsetof(objectIter.State0), 8)
	assertOffset(t, "ObjectIter.State1", unsafe.Offsetof(objectIter.State1), 16)
	assertOffset(t, "ObjectIter.Index", unsafe.Offsetof(objectIter.Index), 24)
	assertOffset(t, "ObjectIter.Tag", unsafe.Offsetof(objectIter.Tag), 28)
	assertOffset(t, "ObjectIter.Reserved", unsafe.Offsetof(objectIter.Reserved), 30)
}

func TestNativeAllocStatsLayout(t *testing.T) {
	var stats NativeAllocStats

	assertSize(t, "NativeAllocStats", unsafe.Sizeof(stats), 48)
	assertOffset(t, "NativeAllocStats.Epoch", unsafe.Offsetof(stats.Epoch), 0)
	assertOffset(t, "NativeAllocStats.LiveBytes", unsafe.Offsetof(stats.LiveBytes), 8)
	assertOffset(t, "NativeAllocStats.TotalAllocBytes", unsafe.Offsetof(stats.TotalAllocBytes), 16)
	assertOffset(t, "NativeAllocStats.AllocCount", unsafe.Offsetof(stats.AllocCount), 24)
	assertOffset(t, "NativeAllocStats.FreeCount", unsafe.Offsetof(stats.FreeCount), 32)
	assertOffset(t, "NativeAllocStats.UntrackedFreeCount", unsafe.Offsetof(stats.UntrackedFreeCount), 40)
}

func TestInternalFrameLayout(t *testing.T) {
	var frame InternalFrame

	if got := unsafe.Sizeof(frame); got != 72 {
		t.Fatalf("unsafe.Sizeof(InternalFrame{}) = %d, want 72", got)
	}
	if got := unsafe.Offsetof(frame.Kind); got != 0 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.Kind) = %d, want 0", got)
	}
	if got := unsafe.Offsetof(frame.Flags); got != 4 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.Flags) = %d, want 4", got)
	}
	if got := unsafe.Offsetof(frame.ChildCount); got != 8 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.ChildCount) = %d, want 8", got)
	}
	if got := unsafe.Offsetof(frame.Reserved); got != 12 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.Reserved) = %d, want 12", got)
	}
	if got := unsafe.Offsetof(frame.KeyPtr); got != 16 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.KeyPtr) = %d, want 16", got)
	}
	if got := unsafe.Offsetof(frame.KeyLen); got != 24 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.KeyLen) = %d, want 24", got)
	}
	if got := unsafe.Offsetof(frame.StringPtr); got != 32 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.StringPtr) = %d, want 32", got)
	}
	if got := unsafe.Offsetof(frame.StringLen); got != 40 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.StringLen) = %d, want 40", got)
	}
	if got := unsafe.Offsetof(frame.Int64Value); got != 48 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.Int64Value) = %d, want 48", got)
	}
	if got := unsafe.Offsetof(frame.Uint64Value); got != 56 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.Uint64Value) = %d, want 56", got)
	}
	if got := unsafe.Offsetof(frame.Float64Value); got != 64 {
		t.Fatalf("unsafe.Offsetof(InternalFrame{}.Float64Value) = %d, want 64", got)
	}
}

func TestInternalMaterializeBuildReturnsBorrowedFrames(t *testing.T) {
	view := &ValueView{}
	nativeFrames := []InternalFrame{
		{Kind: uint32(ValueKindObject), ChildCount: 1},
		{Kind: uint32(ValueKindString)},
	}
	b := &Bindings{
		internalMaterializeBuild: func(gotView *ValueView, outFrames **InternalFrame, outCount *uintptr) int32 {
			if gotView != view {
				t.Fatalf("view = %p, want %p", gotView, view)
			}
			*outFrames = &nativeFrames[0]
			*outCount = uintptr(len(nativeFrames))
			return int32(OK)
		},
	}

	frames, rc := b.InternalMaterializeBuild(view)
	if rc != int32(OK) {
		t.Fatalf("InternalMaterializeBuild() rc = %d, want %d", rc, OK)
	}
	if len(frames) != len(nativeFrames) {
		t.Fatalf("len(frames) = %d, want %d", len(frames), len(nativeFrames))
	}
	if &frames[0] != &nativeFrames[0] {
		t.Fatal("InternalMaterializeBuild() copied the frame span")
	}
}

func TestInternalMaterializeBuildNilPointerWithCountReturnsInternal(t *testing.T) {
	b := &Bindings{
		internalMaterializeBuild: func(_ *ValueView, outFrames **InternalFrame, outCount *uintptr) int32 {
			*outFrames = nil
			*outCount = 1
			return int32(OK)
		},
	}

	frames, rc := b.InternalMaterializeBuild(&ValueView{})
	if rc != int32(ErrInternal) {
		t.Fatalf("InternalMaterializeBuild() rc = %d, want %d", rc, ErrInternal)
	}
	if frames != nil {
		t.Fatalf("InternalMaterializeBuild() frames = %v, want nil", frames)
	}
}

func TestInternalMaterializeBuildAllowsNilPointerWhenCountIsZero(t *testing.T) {
	b := &Bindings{
		internalMaterializeBuild: func(_ *ValueView, outFrames **InternalFrame, outCount *uintptr) int32 {
			*outFrames = nil
			*outCount = 0
			return int32(OK)
		},
	}

	frames, rc := b.InternalMaterializeBuild(&ValueView{})
	if rc != int32(OK) {
		t.Fatalf("InternalMaterializeBuild() rc = %d, want %d", rc, OK)
	}
	if len(frames) != 0 {
		t.Fatalf("len(frames) = %d, want 0", len(frames))
	}
}

func assertSize(t *testing.T, name string, got, want uintptr) {
	t.Helper()
	if got != want {
		t.Fatalf("unsafe.Sizeof(%s{}) = %d, want %d", name, got, want)
	}
}

func assertOffset(t *testing.T, name string, got, want uintptr) {
	t.Helper()
	if got != want {
		t.Fatalf("unsafe.Offsetof(%s) = %d, want %d", name, got, want)
	}
}
