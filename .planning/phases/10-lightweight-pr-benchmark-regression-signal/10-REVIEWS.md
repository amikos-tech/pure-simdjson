---
phase: 10
reviewers: [gemini, claude]
reviewed_at: 2026-04-27T15:14:33Z
plans_reviewed: [10-01-PLAN.md, 10-02-PLAN.md, 10-03-PLAN.md]
---

# Cross-AI Plan Review — Phase 10

## Gemini Review

This review evaluates the implementation plans for **Phase 10: Lightweight PR benchmark regression signal**. 

### 1. Summary
The plans provide a comprehensive and robust strategy for introducing performance regression detection into the PR pipeline. By utilizing a rolling baseline stored in GitHub Actions cache and a specialized Python parser, the design achieves a "cheap" yet statistically significant signal within the 10-minute wall-clock budget. The architecture is highly idiomatic for the `pure-simdjson` project, favoring script-driven logic over complex YAML, pinning all dependencies to immutable SHAs, and explicitly handling the security hazards of fork-based PRs.

### 2. Strengths
- **Surgical Performance Optimization:** The 10-minute budget (D-01) is aggressively defended by caching `target/release/` (Pitfall 3) and subsetting the benchmarks to Tier 1/2/3 on only two fixtures.
- **Statistical Rigor:** Maintaining `-count=5` and enforcing both a $p < 0.05$ threshold and a $5\%$ delta floor (D-11) ensures the bot provides an actionable signal rather than noise.
- **Security Posture:** The decision to avoid `pull_request_target` (T-10-11) and use a restore-only cache in PR jobs (T-10-12) effectively mitigates the risk of token exfiltration and cache poisoning from untrusted forks.
- **Graceful Degradation:** The "summary-first, sticky-comment-best-effort" pattern (D-19) ensures fork PRs still receive diagnostic data even when GitHub's standard token policy denies write access for comments.
- **Maintainability:** Reusing `parse_benchmark_file()` from Phase 9 rather than duplication (Plan 01) and encoding the benchmark regex in a shell script rather than YAML (Plan 02) makes the system easily testable locally.
- **Clear Migration Path:** The use of the `REQUIRE_NO_REGRESSION` env var as a discoverable control knob (D-14) provides a low-friction path to graduating this check from advisory to blocking.

### 3. Concerns
- **Submodule Cache Staleness (Severity: LOW):** Plan 03 notes that `hashFiles` for `target/release/` caching excludes the `simdjson` submodule SHA, requiring a manual "touch" of `CMakeLists.txt` on submodule bumps. 
    *   *Mitigation:* While brittle, this is explicitly documented and accepted as a workaround for GitHub's file-hashing limits.
- **`benchstat` Version Drift (Severity: LOW):** Since both workflows install via `@latest` (Pitfall 7), a breaking change in `benchstat` output format could theoretically desynchronize a baseline captured today with a PR run tomorrow.
    *   *Mitigation:* Plan 01 requires recording the resolved version in `summary.json`, aiding rapid diagnosis if regex parsing fails.
- **Negation Expressibility (Severity: LOW):** The plan acknowledges that `paths-ignore` cannot negate (D-17), choosing to ignore all workflow edits.
    *   *Mitigation:* This is the correct operational choice to avoid complex YAML logic; a workflow-only change rarely necessitates a performance regression check.

### 4. Suggestions
- **Submodule Cache Invalidation:** Instead of relying on manual `CMakeLists.txt` touches, consider adding `hashFiles('.gitmodules')` to the Rust build cache key. While not perfectly granular for one submodule, it provides an automated invalidation path whenever submodules are modified.
- **Regex Robustness:** In `scripts/bench/run_pr_benchmark.sh`, consider adding a short comment above the `PR_BENCH_REGEX` variable that lists the specific comparators excluded (sonic, minio, etc.), as the regex group `(pure-simdjson|encoding-json-any|encoding-json-struct)` is the definitive filter for what costs CI minutes.

### 5. Risk Assessment: LOW
The overall risk is **LOW**. The plans are highly detailed, acknowledge every pitfall identified during research, and are backed by extensive unit and integration tests (Wave 0). The advisory-only start ensures that any unforeseen noise in the hosted-runner environment will not block the development velocity of the core library. The dependency on the manual `workflow_dispatch` seed run is well-documented in the operator checkpoint.

**Approved for execution.**

---

## Claude Review

# Cross-AI Plan Review: pure-simdjson Phase 10

**Reviewer:** Claude Opus 4.7 (1M)
**Date:** 2026-04-27
**Scope:** Plans 10-01, 10-02, 10-03 — Lightweight PR benchmark regression signal

---

## Overall Summary

The three plans form a well-sequenced, TDD-driven implementation of a PR-time benchmark regression check. They mirror existing Phase 9 patterns closely, pin third-party action SHAs, and correctly choose `pull_request` over `pull_request_target` for fork safety. The decomposition (parser → orchestrator → workflows) is clean: each plan owns a single contract that the next consumes as a black box. **However, two HIGH-severity correctness bugs and several MEDIUM gaps would cause the implementation to fail on first real-world execution.** Both HIGH issues are missed during planning because synthetic fixtures don't reflect the structural multi-table shape of real `benchstat` output, and the cache `path:` semantics in `actions/cache` were not cross-checked between producer and consumer workflows.

---

## Plan 10-01: Bidirectional regression parser, contract tests, fixture corpus

### Strengths

- TDD structure with RED-state acceptance is genuinely tested (the verify command requires the unittest run to fail in a specific way).
- 16-test floor with explicit ceiling-permissive language ("at least 16") avoids the common pitfall of forcing executors to delete useful tests.
- Bidirectional `DELTA_RE` is correctly distinct from Phase 9's unidirectional `SIGNIFICANT_WIN_RE` — the asymmetry is called out and Phase 9's gate is left untouched.
- D-14 control surface (`REQUIRE_NO_REGRESSION`) is named and locked at module level, making the future blocking-flip a one-line PR.
- `seen_any_row` empty-input fail-closed (Pitfall 5) is correctly distinct from "row found, but `~` sentinel — skip."
- Real Phase 9 fixture (`real-tier1-vs-stdlib.benchstat.txt`) is included verbatim as the Phase 9 LEARNINGS fence — exactly the right instinct.
- `unittest.main()` shim mandate is correct given the repo has no `tests/__init__.py`.

### Concerns

- **HIGH — `ROW_PREFIX_RE` will match across ALL benchstat metric tables, not just sec/op.** Real `benchstat` output emits multiple tables for each metric (`sec/op`, `B/s`, `native-allocs/op`, `native-bytes/op`, `native-live-bytes`, `B/op`, `allocs/op`). The same row name appears in each. Critically, in the **B/s** table, a **positive** delta means FASTER (more bytes/second), not slower. Looking at the real fixture:
  ```
  # sec/op (lower is faster)
  Tier1FullParse_twitter_json-4  6.443m ± 1%  2.044m ± 2%  -68.27% (p=0.000 n=10)
  # B/s (higher is faster)
  Tier1FullParse_twitter_json-4  93.47Mi ± 1%  294.62Mi ± 2%  +215.19% (p=0.000 n=10)
  ```
  The parser as designed (`ROW_PREFIX_RE` with no metric-section awareness) would see the +215.19% B/s row, match `sign == "+"`, `pct >= 5.0`, `p < 0.05`, and **flag it as a regression**. This means:
  1. `test_real_phase9_benchstat_format` will fail at GREEN state — `flagged_rows` will NOT be `[]`; the all-faster Phase 9 fixture will produce ~3 false-positive regression flags from the B/s table alone.
  2. Even worse, in production: a real performance improvement will be flagged as a regression because the B/s and `?`-allocs sections all show "+" deltas.

  **Impact:** This is the load-bearing parser for the entire phase. If the test `test_real_phase9_benchstat_format` is added per spec, it will fail and the executor will face a forced choice between (a) deleting the test (silent regression hazard), (b) revising the parser to filter to sec/op (correct fix, requires plan amendment), or (c) hand-tuning the assertion to match observed false positives (the worst option).

  **Fix needed before execution:** Plan must specify a section-aware parser. Suggested approach: track current metric section by detecting the `│ sec/op │` / `│ B/s │` / etc. column-divider lines that benchstat emits between tables, and only emit regressions when the current section is `sec/op`. Alternatively, parse only the FIRST table (sec/op is always first in benchstat's default output). The fixture set should also include a real-format fixture with both fast sec/op and fast B/s tables to exercise this.

- **MEDIUM — Synthetic fixtures may not match real benchstat row spacing.** The action step says "Match whitespace structure of the real file as closely as needed for the parser to parse rows uniformly." This is hand-wavy. Real benchstat right-aligns numeric columns with variable padding. A fixture that is parseable by `DELTA_RE` may not stress every padding edge case. Recommendation: copy one row verbatim from the real file and substitute only the delta/p-value, rather than retyping.

- **MEDIUM — `parse_benchmark_file` import is only used to pull `EvidenceError`.** The plan emphasizes "reuse, no copy-paste of parser body" but the actual usage in `parse_benchstat_for_regressions` only references `EvidenceError`. `parse_benchmark_file` is imported but unused. This invites a future drift where someone removes `parse_benchmark_file` from Phase 9 and breaks Phase 10 silently. Either (a) actually use `parse_benchmark_file` for something (e.g., metadata extraction from the input), or (b) just import `EvidenceError` to make the dependency explicit and the linter happy.

- **LOW — `test_blocking_flip_via_env_var` does not specify whether `subprocess.run` inherits or replaces env.** If `env={"REQUIRE_NO_REGRESSION": "true"}` is passed without `os.environ.copy()` merge, `PATH` and `HOME` are missing and the script may fail to find `python3` interpreter sub-imports. Specify `env={**os.environ, "REQUIRE_NO_REGRESSION": "true"}`.

- **LOW — `chmod +x` on the Python script is unnecessary** since it's invoked via `python3 scripts/...` (orchestrator) and `sys.executable str(SCRIPT_PATH)` (tests). The shebang is fine but the executable bit is no-op cosmetic. Not a blocker.

### Suggestions

1. **Add metric-section awareness.** Introduce `SECTION_HEADER_RE = re.compile(r"│\s*(sec/op|B/s|B/op|allocs/op|native-\S+)")` and track `current_section`. Only flag regressions when `current_section == "sec/op"`. Document in a code comment.
2. **Add a real-format-with-mixed-metrics fixture.** Today's `real-tier1-vs-stdlib.benchstat.txt` is all-faster; add a synthetic real-shape fixture with a slow sec/op row AND a corresponding slow B/s row to confirm the parser reports the regression once, not twice.
3. **Lock the parser invocation env** in tests to use `os.environ.copy()` merge.
4. **Drop the unused `parse_benchmark_file` import** unless actually used.

### Risk Assessment

**HIGH** — The parser as planned will produce false positives on every real benchstat output that has B/s metrics (which is every Tier 1/2/3 row, since `b.SetBytes()` is called). The correctness contract is broken before the workflow ever runs. The 15 synthetic fixture tests will pass; the one real-fixture test will fail; the executor will be tempted to weaken the assertion rather than amend the parser. This is exactly the failure mode Phase 9 LEARNINGS warned against ("Claim gates need to understand real benchstat output, not idealized fixtures") — and the plan even cites that warning while reproducing the failure.

---

## Plan 10-02: Bash orchestrator + integration smoke test

### Strengths

- Locked benchmark regex is the single source of truth (D-02/D-03/D-05 in one constant).
- `set -euo pipefail` + atomic `mktemp`/`mv` staging mirrors the proven Phase 9 pattern.
- Cache-miss path delegated to caller-provided flag (`--no-baseline` xor `--baseline`) — clean separation, both halves explicitly tested.
- PATH-shadowing stub `go` and `benchstat` binaries is the right test approach: exercises real bash semantics without a 5-minute bench run.
- Negative grep acceptance criteria (`! grep -q "actions/cache"`, `! grep -q "minio-..."`) defensively encode the security and scope boundaries.
- Tier 3 / canada over-match is acknowledged and accepted with reasoning (Go's bench filter silently skips non-existent function names).

### Concerns

- **MEDIUM — `must_haves.truths` claims orchestrator runs `cargo build --release` conditionally; action steps do not implement this.** The truth bullet says:
  > "Bash orchestrator runs `cargo build --release` only if a release library is missing"

  But the action steps and `<behavior>` block do not include any cargo invocation, and the tool preflight per checker WARN-1 explicitly excludes `cargo` and `rustc`. Plan 03 owns `cargo build --release` as a separate workflow step. This is a documentation inconsistency — the truth bullet should be deleted or rewritten to "Bash orchestrator does NOT invoke cargo; the workflow YAML owns the native build step."

- **MEDIUM — Stub `go` binary in tests must produce correctly-formatted multi-row output for benchstat to consume.** The action step says "the `go` stub MUST emit a benchmark output containing the 5 expected rows ... ~15 rows × 5 counts = 75 rows total." This is non-trivial: `go test -bench` output has a specific header/footer (`PASS\nok\t...\t<duration>\n`), the `Benchmark<name>-<gomaxprocs>\t<iterations>\t<value> ns/op\t...` row format, and `benchstat` itself parses metadata from `goos:` / `goarch:` / `pkg:` / `cpu:` lines. A stub that gets these wrong will cause `run_benchstat.sh` to fail or produce empty output. Recommend the stub `cat`s a static fixture file checked into `tests/bench/fixtures/pr-regression/` (e.g., a slimmed-down real-format file) rather than synthesizing it inline.

- **MEDIUM — `benchstat` stub is invoked through `scripts/bench/run_benchstat.sh`, not directly.** The stub must be on PATH as the literal name `benchstat`, and `run_benchstat.sh` must find it before any system-installed benchstat. The plan's PATH-prepend pattern (`f"{stub_dir}:{os.environ['PATH']}"`) is correct. But `run_benchstat.sh` does `command -v benchstat` first — the stub must `chmod +x` AND not include shebang issues. Verify the stub bash files have `#!/usr/bin/env bash` and `chmod 0o755`.

- **LOW — `test_with_baseline_produces_full_output_set` uses the orchestrator's atomic-promote `mv` to write to a test-controlled `out_dir`.** If the test passes `--out-dir /tmp/test-X` and the parent (`/tmp`) is on a different filesystem from `mktemp -d` (uncommon but possible), `mv` fails. Recommend the orchestrator's stage_dir mktemp parent be the same as `out_dir`'s parent, which the action step actually says (`mktemp -d "${out_parent}/.${out_base}.tmp.XXXXXX"`). Verify tests pass `out_dir` under the test's own `tempfile.TemporaryDirectory()` so this invariant holds.

- **LOW — `-timeout 600s` on `go test` is the only DoS guard at the orchestrator level.** If a single bench (`-count=5` × 5 fixtures × 3 comparators = 75 sub-runs) hits the cap, the orchestrator emits truncated output and `set -euo pipefail` kills the script with exit 1, surfacing as a workflow failure (not advisory). This is correct behavior, but worth documenting that a budget overrun is loud, not silent.

### Suggestions

1. **Delete or rewrite the cargo-related must_have truth.** The orchestrator does not invoke cargo; that's Plan 03's job. The plan should be self-consistent.
2. **Replace synthetic `go` stub output with a checked-in fixture.** Create `tests/bench/fixtures/pr-regression/stub-go-output.bench.txt` containing a real-format-shaped multi-row capture, and have the stub `go` script just `cat` it. Avoids drift between the test scaffolding and real Go test output.
3. **Add a test for orchestrator failure when stub `go` exits non-zero.** Currently all three tests assume happy paths; add a fourth that simulates a bench compile error and verifies the orchestrator surfaces exit 1 cleanly without leaving partial output in `out_dir` (atomic-promote invariant).

### Risk Assessment

**MEDIUM** — Plan is sound in shape but the integration test is stub-heavy and relies on those stubs producing benchstat-compatible output. If the stub `go` produces malformed bench output, `run_benchstat.sh` may still succeed-with-empty-output or fail in a way that masks parser bugs from Plan 01. The cargo-truth inconsistency is minor but signals the plan was not fully cross-checked. No security or correctness gaps in the orchestrator itself.

---

## Plan 10-03: PR + main-baseline workflows + CHANGELOG

### Strengths

- Third-party SHA pinning is rigorous and explicit; tag-mutation supply-chain attack surface is minimized.
- `pull_request` (not `pull_request_target`) is correctly chosen with documented reasoning; fork-PR degraded path is planned.
- Manual checkpoint task (Task 2) acknowledges that GitHub-Actions-side behavior cannot be unit-tested and gates merge on operator verification.
- `cancel-in-progress: true` on PR concurrency vs `false` on main-baseline is the correct asymmetric pattern.
- D-17 paths-ignore expressibility issue is explicitly resolved (Option 1 — drop the negation), with rationale.
- `REQUIRE_NO_REGRESSION` future-flip env var appears in three discoverable locations (workflow, parser, CHANGELOG) — a maintainer doing the eventual flip will find it instantly.
- Negative-grep acceptance criteria (`! grep -q 'pull_request_target'`, `! grep -q 'actions/cache/save'` in PR workflow) encode security boundaries as tests.

### Concerns

- **HIGH — `actions/cache` `path:` mismatch between save and restore.** The plan locks:
  ```yaml
  # main-baseline (save)
  actions/cache/save with: { path: pr-bench-summary/head.bench.txt, key: pr-bench-baseline-${{ github.sha }} }
  # pr-benchmark (restore)
  actions/cache/restore with: { path: baseline.bench.txt, key: pr-bench-baseline-NEVER-MATCHES, restore-keys: pr-bench-baseline- }
  ```
  `actions/cache@v4` archives files at the exact paths specified during `save`, and on `restore` writes them back to the **same paths** preserved in the archive. The `path:` input on `restore` does not rename the file — it specifies which path(s) the action should restore from the archive. Concretely, after a successful restore, the file will exist at `pr-bench-summary/head.bench.txt` (the path it was saved at), NOT at `baseline.bench.txt`.

  This means the orchestrator's `--baseline baseline.bench.txt` argument points to a file that was never created. The orchestrator's cargo build step creates `pr-bench-summary/` later (since it's the `--out-dir`), but the cache restore would have already extracted into a possibly-not-yet-existent directory. End result: cache-hit path always falls through to no-baseline behavior, OR the orchestrator fails because `baseline.bench.txt` doesn't exist.

  **Fix needed before execution:** Standardize on one path. Two options:
  1. Save and restore both use `baseline.bench.txt`; main-baseline workflow copies/renames `pr-bench-summary/head.bench.txt → baseline.bench.txt` before the save step.
  2. Save uses `pr-bench-summary/head.bench.txt`, restore uses the same path, and PR workflow invokes orchestrator with `--baseline pr-bench-summary/head.bench.txt`. (But this collides with orchestrator's own `pr-bench-summary/` output dir — orchestrator would see a leftover file from a different workflow's cache.)

  Option 1 is cleaner. Either way, both YAMLs need adjustment, and the manual checkpoint Task 2 step 3 ("`cache-matched-key` is non-empty") would still pass while the actual cache-hit-with-real-data path silently never works.

- **MEDIUM — `cache-matched-key == ''` GHA expression evaluation.** The plan uses:
  ```yaml
  NO_BASELINE: ${{ steps.restore-baseline.outputs.cache-matched-key == '' }}
  ```
  GitHub Actions outputs are always strings; comparing to `''` works when the output is unset. However, when there's a cache hit, the value will be the matched key string (e.g., `pr-bench-baseline-abc123...`), and `== ''` evaluates to `false` correctly. When there's a miss, the output may be the empty string OR may not be set at all (the action's behavior varies between hit and prefix-match). Recommend cross-referencing the `actions/cache/restore` action's documented output: if it's `cache-hit` (boolean string `'true'`/`'false'`) plus `cache-matched-key` (always a string), then `cache-hit != 'true'` is more robust than `cache-matched-key == ''`. Same effective semantics; less fragile to action-version output-shape changes.

- **MEDIUM — `hashFiles('third_party/simdjson/CMakeLists.txt')` for the Rust build cache key.** The plan documents the limitation (submodule SHA not in key; manual mitigation via touching CMakeLists.txt on submodule bumps) but this is fragile. Real-world failure mode: contributor bumps simdjson submodule pointer, doesn't touch CMakeLists.txt, PR runs against stale `target/release/`, benchmark numbers reflect old simdjson code, regression check is meaningless. Documented mitigation requires PR-author discipline (or a future lefthook check). For Phase 10 scope this is acceptable, but flag for future hardening — recommend adding `Cargo.toml` to hashFiles too (build.rs version pinning lives there) and documenting "simdjson bump must touch Cargo.toml or CMakeLists.txt" rather than just CMakeLists.txt.

- **MEDIUM — Sticky-comment SHA verification not actually performed during planning.** The plan claims "Pinned action SHAs (verified via `gh api ...` on 2026-04-27)" but executor has no way to confirm that `0ea0beb66eb9baf113663a64ec522f60e49231c0` is actually `marocchino/sticky-pull-request-comment@v3.0.4`. If the SHA is wrong (typo, wrong repo, hallucinated), the workflow fails with "no commit found." Recommendation: add an executor-side verification step at the top of Task 1: `gh api repos/marocchino/sticky-pull-request-comment/git/refs/tags/v3.0.4 --jq .object.sha` should output the planned SHA. Same for `actions/cache@v4.2.4`. Cheap, self-validating.

- **MEDIUM — The post-merge `workflow_dispatch` seeding step is documented in Task 2 but not blocking.** If the operator skips it AND the Phase 10 PR's squash-merge to main has a diff that's entirely under paths-ignore (e.g., if planning files were excluded and only docs changed in some hypothetical future), the cache won't be seeded automatically, the first PR won't get a useful comparison, and the team may lose confidence in the check before it stabilizes. Recommendation: explicitly state that Task 2 step 2 (manual workflow_dispatch) is **mandatory** the first time, even if step 2 also says auto-trigger may have already fired.

- **LOW — `timeout-minutes: 12` for the PR job.** With `cargo build --release` cold ≈4 min + Go setup 30s + checkout 1 min + Rust setup 30s + benchstat install 15s + bench run 3-5 min + benchstat + summary writes, the median path is 9-10 min and the cold-cache path can blow past 12 min. The 2-minute headroom may not survive a single Rust dep update where the build cache misses. RESEARCH explicitly flagged this (Pitfall 3 + Open Question 2). Recommendation: bump to `timeout-minutes: 15` for safety. The job-level timeout is just a safety net; the orchestrator's `-timeout 600s` is the budget gate.

- **LOW — D-18 risk acknowledgment for skipping `.github/actions/**`.** A change to `setup-rust` composite action could change the toolchain and silently affect bench numbers, but PRs touching it would be skipped. The plan accepts this; consider adding a post-merge verification: when `.github/actions/**` changes on main, the next PR (any PR) runs the bench against a baseline produced under the OLD action — this is a one-cycle stale-baseline window that absorbs any benchmark drift caused by the action change. Worth a one-line CHANGELOG note that toolchain changes have a "delay one PR cycle" effect on regression detection.

### Suggestions

1. **Fix the cache path mismatch BEFORE execution.** Pick Option 1 (both use `baseline.bench.txt`); main-baseline workflow does `cp pr-bench-summary/head.bench.txt baseline.bench.txt` before `actions/cache/save`. PR workflow restores to `baseline.bench.txt` directly. Both `path:` fields match exactly.
2. **Add SHA-verification step at the top of Task 1.** `gh api ...` calls confirm planned SHAs are real. ~5 lines, prevents a hard failure on first workflow run.
3. **Switch NO_BASELINE detection to `cache-hit != 'true'`.** More robust than empty-string compare.
4. **Add `Cargo.toml` to the Rust build cache hashFiles** alongside `Cargo.lock` and `CMakeLists.txt`, and document the contributor convention more visibly (CHANGELOG bullet or CONTRIBUTING.md note).
5. **Bump PR `timeout-minutes` to 15.** Cheap safety; preserves D-01 budget at the orchestrator level.
6. **Make the post-merge workflow_dispatch step mandatory and put it in the operator runbook**, not just the Task 2 checkpoint. The first cache seed is critical and should not depend on diff-path luck.

### Risk Assessment

**HIGH** — The cache path mismatch will cause cache-hit paths to silently fall through to no-baseline behavior, defeating the entire phase's value proposition (regression detection against rolling main baseline). The bug is invisible until the first real PR runs against a real cache, at which point the workflow appears to succeed but emits "no baseline available" forever. The manual checkpoint task would catch it (step 3 expects `cache-matched-key` non-empty, but the orchestrator would still hit no-baseline path because `baseline.bench.txt` doesn't exist) — but only if the operator notices that `--baseline` arg was never actually exercised. Other concerns are MEDIUM-tier hardening.

---

## Cross-Plan Concerns

- **Test discovery pattern inconsistency.** Plans 01 and 02 use `python3 -m unittest discover -s tests/bench -p "test_check_pr_regression.py"` (with explicit pattern matching one file). Plan 03 uses `python3 -m unittest discover -s tests/bench -v` (no pattern). The latter picks up everything; the former is single-file-scoped. Both work, but the per-task quick-run vs full-suite distinction is muddled. Recommend Plan 01 and 02 also support `-p "test_*.py"` for full-suite runs and reserve the per-file pattern for the per-task gate.

- **Phase 9 fixture file naming inconsistency.** Plan 01 references `testdata/benchmark-results/v0.1.2/tier1-vs-stdlib.benchstat.txt`. RESEARCH and CONTEXT also reference `phase9.benchstat.txt`. The actual repo has BOTH (verified in `check_benchmark_claims.py`'s `SNAPSHOT_FILES` tuple). Plan 01 picks `tier1-vs-stdlib.benchstat.txt` which is the more focused (sec/op-and-derived-metric only) file — but it still has the multi-table B/s issue described in Plan 01 HIGH concern.

- **`set -euo pipefail` not used uniformly.** Plan 02 mandates it for the orchestrator. Plan 03 mandates it for the inline workflow shell `Install benchstat` step. But the workflow's other inline run scripts (`Build native release library: cargo build --release`, the conditional NO_BASELINE branch in step 8) don't show the mandate. GitHub Actions defaults to `bash -e` (which is `set -e` only); `pipefail` and `nounset` are off by default. Recommend adding `defaults.run.shell: bash` (already there) AND a top-level `defaults.run.flags: -euo pipefail` (or equivalent — actually, GitHub Actions `defaults.run.shell` accepts a custom invocation like `bash -euo pipefail {0}`). This eliminates the per-step variation.

- **No plan owns "what happens when benchstat itself updates and changes output format."** RESEARCH Pitfall 7 surfaces this; no plan binds it. Recommend the parser's `summary.json` record the `benchstat` resolved version (run `go list -m golang.org/x/perf` in the workflow, pass into the parser as `--benchstat-version` arg). One additional argparse line; saves a future debugging session when format drifts.

- **`autonomous: false` on Plan 03 is correct, but the human task is gated on POST-MERGE behavior.** This means Plan 03 cannot truly be "verified" until after it merges. Standard GSD pattern; just flagging that Phase 10 verification has a real gap between "tests green locally" and "workflow actually does the right thing in prod."

---

## Overall Risk Assessment

**HIGH** — Phase 10 has two HIGH-severity correctness bugs (Plan 01 metric-section blindness, Plan 03 cache path mismatch) that would each independently cause the phase to fail its goal of detecting regressions. Both bugs would pass all locally-runnable tests because:
- Plan 01's tests use synthetic single-table fixtures + a real fixture where the test assertion happens to be wrong-but-loosely-stated.
- Plan 03's tests are actionlint/yq-only (correctly so) and the manual checkpoint may not surface the path mismatch unless the operator specifically inspects `baseline.bench.txt` on disk during a cache-hit run.

Both bugs are pre-execution-fixable with small, targeted plan amendments:
1. Plan 01: amend the parser spec to track current metric section and only flag in `sec/op`.
2. Plan 03: amend cache `path:` to match between save and restore (both use `baseline.bench.txt`; main-baseline copies head.bench.txt first).

The remaining MEDIUM/LOW concerns are quality-of-life improvements that don't block the phase but improve resilience.

**Recommendation:** Do NOT begin execution until Plan 01 §HIGH (metric-section awareness) and Plan 03 §HIGH (cache path mismatch) are resolved via plan amendment. Both are <1-hour edits; both are catastrophic if missed.

---

## Consensus Summary

The reviewers agree that Phase 10 is well decomposed and aligned with the project's existing benchmark/release patterns: Plan 10-01 owns parser semantics and tests, Plan 10-02 owns the local orchestrator, and Plan 10-03 owns GitHub Actions integration. Both reviewers also agree that fork safety, advisory-first enforcement, artifact upload, and the future `REQUIRE_NO_REGRESSION` blocking knob are directionally correct.

The main disagreement is risk severity. Gemini rated the plans LOW risk with only low-severity hardening suggestions. Claude rated the plans HIGH risk because it identified two concrete correctness blockers that would prevent the phase goal from working reliably unless the plans are amended before execution. Given Claude's findings are specific, testable, and tied directly to real benchstat/actions-cache behavior, they should be treated as blocking review feedback.

### Agreed Strengths

- The phase decomposition is clean: parser, orchestrator, then workflows.
- The PR benchmark scope is intentionally cheap and avoids release-grade benchmark capture creep.
- `pull_request` plus restore-only cache handling is the right security posture for fork PRs.
- Advisory-only launch with a named future blocking knob is an appropriate rollout path.
- The planned tests and fixtures are stronger than relying on YAML-only validation.

### Agreed Concerns

- `benchstat` version or output-format drift should be observable in `summary.json` so parser failures can be diagnosed quickly.
- Rust/submodule build-cache invalidation is a known weak point and should be documented or strengthened.
- The workflow skip/path strategy is acceptable for Phase 10 but leaves some workflow/action-change drift risk.

### Blocking Follow-Up Before Execution

- Amend Plan 10-01 so the PR regression parser is metric-section aware and only flags slower `sec/op` rows as regressions. Real benchstat output includes `B/s` sections where a positive delta means faster, not slower.
- Amend Plan 10-03 so the main-baseline workflow saves the cache at the same path the PR workflow restores and passes to the orchestrator. Prefer copying `pr-bench-summary/head.bench.txt` to `baseline.bench.txt` before cache save, then restoring `baseline.bench.txt` in the PR workflow.

### Divergent Views

- Gemini considered submodule cache staleness, benchstat drift, and workflow negation limitations LOW severity and approved execution.
- Claude considered the parser metric-section issue and actions/cache path mismatch HIGH severity and recommended not beginning execution until those plan amendments are made.
