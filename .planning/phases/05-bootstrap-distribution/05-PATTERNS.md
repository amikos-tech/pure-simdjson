# Phase 5: Bootstrap + Distribution — Pattern Map

**Mapped:** 2026-04-20
**Files analyzed:** 17 (new/modified)
**Analogs found:** 16 / 17 (docs/bootstrap.md has no code analog; pattern from pure-onnx docs/releases.md)

---

## File Classification

| New / Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---------------------|------|-----------|----------------|---------------|
| `library_loading.go` | loader | request-response | `library_loading.go` (Phase 3, same file) | exact — extend |
| `internal/bootstrap/version.go` | config | — | `internal/bootstrap/version.go` (pure-onnx `ort/bootstrap.go` const block) | role-match |
| `internal/bootstrap/checksums.go` | config | — | `internal/bootstrap/version.go` sibling | role-match |
| `internal/bootstrap/bootstrap.go` | service | request-response | `~/go/pkg/mod/github.com/amikos-tech/pure-onnx@v0.0.1/ort/bootstrap.go` (EnsureOnnxRuntime…) | near-exact lift |
| `internal/bootstrap/download.go` | network | request-response | `pure-onnx@v0.0.1/ort/bootstrap.go` (downloadRuntimeArchive…) | near-exact lift |
| `internal/bootstrap/cache.go` | cache-lock | file-I/O | `pure-onnx@v0.0.1/ort/bootstrap.go` (withProcessFileLock, defaultBootstrapCacheDir) | near-exact lift |
| `internal/bootstrap/url.go` | utility | transform | `pure-onnx@v0.0.1/ort/bootstrap.go` (runtimeArtifact.downloadURL, resolveRuntimeArtifact) | role-match |
| `internal/bootstrap/bootstrap_lock_unix.go` | middleware | file-I/O | `pure-onnx@v0.0.1/ort/bootstrap_lock_unix.go` | exact lift |
| `internal/bootstrap/bootstrap_lock_windows.go` | middleware | file-I/O | `pure-onnx@v0.0.1/ort/bootstrap_lock_windows.go` | exact lift |
| `internal/bootstrap/bootstrap_test.go` | test | — | `pure-onnx@v0.0.1/ort/bootstrap_test.go` | near-exact adapt |
| `cmd/pure-simdjson-bootstrap/main.go` | CLI | request-response | `pure-onnx@v0.0.1/ort/bootstrap.go` (cobra pattern from RESEARCH.md §Pattern 8) | partial-match |
| `cmd/pure-simdjson-bootstrap/fetch.go` | CLI | request-response | `pure-onnx@v0.0.1/ort/bootstrap.go` (EnsureOnnxRuntime… options surface) | partial-match |
| `cmd/pure-simdjson-bootstrap/verify.go` | CLI | file-I/O | `errors.go` (sentinel pattern) | partial-match |
| `cmd/pure-simdjson-bootstrap/platforms.go` | CLI | transform | `library_loading.go` (rustTargetTriple / platformLibraryName) | role-match |
| `cmd/pure-simdjson-bootstrap/version.go` | CLI | — | none in-repo; stdlib `runtime/debug` | partial-match |
| `errors.go` | errors | — | `errors.go` (same file — add sentinels) | exact — extend |
| `docs/bootstrap.md` | docs | — | `pure-onnx@v0.0.1/docs/releases.md` (cosign recipe structure) | docs-match |

---

## Pattern Assignments

### `library_loading.go` (loader, request-response) — MODIFY

**Analog:** `library_loading.go` (in-repo, same file — lines 66–101)

**What Phase 5 changes:** Replace `libraryCandidates()` + candidate-walk with a three-stage chain:
env-override (unchanged) → cache-hit → `bootstrap.BootstrapSync(internalCtx)` → cache-hit-after-bootstrap → fail.
The `libraryCandidates()` and `rustTargetTriple()` functions are **deleted**. `platformLibraryName()` moves to
`internal/bootstrap/url.go` (also needed there) — or kept here with a call through. Confirm with planner.

**Existing env-override pattern to keep verbatim** (`library_loading.go` lines 67–76):
```go
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
```

**New cache-hit + bootstrap chain to insert** (from RESEARCH.md §Pattern 10):
```go
// Stage 2: cache hit (no SHA-256 re-verify — D-04)
cachePath := bootstrap.CachePath(runtime.GOOS, runtime.GOARCH)
if _, err := os.Stat(cachePath); err == nil {
    return cachePath, []string{cachePath}, nil
}

// Stage 3: auto-bootstrap with internal timeout ctx
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
if err := bootstrap.BootstrapSync(ctx); err != nil {
    return "", []string{cachePath},
        fmt.Errorf("bootstrap failed (set %s to bypass): %w", libraryEnvPath, err)
}

// Stage 4: cache hit after bootstrap
if _, err := os.Stat(cachePath); err == nil {
    return cachePath, []string{cachePath}, nil
}
return "", []string{cachePath}, fmt.Errorf("shared library not found after bootstrap")
```

**What to copy verbatim vs adapt:**
- Env-override block (lines 67–76): copy verbatim, no changes.
- `activeLibrary()` (lines 29–64): copy verbatim, no changes needed — the mutex + cachedLibrary guard is already correct.
- `libraryCandidates()` / `rustTargetTriple()`: **delete**.
- `platformLibraryName()`: **move** to `internal/bootstrap/url.go`; keep a re-export or direct call here.

**Divergence notes:** The `(string, []string, error)` return shape of `resolveLibraryPath` is preserved — the `attempted` slice now only holds the computed cachePath (not a list of candidates). This maintains the `wrapLoadFailure(formatAttemptedPaths(attempted), err)` call in `activeLibrary()` unchanged.

---

### `internal/bootstrap/version.go` (config)

**Analog:** `pure-onnx@v0.0.1/ort/bootstrap.go` lines 29–47 (constants block)

**Pattern** (lines 29–32 of pure-onnx bootstrap.go):
```go
const (
    DefaultOnnxRuntimeVersion = "1.23.1"
    // ...other consts
)
```

**What to write** (adapted for pure-simdjson):
```go
package bootstrap

// Version is the library version pinned at compile time.
// CI-05 updates this constant at release time alongside checksums.go.
// ldflags -X is explicitly rejected (D-06): consumer go build does not
// run our build flags.
const Version = "0.1.0"
```

**What to copy verbatim vs adapt:** Pattern is trivial — one-line const. No verbatim lift needed; adapt package name and const name.

---

### `internal/bootstrap/checksums.go` (config)

**Analog:** none in-repo. Pattern derived from D-08 decision and pure-onnx's `WithBootstrapExpectedSHA256` usage.

**Pattern:**
```go
package bootstrap

// Checksums maps cache-path fragment "v<Version>/<os>-<arch>/<libname>" to
// its expected SHA-256 hex digest.
// CI-05 generates this map at release time. During development the map is
// empty; BootstrapSync returns ErrNoChecksum when an entry is missing.
var Checksums = map[string]string{
    // "v0.1.0/linux-amd64/libpure_simdjson.so":    "<sha256>",
    // "v0.1.0/linux-arm64/libpure_simdjson.so":    "<sha256>",
    // "v0.1.0/darwin-amd64/libpure_simdjson.dylib": "<sha256>",
    // "v0.1.0/darwin-arm64/libpure_simdjson.dylib": "<sha256>",
    // "v0.1.0/windows-amd64/pure_simdjson-msvc.dll": "<sha256>",
}
```

**What to copy verbatim vs adapt:** Write from scratch per D-08. The checksum key format mirrors the R2 URL path fragment exactly (D-07).

---

### `internal/bootstrap/bootstrap.go` (service, request-response)

**Analog:** `pure-onnx@v0.0.1/ort/bootstrap.go` (EnsureOnnxRuntimeSharedLibrary + resolveBootstrapConfig)

**Imports pattern** (from pure-onnx bootstrap.go lines 1–27, adapted):
```go
package bootstrap

import (
    "context"
    "errors"
    "fmt"
    "net"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "sync"
    "time"
)
```

**BootstrapOption + config pattern** (pure-onnx lines 105–121, adapted):
```go
// BootstrapOption configures BootstrapSync.
type BootstrapOption func(*bootstrapConfig) error

type bootstrapConfig struct {
    cacheDir   string
    version    string
    mirrorURL  string
    disableGH  bool
    httpClient *http.Client
    goos       string
    goarch     string
    destDir    string   // CLI --dest override
}

func WithMirror(rawURL string) BootstrapOption {
    return func(cfg *bootstrapConfig) error {
        cfg.mirrorURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
        return validateBaseURL(cfg.mirrorURL)
    }
}

func WithDest(path string) BootstrapOption {
    return func(cfg *bootstrapConfig) error {
        cfg.destDir = filepath.Clean(strings.TrimSpace(path))
        return nil
    }
}
```

**EnsureArtifact / BootstrapSync skeleton** (pure-onnx lines 225–280, adapted — archive extraction removed):
```go
func BootstrapSync(ctx context.Context, opts ...BootstrapOption) error {
    cfg, err := resolveConfig(opts...)
    if err != nil {
        return err
    }
    return ensureArtifact(ctx, cfg, cfg.goos, cfg.goarch)
}

func ensureArtifact(ctx context.Context, cfg bootstrapConfig, goos, goarch string) error {
    cachePath := artifactCachePath(cfg.cacheDir, cfg.version, goos, goarch)

    // Cache hit (no SHA-256 re-verify — D-04)
    if _, err := os.Stat(cachePath); err == nil {
        return nil
    }

    if err := os.MkdirAll(filepath.Dir(cachePath), 0700); err != nil {
        return fmt.Errorf("create cache dir: %w", err)
    }

    lockPath := filepath.Join(filepath.Dir(cachePath), ".lock")
    return withProcessFileLock(lockPath, func() error {
        // Re-check after lock (another process may have populated)
        if _, err := os.Stat(cachePath); err == nil {
            return nil
        }
        return downloadAndVerify(ctx, cfg, cachePath, goos, goarch)
    })
}
```

**resolveConfig pattern** (pure-onnx lines 307–376, heavily trimmed for pure-simdjson):
```go
func resolveConfig(opts ...BootstrapOption) (bootstrapConfig, error) {
    cfg := bootstrapConfig{
        cacheDir:   defaultCacheDir(),
        version:    Version,
        mirrorURL:  strings.TrimRight(strings.TrimSpace(os.Getenv("PURE_SIMDJSON_BINARY_MIRROR")), "/"),
        disableGH:  os.Getenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK") == "1",
        httpClient: newHTTPClient(),
        goos:       runtime.GOOS,
        goarch:     runtime.GOARCH,
    }
    for _, opt := range opts {
        if opt == nil {
            continue
        }
        if err := opt(&cfg); err != nil {
            return bootstrapConfig{}, err
        }
    }
    if cfg.mirrorURL != "" {
        if err := validateBaseURL(cfg.mirrorURL); err != nil {
            return bootstrapConfig{}, err
        }
    }
    return cfg, nil
}
```

**What to copy verbatim vs adapt:**
- `permanentBootstrapError` type (pure-onnx lines 60–89): **lift verbatim** (rename package).
- `validateBootstrapBaseURL` / `isLoopbackBootstrapHost` (lines 378–413): **lift verbatim** (rename).
- `newBootstrapHTTPClient` / `rejectHTTPSDowngradeRedirect` (lines 415–465): **lift verbatim** (rename).
- Archive extraction machinery (extractTGZArchive, etc.): **omit entirely** — pure-simdjson ships flat files.
- `resolveRuntimeArtifact` switch (lines 467–523): **adapt** — pure-simdjson uses a flat filename map, not struct; see `url.go` below.

**Divergence notes:**
- No archive extraction step (RESEARCH.md §Summary: "flat files, not tgz/zip").
- `BootstrapSync` takes `ctx context.Context` as first arg; pure-onnx's `EnsureOnnxRuntime…` is ctx-less.
- `disableGH` is env-driven (D-20); pure-onnx has no GitHub-fallback concept.

---

### `internal/bootstrap/download.go` (network, request-response)

**Analog:** `pure-onnx@v0.0.1/ort/bootstrap.go` lines 856–971 (downloadRuntimeArchive + downloadRuntimeArchiveOnce)

**Retry loop pattern** (pure-onnx lines 856–873 — with the D-13/D-14 deviation applied):
```go
// downloadWithRetry tries URL with Full-Jitter backoff + ctx-aware sleep.
// D-13: Full-Jitter replaces pure-onnx's linear time.Sleep(attempt*1s).
// D-14: ctx-aware select replaces time.Sleep for instant cancellation.
func downloadWithRetry(ctx context.Context, cfg bootstrapConfig, primaryURL, fallbackURL string) (tmpPath, hexDigest string, err error) {
    const maxAttempts = 3
    urls := []string{primaryURL}
    if !cfg.disableGH && fallbackURL != "" {
        urls = append(urls, fallbackURL) // D-15: R2 exhausted first, then GH
    }

    var lastErr error
    for _, url := range urls {
        for attempt := 0; attempt < maxAttempts; attempt++ {
            if attempt > 0 {
                if err := sleepWithJitter(ctx, attempt); err != nil {
                    return "", "", err // ctx cancelled
                }
            }
            tmpPath, hexDigest, err = downloadOnce(ctx, cfg, url)
            if err == nil {
                return tmpPath, hexDigest, nil
            }
            lastErr = fmt.Errorf("attempt %d/%d %s: %w", attempt+1, maxAttempts, url, err)
            if isPermanentBootstrapError(err) {
                break // no retry for permanent errors (404, checksum, redirect)
            }
        }
    }
    return "", "", fmt.Errorf("%w: %v (set %s to bypass)", ErrAllSourcesFailed, lastErr, "PURE_SIMDJSON_LIB_PATH")
}
```

**Full-Jitter sleep** (from RESEARCH.md §Pattern 1 — verified):
```go
// sleepWithJitter implements D-13 Full-Jitter + D-14 ctx-aware select.
// Uses math/rand/v2 which is auto-seeded in Go 1.22+ (no Seed() call needed).
import "math/rand/v2"

func sleepWithJitter(ctx context.Context, attempt int) error {
    const (
        base = 500 * time.Millisecond
        cap  = 8 * time.Second
    )
    shift := uint(attempt)
    if shift > 10 {
        shift = 10
    }
    expBackoff := base << shift
    if expBackoff > cap || expBackoff < 0 {
        expBackoff = cap
    }
    jitter := time.Duration(rand.Float64() * float64(base))
    d := expBackoff + jitter

    select {
    case <-time.After(d):
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Single-attempt download + SHA-256** (pure-onnx lines 875–971, archive machinery removed):
```go
// downloadOnce streams body through MultiWriter(tmpFile, sha256Hasher) in one pass.
// Temp file is written to cacheDir so os.Rename is atomic (same filesystem — D pitfall #3).
func downloadOnce(ctx context.Context, cfg bootstrapConfig, rawURL string) (tmpPath, hexDigest string, err error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
    if err != nil {
        return "", "", markPermanentBootstrapError(fmt.Errorf("create request: %w", err))
    }
    req.Header.Set("User-Agent", "pure-simdjson-bootstrap")

    resp, err := cfg.httpClient.Do(req)
    if err != nil {
        if isBootstrapRedirectPolicyError(err) {
            return "", "", markPermanentBootstrapError(err)
        }
        return "", "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
        statusErr := fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, rawURL, strings.TrimSpace(string(snippet)))
        if !isRetryable(resp.StatusCode, resp.Header, string(snippet)) {
            return "", "", markPermanentBootstrapError(statusErr)
        }
        return "", "", statusErr
    }

    cacheDir := filepath.Dir(cachePathForURL(rawURL)) // pre-created by ensureArtifact
    f, err := os.CreateTemp(cacheDir, "pure-simdjson-*.tmp") // same FS as final dest
    if err != nil {
        return "", "", markPermanentBootstrapError(fmt.Errorf("create temp: %w", err))
    }
    tmpPath = f.Name()
    success := false
    defer func() {
        f.Close()
        if !success {
            os.Remove(tmpPath)
        }
    }()

    h := sha256.New()
    const maxBytes int64 = 128 << 20 // 128 MiB — generous for a flat .so/.dylib/.dll
    written, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, maxBytes+1))
    if err != nil {
        return "", "", fmt.Errorf("write to temp: %w", err)
    }
    if written > maxBytes {
        return "", "", markPermanentBootstrapError(fmt.Errorf("response too large: %d bytes", written))
    }
    hexDigest = hex.EncodeToString(h.Sum(nil))
    success = true
    return tmpPath, hexDigest, nil
}
```

**GitHub 403 body-sniff for rate-limit** (pure-onnx lines 706–731):
```go
func isRetryable(statusCode int, headers http.Header, bodySnippet string) bool {
    switch statusCode {
    case http.StatusRequestTimeout, http.StatusTooManyRequests:
        return true
    }
    if statusCode >= 500 {
        return true
    }
    if statusCode == http.StatusForbidden {
        if headers.Get("Retry-After") != "" || headers.Get("X-RateLimit-Remaining") == "0" {
            return true
        }
        lower := strings.ToLower(bodySnippet)
        return strings.Contains(lower, "rate limit exceeded") ||
            strings.Contains(lower, "secondary rate limit")
    }
    return false
}
```

**What to copy verbatim vs adapt:**
- `permanentBootstrapError` / `markPermanentBootstrapError` / `isPermanentBootstrapError`: lift verbatim from pure-onnx lines 60–89 (into `bootstrap.go` or a shared internal file).
- `newBootstrapHTTPClient` / `rejectHTTPSDowngradeRedirect`: lift verbatim from pure-onnx lines 415–465.
- `isRetryableBootstrapHTTPStatus`: lift verbatim lines 91–96; extend with the GitHub 403 sniff from lines 706–731.
- The SHA-256 MultiWriter streaming pattern (lines 952–968): lift verbatim, remove archive-specific size constants.
- `os.CreateTemp(cacheDir, ...)` then `os.Remove` on failure pattern (lines 924–940): lift verbatim.

**Divergence notes:**
- `http.NewRequestWithContext(ctx, ...)` instead of `http.NewRequest` — ctx propagation for D-14.
- Full-Jitter backoff (D-13) instead of `time.Sleep(time.Duration(attempt)*time.Second)` (pure-onnx line 869).
- `downloadWithRetry` handles R2→GH fallback in a single loop with a URL slice; pure-onnx has no fallback concept.
- Max download size is 128 MiB for a flat shared library (pure-onnx uses 1 GiB for archives).

---

### `internal/bootstrap/cache.go` (cache-lock, file-I/O)

**Analog:** `pure-onnx@v0.0.1/ort/bootstrap.go` (withProcessFileLock lines 1283–1334, defaultBootstrapCacheDir lines 1370–1383)

**withProcessFileLock — lift verbatim** (pure-onnx lines 1283–1334):
```go
// withProcessFileLock acquires an exclusive flock on lockPath, calls fn,
// then releases the lock. Polling interval 200ms, timeout 2 minutes.
// Source: pure-onnx@v0.0.1/ort/bootstrap.go lines 1283–1334 (verbatim, rename package)
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
    const (
        lockAcquireTimeout = 2 * time.Minute
        lockRetryInterval  = 200 * time.Millisecond
        lockLogInterval    = 5 * time.Second
    )
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
            // log.Printf only if a logger is wired; otherwise omit (silent-on-success per D-28)
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
```

**Cache directory + path construction** (pure-onnx lines 1370–1383, adapted for per-version layout D-07):
```go
func defaultCacheDir() string {
    base, err := os.UserCacheDir()
    if err == nil && base != "" {
        return filepath.Join(base, "pure-simdjson")
    }
    // Fallback to temp on systems where UserCacheDir fails (rare).
    return filepath.Join(os.TempDir(), "pure-simdjson")
}

// CachePath returns the absolute path where the library artifact for goos/goarch
// is stored. Layout: <cacheDir>/v<Version>/<goos>-<goarch>/<libname> (D-07).
func CachePath(goos, goarch string) string {
    return artifactCachePath(defaultCacheDir(), Version, goos, goarch)
}

func artifactCachePath(cacheDir, version, goos, goarch string) string {
    return filepath.Join(cacheDir,
        "v"+version,
        goos+"-"+goarch,
        platformLibraryName(goos))
}
```

**Atomic rename after download** (pure-onnx pattern, simplified for flat file):
```go
// atomicInstall renames tmpPath to finalPath (same filesystem — os.CreateTemp
// in cacheDir guarantees same-FS). Removes tmpPath on failure.
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
```

**What to copy verbatim vs adapt:**
- `withProcessFileLock`: lift **verbatim** from pure-onnx lines 1283–1334; rename package, rename internal constants.
- `defaultBootstrapCacheDir`: lift verbatim lines 1370–1383; adapt path from `onnx-purego/onnxruntime` → `pure-simdjson`.
- `secureDirectoryPermission = 0o750` in pure-onnx: **change to 0700** per D-05 / pitfall #4.
- `secureLockFilePermission = 0o600`: lift verbatim.

**Divergence notes:**
- Cache layout is per-version (`v<Version>/<goos>-<goarch>/`) instead of per-archive-name (D-07).
- `os.MkdirAll(dir, 0700)` not `0o750` per DIST-05 / pitfall #4 security requirement.
- No staging directory or rename-directory step — pure-simdjson places a single file, not an extracted subtree.

---

### `internal/bootstrap/url.go` (utility, transform)

**Analog:** `pure-onnx@v0.0.1/ort/bootstrap.go` lines 467–535 (resolveRuntimeArtifact, downloadURL)

**Platform-to-filename map** (adapted from pure-onnx lines 467–523 — struct replaced by direct switch per D-10):
```go
package bootstrap

import "fmt"

// platformLibraryName returns the library filename for the given GOOS (D-10).
func platformLibraryName(goos string) string {
    switch goos {
    case "darwin":
        return "libpure_simdjson.dylib"
    case "linux":
        return "libpure_simdjson.so"
    case "windows":
        return "pure_simdjson-msvc.dll"
    default:
        return "libpure_simdjson"
    }
}

// SupportedPlatforms lists the five release targets (DIST-01).
var SupportedPlatforms = [][2]string{
    {"linux", "amd64"},
    {"linux", "arm64"},
    {"darwin", "amd64"},
    {"darwin", "arm64"},
    {"windows", "amd64"},
}
```

**R2 and GitHub URL constructors** (from RESEARCH.md §Code Examples, aligned with D-01 / D-02):
```go
const defaultR2BaseURL = "https://releases.amikos.tech/pure-simdjson"

func r2ArtifactURL(baseURL, version, goos, goarch string) string {
    osArch := goos + "-" + goarch
    lib := platformLibraryName(goos)
    return fmt.Sprintf("%s/v%s/%s/%s",
        strings.TrimRight(baseURL, "/"), version, osArch, lib)
}

func githubArtifactURL(version, goos, goarch string) string {
    lib := platformLibraryName(goos)
    return fmt.Sprintf(
        "https://github.com/amikos-tech/pure-simdjson/releases/download/v%s/%s",
        version, lib)
}

// checksumKey returns the map key used in checksums.go for a given platform.
func checksumKey(version, goos, goarch string) string {
    return fmt.Sprintf("v%s/%s-%s/%s", version, goos, goarch, platformLibraryName(goos))
}
```

**URL validation (loopback-allow pattern)** — lift verbatim from pure-onnx lines 378–413:
```go
// validateBaseURL rejects non-HTTPS URLs except for loopback hosts (tests).
// Source: pure-onnx@v0.0.1/ort/bootstrap.go lines 378–413 (verbatim, rename)
func validateBaseURL(rawURL string) error { ... }
func isLoopbackHost(host string) bool { ... }
```

**What to copy verbatim vs adapt:**
- `resolveRuntimeArtifact` switch logic: adapt — pure-simdjson has simpler flat-file naming (no archiveExtension / libraryGlob fields needed).
- `validateBootstrapBaseURL` / `isLoopbackBootstrapHost`: lift **verbatim**, rename.
- `platformLibraryName`: this is the existing function in `library_loading.go` lines 133–144 — move here, update call sites.

---

### `internal/bootstrap/bootstrap_lock_unix.go` (middleware, file-I/O)

**Analog:** `pure-onnx@v0.0.1/ort/bootstrap_lock_unix.go` (lines 1–23)

**Lift verbatim** (entire file — only change `package ort` → `package bootstrap`):
```go
//go:build !windows

package bootstrap

import (
    "errors"
    "os"

    "golang.org/x/sys/unix"
)

func lockFile(file *os.File) error {
    return unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

func unlockFile(file *os.File) error {
    return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

func isLockWouldBlock(err error) bool {
    return errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN)
}
```

**What to copy verbatim vs adapt:** Entire file is verbatim except package declaration.

---

### `internal/bootstrap/bootstrap_lock_windows.go` (middleware, file-I/O)

**Analog:** `pure-onnx@v0.0.1/ort/bootstrap_lock_windows.go` (lines 1–27)

**Lift verbatim** (entire file — only change `package ort` → `package bootstrap`):
```go
//go:build windows

package bootstrap

import (
    "errors"
    "os"

    "golang.org/x/sys/windows"
)

func lockFile(file *os.File) error {
    handle := windows.Handle(file.Fd())
    var overlapped windows.Overlapped
    flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY)
    return windows.LockFileEx(handle, flags, 0, 1, 0, &overlapped)
}

func unlockFile(file *os.File) error {
    handle := windows.Handle(file.Fd())
    var overlapped windows.Overlapped
    return windows.UnlockFileEx(handle, 0, 1, 0, &overlapped)
}

func isLockWouldBlock(err error) bool {
    return errors.Is(err, windows.ERROR_LOCK_VIOLATION) ||
        errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}
```

**What to copy verbatim vs adapt:** Entire file is verbatim except package declaration.

---

### `internal/bootstrap/bootstrap_test.go` (test)

**Analog:** `pure-onnx@v0.0.1/ort/bootstrap_test.go` (full file, 58KB)

**Test file skeleton** (pure-onnx lines 1–24, adapted):
```go
package bootstrap_test

import (
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "sync"
    "sync/atomic"
    "testing"
)
```

**httptest server helper** (pure-onnx lines 1785–1806, adapted for flat files instead of archives):
```go
// newFileServer serves a single flat file at the R2 URL path.
// Returns the server and a hit counter (for asserting single-download invariant).
func newFileServer(t *testing.T, goos, goarch, version string, body []byte) (*httptest.Server, *atomic.Int32) {
    t.Helper()
    hits := &atomic.Int32{}
    path := "/v" + version + "/" + goos + "-" + goarch + "/" + platformLibraryName(goos)
    mux := http.NewServeMux()
    mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
        hits.Add(1)
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(body)
    })
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.NotFound(w, r)
    })
    srv := httptest.NewServer(mux)
    t.Cleanup(srv.Close)
    return srv, hits
}
```

**clearBootstrapEnv pattern** (pure-onnx lines 1777–1783):
```go
func clearBootstrapEnv(t *testing.T) {
    t.Helper()
    t.Setenv("PURE_SIMDJSON_LIB_PATH", "")
    t.Setenv("PURE_SIMDJSON_BINARY_MIRROR", "")
    t.Setenv("PURE_SIMDJSON_DISABLE_GH_FALLBACK", "")
}
```

**Table-driven concurrent test** (pure-onnx lines 189–249, adapted):
```go
func TestBootstrapSyncConcurrentSingleDownload(t *testing.T) {
    clearBootstrapEnv(t)
    cacheDir := t.TempDir()
    // ...serve fake library bytes...
    const workers = 8
    var wg sync.WaitGroup
    errCh := make(chan error, workers)
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            if err := BootstrapSync(context.Background(),
                WithDest(cacheDir), withHTTPClient(srv.Client())); err != nil {
                errCh <- err
            }
        }()
    }
    wg.Wait()
    close(errCh)
    for err := range errCh {
        t.Fatalf("unexpected error: %v", err)
    }
    if got := hits.Load(); got != 1 {
        t.Fatalf("expected exactly 1 download, got %d", got)
    }
}
```

**In-repo test patterns to match** (from `library_loading_test.go`):
- `t.TempDir()` for all temp directories.
- `t.Setenv(key, val)` for env overrides (auto-restored on test cleanup).
- `t.Skipf(...)` when optional resources (built library) are absent.
- `withLibraryCacheClearedForTest(t)` pattern for resetting package-level state — create analogous `withCacheClearedForTest` in bootstrap package.
- `errors.Is(err, sentinel)` assertions, not string matching.

**What to copy verbatim vs adapt:**
- `clearBootstrapEnv`: adapt env var names only.
- `newArchiveServer`: adapt to `newFileServer` (flat file, no archive building).
- `buildORTArchive`: **omit** — no archive construction needed.
- Concurrent goroutine test structure (lines 189–249): lift verbatim structure; adapt option names.
- Checksum mismatch test (lines 251–277): adapt `WithBootstrapExpectedSHA256` → checksum key in `checksums.go`.

---

### `cmd/pure-simdjson-bootstrap/main.go` (CLI)

**Analog:** RESEARCH.md §Pattern 8 (cobra root command boilerplate)

**Root command pattern** (RESEARCH.md §Pattern 8, D-22, D-28):
```go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

func main() {
    rootCmd := &cobra.Command{
        Use:           "pure-simdjson-bootstrap",
        Short:         "Bootstrap pure-simdjson shared library artifacts",
        SilenceUsage:  true,
        SilenceErrors: true,
    }
    rootCmd.AddCommand(
        newFetchCmd(),
        newVerifyCmd(),
        newPlatformsCmd(),
        newVersionCmd(),
    )
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**What to copy verbatim vs adapt:** The cobra root boilerplate is ~10 lines; write from the RESEARCH.md pattern. No in-repo analog exists for CLI code. `SilenceUsage: true` and `SilenceErrors: true` are mandatory per D-28 (errors to stderr, not usage print on error).

---

### `cmd/pure-simdjson-bootstrap/fetch.go` (CLI, request-response)

**Analog:** `pure-onnx@v0.0.1/ort/bootstrap.go` (EnsureOnnxRuntime… option surface) + RESEARCH.md §Pattern 8

**fetch subcommand pattern** (D-24 flags, cobra RunE pattern):
```go
func newFetchCmd() *cobra.Command {
    var (
        allPlatforms bool
        targets      []string
        dest         string
        version      string
        mirror       string
    )
    cmd := &cobra.Command{
        Use:   "fetch",
        Short: "Download artifacts to cache or --dest",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runFetch(cmd.Context(), allPlatforms, targets, dest, version, mirror,
                cmd.ErrOrStderr())
        },
    }
    cmd.Flags().BoolVar(&allPlatforms, "all-platforms", false, "fetch for all 5 supported platforms")
    cmd.Flags().StringArrayVar(&targets, "target", nil, "fetch for specific os/arch (repeatable)")
    cmd.Flags().StringVar(&dest, "dest", "", "destination directory (default: OS user cache)")
    cmd.Flags().StringVar(&version, "version", "", "library version (default: embedded Version)")
    cmd.Flags().StringVar(&mirror, "mirror", "", "override R2 base URL")
    return cmd
}
```

**What to copy verbatim vs adapt:** cobra flag declaration shape is mechanical. `runFetch` calls `bootstrap.BootstrapSync(ctx, opts...)` directly — the options map 1:1 to `BootstrapOption` setters (D-24). Progress writes to `cmd.ErrOrStderr()` per D-28.

---

### `cmd/pure-simdjson-bootstrap/verify.go` (CLI, file-I/O)

**Analog:** `errors.go` (sentinel pattern) + `internal/bootstrap/checksums.go`

**verify subcommand pattern** (D-25 — SHA-256 re-verify against checksums.go):
```go
func newVerifyCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "verify",
        Short: "Re-verify SHA-256 of cached artifacts against embedded checksums",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runVerify(cmd.Context(), cmd.ErrOrStderr())
        },
    }
}
```

The `runVerify` function reads the cached file, computes SHA-256 via `crypto/sha256` + `io.Copy`, compares against `bootstrap.Checksums[checksumKey(Version, goos, goarch)]`, returns `bootstrap.ErrChecksumMismatch` on mismatch.

**What to copy verbatim vs adapt:** Error classification uses the same `errors.Is(err, sentinel)` pattern from `errors.go`. Output: "PASS <path>" to stdout, "FAIL <path>: <reason>" + non-zero exit per D-28.

---

### `cmd/pure-simdjson-bootstrap/platforms.go` (CLI, transform)

**Analog:** `library_loading.go` lines 133–144 (`platformLibraryName`) + `internal/bootstrap/url.go` (SupportedPlatforms)

**platforms subcommand pattern** (D-26):
```go
func newPlatformsCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "platforms",
        Short: "List supported platforms and local cache presence",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runPlatforms(cmd.Context(), os.Stdout, cmd.ErrOrStderr())
        },
    }
}
```

`runPlatforms` iterates `bootstrap.SupportedPlatforms`, checks `os.Stat(bootstrap.CachePath(goos, goarch))`, prints "✓ cached" or "✗ missing" per entry.

**What to copy verbatim vs adapt:** The `SupportedPlatforms` list and `CachePath` call are the only non-trivial pieces; both come from `internal/bootstrap/url.go` and `cache.go`. CLI output to stdout per D-28.

---

### `cmd/pure-simdjson-bootstrap/version.go` (CLI)

**Analog:** none in-repo. Stdlib `runtime/debug.ReadBuildInfo()`.

**version subcommand pattern** (D-27):
```go
func newVersionCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "version",
        Short: "Print library version, Go runtime version, and build info",
        RunE: func(cmd *cobra.Command, args []string) error {
            info, ok := debug.ReadBuildInfo()
            if ok {
                fmt.Fprintf(os.Stdout, "library:  %s\ngo:       %s\nmodule:   %s\n",
                    bootstrap.Version, runtime.Version(), info.Main.Version)
            } else {
                fmt.Fprintf(os.Stdout, "library: %s\ngo:      %s\n",
                    bootstrap.Version, runtime.Version())
            }
            return nil
        },
    }
}
```

**What to copy verbatim vs adapt:** Trivial — three Printf lines. No analog needed.

---

### `errors.go` (errors) — MODIFY

**Analog:** `errors.go` (in-repo, same file — existing sentinel var block lines 10–40)

**Existing sentinel pattern to extend** (lines 10–40):
```go
var (
    ErrInvalidHandle      = errors.New("invalid handle")
    ErrClosed             = errors.New("closed")
    // ... existing sentinels ...
    ErrABIVersionMismatch = errors.New("abi version mismatch")
)
```

**New sentinels to add** (D-08/D-21, pitfall §Checksum):
```go
var (
    // ErrChecksumMismatch reports that a downloaded artifact's SHA-256 digest did
    // not match the value in internal/bootstrap/checksums.go. This is a permanent
    // error: the bootstrap chain does not retry on checksum mismatch.
    ErrChecksumMismatch = errors.New("checksum mismatch")

    // ErrAllSourcesFailed reports that all download sources (R2 + GitHub fallback)
    // were exhausted without a successful artifact download.
    ErrAllSourcesFailed = errors.New("all sources failed")

    // ErrNoChecksum reports that checksums.go has no entry for the requested
    // platform/version. This is expected during development before CI-05 runs.
    ErrNoChecksum = errors.New("no checksum for platform")
)
```

**What to copy verbatim vs adapt:** Follow the exact comment style of existing sentinels (`// ErrX reports that ...`). No changes to existing sentinels or `wrapLoadFailure`.

---

### `docs/bootstrap.md` (docs)

**Analog:** `pure-onnx@v0.0.1/docs/releases.md` (cosign recipe structure, D-29)

**Doc structure pattern** (from pure-onnx releases.md and D-21 / D-29 / D-28):
The document must cover:
1. Quick start (env-var override for air-gapped)
2. Environment variables table (`PURE_SIMDJSON_LIB_PATH`, `PURE_SIMDJSON_BINARY_MIRROR`, `PURE_SIMDJSON_DISABLE_GH_FALLBACK`)
3. Corporate firewall / mirror setup
4. Offline pre-fetch with `pure-simdjson-bootstrap fetch --all-platforms --dest ./vendor-libs`
5. `PURE_SIMDJSON_LIB_PATH` bypass for air-gapped runtime

**Cosign recipe** (pure-onnx releases.md lines 63–91, adapted):
```bash
TAG=v0.1.0
BASE_URL="https://releases.amikos.tech/pure-simdjson/${TAG}"

curl -LO "${BASE_URL}/libpure_simdjson.so"
curl -LO "${BASE_URL}/libpure_simdjson.so.sig"
curl -LO "${BASE_URL}/libpure_simdjson.so.pem"

cosign verify-blob \
  --signature libpure_simdjson.so.sig \
  --certificate libpure_simdjson.so.pem \
  --certificate-identity "https://github.com/amikos-tech/pure-simdjson/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  libpure_simdjson.so
```

**What to copy verbatim vs adapt:**
- Cosign command structure: lift verbatim from pure-onnx releases.md; adapt repo/artifact names.
- Mirror setup section: write from scratch per D-19, D-20.
- Env var table: write from scratch per D-21.

---

## Shared Patterns

### Error Wrapping (apply to all `internal/bootstrap/*.go` and modified `library_loading.go`)

**Source:** `errors.go` lines 152–161 (`wrapLoadFailure`)

```go
// Pattern: attach stage string to every wrapping site
func wrapLoadFailure(message string, err error) error {
    loadErr := errLoadLibrary
    if err != nil {
        loadErr = fmt.Errorf("%w: %v", errLoadLibrary, err)
    }
    return newError(0, nativeDetails{message: message, offset: ffi.LastErrorOffsetUnknown}, loadErr)
}
```

Bootstrap errors follow the same human-readable stage pattern:
`"download v0.1.0/linux-amd64 from R2"`, `"verify SHA-256 of <path>"`, `"acquire lock <path>"`.

### permanentBootstrapError (apply to `download.go`, `cache.go`, `bootstrap.go`)

**Source:** `pure-onnx@v0.0.1/ort/bootstrap.go` lines 60–89 (lift verbatim)

Placement: define in `internal/bootstrap/bootstrap.go`; use from `download.go`, `cache.go`.

### env-var read pattern (apply to `bootstrap.go::resolveConfig`)

**Source:** `library_loading.go` line 67 + `pure-onnx@v0.0.1/ort/bootstrap.go` line 314

```go
// Trim + Getenv inline (no helper function — matches existing codebase style)
envPath := strings.TrimSpace(os.Getenv("PURE_SIMDJSON_BINARY_MIRROR"))
```

### Build tag file-pair convention (apply to `bootstrap_lock_unix.go` / `bootstrap_lock_windows.go`)

**Source:** `library_unix.go` line 1 / `library_windows.go` line 1

```go
//go:build !windows   // unix file
//go:build windows    // windows file
```

Same convention already in `library_unix.go` and `library_windows.go`; the bootstrap lock files must match.

### Test helper: env isolation (apply to `bootstrap_test.go`)

**Source:** `library_loading_test.go` lines 157–169 (`withLibraryCacheClearedForTest`)

```go
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
```

Bootstrap tests need an analogous helper that resets any package-level state (no global cache in bootstrap package — but tests must clear env vars via `t.Setenv` for isolation).

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/bootstrap/version.go` | config | — | Trivial one-line const; no meaningful analog |
| `cmd/pure-simdjson-bootstrap/version.go` | CLI | — | `runtime/debug.ReadBuildInfo()` is stdlib-only; no existing CLI in repo |

---

## Key Deviation Table (for Planner)

| Area | pure-onnx pattern | pure-simdjson pattern | Source |
|------|-------------------|-----------------------|--------|
| Retry sleep | `time.Sleep(attempt * time.Second)` at line 869 | Full-Jitter + ctx-aware `select` | D-13/D-14 |
| Cache layout | single-slot archive subdir | per-version `v<Version>/<goos>-<goarch>/` | D-07 |
| Artifacts | tgz/zip archives (extract step) | flat `.so`/`.dylib`/`.dll` (no extract) | DIST-01 |
| Directory perms | `0o750` (`secureDirectoryPermission`) | `0700` | DIST-05 / pitfall #4 |
| ctx on download | ctx-less `http.NewRequest` | `http.NewRequestWithContext(ctx, ...)` | D-14 |
| GH fallback | no concept (primary is GH) | explicit fallback after R2 exhaustion | D-15 |
| Checksum source | `WithBootstrapExpectedSHA256` option | embedded `checksums.go` map | D-08 |

---

## Metadata

**Analog search scope:** `/Users/tazarov/experiments/amikos/pure-simdjson/` (in-repo) + `~/go/pkg/mod/github.com/amikos-tech/pure-onnx@v0.0.1/ort/` (external module cache)
**Files scanned:** 14 source files
**Pattern extraction date:** 2026-04-20
