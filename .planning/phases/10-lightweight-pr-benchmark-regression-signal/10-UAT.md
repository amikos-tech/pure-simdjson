---
status: partial
phase: 10-lightweight-pr-benchmark-regression-signal
source:
  - 10-01-SUMMARY.md
  - 10-02-SUMMARY.md
  - 10-03-LIVE-VERIFICATION.md
started: 2026-04-28T04:04:29Z
updated: 2026-04-28T04:11:12Z
---

## Current Test
<!-- OVERWRITE each test - shows where we are -->

[testing paused - 3 live Actions checks outstanding]

## Tests

### 1. PR Regression Parser Contract
expected: Run the parser contract tests. The tests should pass, showing that `scripts/bench/check_pr_regression.py` detects only significant `sec/op` slowdowns, ignores faster/non-significant/non-runtime rows, emits JSON plus markdown, exits 0 in advisory mode, exits 1 when `REQUIRE_NO_REGRESSION=true` and regressions are flagged, fails closed on malformed input, and reports advisory bypass when no baseline exists.
result: pass
evidence: |
  `python3 -m unittest tests.bench.test_check_pr_regression` failed before test execution because
  `tests/bench` is not a Python package.
  Ran the standalone unittest file instead:
  `python3 tests/bench/test_check_pr_regression.py`
  Result: 17 tests passed.

### 2. PR Benchmark Orchestrator Contract
expected: Run the orchestrator contract tests. The tests should pass, showing that `scripts/bench/run_pr_benchmark.sh` locks the Tier 1/2/3 PR benchmark subset and 10-minute budget, produces `head.bench.txt`, `baseline.bench.txt`, `regression.benchstat.txt`, `summary.json`, and `markdown.md` in baseline mode, skips benchstat and marks bypass in no-baseline mode, and reports a clear error for a missing baseline file.
result: pass
evidence: |
  `python3 -m unittest tests.bench.test_run_pr_benchmark` failed before test execution because
  `tests/bench` is not a Python package.
  Ran the standalone unittest file instead:
  `python3 tests/bench/test_run_pr_benchmark.py`
  Result: 4 tests passed.

### 3. PR Workflow Static Wiring
expected: Inspect `.github/workflows/pr-benchmark.yml`. The workflow should be named `pr benchmark`, trigger on `pull_request`, use the locked `paths-ignore`, set concurrency to `pr-bench-${{ github.event.pull_request.number }}` with cancellation enabled, grant only `contents: read` and `pull-requests: write`, restore `baseline.bench.txt` via `actions/cache/restore@0400d5f644dc74513175e3cd8d07132dd4860809` with key `pr-bench-baseline-NEVER-MATCHES`, determine no-baseline from `cache-matched-key == ''`, call `bash scripts/bench/run_pr_benchmark.sh`, append `markdown.md` to `$GITHUB_STEP_SUMMARY`, post/update a sticky comment with `continue-on-error: true`, and upload `pr-bench-summary/` artifacts for 14 days.
result: pass
evidence: |
  `python3 tests/bench/test_pr_benchmark_workflows.py` passed all 7 workflow contract tests.
  `actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml` exited 0.
  `yq e '.' .github/workflows/pr-benchmark.yml` parsed successfully.

### 4. Main Baseline Workflow Static Wiring
expected: Inspect `.github/workflows/main-benchmark-baseline.yml`. The workflow should be named `main benchmark baseline`, run only on pushes to `main` plus `workflow_dispatch`, use the same docs/planning/workflow/action paths-ignore set, set concurrency to `main-bench-baseline` without cancellation, grant only `contents: read`, call `bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir pr-bench-summary`, copy `pr-bench-summary/head.bench.txt` to `baseline.bench.txt`, save that exact path with key `pr-bench-baseline-${{ github.sha }}`, and upload baseline evidence artifacts for 30 days.
result: pass
evidence: |
  `python3 tests/bench/test_pr_benchmark_workflows.py` passed all 7 workflow contract tests.
  `actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml` exited 0.
  `yq e '.' .github/workflows/main-benchmark-baseline.yml` parsed successfully.

### 5. Changelog and Blocking-Flip Discoverability
expected: Inspect `CHANGELOG.md`, `.github/workflows/pr-benchmark.yml`, and `scripts/bench/check_pr_regression.py`. The changelog should describe the advisory PR regression check and name `REQUIRE_NO_REGRESSION` as the future blocking-flip knob; the PR workflow should set `REQUIRE_NO_REGRESSION: "false"` in the regression-check step; the parser should use the same env var; and `grep -nE "REQUIRE_NO_REGRESSION" .github/workflows/pr-benchmark.yml CHANGELOG.md scripts/bench/check_pr_regression.py` should return exactly three matches.
result: pass
evidence: |
  `python3 tests/bench/test_pr_benchmark_workflows.py` passed the changelog/workflow discoverability assertion.
  `grep -nE "REQUIRE_NO_REGRESSION" .github/workflows/pr-benchmark.yml CHANGELOG.md scripts/bench/check_pr_regression.py`
  returned exactly three matches:
  - `.github/workflows/pr-benchmark.yml:79`
  - `CHANGELOG.md:11`
  - `scripts/bench/check_pr_regression.py:26`

### 6. Live Main Baseline Actions Evidence
expected: After the workflow files are merged to `main`, the Actions sidebar should show `main benchmark baseline`. A `workflow_dispatch` run on `main` should finish green, create a `pr-bench-baseline-<sha>` cache entry, and upload an artifact containing `head.bench.txt`.
result: blocked
blocked_by: third-party
reason: "Requires live GitHub Actions after the workflow files are merged to main and a workflow_dispatch baseline run is available."

### 7. Live PR Benchmark Actions Evidence
expected: Open or update a small non-doc PR after a baseline exists. The `pr benchmark` workflow should run, the `Restore main-baseline cache` step should have a non-empty `cache-matched-key`, the step summary should show the PR benchmark result, the sticky PR comment should post or update unless fork-token denial is harmless because the step is `continue-on-error`, the job should finish green in advisory mode, and diagnostic artifacts should be available.
result: blocked
blocked_by: third-party
reason: "Requires a live PR run after the main baseline cache exists."

### 8. Live Cache-Miss and Concurrency Evidence
expected: Delete `pr-bench-baseline-*` cache entries and rerun a PR benchmark. The cache-miss run should report `advisory bypass`, exit green, and upload `head.bench.txt`, `summary.json`, and `markdown.md`. Then push two commits quickly to the same PR; the earlier run should be cancelled and the latest run should update the sticky comment.
result: blocked
blocked_by: third-party
reason: "Requires live Actions cache mutation and concurrent PR workflow runs."

## Summary

total: 8
passed: 5
issues: 0
pending: 0
skipped: 0
blocked: 3

## Gaps

[]
