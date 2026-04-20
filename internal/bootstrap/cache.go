package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	lockAcquireTimeout = 2 * time.Minute
	lockRetryInterval  = 200 * time.Millisecond
	lockLogInterval    = 5 * time.Second

	// cacheDirEnvVar overrides the OS user cache dir for ephemeral-HOME CI runners
	// and self-contained test runs (L2 from 05-REVIEWS.md).
	cacheDirEnvVar = "PURE_SIMDJSON_CACHE_DIR"
)

// defaultCacheDir returns the base cache directory for pure-simdjson artifacts.
// Precedence (L2 + L6):
//  1. PURE_SIMDJSON_CACHE_DIR env var, if non-empty.
//  2. os.UserCacheDir() + "/pure-simdjson".
//  3. os.TempDir() + "/pure-simdjson-<uid>" with 0700 perms (L6 — private,
//     UID-scoped subdir, never the bare TempDir path).
//
// Source: pure-onnx@v0.0.1/ort/bootstrap.go lines 1370-1383, adapted for L2/L6.
func defaultCacheDir() string {
	if env := os.Getenv(cacheDirEnvVar); env != "" {
		return env
	}
	base, err := os.UserCacheDir()
	if err == nil && base != "" {
		return filepath.Join(base, "pure-simdjson")
	}
	uid := os.Getuid() // POSIX; -1 on Windows but UserCacheDir practically never fails there.
	sub := fmt.Sprintf("pure-simdjson-%d", uid)
	path := filepath.Join(os.TempDir(), sub)
	_ = os.MkdirAll(path, 0700)
	return path
}

// CachePath returns the absolute path where the artifact for goos/goarch is stored.
// Layout: <cacheDir>/v<Version>/<goos>-<goarch>/<libname> (D-07).
// Called from library_loading.go::resolveLibraryPath to check cache presence (D-04).
func CachePath(goos, goarch string) string {
	return artifactCachePath(defaultCacheDir(), Version, goos, goarch)
}

func artifactCachePath(cacheDir, version, goos, goarch string) string {
	return filepath.Join(cacheDir,
		"v"+version,
		goos+"-"+goarch,
		platformLibraryName(goos))
}

// withProcessFileLock acquires an exclusive flock on lockPath, calls fn, then
// releases the lock. Polling: 200ms interval, 2-minute timeout.
// Source: pure-onnx@v0.0.1/ort/bootstrap.go lines 1283-1334 (verbatim, rename pkg).
func withProcessFileLock(lockPath string, fn func() error) (err error) {
	if fn == nil {
		return fmt.Errorf("lock callback is nil")
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return fmt.Errorf("create lock dir: %w", err)
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open lock file %q: %w", lockPath, err)
	}

	start := time.Now()
	nextLogAt := start.Add(lockLogInterval)
	for {
		lockErr := lockFile(file)
		if lockErr == nil {
			break
		}
		if !isLockWouldBlock(lockErr) {
			acquireErr := fmt.Errorf("acquire lock %q: %w", lockPath, lockErr)
			_ = file.Close()
			return acquireErr
		}
		waited := time.Since(start)
		if waited >= lockAcquireTimeout {
			_ = file.Close()
			return fmt.Errorf("timed out acquiring lock %q after %s", lockPath, lockAcquireTimeout)
		}
		if time.Now().After(nextLogAt) {
			fmt.Fprintf(os.Stderr, "pure-simdjson: waiting for install lock at %s (held %s)...\n",
				lockPath, time.Since(start).Truncate(time.Second))
			nextLogAt = time.Now().Add(lockLogInterval)
		}
		time.Sleep(lockRetryInterval)
	}

	defer func() {
		unlockErr := unlockFile(file)
		closeErr := file.Close()
		err = errors.Join(err, unlockErr, closeErr)
	}()

	return fn()
}

// atomicInstall renames tmpPath to finalPath. tmpPath MUST live in the same
// directory as finalPath so os.Rename is atomic (same-filesystem invariant).
// On rename failure the temp file is best-effort removed.
func atomicInstall(tmpPath, finalPath string) error {
	if err := os.MkdirAll(filepath.Dir(finalPath), 0700); err != nil {
		return fmt.Errorf("create artifact dir: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("atomic rename to %s: %w", finalPath, err)
	}
	return nil
}
