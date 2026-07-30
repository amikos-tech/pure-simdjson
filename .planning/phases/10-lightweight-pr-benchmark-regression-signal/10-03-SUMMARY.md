---
phase: 10-lightweight-pr-benchmark-regression-signal
plan: 03
subsystem: ci
tags: [github-actions, benchmarks, actions-cache, benchstat, regression]
requires:
  - phase: 10-lightweight-pr-benchmark-regression-signal
    provides: Plan 01 regression parser and Plan 02 benchmark orchestrator
provides:
  - Pull-request advisory benchmark regression workflow
  - Main-branch baseline cache producer workflow
  - Discoverable future blocking-flip control in the changelog
affects: [phase-10, pull-request-ci, benchmark-regression-gate]
tech-stack:
  added: [actions/cache, marocchino/sticky-pull-request-comment]
  patterns: [restore-only PR baseline cache, SHA-pinned GitHub Actions, step-summary-first reporting]
key-files:
  created:
    - .github/workflows/pr-benchmark.yml
    - .github/workflows/main-benchmark-baseline.yml
  modified:
    - CHANGELOG.md
key-decisions:
  - "PR jobs restore but never save the rolling baseline cache, preventing PR cache poisoning."
  - "The advisory-to-blocking transition remains a one-line REQUIRE_NO_REGRESSION workflow change after observing CI noise."
  - "Workflow-only and composite-action-only changes are intentionally paths-ignored to preserve the PR runtime budget."
patterns-established:
  - "Use pull_request with minimal permissions and a best-effort sticky comment; the job summary remains the fork-safe surface."
  - "Main-only workflow runs create canonical baseline.bench.txt cache entries that PR jobs restore by prefix."
requirements-completed: [D-01, D-06, D-07, D-09, D-10, D-13, D-14, D-16, D-17, D-18, D-19, D-20, D-21]
duration: 8min
completed: 2026-07-30
---

# Phase 10 Plan 03: PR Benchmark Workflows Summary

**SHA-pinned advisory PR benchmark checks with a main-only rolling baseline cache, step summaries, and best-effort sticky comments**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-30T11:29:21Z
- **Completed:** 2026-07-30T11:37:21Z
- **Tasks:** 1 automated task completed; 1 post-merge verification checkpoint pending
- **Files modified:** 3 production files, 1 summary

## Accomplishments

- Verified the PR workflow uses `pull_request`, least-privilege permissions, per-PR cancellation, restore-only baseline caching, a 15-minute job cap, and 14-day diagnostic artifacts.
- Verified the main-only workflow produces `baseline.bench.txt` under the matching cache namespace and exposes a manual dispatch for initial cache seeding.
- Confirmed both workflows pass `actionlint` and `yq`, the local benchmark orchestrator succeeds in no-baseline mode, and all benchmark tests pass.

## Task Commits

1. **Task 1: Author both workflow YAMLs and CHANGELOG note** - `60326e4` (pre-existing production commit, verified against this plan)

## Files Created/Modified

- `.github/workflows/pr-benchmark.yml` - Advisory pull-request regression check with cache restore, step summary, sticky comment, and diagnostic artifact upload.
- `.github/workflows/main-benchmark-baseline.yml` - Main-only baseline producer that caches the captured head benchmark under the canonical path.
- `CHANGELOG.md` - Unreleased note documenting advisory mode and the `REQUIRE_NO_REGRESSION` future blocking knob.

## Decisions Made

- Kept the PR workflow on `pull_request`, never `pull_request_target`, so fork code never receives base-repository write tokens.
- Kept baseline writes exclusively in the main-only workflow; PR jobs use `actions/cache/restore` only.
- Preserved advisory mode with `REQUIRE_NO_REGRESSION: "false"`; promoting it to a required check remains a separate decision after observing noise.

## Validation

- PASS: `actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml`
- PASS: `yq eval` parses both workflow files and confirms the PR timeout, eight path ignores, `main` trigger, dispatch trigger, and cache-save key.
- PASS: pinned action SHAs verified through the GitHub API.
- PASS: `bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir /tmp/pr-bench-sanity-phase10-03-20260730` produced `summary.json` and `markdown.md` with `bypassed: true`.
- PASS: `python3 -m unittest discover -s tests/bench -v`.
- PASS: protected Phase 6 and Phase 9 workflow/script files remain unchanged.

## Deviations from Plan

### Pre-existing Implementation

**1. Production workflow artifacts already committed**
- **Found during:** Task 1
- **Issue:** The two workflow files and CHANGELOG note were already present in ancestor commit `60326e4` before this executor began.
- **Resolution:** Preserved the existing implementation, revalidated every plan acceptance criterion locally, and recorded the existing task commit rather than duplicating or modifying correct files.
- **Files modified:** None during this execution task.
- **Verification:** All mandated workflow, orchestrator, and benchmark-suite checks passed.

**Total deviations:** 1 execution-state accommodation; no code changes outside plan scope.
**Impact on plan:** None; the existing implementation satisfies the planned production deliverable.

## Known Stubs

None. The workflows invoke the implemented Plan 02 benchmark orchestrator and do not render placeholder data.

## User Setup Required

After these files are squash-merged to `main`, run **main benchmark baseline** once with `workflow_dispatch` from the Actions page to seed the cache (an automatic main push run is also acceptable if it completes successfully). Then verify a non-ignored PR gets a cache hit, a step summary, a best-effort sticky comment, and an advisory green result. This live GitHub Actions verification is the remaining Task 2 checkpoint.

## Next Phase Readiness

Phase 10’s local implementation and tests are complete. The only remaining work is the post-merge GitHub Actions seed and live workflow verification.

---
*Phase: 10-lightweight-pr-benchmark-regression-signal*
*Completed: 2026-07-30*

## Self-Check: PASSED

- Confirmed the summary exists and the verified production commit `60326e4` is present in git history.
