package purejson

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
	"github.com/amikos-tech/pure-simdjson/internal/ffi"
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
	start := strings.Index(src, "func activeLibraryWithOps(")
	if start < 0 {
		t.Fatal("activeLibraryWithOps function not found in library_loading.go")
	}
	// Find the end of the function: naive but sufficient — the next line that
	// starts with a top-level '}' after a newline.
	rest := src[start:]
	closingRe := regexp.MustCompile(`(?m)^}\s*$`)
	loc := closingRe.FindStringIndex(rest)
	if loc == nil {
		t.Fatal("end of activeLibraryWithOps not found")
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
		t.Fatalf("activeLibraryWithOps has %d libraryMu.Lock calls, want >=2 for double-checked locking", got)
	}
}

// TestNewParserVariadicSignature pins the immutable option call shape while
// preserving zero-argument construction.
func TestNewParserVariadicSignature(t *testing.T) {
	var f func(...ParserOption) (*Parser, error) = NewParser
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

var phase11MandatoryFixtureSymbols = []string{
	"pure_simdjson_parser_new_configured",
	"pure_simdjson_parser_get_last_error_has_offset",
	"pure_simdjson_element_get_bigint",
	"pure_simdjson_set_implementation",
	"pure_simdjson_lock_implementation_selection",
}

type abiLoaderFixture struct {
	reportedABI        uint32
	missingSymbol      string
	implementationName string
	implementationRC   int32
	events             []string
	lookups            []string
}

func (f *abiLoaderFixture) ops(t *testing.T) libraryLoadOps {
	t.Helper()

	path := filepath.Join(t.TempDir(), "libpure_simdjson_fixture")
	lookup := func(_ uintptr, name string) (uintptr, error) {
		f.events = append(f.events, "lookup:"+name)
		f.lookups = append(f.lookups, name)
		if name == f.missingSymbol {
			return 0, errors.New("symbol not found")
		}
		return 1, nil
	}

	return libraryLoadOps{
		resolvePath: func() (string, []string, error) {
			f.events = append(f.events, "resolve")
			return path, []string{path}, nil
		},
		load: func(gotPath string) (uintptr, error) {
			f.events = append(f.events, "load")
			if gotPath != path {
				t.Fatalf("load path = %q, want %q", gotPath, path)
			}
			return 7, nil
		},
		lookup: lookup,
		probeABI: func(handle uintptr, gotLookup ffi.SymbolLookup) (uint32, error) {
			f.events = append(f.events, "probe")
			if handle != 7 {
				t.Fatalf("probe handle = %d, want 7", handle)
			}
			if _, err := gotLookup(handle, "pure_simdjson_get_abi_version"); err != nil {
				return 0, err
			}
			f.events = append(f.events, "probe-call")
			return f.reportedABI, nil
		},
		bind: func(handle uintptr, gotLookup ffi.SymbolLookup) (*ffi.Bindings, error) {
			f.events = append(f.events, "bind")
			if handle != 7 {
				t.Fatalf("bind handle = %d, want 7", handle)
			}
			for _, name := range phase11MandatoryFixtureSymbols {
				if _, err := gotLookup(handle, name); err != nil {
					return nil, fmt.Errorf("lookup %s: %w", name, err)
				}
			}
			f.events = append(f.events, "bind-complete")
			return &ffi.Bindings{}, nil
		},
		implementationName: func(*ffi.Bindings) (string, int32) {
			f.events = append(f.events, "implementation-name")
			if cachedLibrary != nil {
				t.Fatal("cachedLibrary installed before implementation-name validation")
			}
			return f.implementationName, f.implementationRC
		},
	}
}

func TestABI11RejectedBeforePhase11Lookup(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	fixture := &abiLoaderFixture{
		reportedABI:        0x00010001,
		implementationName: "fallback",
		implementationRC:   int32(ffi.OK),
	}
	_, err := activeLibraryWithOps(fixture.ops(t))
	if !errors.Is(err, ErrABIVersionMismatch) {
		t.Fatalf("activeLibrary() error = %v, want ErrABIVersionMismatch", err)
	}

	var nativeErr *Error
	if !errors.As(err, &nativeErr) {
		t.Fatalf("activeLibrary() error = %v, want *Error", err)
	}
	if nativeErr.Code() != int32(ffi.ErrABIMismatch) {
		t.Fatalf("ABI mismatch code = %d, want %d", nativeErr.Code(), ffi.ErrABIMismatch)
	}
	if nativeErr.Message() == "" {
		t.Fatal("ABI mismatch message is empty")
	}
	if want := []string{"pure_simdjson_get_abi_version"}; !reflect.DeepEqual(fixture.lookups, want) {
		t.Fatalf("ABI 1.1 lookups = %v, want %v", fixture.lookups, want)
	}
	if cachedLibrary != nil {
		t.Fatal("cachedLibrary != nil after ABI 1.1 mismatch")
	}
}

func TestABI12CompleteBindsAndCaches(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	fixture := &abiLoaderFixture{
		reportedABI:        ffi.ABIVersion,
		implementationName: "fallback",
		implementationRC:   int32(ffi.OK),
	}
	library, err := activeLibraryWithOps(fixture.ops(t))
	if err != nil {
		t.Fatalf("activeLibrary() error = %v", err)
	}
	if library == nil || cachedLibrary != library {
		t.Fatalf("cachedLibrary = %#v, loaded library = %#v; want identical non-nil pointers", cachedLibrary, library)
	}
	if library.implementationName != "fallback" {
		t.Fatalf("implementation name = %q, want fallback", library.implementationName)
	}
	assertEventBefore(t, fixture.events, "probe-call", "bind")
	assertEventBefore(t, fixture.events, "bind-complete", "implementation-name")
	for _, name := range phase11MandatoryFixtureSymbols {
		if !containsLookup(fixture.lookups, name) {
			t.Errorf("complete ABI 1.2 lookups = %v, missing %q", fixture.lookups, name)
		}
	}
}

func TestABI12IncompleteFailsClosedWithoutCache(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	const missing = "pure_simdjson_element_get_bigint"
	fixture := &abiLoaderFixture{
		reportedABI:        ffi.ABIVersion,
		missingSymbol:      missing,
		implementationName: "fallback",
		implementationRC:   int32(ffi.OK),
	}
	_, err := activeLibraryWithOps(fixture.ops(t))
	if err == nil {
		t.Fatal("activeLibrary() error = nil, want incomplete ABI 1.2 failure")
	}
	if errors.Is(err, ErrABIVersionMismatch) {
		t.Fatalf("activeLibrary() error = %v, want load failure rather than ABI mismatch", err)
	}
	if !errors.Is(err, errLoadLibrary) {
		t.Fatalf("activeLibrary() error = %v, want errLoadLibrary", err)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("activeLibrary() error = %q, want missing symbol %q", err, missing)
	}
	if cachedLibrary != nil {
		t.Fatal("cachedLibrary != nil after incomplete ABI 1.2 bind")
	}
	if containsLookup(fixture.events, "implementation-name") {
		t.Fatal("implementation name read after incomplete binding")
	}
}

func TestABILaterAdditiveMinorBindsAndCaches(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	fixture := &abiLoaderFixture{
		reportedABI:        0x00010003,
		implementationName: "fallback",
		implementationRC:   int32(ffi.OK),
	}
	library, err := activeLibraryWithOps(fixture.ops(t))
	if err != nil {
		t.Fatalf("activeLibrary() later additive ABI error = %v", err)
	}
	if library == nil || cachedLibrary != library {
		t.Fatal("later additive ABI did not install the complete library")
	}
	if !containsLookup(fixture.events, "bind-complete") {
		t.Fatalf("later additive ABI events = %v, want complete bind", fixture.events)
	}
}

func TestABIWrongMajorRejectedBeforeFullBind(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	fixture := &abiLoaderFixture{
		reportedABI:        0x00020000,
		implementationName: "fallback",
		implementationRC:   int32(ffi.OK),
	}
	_, err := activeLibraryWithOps(fixture.ops(t))
	if !errors.Is(err, ErrABIVersionMismatch) {
		t.Fatalf("activeLibrary() error = %v, want ErrABIVersionMismatch", err)
	}
	if want := []string{"pure_simdjson_get_abi_version"}; !reflect.DeepEqual(fixture.lookups, want) {
		t.Fatalf("wrong-major lookups = %v, want %v", fixture.lookups, want)
	}
	if cachedLibrary != nil {
		t.Fatal("cachedLibrary != nil after wrong-major mismatch")
	}
}

func TestABIImplementationNameFailureDoesNotCache(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	fixture := &abiLoaderFixture{
		reportedABI:        ffi.ABIVersion,
		implementationName: "",
		implementationRC:   int32(ffi.ErrInternal),
	}
	_, err := activeLibraryWithOps(fixture.ops(t))
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("activeLibrary() error = %v, want ErrInternal", err)
	}
	if cachedLibrary != nil {
		t.Fatal("cachedLibrary != nil after implementation-name failure")
	}
	assertEventBefore(t, fixture.events, "bind-complete", "implementation-name")
}

func assertEventBefore(t *testing.T, events []string, first, second string) {
	t.Helper()

	firstIndex := -1
	secondIndex := -1
	for i, event := range events {
		switch event {
		case first:
			firstIndex = i
		case second:
			secondIndex = i
		}
	}
	if firstIndex < 0 || secondIndex < 0 || firstIndex >= secondIndex {
		t.Fatalf("events = %v, want %q before %q", events, first, second)
	}
}

func containsLookup(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
