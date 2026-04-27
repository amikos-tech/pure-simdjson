# Phase 10: Lightweight PR benchmark regression signal - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-27
**Phase:** 10-lightweight-pr-benchmark-regression-signal
**Areas discussed:** Subset + runtime budget, Baseline source, Threshold + advisory/blocking, Skip conditions + PR surface

---

## Subset + runtime budget

### Q1: Total wall-clock budget for the PR benchmark job

| Option | Description | Selected |
|--------|-------------|----------|
| ≤ 10 min total | Tightest cheap signal. Forces a real subset (likely Tier 1+2 only, twitter+canada, count=5). Fast PR feedback, low runner cost. Misses small regressions; loud changes still surface. | ✓ |
| ≤ 15 min total | Allows full Tier 1/2/3 across all 3 fixtures at count=5, or smaller fixtures with diagnostics. Slightly better noise floor. | |
| ≤ 20 min total | Allows Tier 1/2/3 + Tier 1 diagnostics across 3 fixtures at count=5, or all families at reduced fixtures. Approaches upper edge of "cheap." | |
| ≤ 30 min total | Effectively cheap end of release-grade. Hard to justify as "lightweight." | |

**User's choice:** ≤ 10 min total
**Notes:** Budget is binding; Phase 10 trims subset/count/fixture to fit, rather than relaxing the budget.

### Q2: Benchmark subset under the ≤10 min budget

| Option | Description | Selected |
|--------|-------------|----------|
| T1+T2+T3, twitter+canada, count=5 | Drops citm_catalog.json (largest). Skips Tier 1 diagnostics + cold/warm. Keeps all three tiers (matches roadmap goal) on two fixtures. count=5 keeps benchstat trustworthy. | ✓ |
| T1+T2 only, all 3 fixtures, count=5 | Drops Tier 3 placeholder entirely. Cleaner story but diverges from "Tier 1/2/3" wording in roadmap. | |
| T1+T2+T3 all 3 fixtures, count=3 | Full coverage but count=3 puts benchstat at edge of significance — false positives likely on noisy hosted runners. | |
| T1+T2+T3, twitter.json only, count=8 | Single fixture with high count. Strong significance for that fixture but misses fixture-specific regressions. | |

**User's choice:** T1+T2+T3, twitter+canada, count=5
**Notes:** Roadmap explicitly says "Tier 1/Tier 2/Tier 3" — Tier 3 stays in despite being a v0.2 selective-path placeholder.

### Q3: Comparator filter

| Option | Description | Selected |
|--------|-------------|----------|
| Filter to pure-simdjson + encoding/json only | Use a narrower -bench regex. Keeps stdlib as cross-target sanity baseline. Comparator competitiveness is a release/headline question, not a regression question. | ✓ |
| Run all comparators | Same harness as Phase 9. Simpler config, but ~2-3x more sub-benches and the data isn't used by the regression gate. | |
| pure-simdjson only | Tightest. Catches pure-simdjson self-regressions only. Loses encoding/json sanity check. | |

**User's choice:** Filter to pure-simdjson + encoding/json only
**Notes:** encoding/json kept as the cross-runner-noise sanity baseline.

---

## Baseline source

### Q4: What baseline does the PR job compare against?

| Option | Description | Selected |
|--------|-------------|----------|
| Rolling main baseline via actions/cache | Push-to-main workflow captures the same subset, writes to actions/cache. PR job restores the most recent cache entry, runs head, runs benchstat. Fresh per merge, single-runner-class consistency, no third-party action. Cache miss falls back gracefully to advisory-only. | ✓ |
| Same-run base+head capture | PR job captures PR base then PR head on the same runner, then benchstats. Tightest noise control. But doubles wall-clock; under 10 min forces count=3 (weakens significance). | |
| Static committed snapshot | Compare PR runs to latest testdata/benchmark-results/v0.1.x/. Cheapest. Cross-runner offsets degrade signal; can't detect cumulative slowdown. | |
| benchmark-action/github-action-benchmark | Third-party action stores per-commit history on gh-pages. Phase 9 D-11 already considered this auxiliary, not source of truth. Adds gh-pages dependency. | |

**User's choice:** Rolling main baseline via actions/cache
**Notes:** Cache miss = graceful advisory-only degradation, no false blockers. Push-to-main workflow keys baseline on main HEAD SHA with restore-keys prefix matching.

---

## Threshold + advisory/blocking

### Q5: How should a regression be defined?

| Option | Description | Selected |
|--------|-------------|----------|
| benchstat p<0.05 AND δ > 5% per row | Flagged when benchstat marks it significant AND median is ≥5% slower. Matches noise reality on hosted runners. Per-row granularity catches fixture-specific cliffs. | ✓ |
| benchstat p<0.05 AND δ > 3% per row | Tighter. Catches smaller regressions but more false positives expected. Could be the v2 setting once noise profile is trusted. | |
| Tier geomean δ > 5% (with p<0.05) | Aggregate per-tier instead of per-row. Smoother, fewer false positives. Hides fixture-specific regressions. | |
| Hard δ > 10% per row, no p-value gate | Simplest. No statistical filtering. Hosted-runner noise routinely swings 8–12%; expect frequent false alarms. | |

**User's choice:** benchstat p<0.05 AND δ > 5% per row
**Notes:** Both conditions required — p-value filters runner noise, percentage gate filters tiny-but-not-actionable deltas.

### Q6: Advisory or blocking on regression?

| Option | Description | Selected |
|--------|-------------|----------|
| Advisory-only initially, manual flip later | Workflow always exits 0; regressions surface as PR comment + step summary annotation only. Required-status check stays off. After ~1 month / N PRs, separately decide to flip via a config toggle. | ✓ |
| Blocking from day one | Regression → nonzero exit → required status fails → merge blocked. Strongest signal, but any false-positive run blocks an unrelated PR. | |
| Tiered: comment <X%, block ≥Y% | e.g., 5–10% advisory, ≥10% blocking. More nuanced, more workflow logic. Defer until real-world data exists. | |
| Advisory + 'opt-in to block' label | Advisory by default. PR author/maintainer can add bench-strict label to flip per-PR. Useful for perf-sensitive PRs. | |

**User's choice:** Advisory-only initially, manual flip later
**Notes:** "Contributors should not learn to ignore the bot." The blocking flip is explicitly a separate, later decision — not bundled into Phase 10.

---

## Skip conditions + PR surface

### Q7: Skip paths (multi-select)

| Option | Description | Selected |
|--------|-------------|----------|
| Docs-only (*.md, docs/**, LICENSE, NOTICE) | README, methodology docs, changelog edits don't change runtime. Use paths-ignore so mixed PRs still run. | ✓ |
| Planning-only (.planning/**) | GSD artifacts — plans, contexts, learnings — don't change runtime. Heavy churn area. | ✓ |
| Other workflows + actions (.github/workflows/** except this one, .github/actions/**) | Edits to release.yml, claude.yml, etc. Risk: composite-action edits (setup-rust) could affect builds and we'd skip. | ✓ |
| Benchmark evidence (testdata/benchmark-results/**) | Snapshot updates from Phase 9 capture runs. No runtime impact. | ✓ |

**User's choice:** All four (Docs-only, Planning-only, Other workflows + actions, Benchmark evidence)
**Notes:** Risk of skipping `.github/actions/**` (shared composite-action edits affecting builds) explicitly accepted for v0.1; revisit if a real regression is ever masked.

### Q8: How should the regression result be surfaced on the PR?

| Option | Description | Selected |
|--------|-------------|----------|
| Step summary + sticky PR comment | Step summary always works (incl. forks). Sticky comment via marocchino/sticky-pull-request-comment for review-time visibility. Comment falls back gracefully on fork PRs. | ✓ |
| Step summary only | No third-party action. Lives in Checks tab forever. Simplest, most secure, less in-your-face. | |
| Sticky PR comment only | Most visible. Requires third-party action and write permissions; fork PRs need extra handling. | |
| Uploaded artifact + step summary | Step summary for headline; raw .bench.txt + benchstat.txt + summary.json as artifact for deep dives. | |

**User's choice:** Step summary + sticky PR comment
**Notes:** Step summary handles fork PRs and Checks-tab visibility; sticky comment handles review-time visibility for same-repo PRs. Artifact upload also runs (Phase 9 "upload evidence even on failure" pattern); not its own surface, just diagnostic backup.

---

## Claude's Discretion

- Exact workflow filenames (e.g., `pr-benchmark.yml`, `main-benchmark-baseline.yml`)
- Exact `summary.json` shape (must include flagged rows, target metadata, threshold values, regression boolean)
- Exact sticky PR comment markdown layout
- Name of the future "advisory → blocking" control knob
- Whether to extend `scripts/bench/check_benchmark_claims.py` or introduce a new `scripts/bench/check_pr_regression.py` (Phase 9's claim gate has different semantics; new script is acceptable as long as it reuses parser helpers)
- `actions/cache@v4` vs split `actions/cache/restore@v4` + `actions/cache/save@v4` — finer-grained miss handling
- Whether to pin `benchstat` version (planner should pin if Phase 9 pinned)

## Deferred Ideas

- Flipping the PR regression check to required/blocking — explicitly out of Phase 10
- Tier 1 diagnostics + cold/warm coverage on PR — stays in Phase 9 release-grade
- `citm_catalog.json` fixture on PR — out of budget; stays in Phase 9
- Cross-comparator regression detection (sonic, minio, goccy) — release-grade concern
- Multi-platform PR benchmarking (darwin, windows, linux/arm64) — release-grade concern
- Same-run base + head capture — rejected for budget; may be reconsidered if rolling-main drift dominates noise
- `benchmark-action/github-action-benchmark` for gh-pages history — superseded by `actions/cache` rolling baseline
- Push-to-main "did this merge regress?" check (regression-on-main detector) — distinct problem
- `bench-strict` PR label for per-PR opt-in blocking — could be added later
