---
phase: 05-bootstrap-distribution
fixed_at: 2026-04-20
review_path: .planning/phases/05-bootstrap-distribution/05-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 5: Code Review Fix Report

**Fixed at:** 2026-04-20
**Source review:** `.planning/phases/05-bootstrap-distribution/05-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 4 (0 Critical + 4 Warning; Info findings out of scope)
- Fixed: 4
- Skipped: 0

## Fixed Issues

### WR-01: `withProcessFileLock` tracks `nextLogAt` but never emits a log

**Files modified:** `internal/bootstrap/cache.go`
**Commit:** 6ee2938
**Applied fix:** Added the missing `fmt.Fprintf(os.Stderr, ...)` call inside the `if time.Now().After(nextLogAt)` branch, matching the exact message format suggested in the review. The existing `nextLogAt` bookkeeping now drives a 5-second progress log so a process waiting on the flock surfaces output instead of hanging silently.

### WR-02: `BootstrapSync` failure memoization ignores config

**Files modified:** `internal/bootstrap/bootstrap.go`
**Commit:** ea6e6f3
**Applied fix:** Reworked `bootstrapFailureCache` from a single-slot `{lastErr, expiresAt}` pair to a `map[bootstrapFailureKey]bootstrapFailureEntry` keyed on `{mirrorURL, disableGH, goos, goarch, version}` (the subset of config that steers network behaviour — `cacheDir`, `httpClient`, `githubBaseURL`, `destDir` are infrastructure knobs not worth keying on). `peek`, `record`, and the new `forget` methods all take a key; `clear()` (exposed via `ResetBootstrapFailureCacheForTest`) now nukes the whole map. `BootstrapSync` moves `peek()` below `resolveConfig(opts...)` so the config-derived key is available, then uses `forget(key)` on success instead of `clear()` so a success for one target does not discard pending memoizations for others. Existing tests (`TestBootstrapFailureMemoized`, `TestBootstrapSuccessClearsFailureCache`) still pass.

### WR-03: Lock-acquire loop is not context-cancellable

**Files modified:** `internal/bootstrap/cache.go`, `internal/bootstrap/bootstrap.go`, `internal/bootstrap/export_test.go`, `internal/bootstrap/cache_test.go`
**Commit:** bfa7612
**Applied fix:** Added `ctx context.Context` as the first parameter of `withProcessFileLock` and replaced the unconditional `time.Sleep(lockRetryInterval)` with a `select { case <-time.After(lockRetryInterval): case <-ctx.Done(): ... }`. On cancellation the lock file is closed and `ctx.Err()` is returned. Updated the one production caller (`ensureArtifact` in `bootstrap.go`), the test seam (`WithProcessFileLockForTest`), and the single test call site (`TestWithProcessFileLockBasic`) to pass `context.Background()`. A nil-ctx guard matches the pattern already in `BootstrapSync`.

### WR-04: `fetch_test.go` increments `hits` from multiple goroutines without synchronization

**Files modified:** `cmd/pure-simdjson-bootstrap/fetch_test.go`
**Commit:** 814d826
**Applied fix:** Replaced both `var hits int` declarations with `var hits atomic.Int32`, switched `hits++` to `hits.Add(1)`, and updated the two assertion sites to use `hits.Load()`. Added the `sync/atomic` import. Verified the fix with `go test -race`, which now runs cleanly on both `TestFetchCmd` and `TestFetchCmdSingleTarget`. Matches the pattern already used in `internal/bootstrap/bootstrap_test.go` (multiple `var hits atomic.Int32` sites).

## Skipped Issues

None — all four in-scope findings were fixed.

## Verification

- `go vet ./...` — clean after each commit.
- `go build ./...` — clean after each commit.
- `go test ./internal/bootstrap/... ./cmd/pure-simdjson-bootstrap/... -count=1` — passes (15.7s + 0.5s).
- `go test -race` on the affected fetch tests — passes.
- Existing memoization tests (`TestBootstrapFailureMemoized`, `TestBootstrapSuccessClearsFailureCache`) still pass under the new keyed-cache implementation.

---

_Fixed: 2026-04-20_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
