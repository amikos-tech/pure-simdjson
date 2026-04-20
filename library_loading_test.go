package purejson

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
)

// The bootstrap package memoizes failures for 30s via a package-level cache.
// Tests in this file either bypass bootstrap via PURE_SIMDJSON_LIB_PATH or a
// pre-populated cache, or deliberately exercise the failure path exactly once,
// so no reset helper is needed.

// TestResolveLibraryPathAbsolute asserts that resolveLibraryPath never returns
// a relative path or bare filename — DIST-09 / pitfall #29: Windows LoadLibrary
// must always receive a full path to prevent DLL hijacking via CWD.
//
// Plan 05-06 extension: also asserts that every entry in the `attempted` slice
// (returned even on the error paths) is absolute or empty. The original Plan
// 05-04 test exercised only the success path; the new sub-tests cover the
// env-override-missing and bootstrap-failure paths where a regression would
// otherwise leak a bare filename into a Windows LoadLibrary call.
func TestResolveLibraryPathAbsolute(t *testing.T) {
	t.Run("cache-hit-success", func(t *testing.T) {
		t.Setenv(libraryEnvPath, "")
		t.Setenv("PURE_SIMDJSON_CACHE_DIR", t.TempDir())

		cachePath := bootstrap.CachePath(runtime.GOOS, runtime.GOARCH)
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(cachePath, []byte("stub"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		path, attempted, err := resolveLibraryPath()
		if err != nil {
			t.Fatalf("resolveLibraryPath() error = %v", err)
		}
		assertAllAbsoluteOrEmpty(t, path, attempted)
	})

	t.Run("env-override-missing-absolute-input", func(t *testing.T) {
		// The env path points at a file that does not exist. resolveLibraryPath
		// must return an error AND the attempted slice must contain only
		// absolute paths — never the bare filename a hostile actor might
		// substitute via CWD.
		t.Setenv(libraryEnvPath, "/absolute/path/that/does/not/exist.so")
		t.Setenv("PURE_SIMDJSON_CACHE_DIR", t.TempDir())

		path, attempted, err := resolveLibraryPath()
		if err == nil {
			t.Fatalf("resolveLibraryPath() error = nil, want missing-file failure")
		}
		assertAllAbsoluteOrEmpty(t, path, attempted)
	})

	t.Run("env-override-missing-relative-input", func(t *testing.T) {
		// A relative env path triggers filepath.Abs before stat — even on the
		// failure branch the attempted slice MUST hold the absolute form, never
		// the relative bare-filename input.
		t.Setenv(libraryEnvPath, "relative/missing.so")
		t.Setenv("PURE_SIMDJSON_CACHE_DIR", t.TempDir())

		path, attempted, err := resolveLibraryPath()
		if err == nil {
			t.Fatalf("resolveLibraryPath() error = nil, want missing-file failure")
		}
		assertAllAbsoluteOrEmpty(t, path, attempted)
	})
}

// assertAllAbsoluteOrEmpty fails the test if `path` or any entry in `attempted`
// is a non-empty relative path. Empty strings are tolerated so the helper
// composes with both success and failure paths from resolveLibraryPath.
func assertAllAbsoluteOrEmpty(t *testing.T, path string, attempted []string) {
	t.Helper()
	if path != "" && !filepath.IsAbs(path) {
		t.Errorf("returned path = %q, want absolute or empty (DIST-09)", path)
	}
	for i, p := range attempted {
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) {
			t.Errorf("attempted[%d] = %q, want absolute or empty (DIST-09)", i, p)
		}
	}
}

// TestLibPathEnvBypassesDownload asserts that PURE_SIMDJSON_LIB_PATH short-
// circuits the cache + bootstrap stages. The file at the env-provided path is
// returned verbatim (absolute) and no network I/O happens.
func TestLibPathEnvBypassesDownload(t *testing.T) {
	tempDir := t.TempDir()
	fake := filepath.Join(tempDir, "fake.so")
	if err := os.WriteFile(fake, []byte("stub"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv(libraryEnvPath, fake)
	// Even if LIB_PATH is set, point cache at a fresh TempDir so a leaked
	// bootstrap would be observable as a failure rather than a cache hit.
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", t.TempDir())

	path, attempted, err := resolveLibraryPath()
	if err != nil {
		t.Fatalf("resolveLibraryPath() error = %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("resolveLibraryPath() path = %q, want absolute", path)
	}
	if path != fake {
		t.Fatalf("resolveLibraryPath() path = %q, want %q", path, fake)
	}
	if len(attempted) != 1 || attempted[0] != fake {
		t.Fatalf("resolveLibraryPath() attempted = %v, want [%q]", attempted, fake)
	}
}

// TestResolveLibraryPathCacheHit asserts that a pre-populated cache file is
// returned without invoking bootstrap (no network call needed). The test uses
// PURE_SIMDJSON_CACHE_DIR to point the cache layout at a fresh TempDir and
// writes the platform library filename into the expected cache subdirectory.
func TestResolveLibraryPathCacheHit(t *testing.T) {
	t.Setenv(libraryEnvPath, "")
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", t.TempDir())

	cachePath := bootstrap.CachePath(runtime.GOOS, runtime.GOARCH)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("stub"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	path, _, err := resolveLibraryPath()
	if err != nil {
		t.Fatalf("resolveLibraryPath() error = %v", err)
	}
	if path != cachePath {
		t.Fatalf("resolveLibraryPath() path = %q, want %q", path, cachePath)
	}
}

// TestResolveLibraryPathBootstrapError asserts that when no cache exists, no
// LIB_PATH is set, and the mirror points at a dead loopback port, the returned
// error mentions PURE_SIMDJSON_LIB_PATH (D-21) so users know how to bypass.
func TestResolveLibraryPathBootstrapError(t *testing.T) {
	t.Setenv(libraryEnvPath, "")
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", t.TempDir())
	// Force bootstrap to fail fast: redirect R2 at a dead loopback port and
	// disable GitHub fallback so we don't hammer the network in CI.
	t.Setenv("PURE_SIMDJSON_BINARY_MIRROR", "http://127.0.0.1:1")
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	_, _, err := resolveLibraryPath()
	if err == nil {
		t.Fatalf("resolveLibraryPath() error = nil, want bootstrap failure")
	}
	if !strings.Contains(err.Error(), libraryEnvPath) {
		t.Fatalf("resolveLibraryPath() error = %q, want mention of %s (D-21)", err, libraryEnvPath)
	}
}

// TestActiveLibraryLockScope asserts M1 — activeLibrary must call
// resolveLibraryPath() and loadLibrary() OUTSIDE libraryMu. Holding the loader
// mutex across the network-I/O-bearing stages would serialize every concurrent
// NewParser() on the first caller's bandwidth.
//
// Implementation: grep-style walk over the activeLibrary function body in the
// source file, tracking whether libraryMu.Lock is held.
func TestActiveLibraryLockScope(t *testing.T) {
	data, err := os.ReadFile("library_loading.go")
	if err != nil {
		t.Fatalf("read library_loading.go: %v", err)
	}
	src := string(data)

	// Extract the body of func activeLibrary() up to the matching closing brace.
	start := strings.Index(src, "func activeLibrary(")
	if start < 0 {
		t.Fatal("activeLibrary function not found in library_loading.go")
	}
	// Find the end of the function: naive but sufficient — the next line that
	// starts with a top-level '}' after a newline.
	rest := src[start:]
	closingRe := regexp.MustCompile(`(?m)^}\s*$`)
	loc := closingRe.FindStringIndex(rest)
	if loc == nil {
		t.Fatal("end of activeLibrary not found")
	}
	body := rest[:loc[1]]

	lineRe := regexp.MustCompile(`\r?\n`)
	lines := lineRe.Split(body, -1)

	locked := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip single-line comments that might mention libraryMu.Lock.
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(line, "libraryMu.Lock") {
			locked++
			continue
		}
		if strings.Contains(line, "libraryMu.Unlock") {
			locked--
			if locked < 0 {
				locked = 0
			}
			continue
		}
		// Skip defer libraryMu.Unlock — it pairs with the enclosing Lock() but
		// doesn't release yet; we still count it as unlock for lexical scope
		// tracking so any code under a `defer libraryMu.Unlock()` after the
		// recheck-and-install section is not considered "under the lock" for
		// this heuristic.  Callers like resolveLibraryPath must simply not
		// appear textually between Lock and a subsequent Unlock (or defer).
		if locked > 0 {
			if strings.Contains(line, "resolveLibraryPath()") {
				t.Fatalf("M1 violation at line %d: resolveLibraryPath called under libraryMu.Lock\n%s",
					i+1, line)
			}
			if strings.Contains(line, "loadLibrary(") {
				t.Fatalf("M1 violation at line %d: loadLibrary called under libraryMu.Lock\n%s",
					i+1, line)
			}
		}
	}

	// Double-checked locking fingerprint: at least two Lock acquisitions
	// (one for the fast-path read, one for the recheck-insert) must appear.
	if got := strings.Count(body, "libraryMu.Lock"); got < 2 {
		t.Fatalf("activeLibrary has %d libraryMu.Lock calls, want >=2 for double-checked locking", got)
	}
}

// TestNewParserSignatureUnchanged pins NewParser's signature: no ctx argument,
// preserving D-02. Compile-time assertion only — the call itself may fail if
// the native library is not loaded on this machine, which we ignore.
func TestNewParserSignatureUnchanged(t *testing.T) {
	var f func() (*Parser, error) = NewParser
	_ = f
}

// withLibraryCacheClearedForTest is retained from Phase 3 tests so other
// activeLibrary tests can reset the package-level cache between runs.
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

// TestActiveLibraryEnvOverrideMissingWrapsLoadFailure exercises the env-path
// branch when the pointed-at file does not exist — the error chain must still
// Unwrap to errLoadLibrary so callers can use errors.Is.
func TestActiveLibraryEnvOverrideMissingWrapsLoadFailure(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	missing := filepath.Join(t.TempDir(), "missing", "libpure_simdjson.so")
	t.Setenv(libraryEnvPath, missing)

	_, err := activeLibrary()
	if !errors.Is(err, errLoadLibrary) {
		t.Fatalf("activeLibrary() error = %v, want errors.Is(..., errLoadLibrary)", err)
	}
}

// TestActiveLibraryEnvOverrideLoadsBuiltLibrary exercises the happy-path env
// override — if a built library is available locally, setting LIB_PATH to it
// must produce a working loadedLibrary without triggering any download. This
// test is skipped in environments where cargo build --release has not been run.
func TestActiveLibraryEnvOverrideLoadsBuiltLibrary(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	libName := builtLibraryName()
	libPath := filepath.Join(projectRootForTest(t), "target", "release", libName)
	if _, err := os.Stat(libPath); err != nil {
		t.Skipf("built library not present at %s; run `cargo build --release` first", libPath)
	}
	t.Setenv(libraryEnvPath, libPath)

	library, err := activeLibrary()
	if err != nil {
		t.Fatalf("activeLibrary() error = %v", err)
	}
	if library.path == "" {
		t.Fatal("activeLibrary() returned empty library path")
	}
}

// builtLibraryName mirrors the historical platformLibraryName() helper that
// Phase 3 defined in this package and Phase 5 moved to internal/bootstrap.
// Test-only — tests still need to locate the cargo build artefact under
// target/release/ which uses these filenames.
func builtLibraryName() string {
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

func projectRootForTest(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) returned ok=false")
	}
	return filepath.Dir(thisFile)
}
