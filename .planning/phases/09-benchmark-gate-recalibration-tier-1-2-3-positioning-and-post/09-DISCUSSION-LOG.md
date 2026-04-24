# Phase 9: Benchmark gate recalibration, Tier 1/2/3 positioning, and post-ABI evidence refresh - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `09-CONTEXT.md`; this log preserves the alternatives considered.

**Date:** 2026-04-24
**Phase:** 09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
**Areas discussed:** Evidence scope, Public positioning, Result artifact shape, Release decision boundary, Benchmark gate policy

---

## Evidence Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Tier 1/2/3 only | Rerun only the main public benchmark families. | |
| Tier 1/2/3 + diagnostics | Rerun public families and Tier 1 diagnostic split. | |
| Tier 1/2/3 + diagnostics + cold/warm | Rerun public families, diagnostics, and cold/warm lifecycle rows. | yes |

**User's choice:** Tier 1/2/3 plus Tier 1 diagnostics plus cold/warm.
**Notes:** README/result docs currently mention all three evidence types.

| Option | Description | Selected |
|--------|-------------|----------|
| Compare only against Phase 7 `v0.1.1` committed evidence | Treat previous pure-simdjson evidence as the comparison baseline. | |
| Compare against Phase 7 plus Phase 8 diagnostics | Use both historical and post-ABI evidence. | |
| Industry comparator comparison | Compare public positioning against industry comparators, not the older weaker pure-simdjson implementation. | yes |

**User's choice:** Compare against industry comparators.
**Notes:** The earlier weaker implementation is historical context, not the competitive bar.

| Option | Description | Selected |
|--------|-------------|----------|
| Same-host `darwin/arm64` only | Capture on the local same-host target only. | |
| Same-host plus available CI platforms | Use local plus whatever CI can provide. | |
| Require real `linux/amd64` | Require real `linux/amd64` before public wording changes. | yes |

**User's choice:** Require real `linux/amd64`.
**Notes:** Existing CI has dispatchable smoke/release-validation workflows but no benchmark-capture workflow; Phase 9 should add or use a dedicated `workflow_dispatch` benchmark job.

| Option | Description | Selected |
|--------|-------------|----------|
| Raw `.bench.txt` only | Commit raw benchmark output only. | |
| Raw `.bench.txt` plus benchstat | Commit raw benchmark output and benchstat comparisons. | |
| Raw, benchstat, and machine-readable summary | Commit raw output, benchstat, and a small machine-readable gate/summary. | yes |

**User's choice:** Raw, benchstat, and machine-readable summary/gate output.
**Notes:** Matches the Phase 8 evidence style.

---

## Public Positioning

| Option | Description | Selected |
|--------|-------------|----------|
| Lead with Tier 1 as supported headline | If Linux/amd64 Tier 1 beats `encoding/json + any`, README may lead with Tier 1. | yes |
| Mention Tier 1 improvement but lead with Tier 2/Tier 3 | More conservative if Tier 1 wins but is not the central story. | |
| Keep Tier 1 as worst-case benchmark | Continue avoiding Tier 1 as headline even if improved. | |

**User's choice:** Lead with Tier 1 if the evidence is statistically clean on real `linux/amd64`.
**Notes:** Later gate discussion defined "statistically clean" as benchstat-significant wins on every published fixture.

| Option | Description | Selected |
|--------|-------------|----------|
| Tier 1 greatly improved, but typed/selective remain the headline | Honest improvement without overstating a non-winning Tier 1 result. | yes |
| Tier 1 is no longer a weakness | More assertive wording. | |
| Avoid Tier 1 discussion in README | Keep Tier 1 details only in benchmark docs. | |

**User's choice:** If Tier 1 improves but does not beat stdlib, say Tier 1 greatly improved while typed/selective remain the headline.
**Notes:** Keeps README truthful if the post-ABI materializer helps but does not clear the public headline bar.

| Option | Description | Selected |
|--------|-------------|----------|
| Strong caveat | Tie every claim to exact target/toolchain in README. | |
| Moderate caveat | Headline from Linux/amd64; other platforms may differ. | yes |
| Light caveat | Only link methodology. | |

**User's choice:** Moderate caveat.
**Notes:** Exact target metadata belongs in the results doc.

| Option | Description | Selected |
|--------|-------------|----------|
| Full comparator tables in README | README shows all available comparator tables. | |
| Stdlib ratios in README; full tables in docs | README stays readable while docs preserve full evidence. | yes |
| Avoid competitor naming in README | Keep named comparators out of README. | |

**User's choice:** Stdlib-relative ratios in README; full comparator tables in benchmark docs.
**Notes:** Balances public readability with evidence depth.

---

## Result Artifact Shape

| Option | Description | Selected |
|--------|-------------|----------|
| `docs/benchmarks/results-v0.1.2.md` | Tie snapshot to next release/upcoming release. | yes |
| `docs/benchmarks/results-phase9.md` | Tie snapshot to GSD phase. | |
| Update `results-v0.1.1.md` in place | Overwrite existing public snapshot. | |

**User's choice:** Detach benchmarks from phase numbering and tie them to a release or upcoming release.
**Notes:** Use `v0.1.2` as the working next benchmark snapshot label unless planning updates all related paths consistently.

| Option | Description | Selected |
|--------|-------------|----------|
| `testdata/benchmark-results/phase9/` | Phase-scoped raw evidence. | |
| `testdata/benchmark-results/v0.1.2/` | Release/upcoming-release scoped raw evidence. | yes |
| Update `testdata/benchmark-results/v0.1.1/` | Overwrite current public raw evidence. | |

**User's choice:** Use version-scoped raw evidence plus investigate GitHub specialized benchmark storage if available.
**Notes:** Research found GitHub Actions artifacts are retention-limited. `benchmark-action/github-action-benchmark` supports Go benchmark history/charts via GitHub Pages, but committed version-scoped evidence remains durable.

| Option | Description | Selected |
|--------|-------------|----------|
| Pass/fail only | Machine output only gives aggregate pass/fail. | |
| Ratios and metadata | Include fixture ratios and target metadata. | |
| Ratios, metadata, thresholds, and allowed claims | Include enough data to drive docs/README claims. | yes |

**User's choice:** Include ratios, metadata, thresholds, and which public claims are allowed.
**Notes:** This should support generated claim allowances.

| Option | Description | Selected |
|--------|-------------|----------|
| README links directly to new result doc | Direct README pointer to snapshot. | |
| README links only to versioned result docs | Direct versioned docs from README. | |
| README links to `docs/benchmarks.md` | Methodology page points to current snapshot. | yes |

**User's choice:** README links to `docs/benchmarks.md`, which points to the current snapshot.
**Notes:** Keeps the README stable as snapshots advance.

---

## Release Decision Boundary

| Option | Description | Selected |
|--------|-------------|----------|
| Phase 9 locks the next benchmark snapshot version | Version label is fixed during benchmark recalibration. | yes |
| Phase 09.1 locks the version | Phase 9 only prepares evidence. | |
| Use an upcoming label until release day | Avoid a version label during Phase 9. | |

**User's choice:** Phase 9 locks the next benchmark snapshot version label.
**Notes:** The context uses `v0.1.2` as the working next snapshot label.

| Option | Description | Selected |
|--------|-------------|----------|
| Phase 9 can tag if Linux/amd64 gates pass | Phase 9 directly enables release tagging. | |
| Phase 9 recommends; Phase 09.1 aligns release/bootstrap | Phase 9 can recommend but does not complete release alignment. | yes |
| Phase 9 is docs/evidence only | No release recommendation. | |

**User's choice:** Phase 9 may recommend a patch release, but Phase 09.1 performs release/bootstrap alignment first.
**Notes:** Research confirmed the clean order: benchmark a release-candidate commit, commit evidence/docs, then tag after release/bootstrap alignment. A release workflow may attach/publish already-produced evidence, but should not discover the benchmark story after tagging.

| Option | Description | Selected |
|--------|-------------|----------|
| Update docs but do not tag | Public docs update without release framing. | |
| Block README headline until Phase 09.1 | No README headline until default installs work. | |
| Publish benchmark docs, README says upcoming release | Benchmark docs are public but release language stays future-facing. | yes |

**User's choice:** Publish benchmark docs, but README says "upcoming release" until Phase 09.1 validates default installs.
**Notes:** Avoids claiming a default-install release path before bootstrap alignment.

| Option | Description | Selected |
|--------|-------------|----------|
| Add Unreleased benchmark notes | Update changelog in Phase 9. | yes |
| Changelog waits for Phase 09.1 | Only benchmark docs and README in Phase 9. | |
| No public docs beyond benchmark results | Minimal public doc change. | |

**User's choice:** Add `Unreleased` benchmark-positioning notes to `CHANGELOG.md`.
**Notes:** Keeps release notes aligned with benchmark docs.

---

## Benchmark Gate Policy

| Option | Description | Selected |
|--------|-------------|----------|
| Median faster on every fixture | Easier to pass but weaker under benchmark noise. | |
| Benchstat-significant faster on every fixture | Stricter headline gate. | yes |
| Geomean faster, misses allowed if explained | Allows overall wins with individual fixture losses. | |

**User's choice:** Tier 1 headline requires a benchstat-significant win over `encoding/json + any` on every published fixture.
**Notes:** User requested explanation before selecting; this is the defensible public headline gate.

| Option | Description | Selected |
|--------|-------------|----------|
| No regression versus current public snapshot | Protects historical pure-simdjson results only. | |
| Still faster than `encoding/json + struct` | Protects public stdlib-relative claim only. | |
| Both no material regression and still faster than stdlib | Protects continuity and public claim validity. | yes |

**User's choice:** Tier 2/Tier 3 must both avoid material regression and still beat `encoding/json + struct`.
**Notes:** Preserves the existing strength story while preventing hidden regressions.

| Option | Description | Selected |
|--------|-------------|----------|
| Fail closed | Rerun until significance or keep conservative claims. | yes |
| Publish medians with caveat | Allow noisy directionally-clear claims. | |
| Maintainer decides case by case | Manual judgment. | |

**User's choice:** Fail closed on noisy runs.
**Notes:** Applies especially to headline claims.

| Option | Description | Selected |
|--------|-------------|----------|
| Pass/fail only | Aggregate gate result. | |
| Per-tier statuses | Tier-level claim statuses. | |
| Per-fixture statuses plus claim allowances | Detailed statuses and generated README/doc claim allowances. | yes |

**User's choice:** Per-fixture statuses plus generated README/doc claim allowances.
**Notes:** The gate should inform docs directly.

---

## the agent's Discretion

- Exact workflow filename and script names.
- Exact machine-readable summary format.
- Exact public wording, constrained by claim gates.

## Deferred Ideas

- Release/bootstrap artifact alignment belongs to Phase 09.1.
- Any new parser/API/materializer optimization discovered during benchmarking belongs in a future phase or backlog.
- GitHub Pages benchmark history is auxiliary unless planning explicitly accepts the third-party action and its write/Pages tradeoffs.
