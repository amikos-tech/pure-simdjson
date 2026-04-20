---
phase: 05-bootstrap-distribution
plan: 02
subsystem: infra
tags: [bootstrap, http, retry, sha256, flock, memoization, full-jitter, purego]

# Dependency graph
requires:
  - phase: 05-bootstrap-distribution
    plan: 01
    provides: Version, Checksums map, URL helpers (r2ArtifactURL, githubArtifactURL, ChecksumKey, validateBaseURL), canonical error sentinels, platform flock primitives (lockFile/unlockFile/isLockWouldBlock)
provides:
  - BootstrapSync(ctx, opts...) public entry point — the function Plan 04 loader and Plan 05 CLI call
  - Full HTTP download pipeline: Full-Jitter retry, HTTPS-downgrade rejection, streaming SHA-256, version-stamped User-Agent
  - Cache layout helpers: CachePath, artifactCachePath, withProcessFileLock, atomicInstall with 0700 perms
  - BootstrapOption surface: WithMirror, WithDest, WithVersion, WithTarget (public); withHTTPClient, withGitHubBaseURL (internal, exported-to-tests)
  - 30-second failure memoization (M2) so blocked-network NewParser() calls don't re-run the retry ladder on every attempt
  - export_test.go M3 seam set (12 entries) for the external bootstrap_test package
affects: [05-03-tls-ca-handling, 05-04-loader-integration, 05-05-cli-bootstrap, 05-06-tests-ci-matrix]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Full-Jitter exponential backoff with ctx-aware select for instant cancellation (replaces pure-onnx's linear time.Sleep(attempt*1s))"
    - "30-second failure memoization via package-level sync.Mutex-guarded struct — peek/record/clear API"
    - "os.CreateTemp(cacheDir, ...) + io.MultiWriter(file, sha256Hasher) + os.Rename for one-pass atomic download-and-hash"
    - "HTTPS-downgrade rejection via CheckRedirect that compares via[0].URL.Scheme against req.URL.Scheme"
    - "BootstrapConfigView accessor pattern keeps internal bootstrapConfig layout private while exposing fields tests need"
    - "Test seam M3 convention: lowercase production options re-exported under capitalized names in export_test.go"

key-files:
  created:
    - internal/bootstrap/cache.go
    - internal/bootstrap/bootstrap.go
    - internal/bootstrap/download.go
    - internal/bootstrap/export_test.go
    - internal/bootstrap/cache_test.go
    - internal/bootstrap/bootstrap_test.go
  modified: []

key-decisions:
  - "PURE_SIMDJSON_CACHE_DIR env var (L2) is read by defaultCacheDir with highest precedence so CI runners with ephemeral HOME and test suites with t.Setenv+t.TempDir can self-isolate."
  - "UID-scoped 0700 subdir under os.TempDir is the L6 fallback when os.UserCacheDir fails — never the bare TempDir, so the cache is never world-writable."
  - "Env-supplied mirror URL is validated at resolveConfig time (not deferred to first HTTP call) so misconfigured PURE_SIMDJSON_BINARY_MIRROR fails fast, before any download attempt."
  - "bootstrapFailureTTL is 30s and not configurable in v0.1; the retry ladder exhausts in ~10-15s, so 30s gives user time to fix env vars while still letting transient network issues self-heal."
  - "BootstrapSync checks ctx.Err() BEFORE consulting the memoization cache, so a cancelled ctx returns ctx.Err() even when a memoized failure exists."
  - "Config errors (bad mirror URL, etc.) are NOT memoized — they're caller bugs, not network state. Only ensureArtifact failures feed the cache."
  - "RegisterChecksumForTest mutates the package-global Checksums map and returns a cleanup closure that restores the prior entry; tests serial by default, no locking needed."
  - "BootstrapConfigView exposes resolved fields via accessor methods (VersionField, MirrorURL, DisableGH, GOOS, GOARCH, DestDir, GitHubBaseURL) instead of returning the raw struct — prevents tests from depending on field order or triggering vet struct-tag warnings."

requirements-completed: [DIST-02, DIST-03, DIST-04, DIST-05, DIST-07]

# Metrics
duration: 7min
completed: 2026-04-20
---

# Phase 5 Plan 2: BootstrapSync HTTP + Cache Pipeline Summary

**BootstrapSync(ctx, opts...) is now a working function: it downloads a flat shared-library artifact via Full-Jitter-retried R2-then-GitHub HTTP, streams SHA-256 through io.MultiWriter, verifies against the embedded Checksums map, and atomically installs to a 0700-permissioned cache — with version-stamped User-Agent, 30-second failure memoization, and a full M3 test-seam set for the external bootstrap_test package.**

## Performance

- **Duration:** ~7 min
- **Started:** 2026-04-20T11:29:05Z
- **Completed:** 2026-04-20T11:35:59Z
- **Tasks:** 2 (both TDD: RED + GREEN)
- **Commits:** 4
- **Files created:** 6
- **Files modified:** 0

## Accomplishments

- `internal/bootstrap/cache.go` — `defaultCacheDir` with PURE_SIMDJSON_CACHE_DIR override (L2) and UID-scoped 0700 TempDir fallback (L6); `CachePath`/`artifactCachePath` computing the per-version layout `v<Version>/<goos>-<goarch>/<libname>`; `withProcessFileLock` lifted verbatim from pure-onnx with 0700 directory perms; `atomicInstall` performing the same-filesystem `os.Rename`.
- `internal/bootstrap/bootstrap.go` — public `BootstrapSync(ctx, opts...)`, `BootstrapOption` type, `WithMirror`/`WithDest`/`WithVersion`/`WithTarget`, internal `withHTTPClient`/`withGitHubBaseURL`, `resolveConfig` wiring env vars PURE_SIMDJSON_BINARY_MIRROR (D-19) and PURE_SIMDJSON_DISABLE_GH_FALLBACK (D-20), `bootstrapFailureCache` with 30s TTL (M2), and the `permanentBootstrapError` tag used by the retry loop.
- `internal/bootstrap/download.go` — `newHTTPClient` with a nested-timeout `Transport` and `rejectHTTPSDowngrade` redirect policy (T-05-04); `userAgent = "pure-simdjson-go/v" + Version` (L3); `sleepWithJitter` using math/rand/v2 Full-Jitter (D-13) inside a ctx-aware select (D-14); `isRetryable` with GitHub 403 body-sniff for rate-limit errors; `downloadWithRetry` primary-then-fallback ladder (D-15); `downloadOnce` streaming through `io.MultiWriter(f, h)` into `os.CreateTemp(cacheDir, ...)`; `downloadAndVerify` that returns `ErrNoChecksum`/`ErrChecksumMismatch` as permanent errors.
- `internal/bootstrap/export_test.go` — 12 M3 test seams: `BootstrapConfigView`, `ResolveConfig`, `WithHTTPClient`, `WithGitHubBaseURL`, `DefaultCacheDir`, `WithProcessFileLockForTest`, `AtomicInstallForTest`, `SleepWithJitterForTest`, `IsRetryableForTest`, `MarkPermanentForTest`, `IsPermanentForTest`, `ResetBootstrapFailureCacheForTest`, `RegisterChecksumForTest`.
- `internal/bootstrap/cache_test.go` and `bootstrap_test.go` — external-package tests exercising every truth from the plan's `must_haves.truths`.

## Task Commits

Each task was committed atomically (TDD RED + GREEN for both):

1. **Task 1 RED — failing cache.go tests** — `7dc7ebc` (test)
2. **Task 1 GREEN — implement cache.go** — `b0e094d` (feat)
3. **Task 2 RED — failing bootstrap.go + download.go tests** — `5096448` (test)
4. **Task 2 GREEN — implement BootstrapSync + HTTP pipeline** — `aaec9db` (feat)

## Files Created/Modified

**Created:**

- `internal/bootstrap/cache.go` — cache layout, 0700 perms, flock, atomic rename, L2/L6 fallbacks.
- `internal/bootstrap/bootstrap.go` — `BootstrapSync` orchestrator, options, memoization, permanent-error tag.
- `internal/bootstrap/download.go` — HTTP client, Full-Jitter retry, User-Agent, SHA-256 streaming, HTTPS-downgrade rejection.
- `internal/bootstrap/export_test.go` — full M3 test seam set.
- `internal/bootstrap/cache_test.go` — Task 1 test file.
- `internal/bootstrap/bootstrap_test.go` — Task 2 test file.

**Modified:** none — Plan 01's files (`version.go`, `checksums.go`, `url.go`, `errors.go`, both `bootstrap_lock_*.go`) are consumed as-is.

## Decisions Made

- **PURE_SIMDJSON_CACHE_DIR takes priority over os.UserCacheDir (L2):** `defaultCacheDir` reads the env var first because CI runners frequently have an ephemeral or absent HOME. Setting the variable to a `t.TempDir()` gives every test a clean slate.
- **UID-scoped 0700 TempDir fallback (L6):** when `os.UserCacheDir` fails (rare; headless CI without HOME and without XDG_CACHE_HOME), `defaultCacheDir` falls back to `os.TempDir() + /pure-simdjson-<uid>` with 0700 perms so the cache is never world-writable. Bare `os.TempDir()` is forbidden.
- **Validate env-supplied mirror at resolve time:** `resolveConfig` calls `validateBaseURL` on `PURE_SIMDJSON_BINARY_MIRROR` when it is non-empty, before any HTTP activity. A misconfigured mirror fails fast instead of silently attacking the DNS.
- **30s memoization TTL (M2):** short enough that a user retrying after an env-var fix re-attempts within a minute, long enough that the retry ladder (~10-15s) has time to exhaust. Not configurable in v0.1 (D-13 simplicity rule).
- **ctx.Err() check precedes the memoization cache:** `BootstrapSync` returns `ctx.Err()` for a cancelled caller even when a memoized failure exists. This matches Go's "ctx wins" convention and prevents the memoization layer from surprising callers who explicitly cancelled.
- **Config errors are not memoized:** only `ensureArtifact` failures feed `globalBootstrapFailureCache`. A bad `WithMirror("http://example.com")` is a caller bug, not a network condition.
- **BootstrapConfigView accessor pattern:** `ResolveConfig` returns a view with getter methods rather than the bare struct. Keeps `bootstrapConfig`'s field layout private while giving tests everything they need.
- **`RegisterChecksumForTest` mutates Checksums directly:** the map is a package-global and bootstrap tests are sequential by default (no `t.Parallel`), so mutate-and-restore via a deferred cleanup closure is simpler than a separate test-only override map.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Critical Functionality] Added `f.Sync()` before temp-file rename**

- **Found during:** Task 2 implementation review.
- **Issue:** The plan's `downloadOnce` sketch omits `f.Sync()`. Without it, a crash between `io.Copy` returning and `atomicInstall` can leave a zero-length artifact on disk that the next `BootstrapSync` mistakes for a cache hit (D-04 "no SHA-256 re-verify" trusts the file system).
- **Fix:** Added `if err := f.Sync(); err != nil { ... }` immediately after `io.Copy`, before the defer closes the file. Ensures durability across power loss.
- **Files modified:** `internal/bootstrap/download.go`.
- **Commit:** `aaec9db`.

**2. [Rule 2 - Critical Functionality] Added `ctx.Err()` guard before every `downloadOnce`**

- **Found during:** Task 2 implementation.
- **Issue:** The plan's retry-loop sketch only checks ctx cancellation inside `sleepWithJitter`. A caller who cancels during the actual HTTP body-read between attempts wouldn't see ctx.Err() bubble up until the next sleep.
- **Fix:** Added `if err := ctx.Err(); err != nil { return "", "", err }` at the top of each retry iteration after the sleep, before the `downloadOnce` call.
- **Files modified:** `internal/bootstrap/download.go` (downloadWithRetry).
- **Commit:** `aaec9db`.

**3. [Rule 2 - Critical Functionality] Added `os.MkdirAll(cacheDir, 0700)` in `downloadOnce`**

- **Found during:** Task 2 implementation.
- **Issue:** `downloadOnce` calls `os.CreateTemp(cacheDir, ...)` but nothing guarantees cacheDir exists — in isolated test runs (no previous `ensureArtifact` path) the directory won't exist yet. `CreateTemp` returns a confusing "no such file or directory" instead of creating it.
- **Fix:** Added `os.MkdirAll(cacheDir, 0700)` guard inside `downloadOnce`, immediately before `os.CreateTemp`. Consistent with the 0700 perm invariant in `ensureArtifact`.
- **Files modified:** `internal/bootstrap/download.go`.
- **Commit:** `aaec9db`.

**4. [Rule 2 - Critical Functionality] Made permanent-error wrapping idempotent**

- **Found during:** Task 2 implementation.
- **Issue:** `markPermanentBootstrapError(markPermanentBootstrapError(err))` would produce a nested `*permanentBootstrapError{err: *permanentBootstrapError{...}}`. Functionally equivalent but ugly in stack traces.
- **Fix:** Added an `errors.As` check at the top of `markPermanentBootstrapError` that returns the already-marked error unchanged. Standard idempotent-wrapping pattern.
- **Files modified:** `internal/bootstrap/bootstrap.go`.
- **Commit:** `aaec9db`.

### Planned additions that required test-only adjustments

**5. [Rule 3 - Blocking Issue] `BootstrapConfigView` accessor wrapper instead of raw struct**

- **Found during:** Writing `TestResolveConfigEnvMirror` and `TestWithVersionAndWithTarget`.
- **Issue:** The plan's sketch has `ResolveConfig` return `bootstrapConfig` directly. Exposing the internal struct would leak field names into the test package as a public surface (callers could depend on the exact layout). Go's `go vet` also complains about copying mutex-carrying structs when `bootstrapConfig` later grows.
- **Fix:** Introduced `BootstrapConfigView{cfg bootstrapConfig}` in `export_test.go` with 8 accessor methods (`CacheDir`, `VersionField`, `MirrorURL`, `DisableGH`, `GOOS`, `GOARCH`, `DestDir`, `GitHubBaseURL`). Keeps the internal layout private and makes the seam explicitly read-only.
- **Files modified:** `internal/bootstrap/export_test.go`, `internal/bootstrap/bootstrap_test.go`.
- **Commit:** `aaec9db`.

## Authentication Gates

None — no external services touched. `newFileServer` pattern uses `httptest.NewServer`/`NewTLSServer` entirely in-process.

## Issues Encountered

None. All tests pass on first GREEN attempt after writing the production code. The two memoization tests (`TestBootstrapFailureMemoized`, `TestBootstrapSuccessClearsFailureCache`) take ~3.5s each because they deliberately exercise the full retry ladder against a 503-returning server; the fast-path `TestUserAgentStamp` hits the success path in ~10ms.

## Known Stubs

None. `grep -ri 'TODO|FIXME|placeholder|coming soon|not available' internal/bootstrap/` returns no results.

## User Setup Required

None — no external service configuration needed for this plan. The following env vars are OPTIONAL and documented in Phase 6 docs:

- `PURE_SIMDJSON_BINARY_MIRROR` — override the R2 base URL (e.g., corporate mirror).
- `PURE_SIMDJSON_DISABLE_GH_FALLBACK=1` — disable the GitHub fallback source.
- `PURE_SIMDJSON_CACHE_DIR` — override the OS user cache directory.
- `PURE_SIMDJSON_LIB_PATH` — bypass bootstrap entirely with an explicit library path.

## Next Phase Readiness

- **Plan 03 (TLS/CA handling):** can now modify `newHTTPClient` in `download.go` to layer in corporate CA bundles / mTLS. The `withHTTPClient` seam already lets tests swap the client.
- **Plan 04 (loader integration):** `library_loading.go::resolveLibraryPath` can call `bootstrap.CachePath` for the cache-hit check and `bootstrap.BootstrapSync(ctx)` for the auto-bootstrap stage. The `ErrAllSourcesFailed` and `ErrChecksumMismatch` sentinels are wired for `errors.Is` matching.
- **Plan 05 (CLI):** can import `bootstrap.BootstrapSync`, `bootstrap.SupportedPlatforms`, `bootstrap.CachePath`, `bootstrap.ChecksumKey`, and `bootstrap.Checksums` to implement `fetch`, `verify`, and `platforms` subcommands. `WithMirror`/`WithDest`/`WithVersion`/`WithTarget` map 1:1 to the planned `--mirror`/`--dest`/`--version`/`--target` flags.
- **Plan 06 (tests + CI):** the M3 test seams enable targeted fault-injection tests (SHA mismatch, HTTPS downgrade, R2-503-then-GH-200, etc.) without forking the package.

## Self-Check: PASSED

All created files exist and all commits are present on the branch:

- FOUND: `internal/bootstrap/cache.go`
- FOUND: `internal/bootstrap/bootstrap.go`
- FOUND: `internal/bootstrap/download.go`
- FOUND: `internal/bootstrap/export_test.go`
- FOUND: `internal/bootstrap/cache_test.go`
- FOUND: `internal/bootstrap/bootstrap_test.go`
- FOUND: commit `7dc7ebc` (Task 1 RED)
- FOUND: commit `b0e094d` (Task 1 GREEN)
- FOUND: commit `5096448` (Task 2 RED)
- FOUND: commit `aaec9db` (Task 2 GREEN)

Plan-level verification:

- `go build ./internal/bootstrap/...` exit 0
- `go vet ./internal/bootstrap/...` exit 0
- `go test ./internal/bootstrap/... -count=1 -timeout 60s` — PASS (7.6s, 26 tests incl. subtests)
- Every grep acceptance criterion from the plan matches:
  - `PURE_SIMDJSON_CACHE_DIR` in cache.go (L2): line 18
  - `0700` in cache.go: lines 40, 65, 109
  - `os.UserCacheDir` in cache.go: line 33
  - `pure-simdjson-%d` UID-scoped fallback: line 38
  - `math/rand/v2` in download.go: line 10
  - `NewRequestWithContext` in download.go: line 242
  - `User-Agent` + `pure-simdjson-go/v` in download.go: lines 23, 246
  - `rejectHTTPSDowngrade` in download.go: lines 41, 65, 70
  - `CreateTemp` in download.go: line 275
  - `MultiWriter` in download.go: line 290
  - `PURE_SIMDJSON_BINARY_MIRROR` in bootstrap.go: line 26
  - `PURE_SIMDJSON_DISABLE_GH_FALLBACK` in bootstrap.go: line 27
  - `bootstrapFailureCache` / `bootstrapFailureTTL` in bootstrap.go: lines 33, 152
  - `ResetBootstrapFailureCacheForTest` in export_test.go: line 85
  - No `time.Sleep.*attempt` linear-sleep pattern in download.go (Full-Jitter replaces it)

## TDD Gate Compliance

Both Task 1 and Task 2 followed the full RED → GREEN cycle with separate commits. No REFACTOR phase needed — the pure-onnx lift combined with the L2/L3/L6/M2/M3 review-driven additions produced code that was already idiomatic on first GREEN.

---
*Phase: 05-bootstrap-distribution*
*Completed: 2026-04-20*
