# Phase 9: Benchmark gate recalibration, Tier 1/2/3 positioning, and post-ABI evidence refresh - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Reframe the public benchmark story after the Phase 8 low-overhead materializer work. Phase 9 must rerun the benchmark evidence on a real `linux/amd64` target, decide which Tier 1/Tier 2/Tier 3 claims are allowed, and update the benchmark docs, README pointer, and changelog language so public claims match measured results.

This phase does not add new parser APIs, tune the materializer, publish native artifacts, or align bootstrap defaults. Phase 09.1 owns release/bootstrap artifact alignment before any tag or default-install release path is completed.

</domain>

<decisions>
## Implementation Decisions

### Evidence Scope
- **D-01:** Rerun the full public benchmark evidence set: Tier 1/2/3, Tier 1 diagnostics, and cold/warm parser lifecycle rows.
- **D-02:** Public positioning compares against current industry comparators, especially `encoding/json`, not against the older weaker pure-simdjson implementation. Phase 7 and Phase 8 evidence remain historical context and regression baselines, not the public competitive bar.
- **D-03:** Require real `linux/amd64` evidence before changing public benchmark wording. Existing workflows include dispatchable smoke/release-validation jobs, but no benchmark-capture workflow; Phase 9 should add or use a dedicated `workflow_dispatch` benchmark job on real `linux/amd64`.
- **D-04:** Commit raw `.bench.txt`, `benchstat` output, and a machine-readable gate/summary output for the release-scoped benchmark snapshot.

### Public Positioning
- **D-05:** If real `linux/amd64` evidence shows a benchstat-significant Tier 1 win over `encoding/json + any` on every published Tier 1 fixture, README may lead with Tier 1 as a supported headline.
- **D-06:** If Tier 1 greatly improves but does not beat `encoding/json + any` under the headline gate, README should say Tier 1 greatly improved while typed extraction and selective traversal remain the headline.
- **D-07:** Use moderate platform caveats in public copy: headline numbers come from `linux/amd64`; other platforms may differ. Exact GOOS/GOARCH/CPU/toolchain metadata belongs in the results document.
- **D-08:** README should show only stdlib-relative ratios. Full comparator tables, including named industry comparators, belong in benchmark docs.

### Result Artifact Shape
- **D-09:** Detach benchmark docs from phase numbering. Do not create `results-phase9.md`; public benchmark snapshots are tied to a release or upcoming release.
- **D-10:** Use `v0.1.2` as the working next benchmark snapshot label for planning: `docs/benchmarks/results-v0.1.2.md` and `testdata/benchmark-results/v0.1.2/`. If planning discovers the release train requires a different semver label, update every docs/raw/workflow path consistently before capture.
- **D-11:** Investigate GitHub-hosted benchmark history as an auxiliary surface, not the durable source of truth. Native GitHub Actions artifacts are useful for workflow bundles but retention-limited. `benchmark-action/github-action-benchmark` is the benchmark-specific option for Go benchmark history/charts through GitHub Pages.
- **D-12:** The machine-readable summary must include ratios, target/toolchain metadata, thresholds, and which public claims are allowed.
- **D-13:** README should link to `docs/benchmarks.md`; that methodology page owns the pointer to the current benchmark snapshot.

### Release Decision Boundary
- **D-14:** Phase 9 locks the next benchmark snapshot version label so docs and evidence paths are release/upcoming-release scoped.
- **D-15:** Phase 9 may recommend a patch release, but Phase 09.1 performs release/bootstrap artifact alignment first.
- **D-16:** Benchmark capture should run before release tagging against a release-candidate commit. A release workflow may verify, attach, or publish already-produced benchmark artifacts, but it should not be the first place where public benchmark claims are discovered.
- **D-17:** If benchmark evidence is strong while the default bootstrap path is still pinned to old artifacts, publish benchmark docs but keep README language framed as an upcoming-release claim until Phase 09.1 validates default installs.
- **D-18:** Update `CHANGELOG.md` under `Unreleased` with benchmark-positioning notes during Phase 9.

### Benchmark Gate Policy
- **D-19:** Tier 1 headline claims require a benchstat-significant win over `encoding/json + any` on every published Tier 1 fixture.
- **D-20:** Tier 2 and Tier 3 claims require both no material regression versus the current public snapshot and continued wins over `encoding/json + struct` on every published fixture.
- **D-21:** Noisy headline runs fail closed: rerun until significance or keep older/more conservative claims.
- **D-22:** Gate output should emit per-fixture statuses plus generated README/doc claim allowances, not just a single pass/fail bit.

### the agent's Discretion
- Exact workflow filename and script names for benchmark capture.
- Exact machine-readable gate output format, as long as it is committed, deterministic, and includes the fields in D-12 and D-22.
- Exact wording of README/changelog/result docs, as long as the claim gates above are enforced.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and prior benchmark decisions
- `.planning/ROADMAP.md` - Phase 9 goal, dependency on Phase 8, and Phase 09.1 follow-up boundary.
- `.planning/PROJECT.md` - current project state, benchmark positioning principles, release constraints, and public artifact constraints.
- `.planning/REQUIREMENTS.md` - `BENCH-01..07` and benchmark/docs requirements that Phase 9 recalibrates.
- `.planning/STATE.md` - current focus and the handoff from Phase 8 into Phase 9.
- `.planning/phases/07-benchmarks-v0.1-release/07-CONTEXT.md` - Tier definitions, truthful positioning, and public benchmark artifact constraints.
- `.planning/phases/07-benchmarks-v0.1-release/07-LEARNINGS.md` - Phase 7 benchmark lessons, especially Tier 1 limitation and Tier 2/Tier 3 strength story.
- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-CONTEXT.md` - Phase 8 boundary and public-positioning handoff to Phase 9.
- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-BENCHMARK-NOTES.md` - internal Phase 8 diagnostic evidence summary and Phase 9 handoff.

### Benchmark docs and evidence
- `docs/benchmarks.md` - current methodology page, tier definitions, rerun commands, and current snapshot pointer.
- `docs/benchmarks/results-v0.1.1.md` - current public benchmark snapshot to supersede with a release-scoped post-ABI snapshot.
- `testdata/benchmark-results/v0.1.1/phase7.bench.txt` - current public Tier 1/2/3 raw evidence.
- `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt` - current public cold/warm raw evidence.
- `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt` - current public Tier 1 diagnostic raw evidence.
- `testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` - post-ABI internal diagnostic raw evidence.
- `testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt` - post-ABI internal diagnostic benchstat comparison.
- `testdata/benchmark-results/phase8/tier1-diagnostics.improvement.txt` - post-ABI internal machine gate output.

### Benchmark code and workflow anchors
- `benchmark_comparators_test.go` - comparator registry, comparator omission behavior, and shared shape checks.
- `benchmark_diagnostics_test.go` - Tier 1 diagnostic family and stable diagnostic row names.
- `benchmark_coldstart_test.go` - cold/warm parser lifecycle benchmark family.
- `benchmark_typed_test.go` - Tier 2 typed extraction benchmark family.
- `benchmark_selective_test.go` - Tier 3 selective placeholder benchmark family.
- `scripts/bench/run_benchstat.sh` - existing benchstat wrapper.
- `scripts/bench/check_phase8_tier1_improvement.py` - prior machine gate style to reuse or generalize for Phase 9 claim gating.
- `.github/workflows/` - existing dispatchable smoke/release-validation workflows; Phase 9 needs a benchmark-capture workflow rather than assuming these already produce benchmark evidence.

### External references
- `https://docs.github.com/en/actions/concepts/workflows-and-actions/workflow-artifacts` - GitHub Actions artifact behavior and use cases.
- `https://docs.github.com/en/actions/tutorials/store-and-share-data` - `actions/upload-artifact`, custom retention, and artifact sharing/downloading.
- `https://docs.github.com/en/organizations/managing-organization-settings/configuring-the-retention-period-for-github-actions-artifacts-and-logs-in-your-organization` - public/private artifact retention limits.
- `https://docs.github.com/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages` - GitHub Pages deployment from Actions.
- `https://github.com/benchmark-action/github-action-benchmark` - third-party benchmark history/charts action supporting Go `go test -bench`.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `benchmark_*_test.go` files already define the benchmark families Phase 9 needs: Tier 1/2/3, diagnostics, cold/warm, comparator registry, fixtures, and native allocation reporting.
- `scripts/bench/run_benchstat.sh` already standardizes local `benchstat` comparison.
- `scripts/bench/check_phase8_tier1_improvement.py` is a concrete machine-gate precedent for parsing Go benchmark evidence and emitting deterministic gate output.
- `docs/benchmarks.md` and `docs/benchmarks/results-v0.1.1.md` provide the existing methodology/result shape to update rather than redesign from scratch.
- `README.md` already has a benchmark snapshot section that can be simplified to stdlib-relative ratios and methodology links.
- `CHANGELOG.md` already has an `Unreleased` section for benchmark-positioning notes.

### Established Patterns
- Benchmark fixtures are committed under `testdata/bench/`; runtime benchmark evidence must not depend on network or `third_party/` paths.
- Public comparator tables omit unsupported libraries on a target instead of showing fake `N/A` rows.
- Native allocator metrics are reported beside Go `benchmem` data where relevant.
- Public benchmark language must optimize for credibility over dramatic claims.
- Release-facing docs and raw evidence should be committed before a tag is cut.

### Integration Points
- Add or update a dispatchable benchmark workflow under `.github/workflows/` for real `linux/amd64` capture.
- Add release-scoped evidence under `testdata/benchmark-results/v0.1.2/` or the consistently updated next-release equivalent.
- Add a release-scoped result doc under `docs/benchmarks/results-v0.1.2.md` or the consistently updated next-release equivalent.
- Update `docs/benchmarks.md` to point at the current post-ABI snapshot.
- Update `README.md` benchmark language according to generated claim allowances.
- Update `CHANGELOG.md` under `Unreleased`.

</code_context>

<specifics>
## Specific Ideas

- The user explicitly wants benchmark positioning against industry comparators, not a victory lap against the repo's earlier weaker implementation.
- The user prefers benchmark artifacts tied to a release or upcoming release rather than to GSD phase numbers.
- The user asked whether GitHub has specialized benchmark storage. The researched answer is: native Actions artifacts are not durable benchmark storage; `benchmark-action/github-action-benchmark` can provide GitHub Pages history/charts, but committed release-scoped evidence remains the durable source of truth.
- The user challenged release-order inversion. The locked model is: benchmark the release-candidate commit first, commit evidence/docs, then Phase 09.1 aligns bootstrap and release artifacts before tagging.

</specifics>

<deferred>
## Deferred Ideas

- Publishing or aligning native release artifacts is deferred to Phase 09.1.
- Default-install bootstrap validation is deferred to Phase 09.1.
- New parser/API/materializer optimization work discovered during benchmarking belongs in a later phase or backlog item.
- Treating GitHub Pages benchmark history as the primary public source of truth is deferred unless planning explicitly accepts the third-party action and its write/Pages tradeoffs.

</deferred>

---

*Phase: 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post*
*Context gathered: 2026-04-24*
