package purejson

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

const libraryEnvPath = "PURE_SIMDJSON_LIB_PATH"

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

func activeLibrary() (*loadedLibrary, error) {
	libraryMu.Lock()
	defer libraryMu.Unlock()

	if cachedLibrary != nil {
		return cachedLibrary, nil
	}

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

	cachedLibrary = &loadedLibrary{
		path:               path,
		handle:             handle,
		implementationName: implementationName,
		bindings:           bindings,
	}
	return cachedLibrary, nil
}

func resolveLibraryPath() (string, []string, error) {
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

	candidates, err := libraryCandidates()
	if err != nil {
		return "", nil, err
	}

	attempted := make([]string, 0, len(candidates))
	var statErr error
	for _, candidate := range candidates {
		attempted = append(attempted, candidate)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, attempted, nil
		} else if !errors.Is(err, os.ErrNotExist) && statErr == nil {
			statErr = err
		}
	}

	if statErr != nil {
		return "", attempted, statErr
	}
	return "", attempted, fmt.Errorf("shared library not found")
}

func libraryCandidates() ([]string, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("determine working directory: %w", err)
	}

	triple, err := rustTargetTriple()
	if err != nil {
		return nil, err
	}

	libraryName := platformLibraryName()
	raw := []string{
		filepath.Join(root, "target", "release", libraryName),
		filepath.Join(root, "target", "debug", libraryName),
		filepath.Join(root, "target", triple, "release", libraryName),
		filepath.Join(root, "target", triple, "debug", libraryName),
	}

	candidates := make([]string, 0, len(raw))
	for _, candidate := range raw {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			return nil, fmt.Errorf("resolve candidate %s: %w", candidate, err)
		}
		candidates = append(candidates, absPath)
	}
	return candidates, nil
}

func platformLibraryName() string {
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

func rustTargetTriple() (string, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/amd64":
		return "x86_64-apple-darwin", nil
	case "darwin/arm64":
		return "aarch64-apple-darwin", nil
	case "linux/amd64":
		return "x86_64-unknown-linux-gnu", nil
	case "linux/arm64":
		return "aarch64-unknown-linux-gnu", nil
	case "windows/amd64":
		return "x86_64-pc-windows-msvc", nil
	default:
		return "", fmt.Errorf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func formatAttemptedPaths(paths []string) string {
	if len(paths) == 0 {
		return "attempted paths: none"
	}
	return fmt.Sprintf("attempted paths: %s", strings.Join(paths, ", "))
}
