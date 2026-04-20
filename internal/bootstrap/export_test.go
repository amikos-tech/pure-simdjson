package bootstrap

// export_test.go — re-exports internal helpers for the external bootstrap_test
// package. Compiled only during `go test` (the _test.go suffix gates it).
// Rationale: 05-REVIEWS.md M3 — bootstrap_test is an external package so we can
// exercise the public API the way downstream consumers will. These seams avoid
// the "cross-package import gymnastics" alternative.

import (
	"context"
	"net/http"
)

// ---- config introspection seams ------------------------------------------------

// BootstrapConfigView is a read-only view of the resolved bootstrapConfig
// returned by ResolveConfig. Fields are exposed as accessors (not struct tags)
// so the internal layout stays private.
type BootstrapConfigView struct{ cfg bootstrapConfig }

func (v BootstrapConfigView) CacheDir() string      { return v.cfg.cacheDir }
func (v BootstrapConfigView) VersionField() string  { return v.cfg.version }
func (v BootstrapConfigView) MirrorURL() string     { return v.cfg.mirrorURL }
func (v BootstrapConfigView) DisableGH() bool       { return v.cfg.disableGH }
func (v BootstrapConfigView) GOOS() string          { return v.cfg.goos }
func (v BootstrapConfigView) GOARCH() string        { return v.cfg.goarch }
func (v BootstrapConfigView) DestDir() string       { return v.cfg.destDir }
func (v BootstrapConfigView) GitHubBaseURL() string { return v.cfg.githubBaseURL }

// ResolveConfig is the exported test seam for resolveConfig. Returns a view
// that exposes the fields tests need.
func ResolveConfig(opts ...BootstrapOption) (BootstrapConfigView, error) {
	cfg, err := resolveConfig(opts...)
	if err != nil {
		return BootstrapConfigView{}, err
	}
	return BootstrapConfigView{cfg: cfg}, nil
}

// ---- option seams -------------------------------------------------------------

// WithHTTPClient is the exported test seam for withHTTPClient.
func WithHTTPClient(c *http.Client) BootstrapOption { return withHTTPClient(c) }

// WithGitHubBaseURL is the exported test seam for withGitHubBaseURL.
func WithGitHubBaseURL(rawURL string) BootstrapOption { return withGitHubBaseURL(rawURL) }

// ---- cache seams --------------------------------------------------------------

// DefaultCacheDir is the exported test seam for defaultCacheDir.
var DefaultCacheDir = defaultCacheDir

// WithProcessFileLockForTest is the exported test seam for withProcessFileLock.
func WithProcessFileLockForTest(lockPath string, fn func() error) error {
	return withProcessFileLock(lockPath, fn)
}

// AtomicInstallForTest is the exported test seam for atomicInstall.
func AtomicInstallForTest(tmpPath, finalPath string) error {
	return atomicInstall(tmpPath, finalPath)
}

// ---- download / retry seams ---------------------------------------------------

// SleepWithJitterForTest is the exported test seam for sleepWithJitter.
func SleepWithJitterForTest(ctx context.Context, attempt int) error {
	return sleepWithJitter(ctx, attempt)
}

// IsRetryableForTest is the exported test seam for isRetryable.
func IsRetryableForTest(statusCode int, headers http.Header, bodySnippet string) bool {
	return isRetryable(statusCode, headers, bodySnippet)
}

// MarkPermanentForTest wraps err as a permanent bootstrap error.
func MarkPermanentForTest(err error) error { return markPermanentBootstrapError(err) }

// IsPermanentForTest reports whether err is wrapped as a permanent bootstrap error.
func IsPermanentForTest(err error) bool { return isPermanentBootstrapError(err) }

// ---- memoization seam ---------------------------------------------------------

// ResetBootstrapFailureCacheForTest clears the package-level failure cache so
// back-to-back tests don't bleed memoized failures across table entries (M2).
func ResetBootstrapFailureCacheForTest() {
	globalBootstrapFailureCache.clear()
}

// ---- checksum registry seam ---------------------------------------------------

// RegisterChecksumForTest temporarily inserts an entry into Checksums and
// returns a cleanup that restores the previous state. The Checksums map is a
// package-global so parallel tests that touch it must coordinate — bootstrap
// tests run sequentially.
func RegisterChecksumForTest(version, goos, goarch, hexDigest string) func() {
	key := ChecksumKey(version, goos, goarch)
	prev, had := Checksums[key]
	Checksums[key] = hexDigest
	return func() {
		if had {
			Checksums[key] = prev
		} else {
			delete(Checksums, key)
		}
	}
}
