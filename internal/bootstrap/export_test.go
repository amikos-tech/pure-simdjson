package bootstrap

// export_test.go — re-exports internal helpers for the external bootstrap_test
// package. Compiled only during `go test` (the _test.go suffix gates it).
// Rationale: 05-REVIEWS.md M3 — bootstrap_test is an external package so we can
// exercise the public API the way downstream consumers will. These seams avoid
// the "cross-package import gymnastics" alternative.

import (
	"context"
	"net/http"
	"time"
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

// ---- URL construction seams ---------------------------------------------------
// Plan 05-03 exposes the unexported URL helpers so tests can assert H1 platform
// tagging, exact URL layouts, and override-base behaviour without rebuilding the
// string format inline (which would let the test and production drift).

// R2ArtifactURL is the exported test seam for r2ArtifactURL.
var R2ArtifactURL = r2ArtifactURL

// GitHubArtifactURL is the exported test seam for githubArtifactURL.
var GitHubArtifactURL = githubArtifactURL

// R2ChecksumsURL is the exported test seam for r2ChecksumsURL.
var R2ChecksumsURL = r2ChecksumsURL

// GitHubChecksumsURL is the exported test seam for githubChecksumsURL.
var GitHubChecksumsURL = githubChecksumsURL

// GitHubAssetName is the exported test seam for githubAssetName (H1 verification).
var GitHubAssetName = githubAssetName

// ---- cache seams --------------------------------------------------------------

// DefaultCacheDir is the exported test seam for defaultCacheDir.
var DefaultCacheDir = defaultCacheDir

// WithProcessFileLockForTest is the exported test seam for withProcessFileLock.
func WithProcessFileLockForTest(ctx context.Context, lockPath string, fn func() error) error {
	return withProcessFileLock(ctx, lockPath, fn)
}

// AtomicInstallForTest is the exported test seam for atomicInstall.
func AtomicInstallForTest(tmpPath, finalPath string) error {
	return atomicInstall(tmpPath, finalPath)
}

// ---- download / retry seams ---------------------------------------------------

// SleepWithJitterForTest is the exported test seam for sleepWithJitter.
func SleepWithJitterForTest(ctx context.Context, attempt int) error {
	return sleepWithJitter(ctx, attempt, 0)
}

// SleepWithJitterHintForTest exposes the Retry-After-aware sleep signature
// so tests can assert hint-vs-jitter precedence without re-implementing the
// sleep budget.
func SleepWithJitterHintForTest(ctx context.Context, attempt int, hint time.Duration) error {
	return sleepWithJitter(ctx, attempt, hint)
}

// ParseRetryAfterForTest is the exported test seam for parseRetryAfter.
func ParseRetryAfterForTest(h http.Header) time.Duration {
	return parseRetryAfter(h)
}

// RejectHTTPSDowngradeForTest is the exported test seam for rejectHTTPSDowngrade.
// Used by Plan 05-06 to unit-test the redirect-policy contract directly without
// constructing a two-server HTTPS→HTTP redirect topology.
func RejectHTTPSDowngradeForTest(req *http.Request, via []*http.Request) error {
	return rejectHTTPSDowngrade(req, via)
}

// NewHTTPClientForTest is the exported test seam for newHTTPClient. Used by
// Plan 05-06 (T-05-04) to assert CheckRedirect is wired to rejectHTTPSDowngrade
// — the wiring test pairs with RejectHTTPSDowngradeForTest's behaviour test.
func NewHTTPClientForTest() *http.Client {
	return newHTTPClient()
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
