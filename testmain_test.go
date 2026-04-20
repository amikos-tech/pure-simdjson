package purejson

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain bootstraps the test environment for the root purejson package.
//
// Plan 05-04 deleted the legacy target/release auto-discovery walk from
// library_loading.go: resolveLibraryPath now chains env → cache → bootstrap.
// Pre-existing tests (TestParserPoolRoundTrip, TestParserParseInt64, …) were
// written assuming the loader would find a cargo-built library under
// target/release/ automatically. To keep those tests working without changing
// them, we set PURE_SIMDJSON_LIB_PATH to the locally-built library when one is
// present at the canonical cargo path. Tests that need to exercise resolution
// behaviour (e.g. TestResolveLibraryPathCacheHit) override LIB_PATH to ""
// via t.Setenv, so this default is a benign baseline rather than a lock-in.
func TestMain(m *testing.M) {
	if os.Getenv(libraryEnvPath) == "" {
		if builtPath, ok := findBuiltLibraryForTestMain(); ok {
			// os.Setenv is safe here — we're in TestMain, before any t.Setenv
			// frame exists. t.Setenv calls in individual tests will shadow this
			// for their duration and restore the value below on cleanup.
			_ = os.Setenv(libraryEnvPath, builtPath)
		}
	}
	os.Exit(m.Run())
}

func findBuiltLibraryForTestMain() (string, bool) {
	root, err := thisPackageDir()
	if err != nil {
		return "", false
	}
	candidate := filepath.Join(root, "target", "release", testMainLibraryName())
	absPath, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}
	if _, err := os.Stat(absPath); err != nil {
		return "", false
	}
	return absPath, true
}

func thisPackageDir() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}
	return filepath.Dir(thisFile), nil
}

// testMainLibraryName mirrors the original Phase-3 platformLibraryName() that
// Plan 05-04 moved to internal/bootstrap. Test-only; the shipped cargo target
// still emits these filenames regardless of the cache-layer naming.
func testMainLibraryName() string {
	switch runtime.GOOS {
	case "darwin":
		return "libpure_simdjson.dylib"
	case "linux":
		return "libpure_simdjson.so"
	case "windows":
		return "pure_simdjson.dll"
	default:
		return "libpure_simdjson"
	}
}
