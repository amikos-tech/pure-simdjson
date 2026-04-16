package purejson

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLibraryPathUsesDebugFallback(t *testing.T) {
	wd := mustChdir(t, t.TempDir())
	defer wd()

	debugPath := filepath.Join("target", "debug", platformLibraryName())
	if err := os.MkdirAll(filepath.Dir(debugPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(debugPath), err)
	}
	if err := os.WriteFile(debugPath, []byte("stub"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", debugPath, err)
	}

	path, attempted, err := resolveLibraryPath()
	if err != nil {
		t.Fatalf("resolveLibraryPath() error = %v", err)
	}

	wantPath, err := filepath.Abs(debugPath)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", debugPath, err)
	}
	if path != wantPath {
		t.Fatalf("resolveLibraryPath() path = %q, want %q", path, wantPath)
	}
	if len(attempted) < 2 {
		t.Fatalf("resolveLibraryPath() attempted = %v, want at least release and debug candidates", attempted)
	}
}

func TestActiveLibraryEnvOverrideMissingWrapsLoadFailure(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	missing := filepath.Join(t.TempDir(), "missing", platformLibraryName())
	t.Setenv(libraryEnvPath, missing)

	_, err := activeLibrary()
	if !errors.Is(err, errLoadLibrary) {
		t.Fatalf("activeLibrary() error = %v, want errors.Is(..., errLoadLibrary)", err)
	}
	if !strings.Contains(err.Error(), "attempted paths:") {
		t.Fatalf("activeLibrary() error = %q, want attempted paths list", err)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("activeLibrary() error = %q, want missing path %q", err, missing)
	}
}

func TestActiveLibrarySearchMissReportsAttemptedPaths(t *testing.T) {
	restoreCache := withLibraryCacheClearedForTest(t)
	defer restoreCache()
	restoreWD := mustChdir(t, t.TempDir())
	defer restoreWD()

	t.Setenv(libraryEnvPath, "")

	_, err := activeLibrary()
	if !errors.Is(err, errLoadLibrary) {
		t.Fatalf("activeLibrary() error = %v, want errors.Is(..., errLoadLibrary)", err)
	}
	if !strings.Contains(err.Error(), "attempted paths:") {
		t.Fatalf("activeLibrary() error = %q, want attempted paths list", err)
	}
	if !strings.Contains(err.Error(), platformLibraryName()) {
		t.Fatalf("activeLibrary() error = %q, want platform library name %q", err, platformLibraryName())
	}
}

func TestActiveLibraryEnvOverrideLoadsBuiltLibrary(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	t.Setenv(libraryEnvPath, filepath.Join("target", "release", platformLibraryName()))

	library, err := activeLibrary()
	if err != nil {
		t.Fatalf("activeLibrary() error = %v", err)
	}
	if library.path == "" {
		t.Fatal("activeLibrary() returned empty library path")
	}
}

func TestActiveLibraryEnvOverrideInvalidLibraryWrapsLoadFailure(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	tempDir := t.TempDir()
	invalidLibrary := filepath.Join(tempDir, platformLibraryName())
	if err := os.WriteFile(invalidLibrary, []byte("not-a-shared-library"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", invalidLibrary, err)
	}
	t.Setenv(libraryEnvPath, invalidLibrary)

	_, err := activeLibrary()
	if !errors.Is(err, errLoadLibrary) {
		t.Fatalf("activeLibrary() error = %v, want errors.Is(..., errLoadLibrary)", err)
	}
	if !strings.Contains(err.Error(), "attempted paths:") {
		t.Fatalf("activeLibrary() error = %q, want attempted paths list", err)
	}
	if !strings.Contains(err.Error(), invalidLibrary) {
		t.Fatalf("activeLibrary() error = %q, want invalid library path %q", err, invalidLibrary)
	}
}

func TestResolveLibraryPathPreservesCandidatePathError(t *testing.T) {
	restoreWD := mustChdir(t, t.TempDir())
	defer restoreWD()

	if err := os.WriteFile("target", []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", "target", err)
	}

	_, attempted, err := resolveLibraryPath()
	if err == nil {
		t.Fatal("resolveLibraryPath() error = nil, want path error")
	}
	if len(attempted) == 0 {
		t.Fatal("resolveLibraryPath() attempted = nil, want attempted candidates")
	}

	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("resolveLibraryPath() error = %v, want fs.PathError", err)
	}
}

func withLibraryCacheClearedForTest(t *testing.T) func() {
	t.Helper()

	libraryMu.Lock()
	previous := cachedLibrary
	cachedLibrary = nil
	libraryMu.Unlock()

	return func() {
		libraryMu.Lock()
		cachedLibrary = previous
		libraryMu.Unlock()
	}
}

func mustChdir(t *testing.T, dir string) func() {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", dir, err)
	}

	return func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("os.Chdir(%q) restore error = %v", previous, err)
		}
	}
}
