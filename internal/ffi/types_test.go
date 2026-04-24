package ffi

import (
	"testing"
	"unsafe"
)

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
