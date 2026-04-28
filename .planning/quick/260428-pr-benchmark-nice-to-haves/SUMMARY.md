---
status: complete
date: 2026-04-28
---

# Quick Task Summary: PR Benchmark Nice-To-Haves

Applied the follow-up nice-to-have test hardening for Phase 10 PR benchmark review feedback.

## Completed

- Added `non-vs-base-header-retains-section.benchstat.txt` and a parser test that documents the current `ANY_METRIC_HEADER_RE` limitation without broadening the parser heuristic.
- Updated workflow YAML smoke tests to skip cleanly without `yq` and require empty stderr when `yq eval .` succeeds.
- Converted stale-output replacement coverage into a true two-run orchestrator test using distinct benchmark evidence on each run.

## Verification

- `python3 tests/bench/test_check_pr_regression.py`
- `python3 tests/bench/test_run_pr_benchmark.py`
- `python3 tests/bench/test_pr_benchmark_workflows.py`

