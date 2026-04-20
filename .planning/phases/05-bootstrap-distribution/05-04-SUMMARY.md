---
phase: 05-bootstrap-distribution
plan: 04
subsystem: infra
tags: [loader, bootstrap, double-checked-locking, resolveLibraryPath, purego, windows-full-path, m1, dist-09]

# Dependency graph
requires:
  - phase: 05-bootstrap-distribution
    plan: 01
    provides: root purejson sentinels aliased from bootstrap (H2 pointer identity), platformLibraryName moved into internal/bootstrap
  - phase: 05-bootstrap-distribution
    plan: 02
    provides: BootstrapSync, CachePath, BootstrapOption surface — the APIs library_loading.go now consumes
provides:
  - resolveLibraryPath 4-stage chain (env override -> cache hit -> bootstrap -> cache hit after bootstrap)
  - activeLibrary double-checked locking (M1) — resolveLibraryPath/loadLibrary/ffi.Bind run outside libraryMu
  - DIST-09 Windows full-path invariant enforced at the boundary (every return from resolveLibraryPath is absolute)
  - D-21 error hint: bootstrap failures reference PURE_SIMDJSON_LIB_PATH so users learn the bypass
  - D-04 cache-hit fast path: no SHA-256 re-verify on disk hit (verification lives in bootstrap's install step)
  - TestMain test-fixture seam so the existing Phase 3/4 test corpus keeps working without the deleted target/release walk
affects: [05-05-cli-bootstrap, 05-06-tests-ci-matrix, 06-ci-release-pipeline]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Double-checked locking for process-wide lazy init: fast-path read under the lock, slow-path resolve OUTSIDE the lock, recheck-and-install under the lock again (M1)"
    - "4-stage resolution chain with errors.Is-friendly %w wrapping relying on H2 pointer-identity sentinel aliasing (no translation adapter)"
    - "TestMain env-var bootstrap as a compatibility seam when deleting an implicit discovery path that existing tests relied on"

key-files:
  created:
    - testmain_test.go
  modified:
    - library_loading.go
    - library_loading_test.go

key-decisions:
  - "activeLibrary switches to double-checked locking: libraryMu is held only for (a) the fast-path cached-pointer read and (b) the recheck-insert block; resolveLibraryPath, loadLibrary, and ffi.Bind all run OUTSIDE the mutex. First-run bootstrap (multi-minute download) no longer serializes concurrent NewParser() callers on one caller's bandwidth (M1)."
  - "resolveLibraryPath chains env override -> cache hit -> BootstrapSync -> cache hit after bootstrap. Stage 1 preserves the Phase 3 PURE_SIMDJSON_LIB_PATH behaviour verbatim so existing air-gapped deployments continue to work without change (D-01)."
  - "Error wrapping uses fmt.Errorf(\"...: %w\", err) with no translation adapter. Plan 01 H2 aliased root purejson sentinels to bootstrap sentinels via pointer identity, so errors.Is(err, purejson.ErrChecksumMismatch) and errors.Is(err, bootstrap.ErrChecksumMismatch) both match the same underlying pointer across the chain."
  - "Internal 5-minute timeout (bootstrapResolveTimeout) bounds Stage 3 so NewParser() can never stall indefinitely; the timeout is a constant, not a caller option, because NewParser's signature is locked (D-02/D-03)."
  - "TestMain seeds PURE_SIMDJSON_LIB_PATH to target/release/<libname> when the cargo artefact is present, preserving the Phase 3/4 developer workflow that expected implicit discovery of the local build."

patterns-established:
  - "Double-checked locking fingerprint in lazy-init loader code (two libraryMu.Lock sites bracketing an unguarded slow path)"
  - "TestMain as a compatibility seam when removing implicit default paths — production behaviour changes, test ergonomics do not"

requirements-completed: [DIST-05, DIST-06, DIST-09]

# Metrics
duration: 8min
completed: 2026-04-20
---

# Phase 5 Plan 4: Loader Integration Summary

**library_loading.go now drives the Phase 5 bootstrap pipeline: first NewParser() on a fresh machine triggers the download/verify/install cycle automatically, while PURE_SIMDJSON_LIB_PATH remains the zero-network escape hatch. activeLibrary switched to double-checked locking so concurrent callers no longer serialize on the first caller's bandwidth across the multi-minute download window (M1). Legacy target/release walk and rust-target-triple helpers deleted; Phase 3 test corpus preserved via a TestMain compatibility seam.**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-04-20T11:49:57Z
- **Completed:** 2026-04-20T11:57:48Z
- **Tasks:** 1 (TDD: RED + GREEN)
- **Commits:** 2
- **Files created:** 1
- **Files modified:** 2

## Accomplishments

- **`resolveLibraryPath()` 4-stage chain.** Stage 1 preserves the Phase 3 env-override block verbatim (TrimSpace + filepath.Abs + os.Stat). Stage 2 checks `bootstrap.CachePath(runtime.GOOS, runtime.GOARCH)` without SHA-256 re-verify (D-04 — install-time verification in Plan 02 already succeeded). Stage 3 calls `bootstrap.BootstrapSync(ctx)` under an internal `context.WithTimeout(5*time.Minute)` so `NewParser()` cannot stall indefinitely. Stage 4 re-stats the cache path; any miss after a successful BootstrapSync is a broken invariant and surfaces with the same "set PURE_SIMDJSON_LIB_PATH to bypass" hint (D-21).
- **`activeLibrary()` double-checked locking (M1).** Two `libraryMu.Lock` sites bracket the slow path: the first reads `cachedLibrary` under the lock and releases immediately; the second re-checks and installs. `resolveLibraryPath()`, `loadLibrary()`, and `ffi.Bind()` run between them with NO mutex held. The benign race — two concurrent callers both hit the slow path, one installs, the other orphans its dlopen handle — is accepted because purego has no dlclose surface for v0.1 and the race fires at most once per process.
- **Legacy functions deleted.** `libraryCandidates()` (the target/release + target/debug + target/<triple>/release + target/<triple>/debug walk), `rustTargetTriple()`, and the in-package `platformLibraryName()` helper are all gone from `library_loading.go`. The plan's D-01 acceptance criterion (no `target/release` or `target/debug` references) is grep-verified.
- **Error wrapping with H2 pointer-identity aliasing.** No `bootstrapErrToPublicErr` translation adapter. Plain `fmt.Errorf("bootstrap failed (set %s to bypass): %w", libraryEnvPath, err)` preserves the full `errors.Is` chain: `errors.Is(err, purejson.ErrChecksumMismatch)` matches because Plan 01 aliased the root sentinel to `bootstrap.ErrChecksumMismatch` via pointer identity.
- **DIST-09 Windows full-path invariant.** Every return path from `resolveLibraryPath` produces an absolute path: `filepath.Abs(envPath)` for Stage 1, `bootstrap.CachePath()` (absolute by construction) for Stages 2 + 4. Test `TestResolveLibraryPathAbsolute` asserts `filepath.IsAbs(path)` on a successful return.
- **M1 grep-verifiable invariant.** `TestActiveLibraryLockScope` parses library_loading.go, extracts the `activeLibrary` body, walks line-by-line tracking `libraryMu.Lock`/`libraryMu.Unlock` depth, and fails the test if `resolveLibraryPath()` or `loadLibrary(` appears under the lock. The same file also asserts `>= 2` `libraryMu.Lock` sites — the fingerprint of double-checked locking.
- **TestMain compatibility seam.** New `testmain_test.go` sets `PURE_SIMDJSON_LIB_PATH=<absolute target/release/libname>` when the cargo artefact exists, preserving the Phase 3/4 dev workflow that expected implicit discovery. Tests that exercise resolution logic override with `t.Setenv(libraryEnvPath, "")`.

## Task Commits

1. **Task 1 RED — failing tests for 4-stage resolveLibraryPath + M1 lock scope** — `d2c3896` (test)
2. **Task 1 GREEN — rewrite library_loading.go + TestMain seam** — `7c6e45d` (feat)

## Files Created/Modified

**Created:**

- `testmain_test.go` — `TestMain` that seeds `PURE_SIMDJSON_LIB_PATH` to `target/release/<libname>` if the cargo artefact is present. Pure developer-ergonomics — does not affect production behaviour.

**Modified:**

- `library_loading.go` — rewritten. Imports gained `context`, `time`, `github.com/amikos-tech/pure-simdjson/internal/bootstrap`; lost `errors` (no longer directly used). `activeLibrary` rewritten with double-checked locking; `resolveLibraryPath` rewritten with 4-stage chain; `libraryCandidates`, `rustTargetTriple`, `platformLibraryName` deleted.
- `library_loading_test.go` — rewritten around the new contract. Removed tests that exercised the deleted target/release walk (`TestResolveLibraryPathUsesDebugFallback`, `TestActiveLibrarySearchMissReportsAttemptedPaths`, `TestResolveLibraryPathPreservesCandidatePathError`). Added `TestResolveLibraryPathAbsolute`, `TestLibPathEnvBypassesDownload`, `TestResolveLibraryPathCacheHit`, `TestResolveLibraryPathBootstrapError`, `TestActiveLibraryLockScope`, `TestNewParserSignatureUnchanged`. Kept `TestActiveLibraryEnvOverrideMissingWrapsLoadFailure` and `TestActiveLibraryEnvOverrideLoadsBuiltLibrary`.

## Decisions Made

- **Double-checked locking over lock-for-the-entire-slow-path (M1):** holding `libraryMu` across `BootstrapSync(ctx)` would serialize every concurrent `NewParser()` call behind one caller's network bandwidth. A benign-race design where two callers both dlopen is tolerable because purego has no dlclose in v0.1 and the race occurs at most once per process lifetime. This matches the pattern reviewers called out in 05-REVIEWS.md (M1 / codex-MEDIUM finding).
- **5-minute internal timeout on Stage 3:** `NewParser()`'s signature is locked (D-02/D-03 — no ctx arg). The loader still needs SOME bound on the bootstrap stage so `NewParser()` can't block forever. Five minutes is generous enough for a cold download of a ~1 MiB shared library on a slow VPN while preserving a sane failure upper bound.
- **No `bootstrapErrToPublicErr` adapter (H2 locked):** Plan 01 aliased the three root `purejson.Err*` sentinels to the canonical `bootstrap.Err*` sentinels by pointer. `%w` wrapping preserves that pointer across the chain, so `errors.Is` matches on both sentinel paths with no translation step. Writing an adapter would duplicate state and create drift.
- **TestMain compatibility seam (Rule 3 deviation):** deleting the target/release walk broke `TestParserPoolRoundTrip`, `TestParserParseInt64`, and other tests that expected implicit library discovery. The alternative — editing every one of those tests to set the env var manually — would have been a multi-file churn with no production benefit. A single `TestMain` that pre-seeds `PURE_SIMDJSON_LIB_PATH` when the cargo artefact is present is narrower in blast radius and reversible: once Phase 5 lands the bootstrap pipeline end-to-end, the TestMain can go away.
- **Keep `formatAttemptedPaths` and `wrapLoadFailure` unchanged:** the plan did not require touching them and Phase 3 tests (`TestActiveLibraryEnvOverrideMissingWrapsLoadFailure`) still assert the `"attempted paths:"` substring. Silent ABI preservation.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking Issue] Added TestMain to keep Phase 3/4 tests working after the target/release walk was deleted**

- **Found during:** Task 1 verification (`go test ./...` timed out at 2 minutes because `TestParserPoolRoundTrip` triggered a real BootstrapSync against GitHub, which returns a 404 HTML page and then hangs the ladder).
- **Issue:** The plan deletes `libraryCandidates()` (the implicit target/release + target/debug walk) without acknowledging that ~20 existing tests in the package assume that walk exists. Without a compatibility layer every `go test ./...` on a dev machine would take 5+ minutes and fail with a cryptic "bootstrap failed / GitHub 404 HTML" trailer.
- **Fix:** Added `testmain_test.go` with a `TestMain(m *testing.M)` that sets `PURE_SIMDJSON_LIB_PATH` to `target/release/<platformLibraryName>` when the cargo artefact exists. Tests that exercise resolution-chain behaviour override with `t.Setenv(libraryEnvPath, "")`.
- **Files modified:** new `testmain_test.go`.
- **Commit:** `7c6e45d` (bundled with the GREEN implementation commit because they are one atomic rewrite; splitting RED / GREEN was sufficient and adding a third commit for the compat shim would just inflate the log without a separable green state).

## Authentication Gates

None — no external services touched.

## Issues Encountered

- The existing test `bootstrap.ResetBootstrapFailureCacheForTest` lives in `internal/bootstrap/export_test.go`, which is compiled only when running tests **inside** the `internal/bootstrap` package. The root `purejson_test` package can't reach it. I had drafted the new tests with calls to it for isolation, then realised the import was invalid. Resolution: dropped the calls — my new tests either use a fresh `t.TempDir()` cache (Stage 2 hit, no bootstrap run) or exercise the bootstrap-failure path exactly once per run, so the 30-second memoization window has no cross-test effect.

## Known Stubs

None. All tests pass; no TODO / FIXME / placeholder markers introduced.

## User Setup Required

None — no external service configuration needed.

## Next Phase Readiness

- **Plan 05 (CLI):** the CLI `fetch` subcommand can now rely on a clean separation: the loader calls `bootstrap.BootstrapSync(ctx)` directly, which is the same API the CLI imports. Users who want offline pre-fetch continue to use the CLI; users who want inline bootstrap now get it automatically from `NewParser()`.
- **Plan 06 (tests + CI):** the previously-deferred Fault Injection Matrix item 6 ("concurrent bootstrap — N goroutines, exactly one download") can now be exercised through the root `NewParser()` / `ParserPool.Get()` surface because `activeLibrary` is the concurrency choke-point and its lock scope is now correct. The M1 lock-scope test locks that in via source introspection.
- **Phase 6 CI release pipeline:** once real artefacts and checksums ship, the TestMain compatibility seam in `testmain_test.go` can be removed — the `bootstrap.BootstrapSync` path will succeed on a fresh CI runner without the env-var bypass.

## Self-Check: PASSED

Created/modified files all exist and the task commits are present on the branch:

- FOUND: `library_loading.go`
- FOUND: `library_loading_test.go`
- FOUND: `testmain_test.go`
- FOUND: commit `d2c3896` (Task 1 RED)
- FOUND: commit `7c6e45d` (Task 1 GREEN)

Plan-level verification:

- `go build ./...` exit 0
- `go vet ./...` exit 0
- `go test ./... -count=1 -timeout 120s` — PASS (root `purejson` 4.1s; `internal/bootstrap` 9.1s; `internal/ffi` 0.7s)
- `go test ./... -count=1 -race -timeout 180s` — PASS (root 9.3s; bootstrap 9.5s; ffi 2.0s)
- Targeted plan-04 tests: `TestResolveLibraryPathAbsolute`, `TestLibPathEnvBypassesDownload`, `TestResolveLibraryPathCacheHit`, `TestResolveLibraryPathBootstrapError`, `TestActiveLibraryLockScope`, `TestNewParserSignatureUnchanged`, `TestActiveLibraryEnvOverrideMissingWrapsLoadFailure` all PASS
- Every grep acceptance criterion from the plan matches:
  - `grep "target/release\|target/debug" library_loading.go` — 0 matches (D-01)
  - `grep "libraryCandidates\|rustTargetTriple" library_loading.go` — 0 matches
  - `grep "func platformLibraryName" library_loading.go` — 0 matches (moved to internal/bootstrap)
  - `grep "bootstrap.BootstrapSync" library_loading.go` — present on line 141 (Stage 3)
  - `grep "bootstrap.CachePath" library_loading.go` — present on line 133 (Stage 2)
  - `grep "PURE_SIMDJSON_LIB_PATH"` appears in Stage 3 error (D-21): lines 143 and 151
  - `awk '/func activeLibrary/,/^}/' library_loading.go | grep -c 'libraryMu\.Lock'` = 2 (double-checked locking fingerprint, M1)
  - M1 grep-awk invariant (a): resolveLibraryPath called outside libraryMu.Lock — OK
  - M1 grep-awk invariant (b): loadLibrary called outside libraryMu.Lock — OK

## TDD Gate Compliance

Task 1 followed RED -> GREEN separately:
- RED commit `d2c3896` ships failing tests against the then-unchanged library_loading.go. Verified failing: `TestResolveLibraryPathCacheHit`, `TestResolveLibraryPathBootstrapError`, `TestActiveLibraryLockScope`.
- GREEN commit `7c6e45d` rewrites library_loading.go and adds the TestMain compat seam. All seven new tests pass; the full suite (root `purejson`, `internal/bootstrap`, `internal/ffi`) passes both normally and under `-race`.

No REFACTOR phase was needed — the rewrite ships the target shape on first GREEN.

---
*Phase: 05-bootstrap-distribution*
*Completed: 2026-04-20*
