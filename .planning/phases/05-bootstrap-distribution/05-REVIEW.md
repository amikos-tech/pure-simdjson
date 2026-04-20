---
status: issues_found
phase: 05-bootstrap-distribution
depth: standard
reviewed: 2026-04-20
reviewer: gsd-code-reviewer
files_reviewed: 27
findings:
  critical: 0
  warning: 4
  info: 12
  total: 16
---

# Phase 5: Code Review Report

**Reviewed:** 2026-04-20
**Depth:** standard
**Files Reviewed:** 27
**Status:** issues_found

## Files Reviewed

- .gitignore
- cmd/pure-simdjson-bootstrap/fetch.go
- cmd/pure-simdjson-bootstrap/fetch_test.go
- cmd/pure-simdjson-bootstrap/main.go
- cmd/pure-simdjson-bootstrap/platforms.go
- cmd/pure-simdjson-bootstrap/verify.go
- cmd/pure-simdjson-bootstrap/verify_test.go
- cmd/pure-simdjson-bootstrap/version.go
- docs/bootstrap.md
- errors.go
- go.mod
- go.sum
- internal/bootstrap/bootstrap.go
- internal/bootstrap/bootstrap_lock_unix.go
- internal/bootstrap/bootstrap_lock_windows.go
- internal/bootstrap/bootstrap_test.go
- internal/bootstrap/cache.go
- internal/bootstrap/cache_test.go
- internal/bootstrap/checksums.go
- internal/bootstrap/download.go
- internal/bootstrap/errors.go
- internal/bootstrap/export_test.go
- internal/bootstrap/url.go
- internal/bootstrap/version.go
- library_loading.go
- library_loading_test.go
- testmain_test.go

## Summary

`go vet ./...` clean, cross-compile for GOOS=windows passes. No Critical findings. Bootstrap pipeline is carefully written — defensive permanent-error tagging, M1 double-checked locking, atomic renames, context-aware retry sleeps, and redirect-downgrade rejection all implemented correctly. Findings concentrate on (a) two real user-visible behaviors (silent lock wait, config-insensitive failure cache), (b) dead code and naming nits, and (c) future-regression surfaces (duplicated `platformLibraryName`, unsynchronized test globals).

Most impactful: **WR-01** (`nextLogAt` bookkeeping exists but no log statement fires) and **WR-02** (memoized failure leaks across differing configs within 30s). Both are post-v0.1 polish, not ship-blockers.

## Warnings

### WR-01: `withProcessFileLock` tracks `nextLogAt` but never emits a log

**File:** `internal/bootstrap/cache.go:90-92`

The `for` loop computes `nextLogAt` and resets it on every tick, but no `fmt.Fprintf(os.Stderr, ...)` call is emitted inside the `if` branch. Result: a second process waiting on the flock for 2 minutes produces zero output — exactly the silent-hang symptom the fetch CLI's L4 per-platform progress was added to prevent.

**Fix:**
```go
if time.Now().After(nextLogAt) {
    fmt.Fprintf(os.Stderr, "pure-simdjson: waiting for install lock at %s (held %s)...\n",
        lockPath, time.Since(start).Truncate(time.Second))
    nextLogAt = time.Now().Add(lockLogInterval)
}
```
Alternatively delete the `nextLogAt` scaffolding if logging is intentionally suppressed.

### WR-02: `BootstrapSync` failure memoization ignores config — stale error leaks across option sets

**File:** `internal/bootstrap/bootstrap.go:199-212`

`globalBootstrapFailureCache.peek()` runs before `resolveConfig(opts...)`. The cache key is implicit (process-wide), so a failure recorded for `WithMirror("https://broken")` short-circuits a subsequent `BootstrapSync(ctx, WithMirror("https://works"))` for the next 30 seconds. Concrete symptom: user corrects `PURE_SIMDJSON_BINARY_MIRROR`, the second `NewParser()` in the same process still returns the cached error.

**Fix:** Key the cache on `{cfg.mirrorURL, cfg.disableGH, cfg.goos, cfg.goarch, cfg.version}`. Move the `peek()` below `resolveConfig` so the key is available.

### WR-03: Lock-acquire loop is not context-cancellable

**File:** `internal/bootstrap/cache.go:61-103`

`withProcessFileLock` takes no `context.Context` and sleeps up to 2 minutes. Every other wait in the pipeline observes ctx cancellation; this one does not. A caller whose ctx has been cancelled can block the full 2 minutes before returning `ctx.Err()`.

**Fix:** Thread ctx through and use `select { case <-time.After(lockRetryInterval): case <-ctx.Done(): ... }` for the sleep.

### WR-04: `fetch_test.go` increments `hits` from multiple goroutines without synchronization

**File:** `cmd/pure-simdjson-bootstrap/fetch_test.go:33-40, 85-89`

Each `httptest.Server` request runs on its own goroutine. `runFetch` drives them sequentially today, but the test code is a sharp edge — if an `--parallel` flag lands in Phase 6, `hits++` becomes a data race `go test -race` would catch.

**Fix:** Use `atomic.Int32` to match the pattern already used in `internal/bootstrap/bootstrap_test.go`.

## Info

### IN-01: Unused constant `bootstrapRetryBaseMS`

**File:** `internal/bootstrap/download.go:37` — `bootstrapRetryBaseMS = 500` is defined but unreferenced; `sleepWithJitter` hardcodes its own `500 * time.Millisecond`. Either delete or wire them together.

### IN-02: Dead defensive branch in `sleepWithJitter`

**File:** `internal/bootstrap/download.go:113-118` — `shift` is clamped ≤10, so `500ms << 10` ≈ 5.12×10¹¹ ns, well under int64 max. The `expBackoff < 0` overflow guard cannot fire.

### IN-03: Local `cap` shadows built-in

**File:** `internal/bootstrap/download.go:110` — Rename `const cap` to `backoffCap` or `maxBackoff`.

### IN-04: Redundant `errors.As(*url.Error)` branch

**File:** `internal/bootstrap/download.go:86-102` — `*url.Error` has `Unwrap()`, so `errors.Is(err, errHTTPSDowngrade)` already walks through the wrapper. Collapse to one line.

### IN-05: Stale comment references `redirectDowngradeError`

**File:** `internal/bootstrap/download.go:42-44` — Actual variable is `errHTTPSDowngrade`. Rename the comment.

### IN-06: `platformLibraryName` duplicated three ways

**File:** `cmd/pure-simdjson-bootstrap/verify.go:79-90` + test helpers — Three forks (`platformLibraryNameForCLI`, `testmain_test.go::testMainLibraryName`, `library_loading_test.go::builtLibraryName`). Export `bootstrap.PlatformLibraryName(goos) string` and delete the CLI fork; keep the cargo-name helpers separate because cargo's on-disk name differs from the cache name.

### IN-07: Error-body snippet may surface untrusted response bytes

**File:** `internal/bootstrap/download.go:288-293` — First 512 bytes of a non-200 body are embedded in the error. For a misconfigured `PURE_SIMDJSON_BINARY_MIRROR`, could include CRLF. Consider `strings.ReplaceAll(snippet, "\n", " ")`. Low risk.

### IN-08: `os.Getuid() == -1` on Windows could produce `pure-simdjson--1` cache dir

**File:** `internal/bootstrap/cache.go:37-41` — Cosmetic. Fallback to `os.Getpid()` when `Getuid() < 0`, or leave as-is (comment already notes the corner).

### IN-09: `platforms` subcommand ignores `--dest`

**File:** `cmd/pure-simdjson-bootstrap/platforms.go:14-35` — `verify --dest` exists; `platforms --dest` does not. Minor UX gap for operators auditing offline bundles.

### IN-10: Tests mutate package-global `bootstrap.Checksums` without a mutex

**File:** `cmd/pure-simdjson-bootstrap/fetch_test.go:52-60`, `verify_test.go:45-46` — Contract is "tests run sequentially". Document it in the CLI test files the same way `internal/bootstrap/bootstrap_test.go::RegisterChecksumForTest` does.

### IN-11: `firstErr` wins over more-interesting subsequent errors in `verify --all-platforms`

**File:** `cmd/pure-simdjson-bootstrap/verify.go:107-125` — If platform[0] hits `ErrNoChecksum` and platform[2] hits `ErrChecksumMismatch`, the returned sentinel is `ErrNoChecksum` — hiding the security-relevant failure. Prefer `ErrChecksumMismatch` when both are present, or document the arbitrary-first behavior.

### IN-12: `Checksums` map is exported mutable package-global

**File:** `internal/bootstrap/checksums.go:7` — `var Checksums = map[string]string{}`. Package is `internal/bootstrap` so external mutation is gated by the import path, but any future consumer could mutate it. Acceptable for v0.1; revisit when CI-05 populates at release time.

---

_Reviewer: gsd-code-reviewer (Claude)_
_Depth: standard_
