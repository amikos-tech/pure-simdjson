---
phase: 10-lightweight-pr-benchmark-regression-signal
verified: 2026-07-30T11:57:53Z
status: human_needed
score: 55/55 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 54/55
  gaps_closed:
    - "Parser reuses parse_benchmark_file and EvidenceError from check_benchmark_claims.py via sys.path.insert + import (no copy-paste of parser body)."
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Seed the main baseline from main with workflow_dispatch."
    expected: "The run is green, creates pr-bench-baseline-<sha>, and uploads baseline evidence containing head.bench.txt."
    why_human: "Hosted runner execution, Actions cache save, and artifacts cannot be exercised locally."
  - test: "Open or update a non-ignored PR after the baseline seed."
    expected: "pr benchmark restores a matched baseline, posts a step summary and best-effort sticky comment, and remains green in advisory mode."
    why_human: "Trigger filtering, cache restore, token behavior, and GitHub UI reporting run only in Actions."
  - test: "Delete pr-bench-baseline-* and run a non-ignored PR benchmark."
    expected: "The PR takes the advisory-bypass path, remains green, and uploads head.bench.txt, summary.json, and markdown.md."
    why_human: "This requires changing and observing the live Actions cache."
  - test: "Push two commits to one PR within ten seconds."
    expected: "The earlier pr benchmark run is cancelled and only the latest run updates the sticky comment."
    why_human: "GitHub concurrency scheduling is an external runtime behavior."
---

# Phase 10: Lightweight PR Benchmark Regression Signal Verification Report

**Phase Goal:** Promote a lightweight advisory `pull_request` benchmark signal for representative Tier 1/2/3 merge-candidate paths, while `main` refreshes a rolling cache baseline and release-grade evidence remains separate.
**Verified:** 2026-07-30T11:57:53Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | PR parser detects only material, significant `sec/op` regressions; bypass and advisory/blocking controls work. | ✓ VERIFIED | `check_pr_regression.py:19-225`; 25 focused parser contracts pass, including section filtering, malformed input, bypass, and `REQUIRE_NO_REGRESSION`. |
| 2 | Parser reuses both Phase 9 helpers required by Plan 10-01. | ✓ VERIFIED | `check_pr_regression.py:14,55-63,188-192` imports `EvidenceError` and `parse_benchmark_file`, calls the shared helper before the local benchstat parser, and rejects raw samples. The runtime-spy and raw-capture tests pass. |
| 3 | Orchestrator captures exactly the locked Tier 1/2/3 twitter/canada subset five times, with a 600-second benchmark cap, atomic output promotion, and baseline/no-baseline paths. | ✓ VERIFIED | `run_pr_benchmark.sh:4-146`; eight PATH-shadowed integration tests pass. |
| 4 | A non-ignored PR starts an advisory workflow with least privilege, cancellation, restore-only baseline cache, summary/comment/artifact reporting, and no release-evidence baseline. | ✓ VERIFIED (static) | `pr-benchmark.yml:1-109`; `actionlint`, `yq`, and eight workflow-contract tests pass. Live execution remains human verification. |
| 5 | `main` (and manual dispatch) captures head evidence and saves it under the exact cache path/key that PRs restore. | ✓ VERIFIED (static) | `main-benchmark-baseline.yml:1-89`; producer and consumer use `baseline.bench.txt` and the `pr-bench-baseline-` namespace. Live cache behavior remains human verification. |
| 6 | Release-grade evidence remains a separate capture path. | ✓ VERIFIED | Neither Phase 10 workflow references versioned `testdata/benchmark-results`; the PR cache baseline is independently produced from `main`. |

**Score:** 55/55 must-haves verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `scripts/bench/check_pr_regression.py` | Section-aware parser and shared raw-evidence guard | ✓ VERIFIED | Substantive 225-line CLI; shared parser is invoked before `parse_benchstat_for_regressions`; no Phase 9 parser body is duplicated. |
| `tests/bench/test_check_pr_regression.py` + fixtures | Parser contract corpus | ✓ VERIFIED | 25 tests cover valid benchstat behavior, helper invocation, and Phase 9 raw capture rejection. |
| `scripts/bench/run_pr_benchmark.sh` | Locked PR runner | ✓ VERIFIED | 146 lines; strict shell mode, staging, real `run_benchstat.sh`/parser calls, and tested baseline/bypass branches. |
| `tests/bench/test_run_pr_benchmark.py` | Orchestrator integration tests | ✓ VERIFIED | Eight PATH-shadowed tests cover baseline, bypass, cleanup, and replacement behavior. |
| `.github/workflows/pr-benchmark.yml` | Advisory PR workflow | ✓ VERIFIED (static) | Parsed/linted and contract-tested; live Actions behavior is deferred below. |
| `.github/workflows/main-benchmark-baseline.yml` | Main-only cache producer | ✓ VERIFIED (static) | Parsed/linted and contract-tested; cache save/restore requires live validation. |
| `CHANGELOG.md` | Discoverable advisory/flip note | ✓ VERIFIED | Unreleased entry and both implementation locations expose `REQUIRE_NO_REGRESSION`. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| PR workflow | `run_pr_benchmark.sh` | cache-matched-key selects `--baseline` or `--no-baseline` | ✓ WIRED | `pr-benchmark.yml:78-87`; workflow-contract tests exercise both command shapes. |
| Main workflow | `run_pr_benchmark.sh` | `--no-baseline` capture | ✓ WIRED | `main-benchmark-baseline.yml:66-69`; head evidence is then copied to cache path. |
| Orchestrator | `run_benchstat.sh` | staged baseline and head paths | ✓ WIRED | `run_pr_benchmark.sh:132-135`; baseline integration test passes. |
| Orchestrator | PR parser | summary/markdown CLI arguments | ✓ WIRED | `run_pr_benchmark.sh:123-141`; both orchestrator modes pass. |
| Main workflow | cache save | canonical `baseline.bench.txt` | ✓ WIRED | `main-benchmark-baseline.yml:71-80` matches the PR restore path at `pr-benchmark.yml:65-75`. |
| PR parser | Phase 9 helper module | imported shared guard before section-aware parse | ✓ WIRED | `check_pr_regression.py:14,55-63,188-192`; `test_real_phase9_benchstat_uses_shared_raw_evidence_guard` proves the call; raw Phase 9 evidence exits 1. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| PR parser | `flagged_rows` / markdown | benchstat text after shared raw-evidence guard | Real Phase 9 benchstat fixture is accepted; 25 parser tests cover positive, negative, malformed, and wrong-evidence flows. | ✓ FLOWING |
| Orchestrator | `head.bench.txt` | locked `go test` output | PATH-shadowed end-to-end tests assert promotion and parser inputs. | ✓ FLOWING locally |
| PR workflow | restored `baseline.bench.txt` | `actions/cache/restore` output | Static producer/consumer wiring is correct; hosted cache restore remains unexecuted. | ? HUMAN |
| Main workflow | cached baseline | copied `head.bench.txt` | Static cache-save wiring is correct; hosted cache save remains unexecuted. | ? HUMAN |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Parser contracts and shared-helper guard | `python3 -m unittest discover -s tests/bench -p 'test_check_pr_regression.py' -v` | 25 passed | ✓ PASS |
| Orchestrator contracts | `python3 -m unittest discover -s tests/bench -p 'test_run_pr_benchmark.py' -v` | 8 passed | ✓ PASS |
| Workflow contracts | `python3 -m unittest discover -s tests/bench -p 'test_pr_benchmark_workflows.py' -v` | 8 passed | ✓ PASS |
| Full benchmark test suite | `python3 -m unittest discover -s tests/bench -v` | 71 passed | ✓ PASS |
| Python and shell syntax | `python3 -m py_compile scripts/bench/check_pr_regression.py && bash -n scripts/bench/run_pr_benchmark.sh` | exit 0 | ✓ PASS |
| Workflow lint and YAML parsing | `actionlint ... && yq eval '.' ...` | exit 0 | ✓ PASS |
| Raw Phase 9 capture supplied as benchstat | `check_pr_regression.py --benchstat-output testdata/.../phase9.bench.txt ...` | exit 1; `wrong benchmark evidence ... expected benchstat output`; no summary emitted | ✓ PASS |

### Probe Execution

Step 7c: SKIPPED — no Phase 10 probe scripts were declared or discovered.

### Requirements Coverage

`REQUIREMENTS.md` contains no Phase 10 requirement IDs because the phase was promoted from backlog with requirements marked TBD. The D-01–D-21 planning contracts are supported by the artifacts, links, tests, and spot-checks above. The former Plan 10-01 helper-reuse contract is now satisfied; no later-phase deferral applies.

### Anti-Patterns Found

No blocker or warning anti-patterns were found in Phase 10 implementation files. No unreferenced `TBD`, `FIXME`, or `XXX` markers were found. The `Tier3SelectivePlaceholder` identifier is a benchmark name, not an implementation placeholder.

### Human Verification Required

### 1. Seed and verify the main baseline

**Test:** After merge, run **main benchmark baseline** with `workflow_dispatch` from `main` and wait for completion.
**Expected:** Green run; a `pr-bench-baseline-<sha>` cache entry exists; the baseline artifact contains `head.bench.txt`.
**Why human:** Hosted runner execution, Actions cache save, and artifacts cannot be proven locally.

### 2. Verify a cache-hit PR run

**Test:** Open or update a non-ignored PR after seeding the baseline.
**Expected:** `pr benchmark` restores a matched baseline, emits a step summary and best-effort sticky comment, and is green in advisory mode.
**Why human:** Trigger filtering, cache restore, PR token behavior, and GitHub UI surfaces run only in Actions.

### 3. Verify the cache-miss bypass

**Test:** Delete `pr-bench-baseline-*` and run a non-ignored PR benchmark.
**Expected:** It reports advisory bypass, exits green, and uploads `head.bench.txt`, `summary.json`, and `markdown.md`.
**Why human:** Requires changing and observing the live Actions cache.

### 4. Verify PR cancellation

**Test:** Push two commits to one PR within ten seconds.
**Expected:** The earlier run is cancelled; only the latest run updates the sticky comment.
**Why human:** GitHub concurrency scheduling is external runtime behavior.

---

_Verified: 2026-07-30T11:57:53Z_
_Verifier: the agent (gsd-verifier)_
