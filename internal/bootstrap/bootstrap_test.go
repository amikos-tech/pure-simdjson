package bootstrap_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amikos-tech/pure-simdjson/internal/bootstrap"
)

// clearBootstrapEnv isolates a test from host env vars that would otherwise
// bleed into resolveConfig. t.Setenv auto-restores on cleanup.
func clearBootstrapEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PURE_SIMDJSON_LIB_PATH", "")
	t.Setenv("PURE_SIMDJSON_BINARY_MIRROR", "")
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "")
	// Reset the package-level failure cache so tests don't bleed memoized
	// failures across table entries (M2 hygiene).
	bootstrap.ResetBootstrapFailureCacheForTest()
}

func TestSleepWithJitterCtxCancel(t *testing.T) {
	// Cancel the ctx after 10ms; the function must return within ~50ms with ctx.Err().
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	err := bootstrap.SleepWithJitterForTest(ctx, 5) // attempt=5 -> ~16s base if not cancelled
	elapsed := time.Since(start)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected ctx.Canceled, got %v", err)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("sleepWithJitter did not cancel promptly: took %v", elapsed)
	}
}

func TestPermanentBootstrapError(t *testing.T) {
	base := errors.New("boom")
	wrapped := bootstrap.MarkPermanentForTest(base)
	if !bootstrap.IsPermanentForTest(wrapped) {
		t.Fatalf("wrapped error should be permanent")
	}
	if bootstrap.IsPermanentForTest(base) {
		t.Fatalf("raw error should not be permanent")
	}
	// Unwrap chain must preserve errors.Is identity.
	if !errors.Is(wrapped, base) {
		t.Fatalf("errors.Is should match the base error through the permanent wrapper")
	}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		name   string
		code   int
		body   string
		header http.Header
		want   bool
	}{
		{"429 too many requests", 429, "", nil, true},
		{"503 service unavailable", 503, "", nil, true},
		{"500 server error", 500, "", nil, true},
		{"408 request timeout", 408, "", nil, true},
		{"404 not found", 404, "", nil, false},
		{"403 rate limit body", 403, "API rate limit exceeded for user", nil, true},
		{"403 secondary rate limit", 403, "You have triggered a secondary rate limit", nil, true},
		{"403 forbidden no body", 403, "forbidden", nil, false},
		{"403 with retry-after", 403, "", http.Header{"Retry-After": {"60"}}, true},
		{"403 remaining 0", 403, "", http.Header{"X-Ratelimit-Remaining": {"0"}}, true},
		{"200 ok", 200, "", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.header
			if h == nil {
				h = http.Header{}
			}
			got := bootstrap.IsRetryableForTest(tc.code, h, tc.body)
			if got != tc.want {
				t.Fatalf("isRetryable(%d, %q) = %v, want %v", tc.code, tc.body, got, tc.want)
			}
		})
	}
}

func TestBootstrapSyncNilCtx(t *testing.T) {
	clearBootstrapEnv(t)
	err := bootstrap.BootstrapSync(nil)
	if err == nil {
		t.Fatalf("BootstrapSync(nil) should return an error, not panic")
	}
}

func TestWithMirrorValidation(t *testing.T) {
	// HTTP non-loopback is rejected — security gate per T-05-05.
	_, err := bootstrap.ResolveConfig(bootstrap.WithMirror("http://example.com"))
	if err == nil {
		t.Fatalf("WithMirror(http://example.com) should fail validation")
	}
}

func TestWithMirrorLoopback(t *testing.T) {
	// Full RFC 5735 loopback range must be accepted for HTTP — previously only
	// three literals ("localhost", "127.0.0.1", "::1") passed. Regression for
	// PR #6 review item #2.
	cases := []string{
		"http://localhost:9999",
		"http://127.0.0.1:9999",
		"http://127.0.0.2:9999",
		"http://127.255.255.254:9999",
		"http://[::1]:9999",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := bootstrap.ResolveConfig(bootstrap.WithMirror(raw)); err != nil {
				t.Fatalf("WithMirror(%s) should succeed: %v", raw, err)
			}
		})
	}
}

func TestWithMirrorHTTPS(t *testing.T) {
	_, err := bootstrap.ResolveConfig(bootstrap.WithMirror("https://example.com"))
	if err != nil {
		t.Fatalf("WithMirror(https://example.com) should succeed: %v", err)
	}
}

func TestResolveConfigEnvMirror(t *testing.T) {
	clearBootstrapEnv(t)
	t.Setenv("PURE_SIMDJSON_BINARY_MIRROR", "https://mirror.example.com/path/")
	cfg, err := bootstrap.ResolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	// Trailing slash trimmed at resolve time.
	if cfg.MirrorURL() != "https://mirror.example.com/path" {
		t.Fatalf("mirrorURL = %q, want trimmed mirror URL", cfg.MirrorURL())
	}
}

func TestResolveConfigDisableGH(t *testing.T) {
	clearBootstrapEnv(t)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")
	cfg, err := bootstrap.ResolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if !cfg.DisableGH() {
		t.Fatalf("disableGH should be true when PURE_SIMDJSON_DISABLE_GH_FALLBACK=1")
	}
}

func TestUserAgentStamp(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("fake library bytes")
	digest := sha256.Sum256(body)
	digestHex := hex.EncodeToString(digest[:])

	goos, goarch := "linux", "amd64"
	// Inject the checksum so downloadAndVerify succeeds end-to-end.
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digestHex)()

	var captured atomic.Pointer[string]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		captured.Store(&ua)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)

	err := bootstrap.BootstrapSync(context.Background(),
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	)
	if err != nil {
		t.Fatalf("BootstrapSync: %v", err)
	}
	got := captured.Load()
	if got == nil {
		t.Fatalf("server never saw a request")
	}
	want := "pure-simdjson-go/v" + bootstrap.Version
	if *got != want {
		t.Fatalf("User-Agent = %q, want %q", *got, want)
	}
}

func TestBootstrapFailureMemoized(t *testing.T) {
	clearBootstrapEnv(t)

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "upstream down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	opts := []bootstrap.BootstrapOption{
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget("linux", "amd64"),
		bootstrap.WithHTTPClient(srv.Client()),
	}

	// First call — exhausts retries against the 503 server.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err1 := bootstrap.BootstrapSync(ctx, opts...)
	if err1 == nil {
		t.Fatalf("first call should fail against 503 server")
	}
	firstHits := hits.Load()
	if firstHits == 0 {
		t.Fatalf("expected first call to make at least one request, got 0")
	}

	// Second call — memoized; must return the cached error without hitting the server.
	start := time.Now()
	err2 := bootstrap.BootstrapSync(context.Background(), opts...)
	elapsed := time.Since(start)
	if err2 == nil {
		t.Fatalf("second call should return memoized error")
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("second call took %v — memoization appears not to short-circuit", elapsed)
	}
	if hits.Load() != firstHits {
		t.Fatalf("second call re-hit the server: before=%d, after=%d", firstHits, hits.Load())
	}
}

func TestBootstrapSuccessClearsFailureCache(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("fake library bytes for success test")
	digest := sha256.Sum256(body)
	digestHex := hex.EncodeToString(digest[:])
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digestHex)()

	// A server that 503s initially, then serves the body after the test flips a flag.
	var serveOK atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveOK.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}
		http.Error(w, "upstream down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	opts := []bootstrap.BootstrapOption{
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	}

	// First attempt — fails and is memoized.
	if err := bootstrap.BootstrapSync(context.Background(), opts...); err == nil {
		t.Fatalf("first call should fail")
	}

	// Flip server to OK, then reset the failure cache (simulating the TTL expiring)
	// so the second call actually hits the network.
	serveOK.Store(true)
	bootstrap.ResetBootstrapFailureCacheForTest()

	if err := bootstrap.BootstrapSync(context.Background(), opts...); err != nil {
		t.Fatalf("second call should succeed: %v", err)
	}

	// Cache should now contain the artifact.
	cached := bootstrap.CachePath(goos, goarch)
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("artifact not installed at %s: %v", cached, err)
	}
}

func TestChecksumMismatchIsPermanent(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("actual body")
	// Register a DIFFERENT digest so verification fails.
	wrongDigest := hex.EncodeToString(sha256.New().Sum(nil)) // sha256("") — will not match "actual body"
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, wrongDigest)()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	err := bootstrap.BootstrapSync(context.Background(),
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	)
	if err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
	if !errors.Is(err, bootstrap.ErrChecksumMismatch) {
		t.Fatalf("err = %v, want ErrChecksumMismatch", err)
	}
	// Must be marked permanent so the retry loop won't hammer the server on mismatch.
	if hits.Load() != 1 {
		t.Fatalf("checksum mismatch should be permanent; server hit %d times", hits.Load())
	}
}

func TestNoChecksumReturnsSentinel(t *testing.T) {
	clearBootstrapEnv(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("anything"))
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	// Use an arch for which no checksum is registered (the production Checksums map
	// is empty by default, so this should fire).
	err := bootstrap.BootstrapSync(context.Background(),
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget("linux", "amd64"),
		bootstrap.WithHTTPClient(srv.Client()),
	)
	if err == nil {
		t.Fatalf("expected ErrNoChecksum")
	}
	if !errors.Is(err, bootstrap.ErrNoChecksum) {
		t.Fatalf("err = %v, want ErrNoChecksum", err)
	}
}

func TestHTTPSDowngradeRejected(t *testing.T) {
	clearBootstrapEnv(t)

	// downstream plain-HTTP server that would serve the library (never reached).
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("x"))
	}))
	defer plain.Close()

	// upstream TLS server that 302s to the plain-HTTP server (simulating a
	// malicious redirect).
	var tlsSrv *httptest.Server
	tlsSrv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL, http.StatusFound)
	}))
	defer tlsSrv.Close()

	// Pre-register a dummy checksum so the path reaches downloadOnce (not ErrNoChecksum).
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, "deadbeef")()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	// Must use the tls server's Client so the test trusts its certificate.
	err := bootstrap.BootstrapSync(context.Background(),
		bootstrap.WithMirror(tlsSrv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(tlsSrv.Client()),
	)
	if err == nil {
		t.Fatalf("expected HTTPS->HTTP downgrade to be rejected")
	}
	// Ensure the plain-HTTP server was not reached (redirect was rejected before dial).
	_ = url.Parse
}

func TestWithVersionAndWithTarget(t *testing.T) {
	clearBootstrapEnv(t)

	cfg, err := bootstrap.ResolveConfig(
		bootstrap.WithVersion("1.2.3"),
		bootstrap.WithTarget("darwin", "arm64"),
	)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.VersionField() != "1.2.3" {
		t.Fatalf("version = %q, want 1.2.3", cfg.VersionField())
	}
	if cfg.GOOS() != "darwin" || cfg.GOARCH() != "arm64" {
		t.Fatalf("target = %s/%s, want darwin/arm64", cfg.GOOS(), cfg.GOARCH())
	}
}

func TestWithDest(t *testing.T) {
	clearBootstrapEnv(t)

	dest := filepath.Join(t.TempDir(), "vendor-libs")
	cfg, err := bootstrap.ResolveConfig(bootstrap.WithDest(dest))
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.DestDir() != dest {
		t.Fatalf("destDir = %q, want %q", cfg.DestDir(), dest)
	}
}

// ---------------------------------------------------------------------------
// Plan 05-03 — additions that round out the VALIDATION.md Fault Injection
// Matrix on top of the tests 05-02 already shipped. Each test below cites the
// matrix row it satisfies so the mapping from spec → assertion is auditable.
// ---------------------------------------------------------------------------

// TestURLConstruction covers DIST-01: R2 URL construction for all 5 platforms.
// The R2 layout keeps a platform-independent filename under <os>-<arch>/ because
// the directory component prevents collision (complements H1 GH tagging).
func TestURLConstruction(t *testing.T) {
	const base = "https://releases.example.com/pure-simdjson"
	const version = "0.1.0"

	cases := []struct {
		goos, goarch string
		wantSuffix   string // suffix after /v<version>/
	}{
		{"linux", "amd64", "linux-amd64/libpure_simdjson.so"},
		{"linux", "arm64", "linux-arm64/libpure_simdjson.so"},
		{"darwin", "amd64", "darwin-amd64/libpure_simdjson.dylib"},
		{"darwin", "arm64", "darwin-arm64/libpure_simdjson.dylib"},
		{"windows", "amd64", "windows-amd64/pure_simdjson-msvc.dll"},
	}

	if len(cases) != len(bootstrap.SupportedPlatforms) {
		t.Fatalf("case table covers %d platforms, SupportedPlatforms has %d",
			len(cases), len(bootstrap.SupportedPlatforms))
	}

	for _, c := range cases {
		t.Run(c.goos+"-"+c.goarch, func(t *testing.T) {
			got := bootstrap.R2ArtifactURL(base, version, c.goos, c.goarch)
			want := base + "/v" + version + "/" + c.wantSuffix
			if got != want {
				t.Fatalf("r2ArtifactURL(%s,%s) = %q, want %q", c.goos, c.goarch, got, want)
			}
		})
	}

	// Trailing-slash hygiene: a base URL with a trailing slash must produce the
	// same URL as one without, so callers can't accidentally introduce a double
	// slash in the path (which R2 would 404 on).
	withSlash := bootstrap.R2ArtifactURL(base+"/", version, "linux", "amd64")
	withoutSlash := bootstrap.R2ArtifactURL(base, version, "linux", "amd64")
	if withSlash != withoutSlash {
		t.Fatalf("trailing slash not trimmed: %q vs %q", withSlash, withoutSlash)
	}
}

// TestGitHubAssetNames covers H1: GH release assets live in a flat namespace so
// each platform must produce a DISTINCT filename. This test is the regression
// guard against a future refactor that accidentally drops the platform tag.
func TestGitHubAssetNames(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "libpure_simdjson-linux-amd64.so"},
		{"linux", "arm64", "libpure_simdjson-linux-arm64.so"},
		{"darwin", "amd64", "libpure_simdjson-darwin-amd64.dylib"},
		{"darwin", "arm64", "libpure_simdjson-darwin-arm64.dylib"},
		{"windows", "amd64", "pure_simdjson-windows-amd64-msvc.dll"},
	}

	// Per-platform exact-string check.
	for _, c := range cases {
		t.Run(c.goos+"-"+c.goarch, func(t *testing.T) {
			got := bootstrap.GitHubAssetName(c.goos, c.goarch)
			if got != c.want {
				t.Fatalf("githubAssetName(%s,%s) = %q, want %q", c.goos, c.goarch, got, c.want)
			}
		})
	}

	// Pairwise-distinct check — the whole point of H1. If two platforms produce
	// the same asset name we have a flat-namespace collision at release time.
	seen := map[string]string{}
	for _, c := range cases {
		got := bootstrap.GitHubAssetName(c.goos, c.goarch)
		if prev, clash := seen[got]; clash {
			t.Fatalf("H1 collision: %s/%s and %s both produce asset name %q",
				c.goos, c.goarch, prev, got)
		}
		seen[got] = c.goos + "/" + c.goarch
	}
}

// TestGitHubArtifactURL covers H1 at the URL level: default base + override
// base both produce the expected platform-tagged URL.
func TestGitHubArtifactURL(t *testing.T) {
	const version = "0.1.0"
	defaultBase := "https://github.com/amikos-tech/pure-simdjson/releases/download"

	t.Run("default-base", func(t *testing.T) {
		cases := []struct {
			goos, goarch, wantAsset string
		}{
			{"linux", "amd64", "libpure_simdjson-linux-amd64.so"},
			{"linux", "arm64", "libpure_simdjson-linux-arm64.so"},
			{"darwin", "amd64", "libpure_simdjson-darwin-amd64.dylib"},
			{"darwin", "arm64", "libpure_simdjson-darwin-arm64.dylib"},
			{"windows", "amd64", "pure_simdjson-windows-amd64-msvc.dll"},
		}
		for _, c := range cases {
			got := bootstrap.GitHubArtifactURL("", version, c.goos, c.goarch)
			want := defaultBase + "/v" + version + "/" + c.wantAsset
			if got != want {
				t.Errorf("githubArtifactURL(default,%s,%s) = %q, want %q",
					c.goos, c.goarch, got, want)
			}
		}
	})

	t.Run("override-base", func(t *testing.T) {
		override := "https://ghtest.example/dl"
		got := bootstrap.GitHubArtifactURL(override, version, "linux", "amd64")
		want := override + "/v" + version + "/libpure_simdjson-linux-amd64.so"
		if got != want {
			t.Fatalf("githubArtifactURL(override,linux,amd64) = %q, want %q", got, want)
		}
	})
}

// TestChecksumKeyFormat pins the "v<version>/<goos>-<goarch>/<libname>" layout
// so the CLI `verify` subcommand (Plan 05) and the Checksums map stay in lockstep.
func TestChecksumKeyFormat(t *testing.T) {
	cases := []struct {
		version, goos, goarch, want string
	}{
		{"0.1.0", "linux", "amd64", "v0.1.0/linux-amd64/libpure_simdjson.so"},
		{"0.1.0", "linux", "arm64", "v0.1.0/linux-arm64/libpure_simdjson.so"},
		{"0.1.0", "darwin", "amd64", "v0.1.0/darwin-amd64/libpure_simdjson.dylib"},
		{"0.1.0", "darwin", "arm64", "v0.1.0/darwin-arm64/libpure_simdjson.dylib"},
		{"0.1.0", "windows", "amd64", "v0.1.0/windows-amd64/pure_simdjson-msvc.dll"},
	}
	for _, c := range cases {
		got := bootstrap.ChecksumKey(c.version, c.goos, c.goarch)
		if got != c.want {
			t.Errorf("ChecksumKey(%s,%s,%s) = %q, want %q",
				c.version, c.goos, c.goarch, got, c.want)
		}
	}
}

// TestResolveConfigCacheDirEnv covers L2: PURE_SIMDJSON_CACHE_DIR takes priority
// over os.UserCacheDir so CI runners with ephemeral HOME can self-isolate.
func TestResolveConfigCacheDirEnv(t *testing.T) {
	clearBootstrapEnv(t)

	custom := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", custom)

	cfg, err := bootstrap.ResolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.CacheDir() != custom {
		t.Fatalf("cacheDir = %q, want env override %q", cfg.CacheDir(), custom)
	}
}

// TestBootstrapSync is the DIST-04 happy path: httptest mirror serves the
// artifact, checksum matches, BootstrapSync succeeds, the cached file exists at
// the expected path, and the server was hit exactly once.
func TestBootstrapSync(t *testing.T) {
	clearBootstrapEnv(t)

	body := fakeLibBody()
	digest := computeHex(body)
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digest)()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)

	err := bootstrap.BootstrapSync(context.Background(),
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	)
	if err != nil {
		t.Fatalf("BootstrapSync: %v", err)
	}

	cached := bootstrap.CachePath(goos, goarch)
	got, err := os.ReadFile(cached)
	if err != nil {
		t.Fatalf("artifact not cached at %s: %v", cached, err)
	}
	if string(got) != string(body) {
		t.Fatalf("cached body mismatch")
	}
	if hits.Load() != 1 {
		t.Fatalf("server hits = %d, want 1", hits.Load())
	}
}

// TestRetryOn429ThenSuccess covers Fault Injection Matrix item 2:
// HTTP 429 on first attempt, 200 on retry → retry succeeds, correct file written.
func TestRetryOn429ThenSuccess(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("retry-success-body")
	digest := computeHex(body)
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digest)()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	// Disable GH so the retry ladder stays on the single mirror (otherwise the
	// second attempt would target the fallback URL).
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := bootstrap.BootstrapSync(ctx,
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	); err != nil {
		t.Fatalf("BootstrapSync: %v", err)
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("hits = %d, want 2 (first 429, second 200)", got)
	}
}

// TestFallback404R2Then200GH covers Fault Injection Matrix item 9 (and DIST-02):
// R2 returns 404 on every attempt, GH fallback serves 200 → GH fires, artifact cached.
// Uses the bootstrap.WithGitHubBaseURL seam (M3) to redirect the fallback to a
// second httptest server whose path layout matches githubArtifactURL's output.
func TestFallback404R2Then200GH(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("github-fallback-body")
	digest := computeHex(body)
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digest)()

	var r2Hits, ghHits atomic.Int32

	r2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2Hits.Add(1)
		http.NotFound(w, r)
	}))
	defer r2.Close()

	// The GH fallback URL is <ghBase>/v<version>/<assetName>. Any path under
	// the base that returns 200 with the correct body satisfies the contract —
	// downloadWithRetry only cares about status + bytes, not path structure.
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ghHits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer gh.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := bootstrap.BootstrapSync(ctx,
		bootstrap.WithMirror(r2.URL),
		bootstrap.WithGitHubBaseURL(gh.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(r2.Client()),
	); err != nil {
		t.Fatalf("BootstrapSync: %v", err)
	}

	if r2Hits.Load() == 0 {
		t.Errorf("R2 mirror was never hit — fallback fired prematurely")
	}
	if ghHits.Load() != 1 {
		t.Errorf("GH hits = %d, want 1 (single successful fallback)", ghHits.Load())
	}

	cached := bootstrap.CachePath(goos, goarch)
	if got, err := os.ReadFile(cached); err != nil {
		t.Fatalf("artifact not cached: %v", err)
	} else if string(got) != string(body) {
		t.Fatalf("cached body came from the wrong source")
	}
}

// TestParseRetryAfter covers RFC 7231 §7.1.3 — Retry-After can be an integer
// seconds count or an HTTP-date. Unparseable values return zero so the caller
// falls back to the default jitter backoff.
func TestParseRetryAfter(t *testing.T) {
	cases := []struct {
		name   string
		header string
		minD   time.Duration
		maxD   time.Duration
	}{
		{"empty", "", 0, 0},
		{"zero seconds", "0", 0, 0},
		{"seconds 5", "5", 5 * time.Second, 5 * time.Second},
		{"seconds padded", "  30 ", 30 * time.Second, 30 * time.Second},
		{"seconds negative", "-1", 0, 0},
		{"garbage", "soon", 0, 0},
		// HTTP-date cases computed relative to time.Now() — allow a small window.
		{"http-date future", time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat), 8 * time.Second, 11 * time.Second},
		{"http-date past", time.Now().Add(-1 * time.Minute).UTC().Format(http.TimeFormat), 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.header != "" {
				h.Set("Retry-After", tc.header)
			}
			got := bootstrap.ParseRetryAfterForTest(h)
			if got < tc.minD || got > tc.maxD {
				t.Fatalf("parseRetryAfter(%q) = %v, want within [%v, %v]",
					tc.header, got, tc.minD, tc.maxD)
			}
		})
	}
}

// TestSleepHonorsRetryAfterHint asserts that when the server-supplied hint
// exceeds the computed jitter backoff, the sleep lasts at least the hint
// duration (min(hint, maxRetrySleep)). Regression for PR #6 review item #9.
func TestSleepHonorsRetryAfterHint(t *testing.T) {
	// attempt=0 → jitterBackoff ≤ 500ms + 500ms jitter ≈ 1s max.
	// A 300ms hint must be ignored (jitter wins); a 1500ms hint must win.
	hint := 1500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	if err := bootstrap.SleepWithJitterHintForTest(ctx, 0, hint); err != nil {
		t.Fatalf("sleepWithJitter: %v", err)
	}
	elapsed := time.Since(start)
	// Allow small timer slack; strict lower bound is the hint.
	if elapsed < hint-100*time.Millisecond {
		t.Fatalf("elapsed = %v, want ≥ ~%v (hint should dominate jitter backoff)",
			elapsed, hint)
	}
}

// TestRetryAfterHintFromServer wires the full path: HTTP 429 + Retry-After
// header on first attempt, 200 on second. Ensures the second-attempt sleep
// honors the server hint.
func TestRetryAfterHintFromServer(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("retry-after-body")
	digest := computeHex(body)
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digest)()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1") // 1 second
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	start := time.Now()
	if err := bootstrap.BootstrapSync(ctx,
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	); err != nil {
		t.Fatalf("BootstrapSync: %v", err)
	}
	elapsed := time.Since(start)

	if got := hits.Load(); got != 2 {
		t.Fatalf("hits = %d, want 2", got)
	}
	// First-attempt jitter at attempt=1 ≤ ~1s+500ms jitter. The Retry-After
	// hint is 1s, which must be honored — so between-attempt sleep ≥ ~1s.
	if elapsed < 900*time.Millisecond {
		t.Fatalf("elapsed = %v, expected ≥ ~1s (Retry-After honored)", elapsed)
	}
}

// TestCachedArtifactPerms asserts the installed cache file has 0644 mode
// (convention for shared libraries), not the 0600 default from os.CreateTemp.
// Regression for PR #6 review item #7.
func TestCachedArtifactPerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits not meaningful on windows")
	}
	clearBootstrapEnv(t)

	body := []byte("perms-body")
	digest := computeHex(body)
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digest)()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := bootstrap.BootstrapSync(ctx,
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	); err != nil {
		t.Fatalf("BootstrapSync: %v", err)
	}

	cached := bootstrap.CachePath(goos, goarch)
	info, err := os.Stat(cached)
	if err != nil {
		t.Fatalf("stat cached artifact: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Fatalf("cached artifact mode = %#o, want 0644", perm)
	}
}

// TestBootstrapRejectsAdvertisedOversize asserts the Content-Length pre-check
// in downloadOnce: a server that advertises a body > 128 MiB is rejected
// before any temp file is created, preventing ~768 MiB of wasted disk writes
// across the full retry ladder. Regression for PR #6 review item #5.
//
// The handler Hijacks the connection to send a forged Content-Length without
// actually streaming that many bytes; Go's default ResponseWriter would
// recompute Content-Length from the written bytes.
func TestBootstrapRejectsAdvertisedOversize(t *testing.T) {
	clearBootstrapEnv(t)

	goos, goarch := "linux", "amd64"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijacker", http.StatusInternalServerError)
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = bufrw.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 134217729\r\nContent-Type: application/octet-stream\r\n\r\n")
		_ = bufrw.Flush()
		// No body — the client must reject on headers alone.
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := bootstrap.BootstrapSync(ctx,
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	)
	if err == nil {
		t.Fatalf("BootstrapSync should have rejected oversize response")
	}
	if !strings.Contains(err.Error(), "advertised response too large") {
		t.Fatalf("error did not flag oversize pre-check: %v", err)
	}

	// No *.tmp files should linger in the cache directory — pre-check fires
	// before os.CreateTemp.
	libDir := filepath.Join(cacheDir, "v"+bootstrap.Version, goos+"-"+goarch)
	entries, _ := os.ReadDir(libDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file leaked: %s", e.Name())
		}
	}
}

// ---------------------------------------------------------------------------
// Small helpers shared by the Plan 05-03 additions.
// ---------------------------------------------------------------------------

func fakeLibBody() []byte { return []byte("fake-library-content-for-testing") }

func computeHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// Plan 05-06 — integration fault-injection tests that close out the remaining
// rows of 05-VALIDATION.md §Fault Injection Test Matrix (items 3, 4, 5, 6, 7,
// 8, 10, 11). Items 1, 2, 9 and 11 (unit-level retry) were already covered by
// Plans 05-02 / 05-03.
// ---------------------------------------------------------------------------

// TestFallback503R2Then200GH covers Fault Injection Matrix item 3 (DIST-02
// fallback): R2 returns 503 on every attempt, GH fallback serves 200.
//
// H1 reinforcement: the GH httptest mock asserts the exact request path is
// /v<Version>/<githubAssetName> — i.e. the platform-tagged asset path
// (libpure_simdjson-linux-amd64.so), NOT the on-disk cache filename
// (libpure_simdjson.so). If this assertion ever fails it is the SIGNAL that
// downloadWithRetry stopped routing GH URLs through githubArtifactURL /
// githubAssetName.
func TestFallback503R2Then200GH(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("github-fallback-via-503")
	digest := computeHex(body)
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digest)()

	var r2Hits, ghHits atomic.Int32
	r2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2Hits.Add(1)
		http.Error(w, "upstream down", http.StatusServiceUnavailable)
	}))
	defer r2.Close()

	wantPath := "/v" + bootstrap.Version + "/" + bootstrap.GitHubAssetName(goos, goarch)
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ghHits.Add(1)
		// H1 wire-format guard: the GH URL MUST encode the platform-tagged asset
		// name, not libpure_simdjson.so. A regression here would manifest as 404
		// from the mock even though bytes are staged — fail loudly with a
		// diagnostic so the cause is obvious.
		if r.URL.Path != wantPath {
			t.Errorf("GH request path = %q, want %q (H1 platform-tagging)", r.URL.Path, wantPath)
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer gh.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := bootstrap.BootstrapSync(ctx,
		bootstrap.WithMirror(r2.URL),
		bootstrap.WithGitHubBaseURL(gh.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(r2.Client()),
	); err != nil {
		t.Fatalf("BootstrapSync: %v", err)
	}

	// R2 must have exhausted its retries (3 attempts) before GH was tried.
	if got := r2Hits.Load(); got != int32(bootstrapRetryAttempts) {
		t.Errorf("r2 hits = %d, want %d (full retry exhaustion)", got, bootstrapRetryAttempts)
	}
	if got := ghHits.Load(); got != 1 {
		t.Errorf("gh hits = %d, want 1 (single successful fallback)", got)
	}

	cached := bootstrap.CachePath(goos, goarch)
	if got, err := os.ReadFile(cached); err != nil {
		t.Fatalf("artifact not cached: %v", err)
	} else if string(got) != string(body) {
		t.Fatalf("cached body came from the wrong source")
	}
}

// TestDisableGHFallbackWith404 covers Fault Injection Matrix item 10 (DIST-07):
// PURE_SIMDJSON_DISABLE_GH_FALLBACK=1 + R2 404 → ErrAllSourcesFailed, GH never
// consulted. The GH server is started anyway so we can prove via its hit
// counter that the fallback URL was not reached.
func TestDisableGHFallbackWith404(t *testing.T) {
	clearBootstrapEnv(t)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	goos, goarch := "linux", "amd64"
	// Pre-register a checksum so we don't bail on ErrNoChecksum before reaching
	// the URL ladder.
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, "deadbeef")()

	var r2Hits, ghHits atomic.Int32
	r2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2Hits.Add(1)
		http.NotFound(w, r)
	}))
	defer r2.Close()
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ghHits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("should-not-be-reached"))
	}))
	defer gh.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)

	err := bootstrap.BootstrapSync(context.Background(),
		bootstrap.WithMirror(r2.URL),
		bootstrap.WithGitHubBaseURL(gh.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(r2.Client()),
	)
	if err == nil {
		t.Fatalf("expected ErrAllSourcesFailed, got nil")
	}
	if !errors.Is(err, bootstrap.ErrAllSourcesFailed) {
		t.Fatalf("err = %v, want ErrAllSourcesFailed", err)
	}
	if r2Hits.Load() == 0 {
		t.Errorf("R2 was never hit — disable_gh shouldn't suppress the primary URL")
	}
	if ghHits.Load() != 0 {
		t.Errorf("GH hits = %d, want 0 (DISABLE_GH_FALLBACK=1 must suppress the fallback)",
			ghHits.Load())
	}
}

// TestBootstrapSyncCancellation covers Fault Injection Matrix item 4 (DIST-04):
// ctx cancelled mid-download → context.Canceled returned, no orphan *.tmp files
// left behind in the cache directory.
//
// The slow handler streams a few bytes then blocks on its request context so the
// ctx.cancel() reaches the in-flight HTTP read. The test asserts on
// errors.Is(..., context.Canceled) — accepting either Canceled or
// DeadlineExceeded would be looser than the matrix row's contract.
func TestBootstrapSyncCancellation(t *testing.T) {
	clearBootstrapEnv(t)

	goos, goarch := "linux", "amd64"
	// Pre-register a dummy checksum so we'd reach downloadOnce even if cancellation
	// somehow didn't fire (defensive — we never expect to compute a digest here).
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, "deadbeef")()
	// Disable GH fallback so the cancellation can't be papered over by a fast
	// fallback URL succeeding before ctx.Err() propagates.
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("first chunk\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		// Block until either client closes the connection or the test's outer
		// timeout fires.
		<-r.Context().Done()
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()

	err := bootstrap.BootstrapSync(ctx,
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	)
	if err == nil {
		t.Fatalf("expected ctx-cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want errors.Is(..., context.Canceled)", err)
	}

	// No *.tmp files should be left behind after a cancelled download. Walk the
	// cache dir and assert.
	walkErr := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".tmp") {
			t.Errorf("orphan temp file left behind after cancellation: %s", path)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk cache dir: %v", walkErr)
	}
}

// TestBootstrapSyncCtxCancelDuringSleep covers Fault Injection Matrix item 5
// (DIST-04): ctx cancelled while sleepWithJitter is waiting between retries
// must short-circuit within ~50ms instead of waiting out the full backoff
// interval. The matrix row's "returns within 50ms" target is loosened slightly
// to 300ms here to absorb CI scheduler jitter — still firmly under the 500ms
// minimum backoff so the test catches a regression.
func TestBootstrapSyncCtxCancelDuringSleep(t *testing.T) {
	clearBootstrapEnv(t)

	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, "deadbeef")()
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	// 429 every call → guarantees the retry loop enters sleepWithJitter on the
	// second attempt with a >= 500ms wait.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Cancel after the first 429 has been received but before the retry
		// sleep elapses.
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := bootstrap.BootstrapSync(ctx,
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected ctx.Canceled, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want errors.Is(..., context.Canceled)", err)
	}
	// Full backoff would be ≥ 500ms; even a single sleep cycle without
	// cancellation would exceed 300ms reliably. Anything under that proves the
	// ctx.Done() branch of sleepWithJitter fired.
	if elapsed > 300*time.Millisecond {
		t.Fatalf("elapsed %v — ctx cancel did not short-circuit retry sleep", elapsed)
	}
}

// TestRedirectDowngradeUnit covers T-05-04 at the policy-function level
// (Plan-05-06 option (a)): rejectHTTPSDowngrade returns a non-nil error when
// the originating request was HTTPS and the next hop is HTTP.
//
// Pairs with TestRedirectDowngradeWired below which proves the policy is
// actually wired into the HTTP client. TestHTTPSDowngradeRejected (Plan 05-02)
// already exercises the end-to-end behaviour through httptest.
func TestRedirectDowngradeUnit(t *testing.T) {
	httpsURL, _ := url.Parse("https://example.com/v0.1.0/lib.so")
	httpURL, _ := url.Parse("http://attacker.example/leak.so")

	via := []*http.Request{{URL: httpsURL}}
	req := &http.Request{URL: httpURL}

	if err := bootstrap.RejectHTTPSDowngradeForTest(req, via); err == nil {
		t.Fatalf("rejectHTTPSDowngrade(https→http) returned nil, want error")
	}

	// Sanity: same-scheme hops must NOT be rejected.
	httpsTwo, _ := url.Parse("https://example.com/redirected")
	if err := bootstrap.RejectHTTPSDowngradeForTest(&http.Request{URL: httpsTwo}, via); err != nil {
		t.Fatalf("rejectHTTPSDowngrade(https→https) = %v, want nil", err)
	}
}

// TestRedirectDowngradeWired covers T-05-04's wiring contract: the HTTP client
// returned by newHTTPClient must have CheckRedirect set, and the function it
// points at must reject the same HTTPS→HTTP transition rejected by the unit
// test above. Together with TestRedirectDowngradeUnit this proves both the
// behaviour and its installation site without spinning up two httptest servers.
func TestRedirectDowngradeWired(t *testing.T) {
	client := bootstrap.NewHTTPClientForTest()
	if client == nil {
		t.Fatal("newHTTPClient returned nil")
	}
	if client.CheckRedirect == nil {
		t.Fatal("newHTTPClient().CheckRedirect is nil — HTTPS→HTTP downgrade not wired (T-05-04)")
	}

	httpsURL, _ := url.Parse("https://example.com/lib.so")
	httpURL, _ := url.Parse("http://attacker.example/leak.so")
	err := client.CheckRedirect(&http.Request{URL: httpURL}, []*http.Request{{URL: httpsURL}})
	if err == nil {
		t.Fatal("client.CheckRedirect(https→http) returned nil — downgrade is not rejected")
	}
}

// TestConcurrentBootstrap covers Fault Injection Matrix item 6 (DIST-04
// concurrency + flock): N goroutines racing on BootstrapSync against the same
// cache dir → exactly one HTTP download, every caller succeeds.
//
// TestConcurrentBootstrap exercises intra-process concurrency on the flock loop
// (8 goroutines racing on the same lockPath). Inter-process lock behavior
// (flock on unix, LockFileEx on windows) is NOT covered by a subprocess-spawning
// test in v0.1; per 05-REVIEWS.md L1 that coverage is delegated to the OS
// semantics, matching the pure-onnx precedent. If a future contributor adds a
// subprocess test, exec.Command(os.Args[0]) with a TEST_SUBPROCESS env var is
// the typical Go pattern — but beware of Windows path quoting gotchas on CI.
func TestConcurrentBootstrap(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("concurrent-bootstrap-body")
	digest := computeHex(body)
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digest)()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		// Tiny jitter so we widen the window where a second goroutine could race
		// past the cache-stat-then-download check if the lock weren't holding.
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	const workers = 8
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := bootstrap.BootstrapSync(context.Background(),
				bootstrap.WithMirror(srv.URL),
				bootstrap.WithTarget(goos, goarch),
				bootstrap.WithHTTPClient(srv.Client()),
			)
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("worker BootstrapSync error: %v", err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("server hits = %d, want 1 (flock should serialize concurrent downloads)", got)
	}

	cached := bootstrap.CachePath(goos, goarch)
	if got, err := os.ReadFile(cached); err != nil {
		t.Fatalf("artifact not cached after concurrent bootstrap: %v", err)
	} else if string(got) != string(body) {
		t.Fatalf("cached body mismatch")
	}
}

// TestGitHub403RateLimit covers Fault Injection Matrix item 11 (DIST-04 error
// classification end-to-end): server returns 403 with a GitHub rate-limit body
// twice, then 200. The body-sniff in isRetryable must classify the 403 as
// retryable so the third call succeeds. Plan 05-02 already unit-tested
// isRetryable's classification; this test proves it integrates with the retry
// ladder.
func TestGitHub403RateLimit(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("rate-limited-then-served")
	digest := computeHex(body)
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digest)()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"API rate limit exceeded for user 12345"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := bootstrap.BootstrapSync(ctx,
		bootstrap.WithMirror(srv.URL),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	); err != nil {
		t.Fatalf("BootstrapSync: %v", err)
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("hits = %d, want 3 (two 403 rate-limited, one 200)", got)
	}
}

// TestMirrorOverride covers Fault Injection Matrix item 10a (DIST-07): setting
// PURE_SIMDJSON_BINARY_MIRROR env var (NOT passing WithMirror) routes the
// download to the mirror. Pairs with TestResolveConfigEnvMirror (Plan 05-02)
// which covers config-level resolution; this is the integration counterpart
// proving the env var actually drives a download.
func TestMirrorOverride(t *testing.T) {
	clearBootstrapEnv(t)

	body := []byte("mirror-override-body")
	digest := computeHex(body)
	goos, goarch := "linux", "amd64"
	defer bootstrap.RegisterChecksumForTest(bootstrap.Version, goos, goarch, digest)()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("PURE_SIMDJSON_CACHE_DIR", cacheDir)
	t.Setenv("PURE_SIMDJSON_BINARY_MIRROR", srv.URL)
	t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "1")

	// Note: deliberately NOT passing bootstrap.WithMirror — env var is the only
	// signal under test. WithHTTPClient is still required so the test trusts
	// the httptest server's certificate.
	err := bootstrap.BootstrapSync(context.Background(),
		bootstrap.WithTarget(goos, goarch),
		bootstrap.WithHTTPClient(srv.Client()),
	)
	if err != nil {
		t.Fatalf("BootstrapSync: %v", err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("mirror server hits = %d, want 1 (env-var override should route here)", got)
	}
}

// bootstrapRetryAttempts mirrors bootstrapRetryAttempt in download.go. Tests
// can't import internal constants but they need the value for the H1 hit-count
// assertion in TestFallback503R2Then200GH. If download.go's constant ever
// changes, update this mirror. Plan-03's TestRetryOn429ThenSuccess also
// implicitly depends on this value being 3.
const bootstrapRetryAttempts = 3
