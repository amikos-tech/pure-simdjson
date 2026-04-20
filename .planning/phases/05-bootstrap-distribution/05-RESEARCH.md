# Phase 5: Bootstrap + Distribution - Research

**Researched:** 2026-04-20
**Domain:** Go binary bootstrap: R2 download, SHA-256 verification, OS cache, flock, atomic rename, Cobra CLI
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Loader chain: `PURE_SIMDJSON_LIB_PATH` → cache-hit → download-then-cache → fail. Phase-3 local `target/` branch dropped.
- **D-02:** `NewParser()` auto-downloads on cache miss using internal `http.Client{Timeout: 2*time.Minute}` + 30s dial/TLS sub-timeouts.
- **D-03:** `BootstrapSync(ctx context.Context, opts ...BootstrapOption) error` is the caller-owned preflight.
- **D-04:** Cache-hit path does NOT re-verify SHA-256 on every `NewParser()` call. ABI-version handshake (`get_abi_version()`) is the post-dlopen guard.
- **D-05:** Concurrent first-run bootstrap across multiple processes coordinated via `.lock` file with `flock` (lifted from pure-onnx).
- **D-06:** Library version pinned via `const Version = "0.1.0"` in `internal/bootstrap/version.go`. `ldflags -X` rejected.
- **D-07:** Cache layout: `<userCacheDir>/pure-simdjson/v<Version>/<os>-<arch>/lib<name>.<ext>`.
- **D-08:** `internal/bootstrap/checksums.go` is a `map[string]string` keyed by `"v<Version>/<os>-<arch>/lib<name>.<ext>"`.
- **D-09:** On module upgrade, `NewParser()` routes to fresh subdirectory automatically. Previous version preserved.
- **D-10:** Filenames: `libpure_simdjson.so` (linux), `libpure_simdjson.dylib` (darwin), `pure_simdjson-msvc.dll` (windows).
- **D-11:** `get_abi_version()` remains post-dlopen runtime check — orthogonal to version pinning.
- **D-12:** `net/http` stdlib only — no third-party retry wrappers.
- **D-13:** Full-Jitter backoff: `min(500ms * 2^attempt, 8s) + rand.Float64()*500ms`, 3–4 attempts, `math/rand/v2`.
- **D-14:** Ctx-aware sleep: `select { case <-time.After(d): case <-ctx.Done(): return ctx.Err() }`.
- **D-15:** R2 → GitHub Releases fallback fires after all R2 attempts exhausted.
- **D-16:** Retryable HTTP statuses: 408, 429, 500, 502, 503, 504 + GitHub 403 body-sniff for "rate limit".
- **D-17:** Permanent failures: 404, non-ratelimit 403, SHA-256 mismatch, ABI mismatch, HTTPS→HTTP redirect. 404 against R2 does fall through to GH fallback; checksum mismatch does not retry.
- **D-18:** Two timeout clocks: per-request `http.Client{Timeout: 2*time.Minute}` + transport sub-timeouts; total: caller ctx deadline on `BootstrapSync`.
- **D-19:** `PURE_SIMDJSON_BINARY_MIRROR` overrides R2 base URL. GH fallback still fires on mirror failure.
- **D-20:** `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1` disables GitHub fallback for hermetic deployments.
- **D-21:** Error classification uses `errors.Is/As` on stdlib errors; final error wraps with hint pointing to `PURE_SIMDJSON_LIB_PATH`.
- **D-22:** CLI framework is `spf13/cobra`.
- **D-23:** v0.1 verbs: `fetch`, `verify`, `platforms`, `version`.
- **D-24:** `fetch` flags: `--all-platforms`, `--target=os/arch`, `--dest=<path>`, `--version=<semver>`, `--mirror=<url>`.
- **D-25:** `verify` — re-verifies SHA-256 of locally cached artifacts against `internal/bootstrap/checksums.go`. No cosign.
- **D-26:** `platforms` — lists 5 supported targets with cache presence indicator.
- **D-27:** `version` — prints library version, Go runtime version, `runtime/debug.ReadBuildInfo()` output.
- **D-28:** Output: human-friendly progress to stderr, silent-on-success to stdout, non-zero exit on failure.
- **D-29:** Cosign verification is docs-only in v0.1.
- **D-30:** No Go code imports `sigstore/sigstore-go` in v0.1.
- **D-31:** SHA-256 integrity always-on, verified before `dlopen`.
- **D-32:** v0.2 escape hatch for in-process cosign verification if demand materializes.

### Claude's Discretion

- Exact Go file layout under `internal/bootstrap/` and `cmd/pure-simdjson-bootstrap/`.
- Exact `BootstrapOption` functional-options surface.
- Exact progress-reporting shape in the CLI.
- Exact typed error surface for permanent-vs-retryable distinction.
- Exact cache-lock file name and acquisition timeout budget.
- Whether to add `PURE_SIMDJSON_CACHE_DIR` override.

### Deferred Ideas (OUT OF SCOPE)

- `Purge(ctx, keepLast int) error` cache-cleanup helper (v0.2).
- In-process cosign verification via `sigstore-go` (v0.2).
- `--json` output mode for the CLI.
- `PURE_SIMDJSON_CACHE_DIR` override (may add if ergonomics warrant).
- Cold-start benchmark (Phase 7).
- HTTP Range resume for partial downloads.
- `PURE_SIMDJSON_QUIET`.
- Shell-out to `cosign` CLI if on PATH.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DIST-01 | Pre-built shared libraries uploaded to CloudFlare R2 at `releases.amikos.tech/pure-simdjson/v<version>/<os>-<arch>/lib<name>.<ext>` | URL construction pattern documented; no archive extraction needed (flat file, not tgz/zip) |
| DIST-02 | GitHub Releases mirror as fallback source | pure-onnx fallback pattern confirmed; GitHub 403 body-sniff verified |
| DIST-03 | SHA-256 table embedded in Go source (`internal/bootstrap/checksums.go`) | `crypto/sha256` + `io.Copy` streaming pattern verified in Go 1.24 |
| DIST-04 | `BootstrapSync(ctx)` downloads, verifies, and caches the library | Full functional-options API pattern lifted from pure-onnx |
| DIST-05 | Auto-download on first `NewParser()` if not cached; OS user-cache-dir with 0700 perms | `os.UserCacheDir()` verified; 0700 perm confirmed via test |
| DIST-06 | `PURE_SIMDJSON_LIB_PATH` env var overrides download entirely | Already implemented in Phase 3 `library_loading.go` |
| DIST-07 | `PURE_SIMDJSON_BINARY_MIRROR` env var overrides R2 base URL | Env var surface documented; validated URL-only override pattern |
| DIST-08 | `cmd/pure-simdjson-bootstrap` CLI pre-downloads artifacts | cobra v1.10.2 confirmed stable; subcommand structure mapped |
| DIST-09 | Windows `LoadLibrary` uses full path | Already enforced in Phase 3 `library_windows.go` via `windows.LoadLibrary(path)` |
| DIST-10 | Cosign keyless OIDC signing; verification documented but optional | docs-only pattern from pure-onnx `docs/releases.md` |
| DOC-05 | `docs/bootstrap.md` — env vars, mirror setup, air-gapped install flow | Content outline derived from requirements and D-21 error hint pattern |
</phase_requirements>

---

## Summary

Phase 5 delivers the full binary bootstrap pipeline for `pure-simdjson`. The canonical implementation model is `pure-onnx@v0.0.1/ort/bootstrap.go`, which provides a near-complete template for the HTTP download loop, flock-based concurrency control, atomic rename, checksum verification, and functional-options API. Two deliberate deviations from pure-onnx are: (1) **Full-Jitter exponential backoff** (replacing linear `time.Duration(attempt)*time.Second`) using `math/rand/v2.Float64()`, which is auto-seeded in Go 1.22+ and requires no `Seed()` call; (2) **per-version cache subdirectories** matching the R2 URL layout one-to-one, enabling parallel-version coexistence and clean rollback.

The distribution artifacts are flat shared-library files (`.so`, `.dylib`, `.dll`), NOT archives — this is a critical simplification vs pure-onnx which downloads and extracts tgz/zip archives. Phase 5's download loop writes the file directly with `os.CreateTemp` + SHA-256 in a single `io.Copy` pass, then renames atomically. There is no extraction step.

The CLI uses `spf13/cobra v1.10.2` (latest stable, Go 1.24-compatible). The four verbs (`fetch`, `verify`, `platforms`, `version`) map cleanly to cobra subcommands with `SilenceUsage: true` and `SilenceErrors: true` on the root command. Windows LoadLibrary full-path invariant is already enforced in Phase 3's `library_windows.go`; Phase 5 must only ensure the cache path it writes to is always absolute before calling `activeLibrary()`.

**Primary recommendation:** Lift pure-onnx's `bootstrap.go` + `bootstrap_lock_*.go` skeleton, strip the archive-extraction machinery (not needed), apply the two deliberate deviations, and wire into the existing `resolveLibraryPath()` extension point.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Env-var override (`PURE_SIMDJSON_LIB_PATH`) | Go library (loader) | — | Already in Phase 3 `resolveLibraryPath()` |
| Cache lookup | Go library (`internal/bootstrap`) | — | Stateless path check; no network |
| Download + verify (R2 primary) | Go library (`internal/bootstrap`) | — | Pure Go stdlib HTTP |
| Download fallback (GitHub Releases) | Go library (`internal/bootstrap`) | — | Triggered only after R2 exhaustion |
| Flock / concurrent process safety | Go library (`internal/bootstrap/cache.go`) | OS kernel | `golang.org/x/sys/unix.Flock` / `windows.LockFileEx` |
| SHA-256 verification | Go library (`internal/bootstrap`) | — | Always before dlopen; pure crypto/sha256 |
| ABI version check post-dlopen | Go library (`library_loading.go`) | — | Carried from Phase 3; orthogonal to bootstrap |
| Cache directory creation / permissions | Go library (`internal/bootstrap/cache.go`) | OS | `os.MkdirAll(dir, 0700)` on unix |
| CLI offline pre-fetch | cmd tier (`cmd/pure-simdjson-bootstrap`) | `internal/bootstrap` (reuse) | Thin cobra wrapper over same bootstrap package |
| cosign verification | External tool (user's shell) | — | docs-only in v0.1; no Go code |

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` | stdlib (Go 1.24) | HTTP download client | D-12: stdlib-only, pure-* family convention |
| `crypto/sha256` | stdlib | SHA-256 digest | DIST-03 requirement; streaming via `io.Copy` |
| `math/rand/v2` | stdlib (Go 1.22+) | Full-Jitter backoff randomization | Auto-seeded; no `Seed()` call needed |
| `golang.org/x/sys` | v0.31.0 (already in go.mod) | `unix.Flock`, `windows.LockFileEx` | Already a project dependency |
| `github.com/spf13/cobra` | v1.10.2 | Bootstrap CLI framework | D-22: justified by 4-verb CLI scope |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os` (stdlib) | stdlib | `UserCacheDir`, `MkdirAll`, `CreateTemp`, `Rename` | All cache/file operations |
| `encoding/hex` | stdlib | SHA-256 hex encoding | `hex.EncodeToString(h.Sum(nil))` |
| `runtime/debug` | stdlib | `ReadBuildInfo()` for CLI `version` verb | CLI only |
| `context` | stdlib | ctx-aware sleep and download cancellation | `BootstrapSync(ctx)` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| stdlib retry loop | `hashicorp/go-retryablehttp` | Extra dep; D-12 locks stdlib-only |
| `math/rand/v2` | `cenkalti/backoff` | Extra dep; D-12 locks stdlib-only |
| `spf13/cobra` | stdlib `flag` | D-22 locked cobra for 4-verb growth path |

**Installation:**
```bash
cd /path/to/pure-simdjson
go get github.com/spf13/cobra@v1.10.2
```

`golang.org/x/sys v0.31.0` is already in `go.mod` — no additional install needed.

**Version verification:** [VERIFIED: go list -m -versions github.com/spf13/cobra] Latest stable: `v1.10.2` (released 2025-12-03). `golang.org/x/sys v0.31.0` confirmed in `go.mod`.

---

## Architecture Patterns

### System Architecture Diagram

```
[NewParser() call]
        |
        v
[resolveLibraryPath()] ──── PURE_SIMDJSON_LIB_PATH set? ──────► [abs path check] ─► [loadLibrary(absPath)]
        |                            (no)
        v
[cache hit? <userCacheDir>/pure-simdjson/v<Ver>/<os>-<arch>/lib<name>.<ext>]
        |                            (yes)
        |────────────────────────────────────────────────────► [loadLibrary(cachePath)]
        | (no)
        v
[BootstrapSync(internal background ctx)]
        |
        v
[acquire .lock file (flock / LockFileEx)]
        |
        v
[cache hit after lock? (another process may have populated)] ──► [loadLibrary(cachePath)]
        | (no)
        v
[buildDownloadURL(R2 primary)] ──► [HTTP GET with retry loop]
        |                              (Full-Jitter, 3–4 attempts)
        |                              408/429/5xx → retry
        |                              404 → fallback to GH
        |                              HTTPS→HTTP redirect → permanent fail
        |                              ctx cancel → return ctx.Err()
        v
[SHA-256 verify against checksums.go] ──► mismatch → ErrChecksumMismatch (permanent, no retry)
        |
        v
[os.CreateTemp(cacheDir, "*.tmp")] ── io.Copy ──► [sha256.New() + tmpFile]
        |
        v
[os.Rename(tmpFile, finalCachePath)]   ← atomic within same filesystem
        |
        v
[release .lock file]
        |
        v
[loadLibrary(finalCachePath)]
        |
        v
[ffi.Bind(handle, lookupSymbol)] ──► [get_abi_version()] ── mismatch ──► ErrABIVersionMismatch
        |
        v
[cachedLibrary = &loadedLibrary{...}]

[GH Releases fallback path]:
After all R2 retries exhausted:
  build GH Releases URL → same retry loop → SHA-256 verify → same file placement
  If GH also fails → ErrAllSourcesFailed (with hint: set PURE_SIMDJSON_LIB_PATH)
```

### Recommended Project Structure

```
internal/
└── bootstrap/
    ├── version.go          # const Version = "0.1.0"
    ├── checksums.go        # var Checksums = map[string]string{...}
    ├── bootstrap.go        # BootstrapSync(ctx, opts...) + BootstrapOption type
    ├── download.go         # HTTP client, retry loop, R2+GH URL construction
    ├── cache.go            # cache layout, MkdirAll 0700, flock, atomic rename
    ├── bootstrap_lock_unix.go    # unix.Flock (build !windows)
    └── bootstrap_lock_windows.go # windows.LockFileEx (build windows)

cmd/
└── pure-simdjson-bootstrap/
    ├── main.go             # cobra root command wiring
    ├── fetch.go            # 'fetch' subcommand
    ├── verify.go           # 'verify' subcommand
    ├── platforms.go        # 'platforms' subcommand
    └── version.go          # 'version' subcommand

docs/
└── bootstrap.md            # new doc: env vars, mirror, air-gapped, cosign recipe

library_loading.go          # rewrite resolveLibraryPath() to chain env→cache→bootstrap→fail
```

### Pattern 1: Full-Jitter Exponential Backoff with ctx-aware sleep

**What:** Capped exponential backoff with full random jitter and immediate ctx propagation.
**When to use:** Every retryable download attempt in `downloadWithRetry`.

```go
// Source: D-13 + D-14; math/rand/v2 verified auto-seeded in Go 1.22+
import (
    "context"
    "math/rand/v2"
    "time"
)

func sleepWithJitter(ctx context.Context, attempt int) error {
    const (
        base = 500 * time.Millisecond
        cap  = 8 * time.Second
    )
    shift := uint(attempt)
    if shift > 10 {
        shift = 10 // guard against overflow at high attempt counts
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

Output verified: attempt 0→~675ms, attempt 1→~1.3s, attempt 2→~2.3s, attempt 3→~4.3s, attempt 4→~8.1s (capped). [VERIFIED: go run /tmp/test_jitter2.go]

### Pattern 2: Download + SHA-256 in Single io.Copy Pass

**What:** Stream body through `io.MultiWriter(tmpFile, sha256Hasher)` so no full-file buffer is needed. Verify hex digest against `checksums.go` before returning path.
**When to use:** Every artifact download.

```go
// Source: pure-onnx bootstrap.go lines 952-968 + crypto/sha256 stdlib docs
import (
    "crypto/sha256"
    "encoding/hex"
    "io"
    "os"
)

func downloadToTemp(cacheDir string, body io.Reader, maxBytes int64) (tmpPath, hexDigest string, err error) {
    f, err := os.CreateTemp(cacheDir, "pure-simdjson-*.tmp")
    if err != nil {
        return "", "", err
    }
    tmpPath = f.Name()
    h := sha256.New()
    written, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(body, maxBytes+1))
    _ = f.Close()
    if err != nil {
        os.Remove(tmpPath)
        return "", "", err
    }
    if written > maxBytes {
        os.Remove(tmpPath)
        return "", "", ErrDownloadTooLarge
    }
    return tmpPath, hex.EncodeToString(h.Sum(nil)), nil
}
```

[VERIFIED: crypto/sha256 streaming pattern confirmed; hex.EncodeToString verified]

### Pattern 3: flock-Based Cross-Process Concurrency

**What:** Acquire an exclusive file lock before attempting download. Check cache again after lock acquisition (another process may have populated it).
**When to use:** Any concurrent process bootstrap scenario.

```go
// Source: pure-onnx bootstrap_lock_unix.go + bootstrap_lock_windows.go (verified)

// unix (bootstrap_lock_unix.go):
//go:build !windows
import "golang.org/x/sys/unix"
func lockFile(f *os.File) error {
    return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}
func unlockFile(f *os.File) error {
    return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
func isLockWouldBlock(err error) bool {
    return errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN)
}

// windows (bootstrap_lock_windows.go):
//go:build windows
import "golang.org/x/sys/windows"
func lockFile(f *os.File) error {
    handle := windows.Handle(f.Fd())
    var ol windows.Overlapped
    return windows.LockFileEx(handle,
        windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
        0, 1, 0, &ol)
}
func unlockFile(f *os.File) error {
    handle := windows.Handle(f.Fd())
    var ol windows.Overlapped
    return windows.UnlockFileEx(handle, 0, 1, 0, &ol)
}
func isLockWouldBlock(err error) bool {
    return errors.Is(err, windows.ERROR_LOCK_VIOLATION) ||
           errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}
```

Lock loop times out after 2 minutes (pure-onnx `bootstrapLockAcquireTimeout`), polling every 200ms. [VERIFIED: pure-onnx bootstrap.go lines 1283–1334]

### Pattern 4: Atomic File Placement

**What:** `os.CreateTemp(cacheDir, prefix)` then `os.Rename(tmp, final)` — atomic within the same filesystem partition.
**When to use:** All file write operations in the bootstrap pipeline.

Verified on darwin/arm64: `os.Rename` succeeds atomically within same FS. [VERIFIED: go run /tmp/test_atomic.go]

On Windows: `os.Rename` in Go 1.22+ uses `MoveFileEx` with `MOVEFILE_REPLACE_EXISTING`, making it atomic for same-volume moves. Cache directory is under `%LocalAppData%`, same volume as temp dir. Write temp to `<cacheDir>/*.tmp` then rename to same directory — guarantees same-volume atomicity. [ASSUMED: Go's os.Rename Windows behavior; verified at code level in Go source but not tested on Windows host]

### Pattern 5: HTTP Client with HTTPS-downgrade rejection

**What:** Reject redirects that downgrade from HTTPS to HTTP — a TLS-interception signal per pitfall #20.
**When to use:** Bootstrap HTTP client construction only (not general use).

```go
// Source: pure-onnx bootstrap.go lines 415–461 (lifted verbatim, adapted)
func newBootstrapHTTPClient() *http.Client {
    transport := &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        DialContext: (&net.Dialer{
            Timeout: 30 * time.Second,
        }).DialContext,
        TLSHandshakeTimeout:   10 * time.Second,
        ResponseHeaderTimeout: 30 * time.Second,
        IdleConnTimeout:       90 * time.Second,
    }
    return &http.Client{
        Timeout:       2 * time.Minute,
        Transport:     transport,
        CheckRedirect: rejectHTTPSDowngradeRedirect,
    }
}

func rejectHTTPSDowngradeRedirect(req *http.Request, via []*http.Request) error {
    if len(via) >= 10 {
        return fmt.Errorf("stopped after 10 redirects")
    }
    if len(via) == 0 {
        return nil
    }
    prev := via[len(via)-1]
    if strings.EqualFold(prev.URL.Scheme, "https") &&
       strings.EqualFold(req.URL.Scheme, "http") {
        return fmt.Errorf("redirect from HTTPS to HTTP rejected: %s -> %s",
            prev.URL.Redacted(), req.URL.Redacted())
    }
    return nil
}
```

### Pattern 6: GitHub 403 Rate-Limit Body Sniff

**What:** GitHub's API returns HTTP 403 for rate-limiting. The body contains `"rate limit exceeded"` or `"secondary rate limit"`. Pure-onnx's pattern checks both body snippet and the `X-RateLimit-Remaining: 0` header.
**When to use:** Only for GitHub Releases fallback path; not for R2.

```go
// Source: pure-onnx bootstrap.go lines 706–730 (verified)
func isRetryableGitHubStatus(statusCode int, headers http.Header, bodySnippet string) bool {
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

Note: GitHub's unauthenticated REST API rate limit is 60 requests/hour. GitHub Releases asset downloads do NOT count against this limit — they are plain HTTPS file downloads from `objects.githubusercontent.com`. The 403 body-sniff is for GitHub API metadata calls only. For direct `.so`/`.dll`/`.dylib` asset downloads, 403 is a permanent failure (private repo or expired URL). [VERIFIED: GitHub documentation + pure-onnx pattern]

### Pattern 7: `permanentBootstrapError` sentinel type

**What:** A thin wrapper type marks errors that must not be retried. `isPermanentBootstrapError` breaks the retry loop immediately without waiting for attempt count exhaustion.
**When to use:** On: SHA-256 mismatch, HTTPS→HTTP redirect, HTTP 404 (permanent not-found), malformed URL.

```go
// Source: pure-onnx bootstrap.go lines 61–89 (lifted verbatim)
type permanentBootstrapError struct{ cause error }
func (e *permanentBootstrapError) Error() string  { return e.cause.Error() }
func (e *permanentBootstrapError) Unwrap() error  { return e.cause }
func markPermanentBootstrapError(err error) error { return &permanentBootstrapError{cause: err} }
func isPermanentBootstrapError(err error) bool {
    var t *permanentBootstrapError
    return errors.As(err, &t)
}
```

### Pattern 8: Cobra CLI structure

**What:** Root command with `SilenceUsage: true` + `SilenceErrors: true`; subcommand errors written to `cmd.ErrOrStderr()`; `os.Exit(1)` only in `main.go` after error from `rootCmd.Execute()`.
**When to use:** All four CLI verbs.

```go
// Source: cobra docs + D-28
rootCmd := &cobra.Command{
    Use:          "pure-simdjson-bootstrap",
    Short:        "Bootstrap pure-simdjson shared library artifacts",
    SilenceUsage: true,
    SilenceErrors: true,
}
// Per verb:
fetchCmd := &cobra.Command{
    Use:   "fetch",
    Short: "Download artifacts to cache or --dest",
    RunE: func(cmd *cobra.Command, args []string) error {
        // returns error for non-zero exit; prints to stderr
        return runFetch(cmd, args)
    },
}
rootCmd.AddCommand(fetchCmd)
```

[VERIFIED: cobra v1.10.2 (latest stable per go list -m -versions)]

### Pattern 9: OS User Cache Directory

**What:** `os.UserCacheDir()` returns platform-appropriate base directory. Permissions `0700` enforced on unix via `os.MkdirAll(dir, 0700)`.
**When to use:** All cache directory creation.

- **macOS**: `$HOME/Library/Caches` (verified: `os.UserCacheDir()` returns `/Users/tazarov/Library/Caches`) [VERIFIED: go run /tmp/test_cache.go]
- **Linux**: `$XDG_CACHE_HOME` if set, else `$HOME/.cache`
- **Windows**: `%LocalAppData%` (e.g., `C:\Users\User\AppData\Local`)

```go
func bootstrapCacheDir() (string, error) {
    base, err := os.UserCacheDir()
    if err != nil {
        return "", fmt.Errorf("resolve user cache dir: %w", err)
    }
    return filepath.Join(base, "pure-simdjson"), nil
}
```

Directory permissions: `0700` on unix gives `drwx------` (owner-only). Verified: [VERIFIED: go run /tmp/test_atomic.go]. Windows ACL is set by the OS to the user's profile directory defaults — no explicit `os.Chmod` needed or meaningful on Windows.

### Pattern 10: `resolveLibraryPath()` rewrite

**What:** Phase 5 rewrites `library_loading.go::resolveLibraryPath()` to chain: env-override → cache-hit → `BootstrapSync` (with internal ctx) → cache-hit → fail.
**Critical:** The Phase-3 local `target/` branch is removed entirely. Maintainers working in-repo set `PURE_SIMDJSON_LIB_PATH` to their `target/release/libpure_simdjson.<ext>` (D-01).

```go
// New resolveLibraryPath() shape (Phase 5)
func resolveLibraryPath() (string, []string, error) {
    // Stage 1: env override (unchanged from Phase 3)
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

    // Stage 2: cache hit (no SHA-256 re-verify per D-04)
    cachePath := bootstrap.CachePath(runtime.GOOS, runtime.GOARCH)
    if _, err := os.Stat(cachePath); err == nil {
        return cachePath, []string{cachePath}, nil
    }

    // Stage 3: auto-bootstrap (uses internal timeout context, not caller ctx)
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
}
```

### Anti-Patterns to Avoid

- **Verifying SHA-256 on every cache hit:** Performance regression, contradicts D-04. ABI version check is the post-dlopen guard.
- **Using `os.TempDir()` instead of `cacheDir` for the temp file:** Must write temp to same filesystem as final path so `os.Rename` is atomic (same-volume rename). pitfall #16.
- **Bare filename to `windows.LoadLibrary`:** Phase 3 already passes full path; bootstrap must only ensure it constructs absolute paths. Pitfall #29.
- **`math/rand` (v1) with explicit `Seed()`:** `math/rand/v2` is auto-seeded since Go 1.22; use it directly.
- **Parallel R2 + GitHub downloads:** D-15 locks sequential fallback. Parallel wastes GitHub's 60/hr unauthenticated limit.
- **Archive extraction logic:** Pure-simdjson ships flat `.so`/`.dylib`/`.dll` files, not tgz/zip archives. Do not add extraction code.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Platform flock | Custom file advisory lock | `golang.org/x/sys/unix.Flock` / `windows.LockFileEx` | Already a dep; handles EINTR, EAGAIN edge cases |
| SHA-256 streaming | Custom hash loop | `crypto/sha256` + `io.Copy(io.MultiWriter(...))` | Stdlib; handles all edge cases; streaming |
| Atomic file write | `os.WriteFile` + hope | `os.CreateTemp` + `os.Rename` | Rename is atomic on same-FS; WriteFile is not |
| CLI argument parsing | `os.Args` manual parsing | `spf13/cobra` | D-22 locked; flag inheritance, --help generation |
| HTTP retry | Custom sleep loop without jitter | Full-Jitter pattern (Pattern 1) | Thundering-herd problem; pure-onnx's linear sleep is the known gap |
| Error classification | String matching on error messages | `errors.Is` / `errors.As` | stdlib `context.DeadlineExceeded`, `*net.DNSError` etc. |

**Key insight:** Pure-onnx's bootstrap.go is 1400+ lines. The complexity is earned — there are ~15 distinct edge cases in cross-platform file locking, HTTP retry, atomic placement, and archive extraction. For pure-simdjson, **no archive extraction** drops the code size by ~40%, but all other complexity remains.

---

## Deviations from pure-onnx Bootstrap Pattern

These are the two deliberate deviations documented in D-13 and D-07:

| Area | pure-onnx pattern | pure-simdjson pattern | Reason |
|------|-------------------|-----------------------|--------|
| Retry sleep | `time.Sleep(time.Duration(attempt)*time.Second)` — linear, not ctx-aware | Full-Jitter + ctx-aware `select` | D-13/D-14: better thundering-herd behavior; D-14: ctx propagates within ms |
| Cache layout | Single slot overwrite (archive name as subdir) | Per-version: `v<Version>/<os>-<arch>/` | D-07: rollback, parallel-version, mirrors R2 URL layout |
| Artifacts | tgz/zip archives with extraction | Flat `.so`/`.dylib`/`.dll` — no extraction | DIST-01 layout: flat files on R2 |
| Retry count | `defaultBootstrapDownloadRetryCount = 3` | 3 attempts (R2) then 3 attempts (GH fallback) | D-15/D-16: sequential fallback model |

---

## Common Pitfalls

### Pitfall 1: SHA-256 from checksums.go must match before dlopen (pitfall #17)

**What goes wrong:** A corrupted or MITM-substituted `.so`/`.dll`/`.dylib` is dlopen'd. The library's code runs in-process with full permissions.
**Why it happens:** Skipping verification "because TLS is secure enough" — corporate proxies terminate TLS.
**How to avoid:** Verify SHA-256 against `checksums.go` in the `downloadToTemp` function, before `os.Rename`. On mismatch, remove the temp file and return `ErrChecksumMismatch` as a `permanentBootstrapError` (no retry).
**Warning signs:** Integration test: serve corrupted bytes from `httptest.Server`, expect `ErrChecksumMismatch` returned, no `dlopen` called.

### Pitfall 2: Windows bare-filename LoadLibrary (pitfall #29)

**What goes wrong:** `windows.LoadLibrary("pure_simdjson-msvc.dll")` triggers DLL search-path order starting from CWD — allows DLL hijacking.
**Why it happens:** Easy to accidentally pass basename when constructing paths.
**How to avoid:** `bootstrap.CachePath()` returns an absolute path. `resolveLibraryPath()` always passes the absolute path to `loadLibrary()`. The existing Phase-3 `library_windows.go` calls `windows.LoadLibrary(path)` where `path` is already absolute — this invariant must be preserved by Phase 5.
**Warning signs:** In-process test: verify that `resolveLibraryPath()` never returns a relative path or a bare filename.

### Pitfall 3: os.CreateTemp must use cacheDir, not os.TempDir (pitfall #16 + atomic rename)

**What goes wrong:** Temp file in `/tmp` (different filesystem from cache in `$HOME/Library/Caches`), `os.Rename` fails with `EXDEV` (cross-device link).
**Why it happens:** `os.CreateTemp(os.TempDir(), ...)` is the natural first instinct.
**How to avoid:** `os.CreateTemp(cacheDir, "pure-simdjson-*.tmp")`. Create the cache dir with `MkdirAll(cacheDir, 0700)` first.
**Warning signs:** Test on a machine where `/tmp` and `$HOME` are on different filesystems (common in Linux container environments).

### Pitfall 4: 0700 perms on unix cache directory (pitfall #16)

**What goes wrong:** Cache directory created with 0755 (world-readable) allows any local user to replace the `.so` with a malicious one before `dlopen`.
**Why it happens:** `os.MkdirAll(dir, 0755)` is the common default.
**How to avoid:** `os.MkdirAll(cacheDir, 0700)` — owner-only access. Verified: `drwx------` perm set correctly. [VERIFIED: go run /tmp/test_atomic.go]
**Warning signs:** Test: after `MkdirAll`, `os.Stat(dir).Mode().Perm() == 0700`.

### Pitfall 5: Missing ctx-aware sleep (pitfall #20)

**What goes wrong:** `time.Sleep(d)` in the retry loop blocks the caller's context cancellation for up to 8 seconds on the final backoff interval.
**Why it happens:** Pure-onnx uses `time.Sleep` (not ctx-aware). This is the documented deviation.
**How to avoid:** Use Pattern 1's `select { case <-time.After(d): case <-ctx.Done(): return ctx.Err() }` for every sleep in the retry loop.
**Warning signs:** Test: cancel context during second retry sleep; measure time from cancel to function return; expect < 50ms, not up to 8s.

### Pitfall 6: Windows `os.Rename` across different volumes (EXDEV)

**What goes wrong:** `os.Rename(tmpPath, cachePath)` fails if temp and cache are on different volumes.
**How to avoid:** Write temp file to `cacheDir` (same volume as final path). `os.CreateTemp(cacheDir, "*.tmp")` handles this.
**Note:** On the same volume, `os.Rename` on Windows uses `MoveFileEx(..., MOVEFILE_REPLACE_EXISTING)` — atomic for same-volume. [ASSUMED: Go 1.22+ Windows os.Rename behavior; cross-referenced with Go stdlib source]

### Pitfall 7: R2 URL must be HTTPS-only for non-loopback hosts

**What goes wrong:** `PURE_SIMDJSON_BINARY_MIRROR=http://internal.corp/...` is accepted by a naive URL parser. If it downgrades to HTTP, the HTTPS-downgrade redirect checker won't catch it (the initial request is already HTTP).
**How to avoid:** In URL validation, reject HTTP scheme for non-loopback hosts. Loopback (`localhost`, `127.0.0.1`) is allowed for tests. Same as pure-onnx's `validateBootstrapBaseURL` + `isLoopbackBootstrapHost`.
**Warning signs:** Test with `PURE_SIMDJSON_BINARY_MIRROR=http://example.com/` — expect validation error on config resolve, not on first download.

---

## Code Examples

### Cache path construction
```go
// Source: D-07 cache layout
func CachePath(goos, goarch string) string {
    base, _ := os.UserCacheDir()
    osArch := goos + "-" + goarch
    return filepath.Join(base, "pure-simdjson",
        "v"+Version,
        osArch,
        platformLibraryName(goos))
}
```

### R2 URL construction
```go
// Source: D-01 (DIST-01 URL layout)
const defaultR2BaseURL = "https://releases.amikos.tech/pure-simdjson"

func r2ArtifactURL(baseURL, version, goos, goarch string) string {
    osArch := goos + "-" + goarch
    libName := platformLibraryName(goos)
    return fmt.Sprintf("%s/v%s/%s/%s",
        strings.TrimRight(baseURL, "/"), version, osArch, libName)
}

func githubReleasesArtifactURL(version, goos, goarch string) string {
    libName := platformLibraryName(goos)
    return fmt.Sprintf(
        "https://github.com/amikos-tech/pure-simdjson/releases/download/v%s/%s",
        version, libName)
}
```

### `BootstrapSync` functional option pattern
```go
// Source: pure-onnx bootstrap.go BootstrapOption pattern (adapted)
type BootstrapOption func(*bootstrapConfig) error

func WithMirror(url string) BootstrapOption {
    return func(cfg *bootstrapConfig) error {
        cfg.mirrorURL = strings.TrimSpace(url)
        return validateBaseURL(cfg.mirrorURL)
    }
}

func WithDest(path string) BootstrapOption {
    return func(cfg *bootstrapConfig) error {
        cfg.destDir = filepath.Clean(strings.TrimSpace(path))
        return nil
    }
}

func BootstrapSync(ctx context.Context, opts ...BootstrapOption) error {
    cfg, err := resolveConfig(opts...)
    if err != nil {
        return err
    }
    return ensureArtifact(ctx, cfg, runtime.GOOS, runtime.GOARCH)
}
```

### Cobra root command boilerplate
```go
// Source: cobra docs; D-22, D-28
func main() {
    rootCmd := &cobra.Command{
        Use:           "pure-simdjson-bootstrap",
        SilenceUsage:  true,
        SilenceErrors: true,
    }
    rootCmd.AddCommand(newFetchCmd(), newVerifyCmd(), newPlatformsCmd(), newVersionCmd())
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `math/rand.Seed(time.Now().UnixNano())` | `math/rand/v2` — auto-seeded, no `Seed()` call | Go 1.22 (2024) | Removes global-seed race in goroutines |
| `time.Sleep(attempt * time.Second)` | Full-Jitter with ctx-aware select | D-13/D-14 decision | Thundering-herd resistance; instant ctx propagation |
| Single-slot artifact cache (overwrite on upgrade) | Per-version subdirs (`v<Version>/`) | D-07 decision | Rollback, parallel coexistence |
| SHA-256 stored in release notes / separate manifest | `checksums.go` embedded in Go source | D-08 decision | Always present at compile time; no network needed for verify |

**Deprecated/outdated:**
- `math/rand.Seed()`: deprecated since Go 1.20, removed from `math/rand/v2`. Use `math/rand/v2` — auto-seeded.
- Phase-3 `libraryCandidates()` target/ search: dropped in Phase 5. Replaced by cache-first lookup.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `os.Rename` on Windows Go 1.22+ uses `MoveFileEx(..., MOVEFILE_REPLACE_EXISTING)` atomically on same volume | Patterns §4, §6 | Torn file on concurrent write — mitigation: flock prevents concurrent write anyway |
| A2 | GitHub Releases asset downloads (.so/.dll/.dylib files) do NOT count against the 60/hr unauthenticated API rate limit | Pattern §6, D-16 | GitHub fallback may be unexpectedly rate-limited — add 403 body-sniff regardless |
| A3 | `PURE_SIMDJSON_BINARY_MIRROR` mirror server returns flat files at same URL layout as R2 | D-19 | Mirror with different layout breaks download; document layout expectation in `docs/bootstrap.md` |

---

## Open Questions (RESOLVED)

1. **`PURE_SIMDJSON_CACHE_DIR` override (Claude's discretion)**
   - What we know: `os.UserCacheDir()` works on all 5 platforms; D-01/D-07 specify layout.
   - What's unclear: Some CI environments override `$XDG_CACHE_HOME` or `$HOME` to control paths. An explicit override would be cleaner for those users.
   - Recommendation: Add `PURE_SIMDJSON_CACHE_DIR` as a `BootstrapOption` and env var. Low cost, clearly in Claude's discretion scope.
   - RESOLVED: Implemented in 05-02 Task 1 (defaultCacheDir reads PURE_SIMDJSON_CACHE_DIR env var). See review L2.

2. **GitHub Releases URL format for flat files**
   - What we know: Pattern is `https://github.com/<owner>/<repo>/releases/download/v<ver>/<filename>`.
   - What's unclear: Whether the actual GH Release tags will be prefixed with `v` (must match Phase 6 CI tag naming).
   - Recommendation: Use `v` prefix in URL (standard Go module convention). Planner should note dependency on Phase 6 tag naming.
   - RESOLVED: Confirmed by D-06/D-07 in CONTEXT.md. v prefix used in url.go githubArtifactURL (05-01 Task 1).

3. **`checksums.go` placeholder content for v0.1 development**
   - What we know: CI-05 generates the real SHA-256 values at release time. Phase 5 implements the verification machinery.
   - What's unclear: What placeholder values to use in `checksums.go` during development (before Phase 6 produces real artifacts).
   - Recommendation: Ship `checksums.go` with empty map `var Checksums = map[string]string{}`. Document that CI-05 populates it. `BootstrapSync` should return a clear error if the entry is missing: `ErrNoChecksum`.
   - RESOLVED: Empty map with commented exemplar entries locked in 05-01 Task 2 checksums.go. Real SHA-256 values populated by Phase 6 CI-05 release-time pipeline.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | All | ✓ | go1.26.2 | — |
| `golang.org/x/sys` | flock (unix + windows) | ✓ | v0.31.0 (in go.mod) | — |
| `github.com/spf13/cobra` | CLI | ✗ (not yet in go.mod) | v1.10.2 (available) | — |
| `math/rand/v2` | Full-Jitter backoff | ✓ | stdlib Go 1.22+ | — |
| `os.UserCacheDir()` | Cache directory | ✓ | stdlib | — |

**Missing dependencies with no fallback:**
- `github.com/spf13/cobra v1.10.2`: must `go get github.com/spf13/cobra@v1.10.2` in Wave 0.

**Missing dependencies with fallback:**
- None that block execution.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib), `net/http/httptest` for HTTP mock |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./internal/bootstrap/... -count=1 -timeout 30s` |
| Full suite command | `go test ./... -count=1 -race -timeout 120s` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DIST-01 | R2 URL construction for all 5 platforms | unit | `go test ./internal/bootstrap/... -run TestURLConstruction` | ❌ Wave 0 |
| DIST-02 | GH Releases URL construction + fallback triggered after R2 exhaustion | unit | `go test ./internal/bootstrap/... -run TestFallback` | ❌ Wave 0 |
| DIST-03 | SHA-256 verify passes on correct hash, fails on corrupted bytes | unit | `go test ./internal/bootstrap/... -run TestChecksumVerify` | ❌ Wave 0 |
| DIST-03 | Corrupted download rejected before dlopen | integration | `go test ./internal/bootstrap/... -run TestCorruptedDownloadRejected` | ❌ Wave 0 |
| DIST-04 | `BootstrapSync(ctx)` downloads, verifies, caches | integration (httptest) | `go test ./internal/bootstrap/... -run TestBootstrapSync` | ❌ Wave 0 |
| DIST-04 | `BootstrapSync(ctx)` cancellation propagates within 50ms of ctx cancel | unit | `go test ./internal/bootstrap/... -run TestBootstrapSyncCancellation` | ❌ Wave 0 |
| DIST-05 | Cache directory created with 0700 perms on unix | unit | `go test ./internal/bootstrap/... -run TestCacheDirPerms` | ❌ Wave 0 |
| DIST-05 | Second `NewParser()` call (cache hit) makes no HTTP requests | integration (httptest) | `go test ./... -run TestNewParserCacheHit` | ❌ Wave 0 |
| DIST-06 | `PURE_SIMDJSON_LIB_PATH` set → no HTTP call made | unit | `go test ./... -run TestLibPathEnvBypassesDownload` | already exists (library_loading_test.go) |
| DIST-07 | `PURE_SIMDJSON_BINARY_MIRROR` overrides R2 base URL | integration (httptest) | `go test ./internal/bootstrap/... -run TestMirrorOverride` | ❌ Wave 0 |
| DIST-08 | `fetch` verb downloads all 5 platform artifacts to --dest | integration (httptest) | `go test ./cmd/pure-simdjson-bootstrap/... -run TestFetchCmd` | ❌ Wave 0 |
| DIST-09 | `resolveLibraryPath()` never returns relative/bare path | unit | `go test ./... -run TestResolveLibraryPathAbsolute` | ❌ Wave 0 |
| DIST-10 | cosign docs-only: no Go code imports sigstore | lint/grep | `grep -r "sigstore" . --include="*.go"` fails if found | — |
| DOC-05 | `docs/bootstrap.md` exists and covers env vars | manual/CI diff | file existence check | ❌ Wave 0 |

### Fault Injection Test Surfaces

| Fault | Test Pattern | Expected Behavior |
|-------|-------------|-------------------|
| Checksum corruption (body tampered) | `httptest.Server` returns file with flipped byte | `ErrChecksumMismatch`, no dlopen |
| HTTP 429 on first attempt, 200 on retry | `httptest.Server` returns 429 then 200 | retry succeeds, correct file written |
| HTTP 503 on all R2 attempts, 200 on GH | `httptest.Server` mux with R2 returning 503 N times | falls back to GH, succeeds |
| ctx cancel mid-download | Cancel ctx after first 1KB received | `context.Canceled`, temp file cleaned up |
| ctx cancel during retry sleep | Cancel ctx during sleep | returns within 50ms |
| Concurrent bootstrap (two goroutines race) | `sync.WaitGroup` with 10 goroutines calling `BootstrapSync` | exactly one download, all find cached artifact |
| HTTPS→HTTP redirect | `httptest.Server` returns 301 to `http://` | redirect rejected with permanent error |
| `.lock` file contention | Two processes: sleep in lock body, verify second waits | second process waits, acquires after first completes |
| 404 on R2, 200 on GH | R2 mock returns 404, GH mock returns 200 | GH fallback fires, artifact cached |
| PURE_SIMDJSON_DISABLE_GH_FALLBACK=1 + R2 404 | env set, R2 returns 404 | fails with `ErrAllSourcesFailed`, no GH attempt |

### Sampling Rate
- **Per task commit:** `go test ./internal/bootstrap/... -count=1 -timeout 30s`
- **Per wave merge:** `go test ./... -count=1 -race -timeout 120s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/bootstrap/bootstrap_test.go` — covers DIST-01..07
- [ ] `cmd/pure-simdjson-bootstrap/fetch_test.go` — covers DIST-08
- [ ] `internal/bootstrap/bootstrap_lock_unix.go` — unix flock implementation
- [ ] `internal/bootstrap/bootstrap_lock_windows.go` — windows LockFileEx implementation
- [ ] Package `github.com/spf13/cobra@v1.10.2`: `go get github.com/spf13/cobra@v1.10.2`

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | yes (cache dir) | `os.MkdirAll(dir, 0700)` on unix |
| V5 Input Validation | yes (URLs, mirror env var) | `url.Parse` + scheme check; reject HTTP for non-loopback |
| V6 Cryptography | yes (SHA-256 verify) | `crypto/sha256` stdlib; never hand-roll |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| MITM substitutes `.so` via corporate TLS proxy | Tampering | SHA-256 verify against embedded `checksums.go` before dlopen |
| DLL hijacking via CWD on Windows | Elevation of privilege | Always absolute path to `windows.LoadLibrary` (pitfall #29) |
| Cache directory world-writable → local priv esc | Tampering | `0700` perms on unix cache dir |
| HTTPS→HTTP redirect downgrade | Tampering | `CheckRedirect` rejects downgrade |
| HTTP mirror URL (no TLS) for non-loopback | Information disclosure | URL validation rejects HTTP for non-loopback hosts |
| GitHub mirror substitution if R2 compromised | Tampering | SHA-256 from `checksums.go` in Go source validates both sources |

---

## Sources

### Primary (HIGH confidence)
- `pure-onnx@v0.0.1/ort/bootstrap.go` — canonical lift target (read directly from Go module cache at `~/go/pkg/mod/github.com/amikos-tech/pure-onnx@v0.0.1/ort/`) [VERIFIED]
- `pure-onnx@v0.0.1/ort/bootstrap_lock_unix.go` — flock implementation [VERIFIED]
- `pure-onnx@v0.0.1/ort/bootstrap_lock_windows.go` — LockFileEx implementation [VERIFIED]
- `pure-simdjson/library_loading.go` — Phase 3 extension point [VERIFIED: read current source]
- `pure-simdjson/library_windows.go` — Windows LoadLibrary full-path pattern [VERIFIED: read current source]
- `pure-simdjson/go.mod` — `golang.org/x/sys v0.31.0` already present [VERIFIED]
- `go run /tmp/test_cache.go` — `os.UserCacheDir()` returns `"/Users/tazarov/Library/Caches"` on darwin [VERIFIED]
- `go run /tmp/test_atomic.go` — `os.MkdirAll(dir, 0700)` gives `drwx------`; `os.CreateTemp+Rename` works [VERIFIED]
- `go run /tmp/test_jitter2.go` — Full-Jitter formula with `math/rand/v2` [VERIFIED]
- `go list -m -versions github.com/spf13/cobra` — latest stable `v1.10.2` [VERIFIED]

### Secondary (MEDIUM confidence)
- AWS Architecture Blog "Exponential Backoff And Jitter" — Full-Jitter formula derivation [CITED: aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/]
- Go 1.22 release notes — `math/rand/v2` auto-seeding [CITED: go.dev/doc/go1.22]
- GitHub REST API docs — unauthenticated rate limit 60/hr for API calls; asset downloads not counted [CITED: docs.github.com/en/rest/rate-limit]

### Tertiary (LOW confidence)
- Go stdlib `os.Rename` Windows behavior (uses `MoveFileEx` with `MOVEFILE_REPLACE_EXISTING`) — [ASSUMED: from Go stdlib source, not directly tested on Windows host]

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages verified via go list or direct source read
- Architecture: HIGH — lifted from verified pure-onnx source; deviations are deliberate and documented
- Pitfalls: HIGH — all P0/P1 distribution pitfalls from PITFALLS.md re-verified for Phase 5 scope
- Validation architecture: HIGH — test patterns derived from pure-onnx test structure and phase requirements

**Research date:** 2026-04-20
**Valid until:** 2026-05-20 (30 days; stdlib and pure-onnx patterns are stable)
