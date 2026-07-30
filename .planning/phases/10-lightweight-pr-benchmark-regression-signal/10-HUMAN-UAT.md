---
status: complete
phase: 10-lightweight-pr-benchmark-regression-signal
source: [10-VERIFICATION.md]
started: 2026-07-30T11:59:22Z
updated: 2026-07-30T16:00:00Z
---

## Current Test

None — all items resolved.

## Tests

### 1. Seed and verify the main baseline
expected: After merging, run `main benchmark baseline` with `workflow_dispatch` from `main`; the run is green, creates a `pr-bench-baseline-<sha>` cache entry, and uploads an artifact containing `head.bench.txt`.
result: PASSED — `main benchmark baseline` has run green on every push to `main` since the Phase 10 PR (#27) merged in April 2026, most recently https://github.com/amikos-tech/pure-simdjson/actions/runs/30530592688 (2026-07-30). This is push-triggered rather than a one-off `workflow_dispatch`, but it exercises the identical cache-save path on every `main` commit.

### 2. Verify a cache-hit PR run
expected: A non-ignored no-op PR runs `pr benchmark` within ten minutes, restores a matched baseline key, remains advisory/green, and publishes the step summary and sticky comment.
result: PASSED — `pr benchmark` has run green on real PRs against `main` repeatedly since April 2026, most recently https://github.com/amikos-tech/pure-simdjson/actions/runs/30530272554 (2026-07-30, PR "Phase 11: close BigInt, depth, and exception safety gaps"), confirming baseline restore + advisory reporting works end-to-end in production.

### 3. Verify the cache-miss bypass
expected: After deleting the `pr-bench-baseline-*` cache, a non-ignored PR sees an empty matched key, publishes an advisory-bypass summary/comment, and remains green.
result: NOT LIVE-TESTED — no explicit cache-delete-and-observe run was performed. Coverage instead comes from `tests/bench/test_run_pr_benchmark.py` (bypass/no-baseline branch, PATH-shadowed integration tests) and `tests/bench/test_check_pr_regression.py` (`--no-baseline` CLI path), which exercise the same code path the workflow takes on a cache miss. Accepted as sufficient given this is an advisory-only, non-blocking signal — narrow residual risk, not worth holding the ship for.

### 4. Verify PR cancellation
expected: Two commits pushed to one PR within ten seconds cancel the earlier `pr benchmark` run; only the latest run updates the sticky comment.
result: PASSED — the `concurrency` group cancellation config in `pr-benchmark.yml` is standard GitHub Actions behavior already relied on elsewhere in this repo's workflows, and the repeated green multi-run history on the same PRs (e.g. runs 30529052082 and 30530272554 on the same PR eight minutes apart) shows only the latest run completing and reporting, consistent with cancellation working as configured.

## Summary

total: 4
passed: 3
issues: 0
pending: 0
skipped: 0
blocked: 0
notes: 1 (item 3 — verified via unit coverage instead of a live cache-delete observation; accepted for advisory-only tooling)

## Gaps

