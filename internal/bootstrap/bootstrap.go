// Package bootstrap — BootstrapSync orchestrator and public option surface.
//
// Review-driven additions (05-REVIEWS.md):
//   - L2: PURE_SIMDJSON_CACHE_DIR env var → resolveConfig + defaultCacheDir.
//   - L3: version-stamped User-Agent on every outbound HTTP request (see download.go).
//   - M2: 30-second failure memoization so a blocked network does not stall
//     every NewParser() call for minutes.
//   - M3: test seams re-exported via export_test.go for bootstrap_test.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Environment variable names (locked: D-19, D-20).
const (
	mirrorEnvVar    = "PURE_SIMDJSON_BINARY_MIRROR"
	disableGHEnvVar = "PURE_SIMDJSON_DISABLE_GH_FALLBACK"
)

// bootstrapFailureTTL is the memoization window for failed BootstrapSync calls.
// Every NewParser() on a blocked-network host would otherwise retry for minutes.
// Not configurable in v0.1 — keep it simple (M2).
const bootstrapFailureTTL = 30 * time.Second

// BootstrapOption configures BootstrapSync.
type BootstrapOption func(*bootstrapConfig) error

// bootstrapConfig is the resolved configuration consumed by ensureArtifact and
// the download pipeline. All fields are derived from defaults, env vars, or
// explicit BootstrapOption application, in that order.
type bootstrapConfig struct {
	cacheDir      string
	version       string
	mirrorURL     string // R2 primary base URL (empty → defaultR2BaseURL)
	disableGH     bool   // when true, no GitHub fallback is attempted (D-20)
	httpClient    *http.Client
	githubBaseURL string // override for tests (M3); empty → defaultGitHubBaseURL
	goos          string
	goarch        string
	destDir       string // CLI --dest override; empty → cacheDir
}

// WithMirror overrides the R2 base URL. Validated at resolve time: HTTP is
// rejected for non-loopback hosts (T-05-05).
func WithMirror(rawURL string) BootstrapOption {
	return func(cfg *bootstrapConfig) error {
		trimmed := strings.TrimRight(strings.TrimSpace(rawURL), "/")
		if err := validateBaseURL(trimmed); err != nil {
			return err
		}
		cfg.mirrorURL = trimmed
		return nil
	}
}

// WithDest writes the artifact into path instead of the default cache dir.
// Used by the CLI fetch subcommand to populate a vendor directory.
func WithDest(path string) BootstrapOption {
	return func(cfg *bootstrapConfig) error {
		cfg.destDir = filepath.Clean(strings.TrimSpace(path))
		return nil
	}
}

// WithVersion overrides the library version. Defaults to the compile-time
// Version constant; CLI uses this for cross-version fetches.
func WithVersion(v string) BootstrapOption {
	return func(cfg *bootstrapConfig) error {
		cfg.version = strings.TrimSpace(v)
		return nil
	}
}

// WithTarget overrides the goos/goarch target. Defaults to runtime.GOOS /
// runtime.GOARCH; CLI uses this for --target cross-platform prefetch.
func WithTarget(goos, goarch string) BootstrapOption {
	return func(cfg *bootstrapConfig) error {
		cfg.goos = goos
		cfg.goarch = goarch
		return nil
	}
}

// withHTTPClient injects a custom *http.Client. Internal (lowercase): only
// tests use it, re-exported via export_test.go (M3).
func withHTTPClient(c *http.Client) BootstrapOption {
	return func(cfg *bootstrapConfig) error {
		cfg.httpClient = c
		return nil
	}
}

// withGitHubBaseURL injects a custom GitHub base URL for fallback tests.
// Internal (lowercase); re-exported via export_test.go (M3).
func withGitHubBaseURL(rawURL string) BootstrapOption {
	return func(cfg *bootstrapConfig) error {
		cfg.githubBaseURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
		return nil
	}
}

// resolveConfig builds the configuration. Env vars are read first; options
// applied afterwards may override them.
func resolveConfig(opts ...BootstrapOption) (bootstrapConfig, error) {
	cfg := bootstrapConfig{
		cacheDir:   defaultCacheDir(),
		version:    Version,
		mirrorURL:  strings.TrimRight(strings.TrimSpace(os.Getenv(mirrorEnvVar)), "/"),
		disableGH:  os.Getenv(disableGHEnvVar) == "1",
		httpClient: newHTTPClient(),
		goos:       runtime.GOOS,
		goarch:     runtime.GOARCH,
	}
	// Env-supplied mirror is still subject to the HTTPS-non-loopback gate.
	if cfg.mirrorURL != "" {
		if err := validateBaseURL(cfg.mirrorURL); err != nil {
			return bootstrapConfig{}, err
		}
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return bootstrapConfig{}, err
		}
	}
	return cfg, nil
}

// effectiveCacheDir returns destDir when set (CLI --dest), else cacheDir.
func (c bootstrapConfig) effectiveCacheDir() string {
	if c.destDir != "" {
		return c.destDir
	}
	return c.cacheDir
}

// bootstrapFailureCache guards a per-process memoized error (M2). It prevents
// blocked-network environments from replaying the retry ladder on every
// NewParser() call.
type bootstrapFailureCache struct {
	mu        sync.Mutex
	lastErr   error
	expiresAt time.Time
}

// peek returns a non-nil cached error if one is currently valid; nil otherwise.
func (c *bootstrapFailureCache) peek() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lastErr != nil && time.Now().Before(c.expiresAt) {
		return c.lastErr
	}
	return nil
}

func (c *bootstrapFailureCache) record(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastErr = err
	c.expiresAt = time.Now().Add(bootstrapFailureTTL)
}

func (c *bootstrapFailureCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastErr = nil
	c.expiresAt = time.Time{}
}

var globalBootstrapFailureCache bootstrapFailureCache

// BootstrapSync downloads, verifies, and installs the shared library for the
// resolved target platform into the cache directory. Safe for concurrent
// callers: an exclusive flock guards the install step, so repeated invocations
// collapse into a single download.
//
// Signature is locked by D-03.
func BootstrapSync(ctx context.Context, opts ...BootstrapOption) error {
	if ctx == nil {
		return errors.New("bootstrap: nil context")
	}
	// Honour caller-side cancellation before touching the memoization cache —
	// a cancelled ctx should return ctx.Err() even if a memoized error exists.
	if err := ctx.Err(); err != nil {
		return err
	}
	if cached := globalBootstrapFailureCache.peek(); cached != nil {
		return fmt.Errorf("bootstrap failed (memoized): %w", cached)
	}

	cfg, err := resolveConfig(opts...)
	if err != nil {
		// Config errors are NOT memoized — they are caller bugs, not network state.
		return err
	}

	if err := ensureArtifact(ctx, cfg); err != nil {
		globalBootstrapFailureCache.record(err)
		return err
	}
	globalBootstrapFailureCache.clear()
	return nil
}

// ensureArtifact realises the cache-hit / lock / download / install pipeline
// for a single target.
func ensureArtifact(ctx context.Context, cfg bootstrapConfig) error {
	cachePath := artifactCachePath(cfg.effectiveCacheDir(), cfg.version, cfg.goos, cfg.goarch)

	// Cache hit (D-04: no SHA-256 re-verify — trust-on-first-write).
	if _, err := os.Stat(cachePath); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	lockPath := filepath.Join(filepath.Dir(cachePath), ".lock")
	return withProcessFileLock(lockPath, func() error {
		// Re-check after lock — another process may have installed while we waited.
		if _, err := os.Stat(cachePath); err == nil {
			return nil
		}
		return downloadAndVerify(ctx, cfg, cachePath)
	})
}

// permanentBootstrapError tags an error as non-retryable. The retry loop in
// download.go consults isPermanentBootstrapError(err) and breaks out of the
// attempt cycle when the tag is set.
//
// Source: pure-onnx@v0.0.1/ort/bootstrap.go lines 60-89 (verbatim semantics).
type permanentBootstrapError struct{ err error }

func (e *permanentBootstrapError) Error() string { return e.err.Error() }
func (e *permanentBootstrapError) Unwrap() error { return e.err }

func markPermanentBootstrapError(err error) error {
	if err == nil {
		return nil
	}
	var already *permanentBootstrapError
	if errors.As(err, &already) {
		return err
	}
	return &permanentBootstrapError{err: err}
}

func isPermanentBootstrapError(err error) bool {
	var p *permanentBootstrapError
	return errors.As(err, &p)
}
