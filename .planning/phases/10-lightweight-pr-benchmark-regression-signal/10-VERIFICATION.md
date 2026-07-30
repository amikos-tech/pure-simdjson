---
phase: 10-lightweight-pr-benchmark-regression-signal
verified: 2026-07-30T11:42:36Z
status: gaps_found
score: 54/55 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Parser reuses parse_benchmark_file and EvidenceError from check_benchmark_claims.py via sys.path.insert + import (no copy-paste of parser body)."
    status: failed
    reason: "The PR parser imports and uses EvidenceError, but does not import or call parse_benchmark_file. Its benchstat parsing is locally implemented instead."
    artifacts:
      - path: "scripts/bench/check_pr_regression.py"
        issue: "Line 15 imports only EvidenceError; repository search finds parse_benchmark_file only in check_benchmark_claims.py."
    missing:
      - "Import and use parse_benchmark_file as contracted, or obtain an explicit override documenting why benchstat parsing must remain independent."
---

# Phase 10: Lightweight PR Benchmark Regression Signal Verification Report

**Phase Goal:** Promote a lightweight advisory `pull_request` benchmark signal for representative Tier 1/2/3 merge-candidate paths, while `main` refreshes a rolling cache baseline and release-grade evidence remains separate.
**Verified:** 2026-07-30T11:42:36Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | PR parser detects only material, significant `sec/op` regressions; bypass and advisory/blocking controls work. | ✓ VERIFIED | `check_pr_regression.py:19-210`; 23 parser-contract tests pass, including metric filtering, malformed input, bypass, and `REQUIRE_NO_REGRESSION`. |
| 2 | Parser reuses both Phase 9 helpers required by Plan 10-01. | ✗ FAILED | It imports `EvidenceError` at line 15 but has no `parse_benchmark_file` import/call. |
| 3 | Orchestrator captures exactly the locked Tier 1/2/3 twitter/canada subset five times, with a 600-second benchmark cap, atomic output promotion, and baseline/no-baseline paths. | ✓ VERIFIED | `run_pr_benchmark.sh:1-146`; 8 PATH-shadowed integration tests pass. |
| 4 | A non-ignored PR starts an advisory workflow with least privilege, cancellation, restore-only baseline cache, summary/comment/artifact reporting, and no release-evidence baseline. | ✓ VERIFIED (static) | `pr-benchmark.yml:1-109`, `actionlint`, `yq`, and 8 workflow-contract tests pass. |
| 5 | `main` (and manual dispatch) captures head evidence and saves it under the exact cache path/key that PRs restore. | ✓ VERIFIED (static) | `main-benchmark-baseline.yml:1-89`; cache producer/consumer paths are both `baseline.bench.txt`. |
| 6 | Release-grade evidence remains a separate capture path. | ✓ VERIFIED | PR/main workflows do not reference `testdata/benchmark-results/v*`; `benchmark-capture.yml` independently invokes `capture_release_snapshot.sh`. |

**Score:** 54/55 must-haves verified

### D-01 / D-06 / D-07 / D-09 / D-10 / D-13 / D-14 / D-16–D-21 Contract Coverage

| Contract | Status | Codebase evidence |
| --- | --- | --- |
| D-01 budget | ✓ VERIFIED | PR job `timeout-minutes: 15`; orchestrator passes `-timeout 600s` and `-count=5` exactly once. |
| D-06 PR cache safety | ✓ VERIFIED | PR uses `actions/cache/restore` with a forced-miss key and prefix restore; no `actions/cache/save` appears in the file. |
| D-07 main baseline producer | ✓ VERIFIED | Main-only workflow copies head output to `baseline.bench.txt` then saves `pr-bench-baseline-${{ github.sha }}`. |
| D-09 merge-candidate signal | ✓ VERIFIED (static) | `pull_request` workflow checks out the PR ref and calls the orchestrator after native build/cache setup. |
| D-10 release-evidence separation | ✓ VERIFIED | Neither Phase 10 workflow reads a versioned public benchmark-results baseline. |
| D-13 advisory operation | ✓ VERIFIED | Workflow sets `REQUIRE_NO_REGRESSION: "false"`; parser returns zero on detected regressions unless the control is true. |
| D-14 future blocking flip | ✓ VERIFIED | The env knob, explanatory comment, parser control surface, and Unreleased changelog note all exist. |
| D-16 event safety | ✓ VERIFIED | Uses `pull_request`, never `pull_request_target`; permissions are only `contents: read` and `pull-requests: write`. |
| D-17 / D-18 path filtering | ✓ VERIFIED | Both workflows have the same eight allowed `paths-ignore` entries, including `.github/actions/**`. |
| D-19 reporting/fork degradation | ✓ VERIFIED (static) | Step summary is always appended; sticky comment has stable header and `continue-on-error: true`. |
| D-20 concurrency | ✓ VERIFIED | PR group is per PR and cancels in progress; main group is stable and does not cancel itself. |
| D-21 diagnostics | ✓ VERIFIED | PR uploads `pr-bench-summary/` with `if: always()` and 14-day retention. |

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `scripts/bench/check_pr_regression.py` | Section-aware parser and control surface | ⚠️ PARTIAL | Substantive 214-line implementation and tested data flow; fails the explicit dual-helper reuse contract. |
| `tests/bench/test_check_pr_regression.py` + fixtures | Parser contract corpus | ✓ VERIFIED | 23 tests pass, including real Phase 9-format fixture. |
| `scripts/bench/run_pr_benchmark.sh` | Locked PR runner | ✓ VERIFIED | 146 lines, strict shell mode, atomic staging, actual parser/benchstat calls. |
| `tests/bench/test_run_pr_benchmark.py` | Orchestrator integration tests | ✓ VERIFIED | 8 tests exercise baseline, bypass, failure cleanup, and replacement behavior. |
| `.github/workflows/pr-benchmark.yml` | Advisory PR workflow | ✓ VERIFIED (static) | Parsed and linted; invokes orchestrator in both cache branches. |
| `.github/workflows/main-benchmark-baseline.yml` | Main-only cache producer | ✓ VERIFIED (static) | Parsed and linted; saves canonical baseline cache. |
| `CHANGELOG.md` | Discoverable advisory/flip note | ✓ VERIFIED | Unreleased Added bullet names advisory mode and `REQUIRE_NO_REGRESSION`. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| PR workflow | `run_pr_benchmark.sh` | cache-matched-key selects `--baseline` or `--no-baseline` | ✓ WIRED | Lines 78-88 pass the correct mutually exclusive argument. |
| Main workflow | `run_pr_benchmark.sh` | `--no-baseline` capture | ✓ WIRED | Line 70 captures only head evidence before cache save. |
| Orchestrator | `run_benchstat.sh` | `--old` staged baseline / `--new` head | ✓ WIRED | Lines 132-135; covered by baseline integration test. |
| Orchestrator | PR parser | summary/markdown CLI arguments | ✓ WIRED | Lines 123-141; covered by both orchestrator modes. |
| Main workflow | cache save | canonical `baseline.bench.txt` | ✓ WIRED | Lines 72-80 match PR restore path at lines 68-76. |
| PR parser | Phase 9 helper module | import of required helpers | ✗ NOT WIRED | Only `EvidenceError` is imported; `parse_benchmark_file` is absent. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| PR workflow | `baseline.bench.txt` / `NO_BASELINE` | `actions/cache/restore` output | Requires live Actions run | ? HUMAN |
| Orchestrator | `head.bench.txt` | locked `go test` output | Yes in PATH-shadowed integration tests; live runner still required | ✓ FLOWING locally |
| Parser | `flagged_rows` / markdown | benchstat text | Yes: fixture-driven subprocess tests cover positive, negative, malformed, and bypass inputs | ✓ FLOWING |
| Main workflow | cached baseline | copied `head.bench.txt` | Requires live Actions cache save/restore | ? HUMAN |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Workflow syntax/lint | `actionlint` on both workflow files | exit 0 | ✓ PASS |
| YAML parsing | `yq eval '.'` on both workflow files | exit 0 | ✓ PASS |
| Parser contracts | `python3 -m unittest discover -s tests/bench -p 'test_check_pr_regression.py' -v` | 23 passed | ✓ PASS |
| Orchestrator contracts | `python3 -m unittest discover -s tests/bench -p 'test_run_pr_benchmark.py' -v` | 8 passed | ✓ PASS |
| Workflow contracts | `python3 -m unittest discover -s tests/bench -p 'test_pr_benchmark_workflows.py' -v` | 8 passed | ✓ PASS |
| Shell syntax | `bash -n scripts/bench/run_pr_benchmark.sh` | exit 0 | ✓ PASS |

### Probe Execution

Step 7c: SKIPPED — no Phase 10 probe scripts were declared or discovered.

### Requirements Coverage

`REQUIREMENTS.md` has no Phase 10 requirement IDs because this phase was promoted from backlog with requirements marked TBD. The concrete D-01/D-06/D-07/D-09/D-10/D-13/D-14/D-16–D-21 planning contracts are covered above; all are statically satisfied. The failed helper-reuse contract originates in Plan 10-01 and is not deferred by any later roadmap phase.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `scripts/bench/check_pr_regression.py` | 15 | Required shared parser helper absent | 🛑 Blocker | Breaks the explicit Phase 10 reuse/no-copy-paste must-have. |

No unreferenced `TBD`, `FIXME`, or `XXX` markers were found in Phase 10 implementation files. The `Tier3SelectivePlaceholder` names are benchmark identifiers, not stub markers.

### Human Verification Required

### 1. Seed and verify the main baseline

**Test:** After merge, run `main benchmark baseline` with `workflow_dispatch` from `main` and wait for completion.
**Expected:** Green run; `pr-bench-baseline-<sha>` exists; `baseline-evidence-*` contains `head.bench.txt`.
**Why human:** Actions cache behavior and hosted runner execution cannot be proven locally.

### 2. Verify a cache-hit PR run

**Test:** Open a non-ignored no-op PR after the seed.
**Expected:** `pr benchmark` runs within ten minutes, restore output has a matched key, summary and sticky comment appear, and the advisory job is green.
**Why human:** Trigger filtering, PR token behavior, cache restore, and UI surfaces run only on GitHub Actions.

### 3. Verify the cache-miss bypass

**Test:** Delete the `pr-bench-baseline-*` cache and push/open another non-ignored PR.
**Expected:** Empty matched key, advisory-bypass summary/comment, and green run.
**Why human:** Requires changing the live Actions cache and observing the live workflow.

### 4. Verify PR cancellation

**Test:** Push two commits to one PR within ten seconds.
**Expected:** Earlier `pr benchmark` run is cancelled; only the latest run updates the sticky comment.
**Why human:** GitHub concurrency scheduling is external runtime behavior.

### Gaps Summary

One non-deferred blocker remains: Plan 10-01 explicitly requires importing and reusing both `parse_benchmark_file` and `EvidenceError` from the Phase 9 helper module. The delivered parser imports only `EvidenceError` and independently parses benchstat. This is observable absence, not uncertainty, and no override exists.

The Phase 10 GitHub Actions behavior also still needs the four live checks above after merge. Those checks do not change the blocker classification, but they must be completed before claiming the workflow operates on hosted runners.

---

_Verified: 2026-07-30T11:42:36Z_
_Verifier: the agent (gsd-verifier)_
