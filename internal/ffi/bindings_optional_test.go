package ffi

import (
	"errors"
	"strings"
	"testing"
)

func TestRegisterOptionalFuncMissingSymbolDoesNotFailBinding(t *testing.T) {
	t.Setenv("PURE_SIMDJSON_DEBUG", "1")
	var target func(*ValueView, **InternalFrame, *uintptr) int32

	stderr := captureStderr(t, func() {
		registered, err := registerOptionalFunc(1, func(uintptr, string) (uintptr, error) {
			return 0, errors.New("symbol not found")
		}, "psdj_internal_materialize_build", &target)
		if err != nil {
			t.Fatalf("registerOptionalFunc() error = %v, want nil", err)
		}
		if registered {
			t.Fatal("registerOptionalFunc() registered = true, want false")
		}
	})

	if target != nil {
		t.Fatal("registerOptionalFunc() target != nil, want nil")
	}
	if !strings.Contains(stderr, "purejson debug: optional symbol psdj_internal_materialize_build unavailable: symbol not found") {
		t.Fatalf("stderr = %q, want optional-symbol debug breadcrumb", stderr)
	}
}

func TestRegisterOptionalFuncMissingSymbolIsQuietWithoutDebug(t *testing.T) {
	var target func(*ValueView, **InternalFrame, *uintptr) int32

	stderr := captureStderr(t, func() {
		registered, err := registerOptionalFunc(1, func(uintptr, string) (uintptr, error) {
			return 0, errors.New("symbol not found")
		}, "psdj_internal_materialize_build", &target)
		if err != nil {
			t.Fatalf("registerOptionalFunc() error = %v, want nil", err)
		}
		if registered {
			t.Fatal("registerOptionalFunc() registered = true, want false")
		}
	})

	if stderr != "" {
		t.Fatalf("stderr = %q, want quiet optional-symbol lookup without debug", stderr)
	}
	if target != nil {
		t.Fatal("registerOptionalFunc() target != nil, want nil")
	}
}

func TestRegisterOptionalFuncRegisterFailureStillFailsBinding(t *testing.T) {
	var target int

	registered, err := registerOptionalFunc(1, func(uintptr, string) (uintptr, error) {
		return 1, nil
	}, "psdj_internal_materialize_build", &target)
	if err == nil {
		t.Fatal("registerOptionalFunc() error = nil, want register failure")
	}
	if registered {
		t.Fatal("registerOptionalFunc() registered = true, want false")
	}
}

func TestRegisterOptionalFuncMissingSymbolKeepsTargetNil(t *testing.T) {
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

func TestNativeAllocStatsUnavailableReturnsNotImplemented(t *testing.T) {
	var b Bindings

	if b.HasNativeAllocStats() {
		t.Fatal("HasNativeAllocStats() = true, want false")
	}

	if rc := b.NativeAllocStatsReset(); rc != int32(ErrNotImplemented) {
		t.Fatalf("NativeAllocStatsReset() rc = %d, want %d", rc, ErrNotImplemented)
	}

	stats, rc := b.NativeAllocStatsSnapshot()
	if rc != int32(ErrNotImplemented) {
		t.Fatalf("NativeAllocStatsSnapshot() rc = %d, want %d", rc, ErrNotImplemented)
	}
	if stats != (NativeAllocStats{}) {
		t.Fatalf("NativeAllocStatsSnapshot() stats = %+v, want zero value", stats)
	}
}
