---
phase: 10
slug: lightweight-pr-benchmark-regression-signal
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-27
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | pytest 7.x (Python contract tests) + go test (Go bench compile-check) + actionlint (workflow YAML) |
| **Config file** | `tests/bench/conftest.py` (extended for new fixtures) |
| **Quick run command** | `pytest tests/bench/test_check_pr_regression.py -x` |
| **Full suite command** | `pytest tests/bench/ && actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml && go test -run=^$ -bench=^$ ./...` |
| **Estimated runtime** | ~30 seconds (pytest only); ~60 seconds full suite |

---

## Sampling Rate

- **After every task commit:** Run `pytest tests/bench/test_check_pr_regression.py -x`
- **After every plan wave:** Run full suite (`pytest tests/bench/ && actionlint <workflows>`)
- **Before `/gsd-verify-work`:** Full suite must be green + manual workflow_dispatch dry-run from a feature branch
- **Max feedback latency:** ≤30 seconds (per-task quick run)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | D-01..D-21 (CONTEXT.md) | TBD | TBD | TBD | TBD | TBD | ⬜ pending |

*Filled by gsd-planner during PLAN.md generation. Plan-checker enforces every task has either an automated verify command or a Wave 0 dependency.*

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/bench/test_check_pr_regression.py` — fixture-driven contract tests for the bidirectional regression parser, mirroring `tests/bench/test_check_benchmark_claims.py` style
- [ ] `tests/bench/fixtures/pr-regression/` — synthetic + real benchstat fixtures (regression flagged, non-significant `~`, missing baseline, malformed input, exact-5%-boundary, p=0.05 boundary, multi-row all-significant) — at minimum one fixture must be sourced from real `testdata/benchmark-results/v0.1.2/phase9.bench.txt`-derived output (per Phase 9 LEARNINGS: synthetic-only fixtures pass while production breaks)
- [ ] `tests/bench/conftest.py` — extend with shared fixtures for benchstat output parsing
- [ ] `actionlint` available in CI (already standard; add to local dev tooling if missing) — validates new workflow YAML

*Existing pytest infrastructure under `tests/bench/` covers framework setup. New fixtures and one new test file are the only additions.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| End-to-end PR job runs ≤10 min on `ubuntu-latest` | D-01 | Cannot be tested deterministically in unit tests; depends on hosted-runner timing | Open a draft PR with a small Go change touching benchmark-relevant code; observe workflow run duration in Actions UI; record total wall-clock time. |
| Sticky comment renders correctly on same-repo PR | D-19 | Requires real GitHub PR with sticky-pull-request-comment action having `pull-requests: write` token | Open a draft PR; verify single sticky comment appears on first run, and is **edited** (not appended) on subsequent runs. |
| Sticky comment fails gracefully on fork PR | D-19 | Requires fork-PR token-permission denial path | Have a collaborator open a fork PR; verify workflow logs show 403 / "skipping comment" notice; verify `$GITHUB_STEP_SUMMARY` still renders; verify workflow exits 0. |
| Cache miss handled gracefully (first run, post-eviction) | D-08 | Requires either first-ever run OR forced cache eviction | Either delete the `pr-bench-baseline-*` cache via `gh actions-cache delete` or open the first PR after the workflow ships; verify "no baseline available — advisory bypass" notice in step summary + comment; verify workflow exits 0; verify raw evidence uploaded as artifact. |
| Concurrency cancellation on rapid pushes | D-20 | Requires two pushes to same PR within seconds | Push commit A to a draft PR; immediately push commit B; verify run for commit A is cancelled and only commit B's run completes. |
| Regression flag triggers on real ≥5% slowdown | D-11, D-12 | Requires intentionally regressed code change | In a draft PR, introduce a 10% slowdown to one Tier 1/2/3 benchmark function; verify the row appears in the sticky comment with correct Δ% and p-value; verify other rows are not flagged. |
| Advisory exit-0 holds on regression | D-13 | Requires real regression detection | Same draft PR as above; verify the workflow run shows green even when regression rows are listed. |
| Future blocking-flip control surface is named and discoverable | D-14 | The control knob's existence and label is human-judgment-grade | Inspect the new workflow file; verify a clearly named env var, workflow input, or workflow-file constant exists with a comment indicating it is the future blocking-flip; verify CHANGELOG/Phase 10 SUMMARY references the knob name. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies (planner enforces; plan-checker re-verifies)
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (parser test fixtures, real-benchstat fixture, workflow YAML lint)
- [ ] No watch-mode flags
- [ ] Feedback latency <30s for parser tests; manual verifications gated to verify-phase
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
