package ffi

import (
	"errors"
	"testing"
)

func TestRegisterOptionalFuncMissingSymbolDoesNotFailBinding(t *testing.T) {
	var target func(*ValueView, **InternalFrame, *uintptr) int32

	registered, err := registerOptionalFunc(1, func(uintptr, string) (uintptr, error) {
		return 0, errors.New("symbol not found")
	}, "psdj_internal_materialize_build", &target)
	if err != nil {
		t.Fatalf("registerOptionalFunc() error = %v, want nil", err)
	}
	if registered {
		t.Fatal("registerOptionalFunc() registered = true, want false")
	}
	if target != nil {
		t.Fatal("registerOptionalFunc() target != nil, want nil")
	}
}

func TestInternalMaterializeBuildUnavailableReturnsInternal(t *testing.T) {
	var b Bindings

	if b.HasInternalMaterializeBuild() {
		t.Fatal("HasInternalMaterializeBuild() = true, want false")
	}

	frames, rc := b.InternalMaterializeBuild(&ValueView{})
	if rc != int32(ErrInternal) {
		t.Fatalf("InternalMaterializeBuild() rc = %d, want %d", rc, ErrInternal)
	}
	if frames != nil {
		t.Fatalf("InternalMaterializeBuild() frames = %v, want nil", frames)
	}
}

func TestNativeAllocStatsUnavailableReturnsInternal(t *testing.T) {
	var b Bindings

	if b.HasNativeAllocStats() {
		t.Fatal("HasNativeAllocStats() = true, want false")
	}

	if rc := b.NativeAllocStatsReset(); rc != int32(ErrInternal) {
		t.Fatalf("NativeAllocStatsReset() rc = %d, want %d", rc, ErrInternal)
	}

	stats, rc := b.NativeAllocStatsSnapshot()
	if rc != int32(ErrInternal) {
		t.Fatalf("NativeAllocStatsSnapshot() rc = %d, want %d", rc, ErrInternal)
	}
	if stats != (NativeAllocStats{}) {
		t.Fatalf("NativeAllocStatsSnapshot() stats = %+v, want zero value", stats)
	}
}
