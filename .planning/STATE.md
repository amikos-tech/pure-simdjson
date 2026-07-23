---
gsd_state_version: 1.0
milestone: v0.1
milestone_name: "Tracked in `REQUIREMENTS.md` as v2 — explicitly deferred and will become a separate roadmap:"
status: executing
stopped_at: Completed 11-08-PLAN.md
last_updated: "2026-07-23T14:07:18.871Z"
last_activity: 2026-07-23
progress:
  total_phases: 22
  completed_phases: 11
  total_plans: 64
  completed_plans: 57
  percent: 50
---

# Project State

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-04-23)

**Core value:** Ship a precision-preserving, cgo-free simdjson DOM parser for Go with honest benchmark positioning: typed extraction and selective traversal are the primary story, while full `any` materialization is documented without overstating current wins.
**Current focus:** Phase 11 — upstream-simdjson-refresh-bigint-and-diagnostics

## Current Position

Phase: 11 (upstream-simdjson-refresh-bigint-and-diagnostics) — EXECUTING
Plan: 9 of 14
Status: Ready to execute
Last activity: 2026-07-23
Shipping: Phase 07 PR: https://github.com/amikos-tech/pure-simdjson/pull/18. Phase 08 PR: https://github.com/amikos-tech/pure-simdjson/pull/19. Phase 09 PR: https://github.com/amikos-tech/pure-simdjson/pull/21. Phase 10 PR: https://github.com/amikos-tech/pure-simdjson/pull/27. `v0.1.0` remains the latest published tag. Phase 09.1 now owns bootstrap artifact/default-install alignment before any later release tag.
Progress: [█████████░] 89%

## Performance Metrics

**Velocity:**

- Total plans completed: 35
- Average duration: 11.1m
- Total execution time: 1.4 hours

**By Phase:**
| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
**Recent Trend:**

- Last 5 plans: 08-03, 08-04, 08-05, 09-01, 09.1-01
- Trend: Stable

| Phase Phase 05 PP05 | 5min | 2 tasks | 7 files |
| Phase Phase 05 PP06 | 9min | 2 tasks tasks | 5 files files |
| Phase 09 P01 | 7min | 2 tasks | 7 files |
| Phase 09.1 P01 | 4min | 2 tasks | 9 files |
| Phase 11 P01 | 1 min | 1 tasks | 1 files |
| Phase 11 P02 | 27min | 3 tasks | 16 files |
| Phase 11 P03 | 20min | 2 tasks | 6 files |
| Phase 11 P04 | 16min | 2 tasks | 6 files |
| Phase 11 P05 | 26min | 2 tasks | 7 files |
| Phase 11 P06 | 11min | 2 tasks | 6 files |
| Phase 11 P07 | 11min | 2 tasks | 10 files |
| Phase 11 P08 | 14min | 2 tasks | 7 files |

## Accumulated Context

## Quick Tasks Completed

| Date | Slug | Summary |
|------|------|---------|
| 2026-04-24 | phase8-final-polish | Added executable depth-boundary fence, clarified ERR_INTERNAL split rationale at the ABI enum, expanded cross-ABI numeric comments, and rechecked benchmark gates. |
| 2026-04-24 | phase8-depth-doc-followup | Clarified depth-limit defense-in-depth docs, strengthened user-actionable enum comments, pinned the current accepted nesting boundary, and rechecked benchmark gates. |
| 2026-04-24 | phase8-followup-feedback | Added observable depth-limit status/sentinel coverage, tightened materializer comments, filled adversarial string-span coverage, and rechecked benchmark gates. |
| 2026-04-24 | phase8-pr-review-feedback | Applied Phase 8 PR review fixes for materializer depth guarding, optional-symbol/fallback observability, unsafe frame diagnostics, not-implemented telemetry status, span contract tests/docs, and benchmark regression checks. |
| 2026-04-24 | pr19-review-items-1-2-3-5 | Addressed PR #19 polish items 1/2/3/5: documented `InternalMaterializeBuild` frame-span lifecycle, expanded the LIFO defer ordering comment in the fast materializer, added native-side (Rust + C++) size asserts for `psdj_internal_frame_t` (field-width expression, 32-bit safe), and documented `psimdjson_test_hold_materialize_guard`'s by-design `PARSER_BUSY` return. Comments-and-asserts only — Tier 1 diagnostics benchstat shows no regression (B/op and allocs/op identical, geomean sec/op within noise). |
| 2026-04-27 | apply-pr-22-feedback-items-2-4-6-8-and-9 | Applied 5 of 9 PR #22 review items: bidirectional ABI sync comments between `internal/bootstrap/abi_assertion.go` and `scripts/release/check_bootstrap_abi_state.py`, fixed `semver_tuple` return-type to honor `tuple[int, int, int]` annotation, added `0.1.1` stale-version boundary test, documented + tested pre-release semver acceptance (`0.1.2-dev`), and added a clarifying comment on the layered `bootstrap.Version` check in `scripts/release/check_readiness.sh`. Items #1/#3/#5/#7 explicitly skipped per prior `/pr-feedback` analysis. |
| 2026-04-28 | pr-benchmark-review-feedback | Addressed Phase 10 PR benchmark feedback: empty benchmark captures now fail closed, parser metric-section handling is stricter, baseline cache save is success-only, workflow comments are clearer, asymmetric `NO_BASELINE=1` parsing was removed, and focused regression tests cover the new contracts. |
| 2026-04-28 | pr-benchmark-nice-to-haves | Added follow-up Phase 10 PR benchmark nice-to-have coverage: pinned the current non-`vs base` metric-header limitation, required clean `yq` stderr for workflow YAML smoke checks, and made stale-output replacement a true two-run orchestrator test. |

### Learning Extractions

| Date | Phase | Output |
|------|-------|--------|
| 2026-04-24 | 08 | `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-LEARNINGS.md` |
| 2026-04-24 | 09 | `.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-LEARNINGS.md` |
| 2026-07-22 | 10 | `.planning/phases/10-lightweight-pr-benchmark-regression-signal/10-LEARNINGS.md` |

### Roadmap Evolution

- Phase 06.1 inserted after Phase 06: Fresh-machine end-to-end bootstrap UAT against live R2 + GitHub Releases (promoted from backlog item 999.4)
- Phase 06.1 execution produced the public bootstrap wrapper, hosted-runner validation workflow, contract tests, and operator runbook updates, and was shipped in PR #17; hosted GitHub Actions execution remains pending
- Phase 07 is now planned as six plans: corpus/oracle foundation, Tier 1 + cold/warm harness, allocator telemetry surface, Tier 2/Tier 3 benchmark consumers, public docs/legal artifacts with committed evidence, and a closeout handoff that defers public patch-release work until the later benchmark-positioning phases
- Phase 07 completed on 2026-04-23 as a benchmark/docs/legal baseline rather than a forced patch release: README, methodology doc, results snapshot, changelog, LICENSE, and NOTICE are now committed, and the closeout explicitly routes Tier 1 ABI work to Phase 08 and benchmark/release recalibration to Phase 09
- Phase 07 learnings were extracted on 2026-04-23 into `.planning/phases/07-benchmarks-v0.1-release/07-LEARNINGS.md`, preserving benchmark positioning decisions, execution lessons, reusable patterns, and surprises for Phase 08 and Phase 09 planning
- Phase 08 added: Low-overhead DOM traversal ABI and specialized Go any materializer. This folds the old 999.6 DOM-materialization ABI idea into the active milestone after Tier 1 diagnostics showed materialization, not parse, dominates the current full-`any` path
- Phase 09 added: Benchmark gate recalibration, Tier 1/2/3 positioning, and post-ABI evidence refresh. This phase exists to replace the invalidated BENCH-07 headline with a measured benchmark story after Phase 08 lands
- Phase 09.1 inserted after Phase 09: Bootstrap artifact and ABI alignment for default installs (URGENT)
- Phase 10 added: Lightweight PR benchmark regression signal, promoted from backlog item 999.8 and explicitly scoped to a cheap Tier 1/Tier 2/Tier 3 `pull_request` benchmark check rather than the heavier Phase 9 release-evidence capture
- Phase 11 added: Upstream simdjson refresh, exact big integers, and production diagnostics
- Phase 12 added: High-value DOM navigation and SIMD utility APIs
- Phase 13 added: Batched On-Demand path extraction
- Phase 14 added: Zero-copy pinned input and borrowed value views
- Phase 15 added: NDJSON and JSONL streaming cursor with parallel parse-many
- Phase 16 added: v0.2 cross-platform evidence, API stabilization, and release
- Backlog items 999.6, 999.7, and 999.8 were retired from the parking lot: 999.6 is now active milestone work under Phase 08, 999.7's diagnostic split was implemented during Phase 07 investigation to justify the new direction, and 999.8 is now active milestone work under Phase 10

### Decisions

Decisions are logged in `.planning/PROJECT.md`. Recent decisions affecting current work:

- Pinned Rust setup lives in a local setup-rust composite action that reads rust-toolchain.toml directly.
- verify-shared-artifact hard-fails when native ABI or minimal_parse smoke commands are missing so export audits stay supplemental.
- Bootstrap release-state rewrites are tested in copied TemporaryDirectory workspaces so unittest never mutates the real repo.
- The shared build action now hands manylinux execution to scripts/release/build_linux_manylinux.sh so workflow YAML does not duplicate docker mount logic or arm64 page-size enforcement.
- linux/arm64 page-size proof runs as both an explicit workflow step and a builder-side guard; the prep workflow also uploads linux-arm64-pagesize.txt with the staged artifact bundle.
- verify_glibc_floor.sh derives the expected pure_simdjson export set from include/pure_simdjson.h instead of freezing a separate symbol list in CI.
- The darwin workflow matrix now carries the expected public asset names and asserts them after packaging, so the bootstrap naming contract is executable in CI.
- The windows release bundle preserves pure_simdjson.dll.lib and a dumpbin /DEPENDENTS report alongside the staged DLL so later plans can reuse that evidence without rebuilding.
- The shared release helpers now emit forward-slash artifact paths and Python-created temp directories so the same bash-based composite actions work on windows runners without a separate packaging path.
- CI-04 now runs through scripts/release/run_native_smoke.sh so every platform executes one shared audit -> ffi_export_surface.c -> minimal_parse.c sequence.
- Staged bootstrap smoke consumes one exact v<version>/<os>-<arch>/<libname> tree assembled from per-platform manifest rows and staged artifacts.
- Both staging jobs rewrite bootstrap release state from the combined manifest before go run so packaged-artifact smoke uses real checksum data.
- Release prep now rewrites version.go, checksums.go, and CHANGELOG.md on a release-prep/v<version> branch before any tag is created.
- Tag publication now starts with a verify-tag-source gate that rejects off-main tags and validates committed bootstrap source state before any build begins.
- The publish workflow signs and verifies the raw staged blobs first, then copies those bytes into flat GitHub Release asset names so R2 and GitHub Releases carry the same signed payload.
- docs/releases.md is the single human-readable source of truth for the Phase 6 release-prep -> main -> tag sequence, required repo configuration, artifact layout, and cosign verification commands.
- scripts/release/check_readiness.sh --strict reuses assert_prepared_state.py --check-source and adds origin/main ancestry checks instead of re-implementing release-state validation in shell.
- scripts/release/check_readiness.sh --strict now also delegates bootstrap/Go/Rust ABI source-state validation to scripts/release/check_bootstrap_abi_state.py using an explicit ABI_MINIMUM_VERSION policy table.
- bootstrap.Version is pinned to 0.1.2 for ABI 0x00010001; internal/bootstrap/checksums.go remains empty in source and runtime digest verification still resolves published SHA256SUMS.
- internal/bootstrap/abi_assertion.go provides a bidirectional compile-time array canary so go test ./internal/bootstrap fails if ffi.ABIVersion drifts from the ABI expected by bootstrap version 0.1.2.
- docs/bootstrap.md now points operators at the release runbook and mirrors the exact xattr Gatekeeper workaround, while Phase 06.1 owns the fresh-runner public validation boundary.
- [Phase 11]: approved_abi12_version: 0.1.5 — Operator-approved next patch for the intermediate ABI 1.2 compatibility artifact.
- [Phase 11]: artifact_role: intermediate Phase 11 ABI 1.2 compatibility artifact — Keeps the bootstrap compatibility artifact distinct from the final v0.2 release.
- [Phase 11]: tag_preflight: refs/tags/v0.1.5 absent after fetching tags — Prevents reusing an existing immutable release identity.
- [Phase 11]: phase16_boundary: not the final v0.2 release unless the user explicitly changes Phase 16 scope — Preserves Phase 16 ownership of the final release.
- [Phase 11]: publication_authorized: false — This checkpoint authorizes source preparation only, with CI remaining the sole publication path.
- [Phase 11]: Keep official simdjson v4.6.4 as the audited base and apply exactly one provenance-recorded positive-overflow patch only to a verified build-output copy. — Preserves reproducible upstream identity while closing the confirmed BigInt gap without a fork, dirty submodule, second source tree, or dependency.
- [Phase 11]: Transport additive native kind 9 through the interim Rust boundary as raw u32. — Avoids constructing an undeclared Rust enum discriminant until Plan 11-07 synchronizes all public ABI mirrors.
- [Phase 11]: Expose TypeBigInt and exact frame text in Wave 1 while leaving GetBigInt and Go materialization to Plan 11-10. — Keeps this compatibility wave green without crossing dependency-ordered public accessor ownership.
- [Phase 11]: Reuse one tracked byte-copy path for strings and BigInts; do not expose borrowed BigInt memory or add a second allocator.
- [Phase 11]: Propagate successful native kind hints directly for roots and descendants, including raw kind 9.
- [Phase 11]: Keep precision-loss for in-range integer-to-float conversion only; BigInt numeric getters return wrong type.
- [Phase 11]: Normalize parser limits once and store the exact effective values in native and registry state. — Keeps primary parsing, the Rust pre-copy gate, and later diagnostic replay aligned.
- [Phase 11]: Establish configured native depth with zero-capacity allocation before first parse. — Configures upstream depth without input-sized construction work.
- [Phase 11]: Clear native diagnostics after handle and busy validation, immediately before the capacity gate. — Prevents Rust-side rejection from inheriting stale parse details.
- [Phase 11]: Replay ordinary syntax failures through raw JSON first, then recursive public-accessor consumption only when the first pass is fully valid. — This preserves the validated v4.6.4 locations without parser reuse or an unbounded retry path.
- [Phase 11]: Represent unknown as UINT64_MAX plus false and transport the known bit independently from the numeric offset. — A proven location at byte zero must remain distinguishable from no upstream location.
- [Phase 11]: Allocate both fresh replay parsers with the exact stored effective capacity and depth, terminating on resource or limit errors. — Diagnostic work must not exceed the parser limits selected by the caller.
- [Phase 11]: Use one native mutex/state for setter, explicit lock, and both parser constructors; a valid construction attempt sets locked before allocation.
- [Phase 11]: Keep native state authoritative for production fallback policy while retaining the Rust forced-fallback seam only for isolated tests.
- [Phase 11]: Compile upstream fallback on every supported architecture so explicit diagnostic fallback remains available on arm64 and x86-64.
- [Phase 11]: Bootstrap 0.1.5 requires ABI 0x00010002; an older ABI cannot claim the newer source identity. — Enforces the version/ABI policy in both directions.
- [Phase 11]: BigInt kind 9 and statuses 9/10 are append-only; all existing numeric values and layouts remain unchanged. — Preserves ABI compatibility while adding the Phase 11 surface.
- [Phase 11]: The 0.1.5 pin is source preparation only; tag-driven CI and Phase 06.1 validation remain future gates. — Prevents source readiness from being mistaken for publication.
- [Phase 11]: Accept ABI major 1 values at or above 0x00010002 only after the complete wrapper-required symbol surface binds.
- [Phase 11]: Loader compatibility classification replaces parser-level ABI re-query and test override state.
- [Phase 11]: Cache installation follows complete binding and successful implementation-name retrieval.

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 09.1] Plan 02 remains the release gate: land the prepared source on origin/main, run strict readiness there, publish v0.1.2 through CI, and dispatch public bootstrap validation before any default-install claim.
- [Phase 02 advisory] Review whether parse-time `simdjson::UNSUPPORTED_ARCHITECTURE` should map to `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED` instead of `PURE_SIMDJSON_ERR_INTERNAL`.
- [Phase 02 advisory] Clean up stale public comments for now-live exports and decide whether `last_error_offset` should remain sentinel-only or surface real offsets.

## Session Continuity

Last session: 2026-07-23T14:07:18.862Z
Stopped at: Completed 11-08-PLAN.md
Resume file: None

**Planned Phase:** 09.1 (Bootstrap artifact and ABI alignment for default installs) — context ready, planning next — 2026-04-24T21:30:00Z

**Ready to Execute:** 09.1 (Bootstrap artifact and ABI alignment for default installs) — 2 verified plans in 2 waves — 2026-04-24T22:15:00Z
