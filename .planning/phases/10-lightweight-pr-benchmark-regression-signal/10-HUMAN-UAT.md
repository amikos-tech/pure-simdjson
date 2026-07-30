---
status: partial
phase: 10-lightweight-pr-benchmark-regression-signal
source: [10-VERIFICATION.md]
started: 2026-07-30T11:59:22Z
updated: 2026-07-30T11:59:22Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Seed and verify the main baseline
expected: After merging, run `main benchmark baseline` with `workflow_dispatch` from `main`; the run is green, creates a `pr-bench-baseline-<sha>` cache entry, and uploads an artifact containing `head.bench.txt`.
result: [pending]

### 2. Verify a cache-hit PR run
expected: A non-ignored no-op PR runs `pr benchmark` within ten minutes, restores a matched baseline key, remains advisory/green, and publishes the step summary and sticky comment.
result: [pending]

### 3. Verify the cache-miss bypass
expected: After deleting the `pr-bench-baseline-*` cache, a non-ignored PR sees an empty matched key, publishes an advisory-bypass summary/comment, and remains green.
result: [pending]

### 4. Verify PR cancellation
expected: Two commits pushed to one PR within ten seconds cancel the earlier `pr benchmark` run; only the latest run updates the sticky comment.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps

