# Phase 6: CI Release Matrix + Platform Coverage - Context

**Gathered:** 2026-04-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Turn the existing shim, Go wrapper, and bootstrap pipeline into the only supported release path for `v0.1`: CI builds, verifies, signs, and publishes the five supported shared-library artifacts, produces the checksum manifest consumed by bootstrap, and enforces the documented Alpine smoke strategy.

This phase is about release production and release gating. It does not expand the runtime platform matrix beyond the five shipped targets, and it does not add a first-class musl artifact.

</domain>

<decisions>
## Implementation Decisions

### Alpine and musl strategy
- **D-01:** `v0.1` does not ship a musl runtime artifact. Alpine remains a smoke-test-only path using `PURE_SIMDJSON_LIB_PATH`.
- **D-02:** The Alpine smoke job is a hard release gate, not an advisory signal.

### Release-state preparation
- **D-03:** Release metadata is prepared in a normal source commit before tagging. The publish tag must point at that exact prepared commit.
- **D-04:** The tag must not rely on a post-tag follow-up PR to make bootstrap metadata coherent. Tagged source should already contain the release-ready `version.go`, checksum manifest, and release-facing documentation updates required by the workflow.

### Linux build baseline
- **D-05:** Linux release artifacts use manylinux-style container builds as the default path.
- **D-06:** Linux release jobs must prove a glibc baseline of `<= 2.17` via `objdump -T` checks on the produced `.so`.
- **D-07:** zig/cross remains an escape hatch for planning/implementation if the default Linux build path proves insufficient, but it is not the primary release strategy.

### Release gating depth
- **D-08:** Release publication is blocked on native artifact verification and Go-side bootstrap / consumer verification using the packaged artifacts before publish.
- **D-09:** "Artifact verification" means more than successful compilation: exported symbols, parse smoke, signature/codesign checks, checksum generation, and packaging correctness must all be part of the gate.

### Release-process guidance
- **D-10:** Phase 6 includes an in-repo release runbook that humans and agents both follow.
- **D-11:** The runbook is backed by a scriptable readiness gate so the process is documented and enforceable.
- **D-12:** Release guidance should also be delivered as a repo-local agent skill backed by the same runbook, rather than as prose alone.

### Sequencing with the promoted follow-up
- **D-13:** Promoted follow-up Phase `06.1` is a hard gate before the final `v0.1` release closeout work in Phase 7.
- **D-14:** For Phase `06.1`, "fresh machine" can be satisfied by a clean CI runner / empty-cache validation path rather than requiring a manually managed workstation.

### the agent's Discretion
- Exact workflow split between reusable composite actions, shell scripts, and workflow YAML, as long as the release path stays auditable and deterministic.
- Exact shape of the Go-side release smoke gate, as long as it verifies the shipped artifacts through the real bootstrap / consumer path before publish.
- Exact location and naming of the runbook and repo-local skill, as long as both clearly gate the release process and share one source of truth.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and locked requirements
- `.planning/ROADMAP.md` — Phase 6 goal, must-haves, success criteria, and the newly inserted Phase `06.1` follow-up after release automation.
- `.planning/PROJECT.md` — project-level distribution constraints, five-target support, ad-hoc macOS signing choice, and the bootstrap/release narrative after Phase 5.
- `.planning/REQUIREMENTS.md` — `PLAT-01..06` and `CI-01..07`, including the explicit Alpine smoke-only wording and checksum/signing expectations.

### Prior phase decisions that constrain release work
- `.planning/phases/05-bootstrap-distribution/05-CONTEXT.md` — locked bootstrap/distribution decisions, especially asset naming, checksum-key layout, versioning, and docs-only cosign UX.
- `.planning/phases/05-bootstrap-distribution/05-HUMAN-UAT.md` — the deferred live-artifact bootstrap validation that Phase `06.1` now picks up.

### Research that narrows the implementation space
- `.planning/research/SUMMARY.md` — Phase 6 research flag, Alpine/musl uncertainty, ad-hoc macOS signing choice, and the recommended release-matrix shape.
- `.planning/research/STACK.md` — release automation references, manylinux vs zig guidance, cosign/R2 expectations, and musl constraints.
- `.planning/research/ARCHITECTURE.md` — release pipeline placement within the repo architecture and CI responsibilities.
- `.planning/research/PITFALLS.md` — glibc baseline, macOS signing, Alpine/musl, DLL loading, and binary-integrity risks that Phase 6 must guard against.

### Existing implementation anchors
- `internal/bootstrap/url.go` — supported-platform list, on-disk library names, GitHub asset naming, and checksum-key format the release pipeline must honor exactly.
- `internal/bootstrap/checksums.go` — release-time checksum manifest target populated by CI.
- `internal/bootstrap/version.go` — version pinning model; release flow must update this before tagging.
- `docs/bootstrap.md` — current operator-facing bootstrap and cosign documentation; Phase 6 must keep it aligned with the actual release path.
- `.github/workflows/phase2-rust-shim-smoke.yml` — existing native smoke workflow patterns and export checks.
- `.github/workflows/phase3-go-wrapper-smoke.yml` — existing five-target Go-wrapper smoke workflow pattern that can inform the Go-side release gate.
- `tests/smoke/minimal_parse.c` — existing native smoke harness reusable for release validation.
- `cmd/pure-simdjson-bootstrap/verify.go` — existing packaged-artifact verification primitive that should inform the Go-side release gate.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `.github/workflows/phase2-rust-shim-smoke.yml`: already proves a cross-platform native smoke pattern for Linux, Windows, and macOS.
- `.github/workflows/phase3-go-wrapper-smoke.yml`: already proves the five supported targets in Go CI and can inform the packaged-artifact consumer gate.
- `tests/smoke/minimal_parse.c`: existing native harness for ABI version, parser/doc lifecycle, parse smoke, and diagnostics.
- `internal/bootstrap/url.go`: central source of truth for platform names, artifact names, and GitHub asset naming.
- `internal/bootstrap/checksums.go` and `internal/bootstrap/version.go`: exact files Phase 6 release prep must update coherently before tagging.
- `cmd/pure-simdjson-bootstrap/fetch.go` and `cmd/pure-simdjson-bootstrap/verify.go`: ready-made primitives for post-publish artifact verification and clean-runner bootstrap checks.

### Established Patterns
- Bootstrap and release metadata are source-controlled, not hidden behind opaque release-time state.
- Asset names are already split between R2 directory layout and GitHub flat-namespace layout; release automation must not invent a third naming scheme.
- The repo already treats smoke verification as real gating work rather than "best effort"; Phase 6 should extend that discipline to the release path.

### Integration Points
- New release workflow lives under `.github/workflows/` and becomes the canonical publish path.
- Release automation must update `internal/bootstrap/version.go`, `internal/bootstrap/checksums.go`, and any release-facing docs/runbook artifacts in a coherent pre-tag flow.
- Phase `06.1` should consume the published artifacts through the real bootstrap path on a clean runner with an empty cache to validate the post-publish story end-to-end.

</code_context>

<specifics>
## Specific Ideas

- The release gate should exercise the real packaged artifacts from the Go side, not just verify that native binaries compile and export symbols.
- The release process should be teachable to both humans and agents through one shared source of truth: a runbook plus a repo-local skill that enforces the same checklist.
- "Fresh machine" for the promoted validation phase means an ephemeral CI environment with no prewarmed cache or local-target shortcuts.

</specifics>

<deferred>
## Deferred Ideas

- A global reusable release skill beyond this repository is deferred. Phase 6 only needs the repo-local skill and runbook.
- Shipping a musl runtime artifact is deferred beyond `v0.1`; Alpine remains validated through the smoke path only.

</deferred>

---

*Phase: 06-ci-release-matrix-platform-coverage*
*Context gathered: 2026-04-20*
