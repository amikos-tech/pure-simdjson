---
phase: 9
reviewers: [gemini, claude]
reviewed_at: 2026-04-24T11:16:46Z
plans_reviewed:
  - 09-01-PLAN.md
  - 09-02-PLAN.md
  - 09-03-PLAN.md
---

# Cross-AI Plan Review - Phase 9

## Gemini Review

# Phase 9 Plan Review: Benchmark Gate Recalibration

This review covers implementation plans `09-01-PLAN.md`, `09-02-PLAN.md`, and `09-03-PLAN.md` for the benchmark evidence refresh and positioning recalibration.

## 1. Summary

The Phase 9 plans are exceptionally well-structured and rigorously aligned with the project's "honest benchmarking" core value. By automating the translation of statistical evidence into allowed documentation wording, the plans effectively eliminate "marketing drift" where docs outrun reality. The separation of capture scaffolding (Plan 01), evidence commitment (Plan 02), and documentation refresh (Plan 03) provides a clean, auditable lifecycle for release-scoped evidence. The inclusion of a human provenance checkpoint before modifying public documentation is a critical safety measure for a project that relies on FFI and specific hardware targets.

## 2. Strengths

*   **Evidence-Driven Documentation:** The use of a deterministic claim gate (`check_benchmark_claims.py`) that outputs a `summary.json` with allowed `readme_mode` values is a superior architectural pattern for preventing over-claiming.
*   **Target Integrity:** The strict `--require-target linux/amd64` enforcement ensures that public headline numbers come from the primary production target, not developer workstations.
*   **Benchstat Significance Gate:** Moving beyond simple median ratios to require `benchstat` significance markers for headline wins (D-19) ensures the library's performance story is grounded in statistical rigor.
*   **Least-Privilege CI:** The `benchmark-capture.yml` workflow correctly limits permissions to `contents: read` and `actions: read`, utilizing artifacts only as a transport mechanism rather than granting the workflow write tokens to the repository.
*   **Stable Naming:** The plans preserve Phase 7 diagnostic row names, enabling direct historical comparisons without conflating naming churn with performance changes.

## 3. Concerns

*   **Complex Shell Logic in Capture Script (LOW):** `capture_release_snapshot.sh` is tasked with creating temporary normalized benchmark files to generate stdlib-relative `benchstat` comparisons. While necessary for the gate, this "regex-in-shell" logic can be brittle.
    *   *Severity:* LOW. The plan specifies `set -euo pipefail` and clear commands.
*   **External Dependency on GitHub Runner Availability (MEDIUM):** Plan 02 relies on the existence and availability of `ubuntu-latest` (or the specified runner) to produce the required evidence. In a strictly local environment, this could block the phase.
    *   *Severity:* MEDIUM. Mitigated by the project's clear CI-first distribution model and the manual-dispatch fallback instructions.
*   **Human Checkpoint "approved" Signal (LOW):** Plan 02 uses a resume signal `approved`. In some autonomous environments, this requires a specific tool configuration to wait for user input without timing out.
    *   *Severity:* LOW. Standard operational procedure for the GSD workflow.

## 4. Suggestions

*   **Standardize Normalization Script:** Consider making the benchmark normalization (stripping suffixes for same-snapshot `benchstat` comparison) a small Python helper or a flag in `check_benchmark_claims.py` instead of raw shell/awk/sed in `capture_release_snapshot.sh`. This would make the transformation more testable and less prone to OS-specific shell variance.
*   **Metadata Expansion:** Ensure `metadata.json` explicitly captures the `simdjson` kernel implementation name (e.g., `icelake`, `haswell`) via the library's diagnostic exports. This helps explain performance variance between different `linux/amd64` runners (e.g., different Intel vs AMD generations in GitHub Actions).
*   **Fixture Validation in Gate:** Add a check in `check_benchmark_claims.py` to verify that the byte-size reported in the benchmark output matches the expected size of the committed fixtures. This prevents accidental runs against truncated or modified testdata.

## 5. Risk Assessment

**Risk Level: LOW**

The overall risk is low because the plans are "fail-closed" by design. If benchmarks are noisy, regressions occur, or the target is wrong, the machine gate prevents the documentation from being updated with aggressive claims. The dependency ordering is correct, and the scope is strictly bounded to positioning evidence, leaving artifact publication and bootstrap alignment to the subsequent Phase 09.1. The implementation of Wave 0 unit tests for the claim gate further reduces the risk of logic errors in the statistical gating code.

---

## Claude Review

# Phase 9 Plan Review: Benchmark Gate Recalibration

Insight:

This phase is unusual: it is an evidence-capture and claim-gating phase, not a code-change phase. The real engineering here is the mechanical gate that prevents docs from outrunning evidence - the capture workflow is just transport. Getting the gate's exit semantics right matters more than getting the workflow YAML exactly right.

## Summary

The three-plan decomposition (scaffolding -> evidence capture -> docs refresh) cleanly maps to the phase's `must exist before claims can change` invariant and respects the Phase 09.1 release boundary. The plans are unusually disciplined for a benchmark phase: they treat docs as generated output from `summary.json`, forbid headline wording that isn't mechanically approved, and avoid the usual trap of discovering public claims inside the release workflow. The weakest spots are mechanical rather than architectural - a few acceptance-criteria test snippets have escaping bugs, the "same-snapshot stdlib benchstat" normalization is under-specified and where most execution risk lives, and the claim-gate exit semantics around `conservative_current_strengths` are ambiguous about whether that mode is a pass or a soft failure.

## Strengths

- **Correct problem framing.** The phase explicitly rejects the two anti-patterns that bit Phase 7/8: (a) promoting Phase 8 internal `darwin/arm64` diagnostics to public headline copy, and (b) first-discovering public benchmark claims inside `release.yml`. D-21 "noisy headline runs fail closed" is the right default.
- **Mechanical claim gating.** Generating `summary.json` first and then gating README/CHANGELOG wording on `claims.readme_mode` is a real control, not a convention. The Task 2 Python snippet in 09-03 that asserts "if not tier1_headline_allowed then README must not say 'Tier 1 headline'" makes this executable.
- **Clean separation of transport vs. durable evidence.** Using `actions/upload-artifact` with 30-day retention as transport only, and requiring committed `testdata/benchmark-results/v0.1.2/` files as the durable source of truth, correctly handles the GitHub Actions retention pitfall.
- **Least-privilege workflow.** `contents: read` + `actions: read` only, with an explicit `! rg 'pages: write|id-token: write|github-action-benchmark'` acceptance check in 09-01 Task 2, keeps the door shut on the optional Pages history until someone explicitly decides to open it.
- **Human checkpoint at the right place.** 09-02 Task 2's blocking human-verify checkpoint sits between "evidence captured" and "docs changed" - exactly where provenance drift would otherwise silently propagate.
- **Release boundary discipline.** 09-03 Task 1 requires a specific sentence pointing at `docs/releases.md` + `check_readiness.sh` + `origin/main` anchor, and 09-03 Task 2's `! rg '^## \[0\.1\.2\]|git tag|git push'` acceptance check mechanically prevents Phase 9 from stepping into Phase 09.1's territory.
- **Stable row names preserved.** The plans deliberately do not rename any `BenchmarkTier1FullParse_*` / `BenchmarkTier2Typed_*` rows, which is what makes benchstat comparison to `v0.1.1` meaningful at all.

## Concerns

### HIGH

- **"Same-snapshot stdlib benchstat" normalization is under-specified.** 09-01 Task 2 says to create temp files that copy "only the relevant comparator rows and replace comparator suffixes with the same normalized benchmark names," then run benchstat. But benchstat's significance test requires two *distributions* at the same name - you'd need to produce two files (one with `pure-simdjson` samples renamed to a common name, one with `encoding-json-any` samples renamed to the same common name) and diff them. The plan describes this as "normalized benchmark names" without saying there are two sides. This is the trickiest implementation detail in the whole phase and the most likely to be built wrong; the claim gate will then silently pass/fail on comparisons that aren't what they claim to be.
- **Ambiguous gate exit code for `conservative_current_strengths` mode.** 09-01 Task 1 says the gate emits `readme_mode: "conservative_current_strengths"` as a valid state, but nowhere does the plan say whether that mode produces exit 0 (gate passes, just with conservative wording) or nonzero (gate fails, block docs). 09-02 Task 1 says "if the claim gate exits nonzero ... do not continue to public docs," which suggests the conservative mode must exit 0 - but 09-01 Task 1's Test 6 ("Tier 2 or Tier 3 regression ... exits nonzero") contradicts "conservative" being a passing state if the only way to reach conservative is regressions. This needs pinning down before implementation: is `conservative_current_strengths` a valid publishable state or a soft-failure state?
- **The upload-artifact SHA is pinned to `v4.6.2` while latest is `v7.0.1`.** The research log notes this, and 09-01 Task 2 picks the `v4.6.2` SHA for consistency with `release.yml`. That's a defensible choice, but `v7`'s upload-artifact changed behavior around hidden files and immutable artifacts. Worth a one-line note in the plan confirming the v4 choice is intentional, not stale - otherwise a later `actions/upload-artifact` bump in `release.yml` could silently drift this workflow.

### MEDIUM

- **Acceptance-criteria escaping bugs.** Several `rg` commands in acceptance criteria use `\\s`, `\\d`, `\\.`, `\\|` which are double-escaped for the YAML/XML context but will fail when executed as shell. Examples:
  - 09-01 Task 1: `rg '^(Benchmark[^\\s]+)-\\d+\\s+(.*)$'` - this is in the `<action>` narrative, but the acceptance `rg 'baseline-dir|snapshot-dir|...'` is fine. The heredoc `python3 - <<'PY'` blocks use `'\\n'` in escape sequences that will break when literally executed.
  - 09-03 Task 2: The embedded Python heredoc has `\\n` separators that the executor will interpret inconsistently depending on whether the outer shell interpolates them.
  - Fix: use single-quoted heredocs (already done) and single backslashes, or move the verification scripts to committed files under `scripts/bench/` rather than inlining them in acceptance criteria.
- **`metadata.json` population is the executor's responsibility but the schema isn't anchored.** 09-01 Task 2 lists the required keys (`snapshot`, `goos`, `goarch`, ..., `commands`), but the capture script has to collect these from `go env`, `rustc --version`, `git rev-parse HEAD`, `uname`, `runner_os`/`runner_arch` env vars, etc. The plan doesn't specify which `runner_os` value is expected when the script runs locally vs. in Actions - locally there's no such env var. The claim gate's Test 4 in 09-01 Task 1 rejects missing metadata, so a local run could fail the gate for metadata reasons unrelated to benchmark quality.
- **Tier 3 does not include `canada.json`.** `BenchmarkTier3SelectivePlaceholder_*` only covers `twitter_json` and `citm_catalog_json`; this is correct per the existing code. But the required-rows list in 09-01 Task 1 and the acceptance criteria correctly omit Canada for Tier 3 - good. However, the result doc template in 09-03 Task 1 says headings "Tier 3: Selective Traversal on the Current DOM API" without noting the two-fixture scope; a future reader could add a missing-Canada-row claim that the gate would then correctly reject but docs would be inconsistent. Minor, worth a sentence in the doc template.
- **Capture script error handling under partial failure.** 09-01 Task 2 says "If this command exits nonzero, keep the emitted `summary.json` if it was written and exit nonzero." But if the Tier 1 `go test -bench` command fails midway, the capture script has already written `phase9.bench.txt` with partial data. The next run's gate will either pass on partial data (bad) or fail (fine), but there's no explicit "cleanup on failure" or "atomic swap" semantic. Consider writing to a temp dir and renaming into place only after all steps pass.
- **`purego.Close()` and parser lifetime in benchmark loops.** Not a plan concern per se, but the existing `runTier1FullParseBenchmark` re-creates parsers across iterations for non-pure comparators while reusing a warmed parser for pure-simdjson. This is deliberate (see the "steady-state" comment), but any executor who doesn't read the Phase 7 learnings carefully might "fix" it during capture and invalidate the comparison. Worth an explicit "do not modify benchmark semantics during capture" note in 09-02 Task 1.

### LOW

- **`claims.readme_mode` enum could drift without a schema file.** Three string literals are embedded in three plans plus README-asserting Python. If a fourth mode is added later (e.g., `tier1_headline_with_caveat`), the acceptance-criteria `assert mode in {...}` will need updating in multiple places. A tiny JSON schema or Python enum module imported by tests would centralize this. Not blocking for v0.1.2.
- **Benchstat count guidance is `count=10` but no note on wall-clock budget.** Tier 1 Canada at the Phase 8 same-host number (`~6.14 ms`) * `count=10` * benchstat-recommended samples across three benchmark files is in the multi-minute range - fine on `ubuntu-latest`, but 09-02 Task 1's local linux/amd64 path doesn't warn the executor about expected duration (the 1200s workflow timeout hints at it). Low risk since the workflow has `timeout-minutes: 60`.
- **`docs/benchmarks.md` "rerun commands" block will go stale when v0.1.3 arrives.** The plan updates commands to point at `v0.1.2` paths; the next benchmark snapshot will need the same doc edited again. Consider a `<!-- current-snapshot -->` marker comment to make the next phase's planner grep it reliably.
- **Sonic/minio comparator availability on `ubuntu-latest`.** The workflow runs on `ubuntu-latest` (x86_64), so `minio-simdjson-go` should be available - meaning the `v0.1.2` snapshot will have a `minio-simdjson-go` row that `v0.1.1` does not (Rosetta-only on that snapshot failed). The `phase9.benchstat.txt` comparison against `v0.1.1` will show missing-row warnings for minio. Not a bug, but the claim gate and docs should explicitly handle "new comparator appeared in snapshot" rather than flagging it as malformed.

## Suggestions

1. **Specify the stdlib-comparison benchstat construction explicitly in 09-01 Task 2.** Add a worked example: to produce `tier1-vs-stdlib.benchstat.txt`, generate two tmp files:
   - `tmp-pure.txt`: raw `phase9.bench.txt` metadata + rows filtered to `pure-simdjson`, with `/pure-simdjson` suffix stripped so the row name is `BenchmarkTier1FullParse_twitter_json`
   - `tmp-stdlib.txt`: same filter for `encoding-json-any`, suffix stripped to the same row name

   Then `benchstat tmp-stdlib.txt tmp-pure.txt > tier1-vs-stdlib.benchstat.txt`. This makes "pure beats stdlib" show up as "improvement vs base" in benchstat output, which the gate can then parse. Without this worked example, the implementer will likely produce a benchstat file that compares wrong distributions.

2. **Pin `conservative_current_strengths` semantics.** Either:
   - (a) make it a passing mode (exit 0) that allows docs to publish conservative wording - and adjust 09-01 Task 1 Test 6 to not require regressions to reach this mode, or
   - (b) keep it as a soft-failure (exit nonzero but write `summary.json` anyway) and explicitly say 09-02 Task 1 should then route to "commit evidence, update docs in conservative mode, note the regression in `09-02-SUMMARY.md`."

   Option (a) is cleaner because it lets the gate cover three real public outcomes (Tier 1 headline, Tier 2/3 headline, conservative) without conflating regressions with insufficient evidence.

3. **Extract embedded Python heredocs to a committed verifier.** The `python3 - <<'PY' ... PY` blocks in acceptance criteria (especially 09-02 Task 2 and 09-03 Task 2) are doing real validation work and will drift from reality as escaping breaks. Move them to `scripts/bench/verify_phase9_docs.py` and invoke via `python3 scripts/bench/verify_phase9_docs.py`. Bonus: it can import the claim-gate module directly instead of parsing JSON blindly.

4. **Add a "comparator appearance drift" note.** In 09-01 Task 1, document that when `snapshot` has a comparator row not present in `baseline` (e.g., `minio-simdjson-go` on `ubuntu-latest` vs. Rosetta-only `v0.1.1`), the gate should not treat this as an error - it should only gate on the *pure-simdjson* row movements and present new comparator rows in `fixtures` informationally.

5. **Specify `metadata.json` construction for local vs. CI capture.** Add to the capture script spec: `runner_os` defaults to `uname -s` when `$RUNNER_OS` is unset, `runner_arch` to `uname -m` when `$RUNNER_ARCH` is unset. Otherwise the gate will reject legitimate local captures with "missing runner_os" errors.

6. **Add atomic-write semantics to the capture script.** Capture into `testdata/benchmark-results/v0.1.2.tmp/`, then `mv` into place only after `summary.json` is written successfully. Prevents partial-state commits after a Ctrl-C.

7. **Consider a `--mode local` / `--mode ci` flag on the capture script.** Makes the two different `metadata.json` population paths explicit rather than implicit-on-env-var-presence.

## Risk Assessment

**Overall Risk: MEDIUM**

Justification:
- **Architecture risk: LOW.** The three-plan shape, evidence-before-docs ordering, and Phase 09.1 boundary are all correct. The human checkpoint in 09-02 is well-placed.
- **Execution risk: MEDIUM.** The two HIGH concerns - stdlib-benchstat normalization and `conservative_current_strengths` exit semantics - are both live landmines for the implementer. Either can produce a plausible-looking `summary.json` that grants wrong claim allowances. Both are fixable with a 1-2 sentence spec tightening before execution.
- **Scope-creep risk: LOW.** The plans are disciplined about not touching parser APIs, release workflows, or bootstrap artifacts. The `! rg` negative acceptance criteria are a nice executable fence.
- **Security risk: LOW.** Least-privilege workflow permissions, input-label validation (`^v[0-9]+\.[0-9]+\.[0-9]+...`), and confined output paths are all present.
- **Reversibility: HIGH (good).** Everything committed in this phase is benchmark evidence and documentation; nothing changes the shipping ABI, library binaries, or release artifacts. If the gate says "conservative," the docs go conservative and a later phase can republish stronger claims from a better snapshot without any rollback.

The phase is ready to execute after clarifying the two HIGH items and patching the acceptance-criteria escaping. The 09-02 human checkpoint provides a natural safety catch even if 09-01's gate has subtle bugs, which further lowers overall risk.

Insight:

The best part of this plan set is that it makes the phrase "truthful benchmark positioning" mechanically enforceable rather than aspirational. The `summary.json` -> README.md pipeline means that a future sloppy edit to README that overclaims Tier 1 will fail a committed grep check, not just an honor-system review. That's the right shape for a project whose core value proposition includes "honest benchmark positioning."

---

## Consensus Summary

Both reviewers agree that Phase 9 is architecturally sound and unusually well-scoped for benchmark work. The shared view is that the plan set gets the important sequencing right: build the claim gate first, capture evidence second, and only then update public documentation. They also agree that the phase should stay out of release publication and bootstrap alignment, leaving those decisions to Phase 09.1.

The main difference is severity. Gemini rates the plan set as low risk because it sees the fail-closed claim gate and human checkpoint as strong controls. Claude rates execution risk as medium because two mechanical details could produce plausible but incorrect benchmark conclusions if implemented loosely: same-snapshot benchstat normalization and the exit semantics for `conservative_current_strengths`.

### Agreed Strengths

- The `summary.json` claim gate is the right control for preventing benchmark wording from outrunning evidence.
- The three-plan split cleanly separates scaffolding, evidence capture, and public documentation updates.
- The linux/amd64 requirement and stable benchmark row names preserve comparability for release-facing evidence.
- Least-privilege workflow permissions and artifact-as-transport-only design keep benchmark capture scoped.
- The human checkpoint before documentation changes is correctly placed.
- Phase 09.1 is treated as the release/bootstrap boundary, keeping Phase 9 focused on benchmark positioning and evidence.

### Agreed Concerns

- **Benchmark normalization needs tightening.** Both reviewers flag the stdlib-relative benchstat construction as the most delicate implementation area. The plan should spell out the two-input benchstat shape so pure-simdjson and stdlib distributions are normalized to the same benchmark names on opposite sides of the comparison.
- **Metadata and fixture provenance need explicit validation.** Gemini asks for fixture byte-size validation and simdjson kernel metadata; Claude asks for a clearer local-vs-CI `metadata.json` schema. These are compatible: the capture/gate path should make provenance precise enough to explain variance and reject malformed evidence.
- **Shell or inline-script verification is brittle.** Gemini suggests moving normalization into Python or the claim gate; Claude points out escaping risks in inline acceptance snippets. The shared recommendation is to make the core verification executable and committed rather than relying on fragile shell text.
- **External capture environment remains a practical dependency.** Gemini calls out GitHub runner availability; Claude calls out upload-artifact version behavior and local metadata defaults. The workflow should document the expected CI/local behavior explicitly.

### Divergent Views

- **Overall risk level:** Gemini says LOW; Claude says MEDIUM. This is not a disagreement about architecture. It is a disagreement about how much risk to assign to under-specified mechanical details.
- **Blocking items before execution:** Gemini suggests incremental improvements. Claude recommends clarifying two high-severity issues before execution: benchstat normalization and `conservative_current_strengths` pass/fail semantics.
- **Scope of fixes:** Gemini leans toward small additions inside the existing plan. Claude suggests extracting reusable verifier logic and adding atomic-write semantics to the capture script.

### Highest-Priority Follow-Up Before Execution

1. Clarify `conservative_current_strengths`: decide whether it exits 0 as a publishable conservative mode or exits nonzero as a soft-failure state.
2. Add a worked example for same-snapshot `benchstat` comparison against `encoding-json-any`.
3. Specify `metadata.json` defaults for local execution when GitHub Actions runner variables are absent.
4. Replace or harden inline acceptance snippets whose escaping may not survive execution.
5. Add fixture/provenance checks so benchmark evidence cannot be generated from truncated or unexpected data.
