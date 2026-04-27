# Phase 10: Lightweight PR benchmark regression signal - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Add a cheap `pull_request` benchmark workflow that exercises representative Tier 1/Tier 2/Tier 3 paths and reports a regression signal on the merge candidate within a tight time budget. Every non-trivial PR gets a modest but durable performance check; the heavier release-grade evidence capture (Phase 9 `benchmark-capture.yml`) remains the public-claim and headline gate.

This phase does not change benchmark code, fixtures, comparators, or the public claim-gating logic. It does not publish public benchmark wording or alter `testdata/benchmark-results/v0.1.x/` snapshots. It does not flip the regression check to a required/blocking status — that is a separate, later decision once the noise profile is observed.

</domain>

<decisions>
## Implementation Decisions

### Benchmark Subset and Runtime Budget
- **D-01:** Total wall-clock budget for the PR job (checkout + setup + native build + bench + benchstat + comment) is `≤ 10 minutes` on `ubuntu-latest`. The job is "cheap" by construction; budget overruns are a planning bug, not something to relax.
- **D-02:** Benchmark families run on PR: `BenchmarkTier1FullParse_*`, `BenchmarkTier2Typed_*`, `BenchmarkTier3SelectivePlaceholder_*`. Tier 1 diagnostics and cold/warm families are explicitly out — they belong to Phase 9 release-grade capture.
- **D-03:** Fixtures: `twitter.json` and `canada.json` only. `citm_catalog.json` (largest, ~1.7MB) is dropped to fit the budget; it stays in Phase 9 release-grade capture.
- **D-04:** `-count=5`. This is the minimum count that keeps benchstat's significance test trustworthy on hosted-runner noise; lower counts produce false positives.
- **D-05:** Comparator filter: only `pure-simdjson` and `encoding-json-any` / `encoding-json-struct` rows execute. The `-bench` regex must omit `minio`, `sonic`, `goccy` rows. Comparator competitiveness is a release/headline question (Phase 9), not a regression-detection question.

### Baseline Source
- **D-06:** Rolling main baseline via `actions/cache`. A separate `push: branches: [main]` workflow captures the same subset (D-02..D-05) and writes raw `.bench.txt` to a cache entry; the PR workflow restores the most recent entry as the comparison baseline.
- **D-07:** Cache key strategy: write key `pr-bench-baseline-${{ github.sha }}` (or equivalently date+sha-tagged), restore-keys prefix `pr-bench-baseline-` so the PR job picks up the latest available entry. The push-on-main workflow always writes a new key; old entries evict naturally via GitHub's 7-day-LRU policy.
- **D-08:** Cache miss is handled gracefully: if no baseline cache is restored (first run after Phase 10 ships, or after a long quiet period evicts the cache), the PR job emits an explicit "no baseline available — advisory bypass" notice in the step summary and the sticky comment, exits 0, and uploads its own raw evidence as a workflow artifact so the next push-to-main fills the cache.
- **D-09:** Baseline staleness is acceptable: "rolling latest main" is the comparison target. The PR job does NOT recapture base or PR-base in the same run (rejected to preserve the 10-minute budget). Cross-runner-class drift between consecutive `ubuntu-latest` instances is treated as part of the noise floor that the threshold (D-11) is calibrated against.
- **D-10:** Public release-scoped evidence under `testdata/benchmark-results/v<version>/` is NOT used as the PR baseline — it is cross-target/time-frozen and cannot detect regressions accumulating between releases.

### Regression Threshold and Enforcement
- **D-11:** Regression definition (per row): the row is flagged when benchstat reports it `p < 0.05` AND the candidate (PR) median is `≥ 5%` slower than the baseline median. Both conditions are required — the `p`-value gate filters out hosted-runner noise; the percentage gate filters out tiny-but-significant deltas that are not actionable.
- **D-12:** Granularity is per row, not per-tier-geomean. Per-tier aggregation hides fixture-specific cliffs (e.g., a 15% canada.json regression masked by an unchanged twitter.json + Tier 2). Every flagged row is listed individually with its `Δ%` and `p`-value.
- **D-13:** Advisory-only initially: the PR workflow exits `0` on regressions; regressions are surfaced via PR comment + step summary annotation only. The PR benchmark check is NOT registered as a required status check at Phase 10 ship time.
- **D-14:** Graduating to blocking is explicitly out of scope for Phase 10. After ~1 month / N PRs of observed regression-flag quality, a separate decision (and a small follow-up PR flipping a config knob) makes the check blocking. Phase 10 must leave a clearly named control surface (env var, workflow input, or workflow file constant) that the future flip changes.
- **D-15:** Regression rows in the PR comment cite the exact benchmark row name, the baseline median (in time/op + B/op), the candidate median, the absolute and percentage delta, and the benchstat-reported `p`-value. This format mirrors `scripts/bench/run_benchstat.sh`'s output so reviewers can rerun it locally.

### Skip Conditions and PR Surface
- **D-16:** The PR job uses `paths-ignore` (not `paths`) to skip docs-only / planning-only / benchmark-evidence-only / unrelated-workflow-only PRs. `paths-ignore` ensures mixed PRs (docs + code) still run; `paths` would silently skip them.
- **D-17:** Skip path globs (paths-ignore set):
    - `**.md`, `docs/**`, `LICENSE`, `NOTICE`
    - `.planning/**`
    - `.github/workflows/**` *except* the new PR-bench workflow file itself, and `.github/actions/**`
    - `testdata/benchmark-results/**`
- **D-18:** Risk acknowledged for skipping `.github/actions/**`: edits to shared composite actions (e.g., `setup-rust`) can affect native build behavior. Acceptable tradeoff for v0.1; if this skip ever masks a real regression, planning revisits the glob.
- **D-19:** Surface order: `$GITHUB_STEP_SUMMARY` (always written, works on fork PRs and same-repo PRs) AND a sticky PR comment via `marocchino/sticky-pull-request-comment` (or the planner-selected equivalent). The sticky comment is best-effort: if `pull_request` token permissions deny the write (forks without `pull_request_target`), the workflow logs a notice, skips the comment, and still posts the step summary.
- **D-20:** Workflow concurrency: `group: pr-bench-${{ github.event.pull_request.number }}`, `cancel-in-progress: true`. Rapid pushes to the same PR cancel the previous run instead of stacking. (Push-to-main workflow uses a different group so it never cancels itself.)
- **D-21:** Workflow artifact upload (raw `.bench.txt` for head and baseline, `benchstat.txt`, `summary.json`) runs `if: always()`, mirroring Phase 9's "upload evidence even on gate failure" pattern. PR diagnostic artifacts retain for 14 days (shorter than Phase 9's 30 days — PR runs are higher-volume).

### Claude's Discretion
- Exact workflow filenames (`.github/workflows/pr-benchmark.yml`, `.github/workflows/main-benchmark-baseline.yml` or planner-chosen equivalents).
- Exact `summary.json` shape, as long as it includes flagged rows, target metadata, threshold values, and a `regression: true|false` boolean.
- Exact sticky PR comment markdown layout, as long as it presents flagged rows with their deltas + `p`-values and clearly labels itself as advisory.
- Exact name of the future "blocking flip" control knob (D-14).
- Decision whether to share one Python script with `scripts/bench/check_benchmark_claims.py` (extending it) versus introducing a new `scripts/bench/check_pr_regression.py`. Phase 9's claim gate has different semantics (publishability, asymmetric); a new script is acceptable as long as it reuses `parse_benchmark_file()` and benchstat parsing helpers.
- Whether to use `actions/cache@v4` directly or `actions/cache/restore@v4` + `actions/cache/save@v4` for finer control on miss handling.
- Whether to install `benchstat` from the standard `golang.org/x/perf/cmd/benchstat@latest` or pin to the version Phase 9 uses (planner should pin if Phase 9 pinned).

</decisions>

<specifics>
## Specific Ideas

- The user accepts the runtime budget (≤10 min) as the binding constraint and chose subset / count / fixture trims to fit it, rather than relaxing the budget.
- The user chose advisory-only with a manual flip later — reflecting the principle "contributors should not learn to ignore the bot." The blocking flip is a separate, later decision and is NOT bundled into Phase 10.
- The user explicitly accepted the small risk of skipping `.github/actions/**` even though composite-action edits could affect builds, in exchange for fewer wasted PR runs.
- The user selected step summary + sticky PR comment together — the step summary handles fork PRs and check-history visibility; the sticky comment handles review-time visibility for same-repo PRs.
- The user's roadmap goal explicitly includes Tier 3 ("Tier 1/Tier 2/Tier 3"), so Tier 3 is in even though it's a v0.2 selective-path placeholder. Dropping Tier 3 was offered and rejected.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope, prior benchmark decisions, and project-level constraints
- `.planning/ROADMAP.md` § Phase 10 — phase goal, dependency on Phase 09.1, and the explicit "lightweight, not release-grade" boundary.
- `.planning/PROJECT.md` — current state (post Phase 09.1), benchmark positioning principles, public artifact constraints.
- `.planning/REQUIREMENTS.md` § BENCH-01..07 — committed benchmark contract that Phase 10 inherits.
- `.planning/STATE.md` — current focus, milestone progress, and the Phase 10 entry retiring backlog item 999.8.
- `.planning/phases/07-benchmarks-v0.1-release/07-CONTEXT.md` — Tier 1/2/3 definitions, fixture choice rationale, comparator-omission pattern.
- `.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-CONTEXT.md` — Phase 9 release-grade decisions Phase 10 must NOT duplicate or supersede (D-03, D-07, D-19, D-22 in particular).
- `.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-LEARNINGS.md` — "Same-target baselines matter," "Claim gates need real benchstat output, not idealized fixtures," "Public benchmark docs need their own contract tests."
- `.planning/phases/999.8-pr-head-ci-coverage-for-feature-branches/.gitkeep` — empty placeholder; the 999.8 idea is now this phase, no separate spec doc exists.

### Existing benchmark workflow, scripts, and tests to extend or mirror
- `.github/workflows/benchmark-capture.yml` — Phase 9 dispatchable release-grade capture; the structural model the PR workflow mirrors (checkout → setup-go → setup-rust → install benchstat → cargo build --release → bench → upload artifact).
- `.github/actions/setup-rust/` — pinned Rust toolchain composite action; reuse as-is.
- `scripts/bench/capture_release_snapshot.sh` — Phase 9 capture pipeline; reference for the bench-then-benchstat-then-summary sequence, NOT to be invoked from PR (it is release-grade and oversized for the 10-minute budget).
- `scripts/bench/run_benchstat.sh` — canonical `--old <path> --new <path>` benchstat wrapper; the PR job runs this on the head vs. cached-baseline `.bench.txt` files.
- `scripts/bench/check_benchmark_claims.py` — Phase 9 claim gate; reuse `parse_benchmark_file()` and metadata-validation helpers, but the regression gate is a separate script (different semantics).
- `tests/bench/test_check_benchmark_claims.py` — Phase 9 contract tests; the new PR regression script needs its own equivalent.
- `benchmark_fixtures_test.go`, `benchmark_fullparse_test.go`, `benchmark_typed_test.go`, `benchmark_selective_test.go` — Tier 1/2/3 benchmark families; the `-bench` regex (D-02, D-05) targets these.
- `benchmark_comparators_test.go` — comparator registry; reference for how to filter to pure-simdjson + encoding-json rows only.

### Existing release-grade evidence (for reference, not for use as PR baseline)
- `testdata/benchmark-results/v0.1.2/phase9.bench.txt` — current public Tier 1/2/3 raw evidence; PR job does NOT compare against this (D-10).
- `testdata/benchmark-results/v0.1.2/metadata.json` — example metadata.json shape Phase 10's `summary.json` should align with where overlap exists.

### External references
- `https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows` — `actions/cache` semantics, restore-keys prefix matching, eviction policy.
- `https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request` — `pull_request` event scope, fork token permissions, `paths-ignore` semantics.
- `https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#adding-a-job-summary` — `$GITHUB_STEP_SUMMARY` markdown semantics.
- `https://github.com/marocchino/sticky-pull-request-comment` — sticky PR comment action; planner verifies maintained release + permissions model before committing to it.
- `https://pkg.go.dev/golang.org/x/perf/cmd/benchstat` — benchstat output format the regression parser depends on (`Δ%` notation, `p=...` annotation, `~` sentinel for non-significant rows).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `.github/workflows/benchmark-capture.yml` — structural template for the new push-to-main baseline workflow (and, with `pull_request` trigger swap, partly for the PR workflow). Shares the same setup-go / setup-rust / cargo-build-release / benchstat-install steps.
- `.github/actions/setup-rust` — pinned Rust toolchain action; reuse without changes.
- `scripts/bench/run_benchstat.sh` — already standardizes benchstat invocation; the PR regression script invokes it.
- `scripts/bench/check_benchmark_claims.py` — `parse_benchmark_file()`, metadata-extraction helpers, and the benchstat-output regex (`SIGNIFICANT_WIN_RE`) are directly reusable. The PR regression gate adapts these into a new bidirectional (faster OR slower) parser.
- `BenchmarkTier1FullParse_*`, `BenchmarkTier2Typed_*`, `BenchmarkTier3SelectivePlaceholder_*` — already exist; PR job filters via `-bench` regex (D-02, D-05).
- `tests/bench/` directory — established home for Python contract tests; the PR regression script gets its own `test_check_pr_regression.py`.

### Established Patterns (from Phase 7 / Phase 9)
- Workflow artifact = transport, repo = source of truth — PR job follows this strictly: artifacts hold raw `.bench.txt` and `summary.json` for diagnostics, never committed.
- Upload evidence even on failure — `if: always()` on the upload step (D-21).
- Comparator omission via build tags / regex — never fake `N/A` rows; rows we don't run simply don't appear.
- benchstat is the single comparison tool — no ad-hoc text munging.
- Native allocator metrics next to Go `benchmem` data — already produced by the Tier 1 family; PR summary preserves the columns benchstat emits.
- Public benchmark wording stays out of PR-job scope — PR is internal regression detection, not public claims.

### Integration Points
- New file: `.github/workflows/pr-benchmark.yml` (or planner's equivalent) — `pull_request` trigger with `paths-ignore` set per D-17.
- New file: `.github/workflows/main-benchmark-baseline.yml` (or planner's equivalent) — `push: branches: [main]` trigger; captures baseline subset and writes to `actions/cache`.
- New script: `scripts/bench/run_pr_benchmark.sh` (or planner's equivalent) — orchestrates the bench-and-benchstat-and-summary sequence on the PR head; small, ≤200 LOC, NOT a copy of `capture_release_snapshot.sh`.
- New script: `scripts/bench/check_pr_regression.py` — parses benchstat output, applies D-11/D-12 thresholds, emits `summary.json` and a markdown step-summary fragment.
- New tests: `tests/bench/test_check_pr_regression.py` — fixture-driven contract tests for the regression parser, mirroring `test_check_benchmark_claims.py`'s style.
- README / methodology docs are NOT touched in Phase 10 (D-13: advisory-only, no public claims).
- `CHANGELOG.md` may receive a brief `Unreleased` note describing the new PR check; planner decides.

</code_context>

<deferred>
## Deferred Ideas

- **Flipping the PR regression check to required/blocking** — explicitly out of Phase 10 (D-14). After observation period, a small follow-up flips the control knob.
- **Tier 1 diagnostics + cold/warm coverage on PR** — out of budget for Phase 10. Stays in Phase 9 release-grade capture. Revisit if the 10-min budget grows or if a regression class is observed that Tier 1+2+3 alone misses.
- **`citm_catalog.json` fixture on PR** — same reasoning; out of budget. Stays in Phase 9.
- **Cross-comparator regression detection (sonic, minio, goccy)** — comparator competitiveness is a release/headline concern (Phase 9), not a PR-time concern.
- **Multi-platform PR benchmarking (darwin, windows, linux/arm64)** — out of scope. linux/amd64 is the canonical target (Phase 9 D-07). Other-platform regressions surface in release-time capture.
- **Same-run base + head capture** — rejected for budget reasons (would force `-count=3` and hurt benchstat significance). May be reconsidered if rolling-main baseline drift becomes the dominant noise source.
- **`benchmark-action/github-action-benchmark` for gh-pages history** — Phase 9 D-11 already labeled this auxiliary, not source of truth. The home-grown `actions/cache` baseline (D-06..D-09) supersedes the need.
- **Detecting regressions on `main` itself** (i.e., when a merged PR caused a regression that the rolling baseline now silently absorbs) — distinct problem; could be added later as a push-to-main benchstat-vs-previous-cache check, but is not Phase 10's promise.
- **`bench-strict` PR label to opt-in to blocking on perf-sensitive PRs** — considered and rejected for v0.1 (D-14 keeps the global advisory/blocking decision binary). Could be a thin layer added later.

</deferred>

---

*Phase: 10-lightweight-pr-benchmark-regression-signal*
*Context gathered: 2026-04-27*
