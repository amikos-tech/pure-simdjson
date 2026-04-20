package purejson

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

const libraryEnvPath = "PURE_SIMDJSON_LIB_PATH"

// bootstrapResolveTimeout bounds the auto-bootstrap stage inside
// resolveLibraryPath. Five minutes is generous enough for a cold download of
// the shared library on a slow link while guaranteeing NewParser cannot stall
// indefinitely.
const bootstrapResolveTimeout = 5 * time.Minute

type loadedLibrary struct {
	path               string
	handle             uintptr
	implementationName string
	bindings           *ffi.Bindings
}

var (
	libraryMu     sync.Mutex
	cachedLibrary *loadedLibrary
)

// activeLibrary returns the process-wide loaded library, triggering the resolve
// + dlopen + bind chain on first call. Concurrency model is double-checked
// locking (M1): libraryMu guards only the cachedLibrary pointer. Path
// resolution — which may trigger a multi-minute bootstrap download — and the
// dlopen/bind calls all run OUTSIDE libraryMu so concurrent NewParser() calls
// are not serialized on the first caller's bandwidth.
//
// A benign race is possible where two callers both reach the slow path and
// both dlopen the library; the loser discards its handle and returns the
// installed pointer. For v0.1 we accept the rare leak because purego does not
// expose dlclose; the dlopen race happens at most once per process.
func activeLibrary() (*loadedLibrary, error) {
	// Fast path: read the cached pointer under the lock, release immediately.
	libraryMu.Lock()
	cached := cachedLibrary
	libraryMu.Unlock()
	if cached != nil {
		return cached, nil
	}

	// Slow path: path resolution runs without libraryMu held (M1). This is the
	// stage that may trigger a network download.
	path, attempted, err := resolveLibraryPath()
	if err != nil {
		return nil, wrapLoadFailure(formatAttemptedPaths(attempted), err)
	}

	handle, err := loadLibrary(path)
	if err != nil {
		return nil, wrapLoadFailure(formatAttemptedPaths([]string{path}), err)
	}

	bindings, err := ffi.Bind(handle, lookupSymbol)
	if err != nil {
		return nil, wrapLoadFailure(fmt.Sprintf("bind symbols from %s", path), err)
	}

	implementationName, rc := bindings.ImplementationName()
	if rc != int32(ffi.OK) {
		return nil, wrapStatus(rc)
	}

	library := &loadedLibrary{
		path:               path,
		handle:             handle,
		implementationName: implementationName,
		bindings:           bindings,
	}

	// Re-check under the lock and install. If a concurrent caller won the race,
	// drop our handle and return theirs. For v0.1 purego has no dlclose surface
	// so we simply orphan the loser's handle — acceptable because the race fires
	// at most once per process lifetime.
	libraryMu.Lock()
	defer libraryMu.Unlock()
	if cachedLibrary != nil {
		return cachedLibrary, nil
	}
	cachedLibrary = library
	return cachedLibrary, nil
}

// resolveLibraryPath implements the 4-stage resolution chain:
//
//  1. Env override — PURE_SIMDJSON_LIB_PATH: absolute path honoured as-is,
//     relative paths resolved via filepath.Abs (DIST-09 Windows full-path
//     invariant). No network I/O.
//  2. Cache hit — a prior BootstrapSync installed the artifact at
//     bootstrap.CachePath. No SHA-256 re-verify on hit (D-04 — trust-on-first-
//     write; verification happened at install time).
//  3. Auto-bootstrap — call bootstrap.BootstrapSync with an internal 5-minute
//     timeout so NewParser never blocks indefinitely.
//  4. Cache hit after bootstrap — the artifact MUST be present now; if it
//     isn't the install broke an invariant.
//
// Returned path is always absolute (DIST-09). Returned error on bootstrap
// failure references PURE_SIMDJSON_LIB_PATH so users learn the bypass
// mechanism (D-21). Errors from bootstrap are wrapped with %w so
// errors.Is(err, purejson.ErrChecksumMismatch) and the other re-exported
// sentinels continue to match across the chain (H2 pointer-identity aliasing,
// Plan 01).
func resolveLibraryPath() (string, []string, error) {
	// Stage 1: env override.
	if envPath := strings.TrimSpace(os.Getenv(libraryEnvPath)); envPath != "" {
		absPath, err := filepath.Abs(envPath)
		if err != nil {
			return "", []string{envPath}, fmt.Errorf("resolve %s: %w", libraryEnvPath, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return "", []string{absPath}, fmt.Errorf("%s not found: %w", absPath, err)
		}
		return absPath, []string{absPath}, nil
	}

	// Stage 2: cache hit — bootstrap.CachePath returns an absolute path by
	// construction (cacheDir is absolute; filepath.Join preserves that).
	cachePath := bootstrap.CachePath(runtime.GOOS, runtime.GOARCH)
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, []string{cachePath}, nil
	}

	// Stage 3: auto-bootstrap with an internal timeout.
	ctx, cancel := context.WithTimeout(context.Background(), bootstrapResolveTimeout)
	defer cancel()
	if err := bootstrap.BootstrapSync(ctx); err != nil {
		return "", []string{cachePath},
			fmt.Errorf("bootstrap failed (set %s to bypass): %w", libraryEnvPath, err)
	}

	// Stage 4: cache hit after bootstrap.
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, []string{cachePath}, nil
	}
	return "", []string{cachePath},
		fmt.Errorf("shared library not found after bootstrap (set %s to bypass)", libraryEnvPath)
}

// formatAttemptedPaths renders the attempted-paths slice for error messages.
// Retained from Phase 3 so Phase 3's error tests continue to match.
func formatAttemptedPaths(paths []string) string {
	if len(paths) == 0 {
		return "attempted paths: none"
	}
	return fmt.Sprintf("attempted paths: %s", strings.Join(paths, ", "))
}
