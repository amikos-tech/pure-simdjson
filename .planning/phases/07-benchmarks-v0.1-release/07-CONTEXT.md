# Phase 7: Benchmarks + v0.1 Release - Context

**Gathered:** 2026-04-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Close out `v0.1` with a credible, reproducible benchmark story plus the missing public-facing release artifacts: `README.md`, final changelog content, and top-level `LICENSE` / `NOTICE`.

This phase is about benchmark methodology, correctness evidence, and release-facing documentation. It does not add new parser capabilities, redesign the existing release/bootstrap pipeline, or pull `v0.2` On-Demand work into `v0.1`.

</domain>

<decisions>
## Implementation Decisions

### Benchmark fairness and tier definitions
- **D-01:** Tier 1 is strict full-materialization parity. `pure-simdjson` must materialize an equivalent Go tree before timing is counted so the headline comparison against `encoding/json` stays defensible.
- **D-02:** Tier 2 measures schema-shaped end-to-end typed extraction using the current public API (`GetField`, typed accessors, iterators), compared against typed decoding in the baseline libraries.
- **D-03:** Tier 3 is a runnable selective-field benchmark built on the current DOM API and explicitly labeled as a `v0.2` placeholder. It is part of the harness, but it is not the main headline claim.
- **D-04:** Public benchmark tables only compare libraries that actually run on that exact target/toolchain combination. Unsupported comparators are omitted from that table rather than shown as `N/A`.

### the agent's Discretion
- Exact canonical benchmark environment and publication format, as long as cold-start and warm results are separated and the published numbers are reproducible.
- Exact release-close sequencing around the existing `v0.1.0` version/tag state, as long as the final docs, artifacts, and bootstrap validation stay coherent.
- Exact `README.md` structure, as long as it includes installation, quick-start usage, supported-platform guidance, and a benchmark snapshot with methodology caveats.

</decisions>

<specifics>
## Specific Ideas

- The benchmark story should optimize for credibility over drama.
- The typed-extraction tier should reflect how the library is actually intended to be used, not synthetic accessor microbenchmarks.
- The selective-path tier should be clearly marked as a DOM-era placeholder, not presented as a shipped On-Demand capability.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and project constraints
- `.planning/ROADMAP.md` — Phase 7 goal, must-haves, success criteria, and release-close boundary.
- `.planning/PROJECT.md` — core value, product constraints, release narrative, and current project state.
- `.planning/REQUIREMENTS.md` — `BENCH-01..07` and `DOC-01`, `DOC-06`, `DOC-07`.
- `.planning/STATE.md` — current milestone state and the note that the project is at release-close / post-Phase-06.1 sign-off.

### Benchmark methodology and risk guidance
- `.planning/research/STACK.md` — benchmark tooling, corpus choices, comparator set, and `benchstat` conventions.
- `.planning/research/PITFALLS.md` — fairness, warm-up, and native-allocation reporting pitfalls that Phase 7 must avoid.
- `.planning/research/SUMMARY.md` — consolidated benchmark and release guidance for the project.
- `.planning/research/ARCHITECTURE.md` — prior benchmark/workflow structure ideas and repo-level architecture constraints.
- `.planning/research/FEATURES.md` — benchmark expectations tied to user value and representative corpus choices.

### Existing release and bootstrap anchors
- `docs/releases.md` — authoritative tag-driven release runbook and post-publish validation boundary.
- `docs/bootstrap.md` — consumer/operator bootstrap contract and current installation guidance.
- `CHANGELOG.md` — existing Keep-a-Changelog file to extend rather than replace.
- `internal/bootstrap/version.go` — current version pin that must stay aligned with any release-close decision.
- `.github/workflows/release.yml` — current publish workflow.
- `.github/workflows/public-bootstrap-validation.yml` — current fresh-runner public bootstrap validation workflow.
- `scripts/release/run_public_bootstrap_smoke.sh` — existing end-to-end public bootstrap validation shape.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `docs/releases.md`: already defines the supported release-close process and should remain the operator source of truth.
- `docs/bootstrap.md`: already covers runtime installation/bootstrap behavior; `README.md` should link to it, not duplicate it.
- `.github/workflows/release.yml` and `.github/workflows/public-bootstrap-validation.yml`: existing release evidence and post-publish validation paths that Phase 7 should consume, not redesign.
- `scripts/release/run_public_bootstrap_smoke.sh` and `tests/smoke/go_bootstrap_smoke.go`: existing end-to-end smoke assets for validating the release story and README install path.
- `internal/bootstrap/url.go`: source of truth for supported platforms and artifact naming that public docs should mirror accurately.
- `internal/bootstrap/version.go`: current version source that constrains any final tagging/release-close sequence.
- `CHANGELOG.md`: existing changelog scaffold with a `0.1.0` entry already present.
- `third_party/simdjson/LICENSE`: upstream license source for the required top-level `NOTICE` work.

### Established Patterns
- Tag-driven CI publish is the only supported release path.
- Public bootstrap validation is intentionally separate from the publish workflow.
- Existing release/bootstrap docs are detailed and operator-focused; the missing top-level docs should stay consumer-focused.
- There is no first-class in-repo benchmark harness yet, so Phase 7 defines that structure from scratch.

### Integration Points
- New benchmark code should integrate with normal `go test -bench` workflows and produce `benchstat`-friendly output.
- New benchmark corpus files will need a stable vendored location inside the repo.
- `README.md`, `CHANGELOG.md`, `LICENSE`, and `NOTICE` form the public release-facing artifact set for this phase.
- Any correctness oracle should live in the normal Go test path so it can run in CI alongside the benchmark harness and release checks.

</code_context>

<deferred>
## Deferred Ideas

- Any parser/API/runtime optimization work discovered during benchmarking that requires new product scope belongs in a follow-up phase or backlog item, not in this release-close phase.
- Real On-Demand / selective-path semantics remain `v0.2` work; Phase 7 only ships a clearly labeled placeholder benchmark on the current DOM API.

</deferred>

---

*Phase: 07-benchmarks-v0.1-release*
*Context gathered: 2026-04-22*
