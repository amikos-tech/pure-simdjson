package bootstrap

// export_test.go — re-exports for the external bootstrap_test package.
// Compiled only during `go test` (the _test.go suffix gates it).
// See 05-REVIEWS.md M3 for the design rationale: bootstrap_test is an
// external package to exercise the public API; these seams keep it from
// needing to live in package bootstrap.

// DefaultCacheDir is the exported test seam for defaultCacheDir.
var DefaultCacheDir = defaultCacheDir

// WithProcessFileLockForTest is the exported test seam for withProcessFileLock.
// Suffix -ForTest keeps production callers from accidentally importing it.
func WithProcessFileLockForTest(lockPath string, fn func() error) error {
	return withProcessFileLock(lockPath, fn)
}

// AtomicInstallForTest is the exported test seam for atomicInstall.
func AtomicInstallForTest(tmpPath, finalPath string) error {
	return atomicInstall(tmpPath, finalPath)
}
