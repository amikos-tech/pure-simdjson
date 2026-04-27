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
| **Framework** | unittest (Python stdlib) — Python contract tests + go test (Go bench compile-check) + actionlint (workflow YAML). Mirrors the analog `tests/bench/test_check_benchmark_claims.py` (per CLAUDE.md "radically simple" — no new framework just for one test module). The repo does NOT carry `tests/__init__.py` or `tests/bench/__init__.py`, so dotted-module invocations like `python3 -m unittest tests.bench.test_X` do NOT work. Use `python3 -m unittest discover -s tests/bench -p "<file>.py"` (matches Phase 9 working convention) or invoke the test module directly via `python3 tests/bench/<file>.py` (each new test module ships a `if __name__ == "__main__": unittest.main()` shim mirroring the analog `tests/bench/test_check_benchmark_claims.py`). |
| **Config file** | None (unittest discovers tests via `python3 -m unittest discover -s tests/bench -p "<file>.py"`; no conftest.py required). |
| **Quick run command** | `python3 -m unittest discover -s tests/bench -p "test_check_pr_regression.py" -v` |
| **Full suite command** | `python3 -m unittest discover -s tests/bench -v && actionlint .github/workflows/pr-benchmark.yml .github/workflows/main-benchmark-baseline.yml && go test -run=^$ -bench=^$ ./...` |
| **Estimated runtime** | ~5 seconds (unittest only); ~30 seconds full suite |

---

## Sampling Rate

- **After every task commit:** Run `python3 -m unittest discover -s tests/bench -p "test_check_pr_regression.py" -v`
- **After every plan wave:** Run full suite (`python3 -m unittest discover -s tests/bench -v && actionlint <workflows>`)
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

- [ ] `tests/bench/test_check_pr_regression.py` — fixture-driven `unittest` contract tests for the bidirectional regression parser, mirroring `tests/bench/test_check_benchmark_claims.py` style (including the `if __name__ == "__main__": unittest.main()` shim at the bottom so direct-script invocation works)
- [ ] `tests/bench/fixtures/pr-regression/` — synthetic + real benchstat fixtures (regression flagged, non-significant `~`, missing baseline, malformed input, exact-5%-boundary, p=0.05 boundary, multi-row all-significant) — at minimum one fixture must be sourced from real `testdata/benchmark-results/v0.1.2/phase9.bench.txt`-derived output (per Phase 9 LEARNINGS: synthetic-only fixtures pass while production breaks)
- [ ] `actionlint` available locally for the Plan 03 executor (`go install github.com/rhysd/actionlint/cmd/actionlint@latest`) — validates new workflow YAML

*Existing `tests/bench/` infrastructure already uses `unittest` (see `test_check_benchmark_claims.py`). Phase 10 adds two new test modules in the same style — no `conftest.py` and no pytest dependency are introduced. The repo intentionally has no `tests/__init__.py` or `tests/bench/__init__.py`; tests are run via `unittest discover` or by invoking the test file directly. Phase 10 MUST NOT add those `__init__.py` files (would change Phase 9's existing test discovery semantics — phase boundary).*

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
