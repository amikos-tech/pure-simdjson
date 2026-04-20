---
phase: 05-bootstrap-distribution
plan: 03
subsystem: testing
tags: [bootstrap, httptest, fault-injection, sha256, retry, fallback, url-construction]

# Dependency graph
requires:
  - phase: 05-bootstrap-distribution
    plan: 01
    provides: Version, Checksums map, SupportedPlatforms, ChecksumKey, validateBaseURL
  - phase: 05-bootstrap-distribution
    plan: 02
    provides: BootstrapSync entry point, BootstrapOption surface, export_test.go M3 seam set, cache_test.go and bootstrap_test.go with 26 tests covering 5 fault-injection rows
provides:
  - Full Fault Injection Matrix coverage for unit + fast-integration rows (items 1, 2, 5, 7, 9, 11)
  - TestURLConstruction (DIST-01 across all 5 platforms) with trailing-slash hygiene
  - TestGitHubAssetNames proving H1 platform-tagging is pairwise-distinct (flat-namespace no-collision)
  - TestGitHubArtifactURL (default + override base)
  - TestChecksumKeyFormat pinning the ChecksumKey map layout so the CLI verify subcommand cannot drift
  - TestResolveConfigCacheDirEnv (L2 env-var precedence)
  - TestBootstrapSync happy-path integration test (DIST-04)
  - TestRetryOn429ThenSuccess (matrix item 2)
  - TestFallback404R2Then200GH (matrix item 9) — the test that uncovered a per-URL vs ladder-fatal error-classification bug in downloadWithRetry
  - Production fix: isLadderFatalError separates "permanent for this URL" (404) from "permanent for the whole ladder" (checksum/no-checksum/HTTPS downgrade), so R2 404 correctly rolls over to GH
  - Three additional export_test.go re-exports: R2ArtifactURL, GitHubArtifactURL, GitHubAssetName
affects: [05-04-loader-integration, 05-05-cli-bootstrap, 05-06-tests-ci-matrix, 06-ci-release-pipeline]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-tier permanent-error classification in downloadWithRetry: ladder-fatal (stop all URLs) vs per-URL fatal (skip remaining retries, try next URL)"
    - "httptest servers for R2 + GH fallback wired via bootstrap.WithGitHubBaseURL test seam (M3)"
    - "Table-driven URL tests that assert exact wire format via re-exported url helpers instead of rebuilding format strings in tests"

key-files:
  created:
    - .planning/phases/05-bootstrap-distribution/05-03-SUMMARY.md
  modified:
    - internal/bootstrap/bootstrap_test.go
    - internal/bootstrap/download.go
    - internal/bootstrap/export_test.go

key-decisions:
  - "404 from R2 must not abort the whole retry ladder — it marks the URL permanently dead but the GH fallback still fires (Fault Injection Matrix item 9); only checksum mismatch, no-checksum, and HTTPS-downgrade stay ladder-fatal."
  - "Expose r2ArtifactURL, githubArtifactURL, githubAssetName via export_test.go so tests assert the exact wire format instead of rebuilding the string in-test — eliminates drift between production and test expectations."

patterns-established:
  - "Per-URL vs ladder-fatal error classification in URL-ladder retry loops."
  - "Flat-namespace-asset pairwise-distinctness as an H1 regression test pattern."

requirements-completed: [DIST-01, DIST-02, DIST-03, DIST-04, DIST-05, DIST-07]

# Metrics
duration: 3min
completed: 2026-04-20
---

# Phase 5 Plan 3: Bootstrap Test Matrix Completion Summary

**Fault Injection Test Matrix is now fully covered: Plan 02 shipped 16 tests hitting items 1, 5, 7, 11; Plan 03 added 8 more tests that close items 2 and 9 and pin URL construction (H1) + cache layout (D-07) against future drift. Uncovered and fixed a per-URL-vs-ladder-fatal error-classification bug that had silently aborted the GH fallback on any 4xx from R2.**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-04-20T11:41:18Z
- **Completed:** 2026-04-20T11:44:52Z
- **Tasks:** 1
- **Commits:** 1
- **Files modified:** 3 (no new files — all changes append to existing 05-02 artifacts)

## Accomplishments

- Added 8 test functions (25 total in bootstrap_test.go, up from 17), all passing under `go test -race`.
- Three new test-seam re-exports (`R2ArtifactURL`, `GitHubArtifactURL`, `GitHubAssetName`) in `internal/bootstrap/export_test.go` so tests compare URLs by exact wire format rather than rebuilding strings in-test.
- Fixed a Plan-02-era production bug in `downloadWithRetry` that marked any non-retryable HTTP status as ladder-fatal. The fix introduces `isLadderFatalError` and flips 404-then-fallback into the expected behaviour.
- Every Fault Injection Matrix row this plan owns (unit-level + fast-integration) now has a passing automated test.

## Fault Injection Matrix → Test Mapping

| # | Fault | Covered By | Plan |
|---|-------|-----------|------|
| 1 | Checksum corruption → ErrChecksumMismatch, no dlopen | `TestChecksumMismatchIsPermanent` | 05-02 |
| 2 | HTTP 429 then 200 → retry succeeds | `TestRetryOn429ThenSuccess` | **05-03 (new)** |
| 3 | HTTP 503 all R2 attempts, 200 GH → fallback succeeds | deferred to Plan 06 (full integration wave) | 05-06 |
| 4 | ctx cancel mid-download → context.Canceled, temp cleaned up | deferred to Plan 06 | 05-06 |
| 5 | ctx cancel during retry sleep → returns within 50ms | `TestSleepWithJitterCtxCancel` | 05-02 |
| 6 | Concurrent bootstrap (N goroutines) → exactly one download | deferred to Plan 06 (needs library_loading.go rewire) | 05-06 |
| 7 | HTTPS→HTTP redirect rejected | `TestHTTPSDowngradeRejected` | 05-02 |
| 8 | .lock file contention → second process waits | deferred to Plan 06 (needs process-level harness) | 05-06 |
| 9 | 404 on R2, 200 on GH → GH fallback fires, artifact cached | `TestFallback404R2Then200GH` | **05-03 (new)** |
| 10 | PURE_SIMDJSON_DISABLE_GH_FALLBACK=1 + R2 404 → ErrAllSourcesFailed | deferred to Plan 06 | 05-06 |
| 11 | GitHub 403 rate-limit body classified retryable | `TestIsRetryable` (403 cases) | 05-02 |

Unit tests also added by Plan 03 (not matrix rows but plan `must_haves`):

| Test | Requirement |
|------|-------------|
| `TestURLConstruction` (5 subtests + trailing-slash hygiene) | DIST-01 |
| `TestGitHubAssetNames` (5 subtests + pairwise-distinct) | H1 regression |
| `TestGitHubArtifactURL` (default + override) | H1 URL-level |
| `TestChecksumKeyFormat` | CLI / Checksums map contract |
| `TestResolveConfigCacheDirEnv` | L2 |
| `TestBootstrapSync` | DIST-04 happy path |

Unit tests already in place from 05-02 (not duplicated):

| Test | File |
|------|------|
| `TestCacheDirPerms`, `TestArtifactCachePath`, `TestCachePathCurrent`, `TestCacheDirEnvOverride`, `TestCacheDirEnvOverrideEmpty`, `TestCacheDirTempDirFallbackPerms`, `TestWithProcessFileLockBasic`, `TestAtomicInstall` | `cache_test.go` |
| `TestSleepWithJitterCtxCancel`, `TestPermanentBootstrapError`, `TestIsRetryable` (11 subtests), `TestBootstrapSyncNilCtx`, `TestWithMirrorValidation`, `TestWithMirrorLoopback`, `TestWithMirrorHTTPS`, `TestResolveConfigEnvMirror`, `TestResolveConfigDisableGH`, `TestUserAgentStamp`, `TestBootstrapFailureMemoized`, `TestBootstrapSuccessClearsFailureCache`, `TestChecksumMismatchIsPermanent`, `TestNoChecksumReturnsSentinel`, `TestHTTPSDowngradeRejected`, `TestWithVersionAndWithTarget`, `TestWithDest` | `bootstrap_test.go` |

## Task Commits

1. **Task 1: Round out Fault Injection Matrix (tests + per-URL fatal-error fix)** — `43e74ee` (test)

_This plan has a single atomic task. The production-code fix (Rule 1 deviation) is bundled into the same commit because the test that surfaced it is the same test that proves the fix — separating them would leave a red-to-green gap on one of the two commits._

## Files Created/Modified

**Modified:**

- `internal/bootstrap/bootstrap_test.go` — appended 8 test functions (`TestURLConstruction`, `TestGitHubAssetNames`, `TestGitHubArtifactURL`, `TestChecksumKeyFormat`, `TestResolveConfigCacheDirEnv`, `TestBootstrapSync`, `TestRetryOn429ThenSuccess`, `TestFallback404R2Then200GH`) + `fakeLibBody` / `computeHex` helpers.
- `internal/bootstrap/export_test.go` — added `R2ArtifactURL`, `GitHubArtifactURL`, `GitHubAssetName` re-exports so URL tests assert exact wire format.
- `internal/bootstrap/download.go` — introduced `isLadderFatalError` and flipped `downloadWithRetry` to distinguish per-URL fatal (break to next URL) from ladder-fatal (abort ladder).

## Decisions Made

- **Per-URL vs ladder-fatal error classification:** Plan 02's `downloadWithRetry` treated every permanent error identically — any 4xx killed the whole ladder, so R2 404 never reached the GH fallback. The new contract keeps checksum mismatch / no-checksum / HTTPS-downgrade as ladder-fatal (they can't get better on a different URL) but lets HTTP-status permanent errors (like 404) skip only the remaining retries for that URL. This is the correct semantics for "primary-then-fallback" ladders and is required by Fault Injection Matrix item 9.
- **Export the three URL helpers instead of reconstructing URLs in-test:** The alternative (sprintf the expected URL in-test) would let production and tests drift silently. Exporting the functions keeps tests aligned with the code under test while preserving the unexported package API for downstream consumers.
- **Bundle production fix and test in a single commit:** Normally `fix(...)` and `test(...)` would be separate TDD commits. Here the task is not marked `tdd="true"` and the single task includes both the matrix verification work AND the deviation fix the verification discovered; splitting them would leave the tree red between commits.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `downloadWithRetry` aborted the whole URL ladder on any non-retryable HTTP status**

- **Found during:** Task 1 (TestFallback404R2Then200GH FAILED on first run — R2 404 propagated to `isPermanentBootstrapError` → immediate `return` on the primary URL, GH fallback never attempted).
- **Issue:** Plan 02 wrote a single `isPermanentBootstrapError` check that treated "permanent for this URL" (e.g. 404) identically to "permanent for the whole ladder" (e.g. checksum mismatch). This contradicts 05-VALIDATION.md §Fault Injection Matrix item 9 which requires "404 on R2, 200 on GH → GH fallback fires, artifact cached". The existing test (`TestRetryOn429ThenSuccess`) did not catch it because 429 IS retryable, so the early-return branch was never exercised with a non-retryable 4xx while a fallback URL existed.
- **Fix:** Added `isLadderFatalError(err)` that returns true only for `ErrChecksumMismatch`, `ErrNoChecksum`, and HTTPS-downgrade redirect policy errors. In `downloadWithRetry`: ladder-fatal → return immediately (same as before); per-URL fatal → `break` out of the attempt loop to move on to the next URL; transient → retry up to 3 times as before.
- **Files modified:** `internal/bootstrap/download.go`.
- **Verification:** `TestFallback404R2Then200GH` now passes (R2 hit ≥1, GH hit = 1, artifact cached with GH body). `TestChecksumMismatchIsPermanent` still passes (checksum mismatch still aborts the ladder after a single hit). All 25 tests pass under `-race`.
- **Committed in:** `43e74ee` (Task 1 commit).

**2. [Rule 3 - Blocking Issue] Extended `export_test.go` with three additional re-exports**

- **Found during:** Writing `TestURLConstruction`, `TestGitHubAssetNames`, `TestGitHubArtifactURL`.
- **Issue:** The plan's acceptance criteria require asserting exact R2 / GH URL output for 5 platforms plus pairwise-distinctness for H1. The helpers (`r2ArtifactURL`, `githubArtifactURL`, `githubAssetName`) are unexported by design and Plan 02's `export_test.go` did not include them. Rebuilding the format strings inline in the test would defeat the purpose (test-production drift).
- **Fix:** Added `var R2ArtifactURL = r2ArtifactURL`, `var GitHubArtifactURL = githubArtifactURL`, `var GitHubAssetName = githubAssetName` to `export_test.go`. This is the idiomatic Go test-seam pattern (stdlib `net/http/export_test.go` does the same), and because `export_test.go` is compiled only during `go test` the production API is unaffected.
- **Files modified:** `internal/bootstrap/export_test.go`.
- **Verification:** `head -1 internal/bootstrap/bootstrap_test.go | grep 'package bootstrap_test'` succeeds; `grep -E 'R2ArtifactURL|GitHubArtifactURL|GitHubAssetName' internal/bootstrap/export_test.go` returns three entries. Plan-03 acceptance criteria mention this possibility explicitly ("If Plan 02 omitted them, expose them here as a first step").
- **Committed in:** `43e74ee` (Task 1 commit).

---

**Total deviations:** 2 (1 Rule 1 bug fix, 1 Rule 3 blocking)
**Impact on plan:** Both deviations are required to satisfy the plan's acceptance criteria — the Rule 1 fix is a correctness bug that would have shipped in v0.1 otherwise, and the Rule 3 extension to `export_test.go` is anticipated by the plan itself.

## Issues Encountered

None beyond the deviations above. The fault-9 bug surfaced on first run, the fix was local (two branches in one function), and the full suite turned green on the next iteration.

## Authentication Gates

None — all tests use `httptest.NewServer` / `httptest.NewTLSServer` entirely in-process.

## Known Stubs

None. `grep -ri 'TODO|FIXME|placeholder|coming soon|not available' internal/bootstrap/*_test.go` returns no results.

## User Setup Required

None — no external services touched.

## Next Phase Readiness

- **Plan 04 (loader integration):** can proceed knowing that the GH fallback actually fires on R2 404 (the fix this plan landed). `library_loading.go::resolveLibraryPath` can trust `bootstrap.BootstrapSync` to exhaust both sources before returning `ErrAllSourcesFailed`.
- **Plan 05 (CLI):** `ChecksumKey` format is pinned by `TestChecksumKeyFormat`, and the Checksums-map layout cannot drift without breaking the test.
- **Plan 06 (tests + CI):** the remaining Fault Injection Matrix rows (items 3, 4, 6, 8, 10) are all items that need either the rewired loader (item 6 concurrent bootstrap) or longer-running process-level harnesses. Plan 06 is scheduled to own them. The matrix mapping table above records ownership explicitly.

## Self-Check: PASSED

Created/modified files all exist and the task commit is present on the branch:

- FOUND: `internal/bootstrap/bootstrap_test.go` (includes 25 `func Test` definitions, `package bootstrap_test` at line 1)
- FOUND: `internal/bootstrap/download.go` with `isLadderFatalError`
- FOUND: `internal/bootstrap/export_test.go` with `R2ArtifactURL`, `GitHubArtifactURL`, `GitHubAssetName`
- FOUND: commit `43e74ee` (Task 1)

Plan-level verification:

- `go build ./...` exit 0
- `go vet ./internal/bootstrap/...` exit 0
- `go test ./internal/bootstrap/... -count=1 -timeout 60s` — PASS (8.4s, 33 test functions including 05-02 + 05-03 + cache_test.go subtests)
- `go test ./internal/bootstrap/... -count=1 -race -timeout 120s` — PASS (9.9s)
- `head -1 internal/bootstrap/bootstrap_test.go` → `package bootstrap_test`
- `grep -c "^func Test" internal/bootstrap/bootstrap_test.go` → 25 (acceptance criterion: >= 20)
- `grep bootstrap.WithHTTPClient internal/bootstrap/bootstrap_test.go` → 9 occurrences (M3 seam usage)
- `grep bootstrap.WithGitHubBaseURL internal/bootstrap/bootstrap_test.go` → 2 occurrences (M3 fallback seam)
- Tests specifically run and pass: `TestURLConstruction`, `TestGitHubAssetNames`, `TestGitHubArtifactURL`, `TestChecksumKeyFormat`, `TestResolveConfigCacheDirEnv`, `TestBootstrapSync`, `TestRetryOn429ThenSuccess`, `TestFallback404R2Then200GH`, `TestCacheDirPerms`

---
*Phase: 05-bootstrap-distribution*
*Completed: 2026-04-20*
