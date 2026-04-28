---
phase: 10
slug: lightweight-pr-benchmark-regression-signal
status: audited
nyquist_compliant: false
wave_0_complete: true
created: 2026-04-27
last_audited: 2026-04-27
---

# Phase 10 - Validation Strategy

Retroactive Nyquist audit for the lightweight PR benchmark regression signal.

Phase 10 has automated coverage for parser behavior, local orchestration, workflow contract invariants, workflow syntax/lint, and release-boundary negative checks. It remains **partial** rather than fully Nyquist-compliant because several requirements depend on live GitHub Actions behavior after merge to `main` and are intentionally tracked as manual-only checks.

## Test Infrastructure

| Property | Value |
|----------|-------|
| Framework | Python stdlib `unittest`; shell syntax checks; `actionlint`; `yq`; Go compile-only benchmark smoke. |
| Config file | None. Tests are discovered with `python3 -m unittest discover -s tests/bench -p "<file>.py"`. |
| Parser command | `python3 -m unittest discover -s tests/bench -p "test_check_pr_regression.py" -v` |
| Orchestrator command | `python3 -m unittest discover -s tests/bench -p "test_run_pr_benchmark.py" -v` |
| Workflow contract command | `python3 -m unittest discover -s tests/bench -p "test_pr_benchmark_workflows.py" -v` |
| Workflow lint command | `actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml && yq eval '.' .github/workflows/pr-benchmark.yml >/dev/null && yq eval '.' .github/workflows/main-benchmark-baseline.yml >/dev/null` |
| Full suite command | `python3 -m unittest discover -s tests/bench -v && actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml && yq eval '.' .github/workflows/pr-benchmark.yml >/dev/null && yq eval '.' .github/workflows/main-benchmark-baseline.yml >/dev/null && go test -run=^$ -bench=^$ ./...` |
| Estimated runtime | ~5 seconds for Python tests and workflow lint; Go compile-only runtime depends on local build cache. |

## Sampling Rate

- **After parser changes:** Run `python3 -m unittest discover -s tests/bench -p "test_check_pr_regression.py" -v`.
- **After orchestrator changes:** Run `bash -n scripts/bench/run_pr_benchmark.sh && python3 -m unittest discover -s tests/bench -p "test_run_pr_benchmark.py" -v`.
- **After workflow changes:** Run `python3 -m unittest discover -s tests/bench -p "test_pr_benchmark_workflows.py" -v && actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml && yq eval '.' .github/workflows/pr-benchmark.yml >/dev/null && yq eval '.' .github/workflows/main-benchmark-baseline.yml >/dev/null`.
- **Before release/merge:** Run the full suite command and complete the manual-only live checklist after merge to `main`.

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Automated Command | Test File / Check | Status |
|---------|------|------|-------------|-------------------|-------------------|--------|
| 10-01-T1/T2 | 10-01 | 1 | D-08 cache-miss bypass in parser, D-11 regression threshold, D-12 per-row reporting, D-13 advisory exit, D-14 blocking flip, D-15 markdown detail | `python3 -m unittest discover -s tests/bench -p "test_check_pr_regression.py" -v` | `tests/bench/test_check_pr_regression.py`; `tests/bench/fixtures/pr-regression/` | GREEN |
| 10-01-T2 | 10-01 | 1 | Reuse Phase 9 parser helper and fail closed on malformed/empty benchstat input | `python3 -m unittest discover -s tests/bench -p "test_check_pr_regression.py" -v` | `test_real_phase9_benchstat_format`, `test_malformed_input_fails_closed`, `test_empty_input_fails_closed` | GREEN |
| 10-02-T1 | 10-02 | 2 | D-02 benchmark families, D-03 fixture subset, D-04 `-count=5`, D-05 comparator subset, D-08 no-baseline orchestration | `python3 -m unittest discover -s tests/bench -p "test_run_pr_benchmark.py" -v` | `tests/bench/test_run_pr_benchmark.py` | GREEN |
| 10-02-T1 | 10-02 | 2 | Orchestrator syntax, staged output promotion, baseline/no-baseline file set, missing-baseline error | `bash -n scripts/bench/run_pr_benchmark.sh && python3 -m unittest discover -s tests/bench -p "test_run_pr_benchmark.py" -v` | `scripts/bench/run_pr_benchmark.sh`; `tests/bench/test_run_pr_benchmark.py` | GREEN |
| 10-03-T1 | 10-03 | 3 | D-01 job timeout headroom, D-16 paths-ignore trigger, D-17 skip list, D-18 `.github/actions/**` accepted skip, D-20 PR concurrency | `python3 -m unittest discover -s tests/bench -p "test_pr_benchmark_workflows.py" -v` | `tests/bench/test_pr_benchmark_workflows.py` | GREEN |
| 10-03-T1 | 10-03 | 3 | D-06 rolling main baseline, D-07 baseline cache key, D-08 cache-miss branch source, D-10 no release evidence baseline | `python3 -m unittest discover -s tests/bench -p "test_pr_benchmark_workflows.py" -v` | `tests/bench/test_pr_benchmark_workflows.py` | GREEN |
| 10-03-T1 | 10-03 | 3 | D-13 advisory workflow mode, D-14 discoverable control knob, D-19 step summary + best-effort sticky comment, D-21 artifact retention | `python3 -m unittest discover -s tests/bench -p "test_pr_benchmark_workflows.py" -v` | `tests/bench/test_pr_benchmark_workflows.py`; `CHANGELOG.md` grep through test | GREEN |
| 10-03-T1 | 10-03 | 3 | Workflow YAML validity and action linting | `actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml && yq eval '.' .github/workflows/pr-benchmark.yml >/dev/null && yq eval '.' .github/workflows/main-benchmark-baseline.yml >/dev/null` | `.github/workflows/pr-benchmark.yml`; `.github/workflows/main-benchmark-baseline.yml` | GREEN |
| 10-03-T2 | 10-03 | 3 | Live GitHub Actions behavior after merge | Manual checklist in `10-03-LIVE-VERIFICATION.md` | Actions UI, cache UI, PR comment, run summaries | MANUAL |

## Requirement Coverage

| Requirement | Coverage | Evidence |
|-------------|----------|----------|
| D-01 PR job budget | PARTIAL | Workflow timeout and orchestrator `600s` timeout are automated in `test_pr_benchmark_workflows.py` and `test_run_pr_benchmark.py`; hosted-runner wall-clock still manual. |
| D-02 benchmark families | COVERED | `test_script_locks_benchmark_subset_and_budget` asserts the locked Tier 1/2/3 regex. |
| D-03 fixture subset | COVERED | Orchestrator regex excludes `citm_catalog`; workflow tests assert no benchmark selection leaks into YAML. |
| D-04 `-count=5` | COVERED | `test_script_locks_benchmark_subset_and_budget`. |
| D-05 comparator subset | COVERED | `test_script_locks_benchmark_subset_and_budget` asserts only the three allowed comparator keys and excludes known out-of-scope comparators. |
| D-06 rolling main baseline | COVERED | `test_main_baseline_workflow_writes_cache_from_main_only`. |
| D-07 cache key strategy | COVERED | `test_main_baseline_workflow_writes_cache_from_main_only` and `test_pr_workflow_restores_baseline_and_stays_advisory`. |
| D-08 cache miss bypass | COVERED | Parser `test_no_baseline_bypass_mode`, orchestrator `test_no_baseline_skips_benchstat`, workflow cache-miss branch assertion. |
| D-09 rolling latest-main staleness model | PARTIAL | Workflow contract verifies rolling cache restore/save split; semantic acceptance of staleness remains design/manual judgment. |
| D-10 no release-scoped baseline | COVERED | `test_workflows_keep_benchmark_selection_inside_orchestrator`. |
| D-11 regression definition | COVERED | Parser threshold, p-value, faster-row, sentinel, and boundary tests. |
| D-12 per-row granularity | COVERED | Parser `test_per_row_granularity` and markdown row assertions. |
| D-13 advisory-only initial mode | COVERED | Parser advisory exit tests and workflow `REQUIRE_NO_REGRESSION: "false"` assertion. |
| D-14 future blocking flip | COVERED | Parser env-var test, workflow env assertion, CHANGELOG assertion. |
| D-15 markdown row detail | COVERED | Parser `test_markdown_renderer_includes_row_delta_pvalue`. |
| D-16 `paths-ignore` trigger style | COVERED | Workflow contract tests. |
| D-17 skip path globs | COVERED | Workflow contract tests assert the locked 8-entry ignore list in both workflows. |
| D-18 `.github/actions/**` skip risk | PARTIAL | Workflow contract tests assert the accepted skip is present; live/toolchain drift risk remains manual/operational. |
| D-19 step summary + sticky comment surface | PARTIAL | Workflow contract tests assert step summary, sticky comment, and `continue-on-error`; actual comment behavior requires live PR verification. |
| D-20 concurrency | PARTIAL | Workflow contract tests assert concurrency groups; cancellation behavior requires live rapid-push verification. |
| D-21 artifact upload | COVERED | Workflow contract tests assert `if: always()`, artifact upload action, paths, and retention. |

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| End-to-end PR job runs within the intended hosted-runner budget | D-01 | Requires GitHub-hosted runner timing and a real PR event | Follow `10-03-LIVE-VERIFICATION.md`; record the PR benchmark run URL and wall-clock duration. |
| First main-baseline cache seed works after merge to `main` | D-06, D-07 | Requires workflow files to exist on `main` and Actions cache side effects | Run `main benchmark baseline` via `workflow_dispatch`; record run URL, cache key, and baseline artifact. |
| Live cache-miss bypass renders expected advisory output | D-08 | Requires a real missing Actions cache entry | Delete or avoid `pr-bench-baseline-*`, rerun PR benchmark, and record summary/comment evidence. |
| Real hosted-runner regression row appears in the PR surface | D-11, D-12, D-15 | Requires an intentionally slowed PR and real benchmark noise | Open a draft PR with a controlled slowdown and confirm the exact row, delta, and p-value are visible. |
| Sticky comment posts or degrades harmlessly on fork PRs | D-19 | Requires GitHub token permission behavior | Verify same-repo comment update and fork-PR `continue-on-error` path. |
| Rapid-push cancellation cancels older PR benchmark runs | D-20 | Requires multiple live pushes to the same PR | Push two commits quickly and record the cancelled run and completed run URLs. |

## Validation Audit 2026-04-27

| Metric | Count |
|--------|-------|
| Gaps found | 2 |
| Resolved | 2 |
| Escalated to manual-only | 6 |

Resolved gaps:

- Added `tests/bench/test_pr_benchmark_workflows.py` to cover Plan 03 workflow contract invariants that were previously only listed as ad hoc grep/yq/actionlint criteria.
- Added an orchestrator script-contract assertion to `tests/bench/test_run_pr_benchmark.py` so D-02/D-03/D-04/D-05 are locked by an automated test, not just by the bash implementation.

Manual-only items are live GitHub Actions behaviors, not local code gaps. They remain tracked in `10-03-LIVE-VERIFICATION.md`.

## Validation Sign-Off

- [x] Nyquist config checked (`workflow.nyquist_validation: true` in `.planning/config.json`).
- [x] Input state detected: State A, existing `10-VALIDATION.md`.
- [x] PLAN/SUMMARY artifacts read and requirement map rebuilt.
- [x] Test infrastructure detected.
- [x] Gaps classified and filled where local automation can prove behavior.
- [x] `VALIDATION.md` updated with current coverage.
- [x] Generated/updated test files for missing automated checks.
- [ ] Live GitHub Actions manual-only checklist complete.
- [ ] `nyquist_compliant: true` set in frontmatter.

**Approval:** partial - local validation covered; live Actions verification remains.
