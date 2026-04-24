---
phase: 7
phase_name: "benchmarks-v0.1-release"
project: "pure-simdjson"
generated: "2026-04-23"
counts:
  decisions: 11
  lessons: 10
  patterns: 10
  surprises: 8
missing_artifacts: []
---

# Phase 7 Learnings: benchmarks-v0.1-release

## Decisions

### Commit Benchmark Inputs Locally
Benchmark fixtures must be loaded only by exact filename from `testdata/bench/`, so benchmark execution cannot drift back to `third_party/` paths or network inputs.

**Rationale:** Phase 7 needed reproducible benchmark inputs with pinned provenance before publishing benchmark numbers or README claims.
**Source:** 07-01-SUMMARY.md

---

### Use A Manifest-Driven Correctness Oracle
`TestJSONTestSuiteOracle` uses `expectations.tsv` as the only runtime source of truth and fails on both missing and extra vendored case files.

**Rationale:** The oracle needs deterministic expectations and must detect manifest/filesystem drift before parsing any case.
**Source:** 07-01-SUMMARY.md

---

### Centralize Comparator Availability
Comparator availability is registered once and split by build tags, with unsupported comparators omitted structurally and accompanied by human-readable reason strings.

**Rationale:** Published benchmark tables should not contain fake rows, zero values, or build failures when target-specific libraries are unavailable.
**Source:** 07-02-SUMMARY.md

---

### Define Cold Start Narrowly
Cold-start benchmarks mean the first `Parse` after `NewParser` inside an already loaded process; bootstrap and download time are excluded.

**Rationale:** Parser lifecycle costs and binary acquisition costs are different questions and need separate measurement boundaries.
**Source:** 07-02-SUMMARY.md

---

### Time Full Materialization For Tier 1
Tier 1 full-materialization benchmarks keep DOM parse and recursive conversion to ordinary Go values inside the timed section for `pure-simdjson`.

**Rationale:** This makes the Tier 1 comparator work equivalent to `encoding/json` into `any` instead of timing only the faster parse portion.
**Source:** 07-02-SUMMARY.md

---

### Scope Native Allocator Telemetry To The Cdylib Path
Native allocation reporting is scoped to allocations routed through the shim/simdjson cdylib path and does not claim process-wide or Go heap totals.

**Rationale:** Benchmark allocator metrics need to be concrete and honest without overclaiming what the C++ telemetry can observe.
**Source:** 07-03-SUMMARY.md

---

### Use Epoch-Based Telemetry Reset Semantics
Native allocator telemetry reset excludes pre-existing live allocations from later snapshots instead of treating reset as a process-wide heap reset.

**Rationale:** Benchmark helpers need clean per-run counters without invalidating persistent native allocations that existed before the benchmark epoch.
**Source:** 07-03-SUMMARY.md

---

### Ship ABI Changes With Contract Coverage
New public ABI structs and exports must be reflected in cbindgen output, contract docs, header audit rules, Rust integration tests, and C export smoke tests in the same plan.

**Rationale:** The allocator diagnostic surface needed to remain auditable across the bridge, Rust, generated header, Go binding, and smoke-test layers.
**Source:** 07-03-SUMMARY.md

---

### Keep Tier 2 On Shared Schema Structs
Tier 2 uses shared schema structs across supported comparators; `pure-simdjson` reaches those structs through DOM traversal rather than adding benchmark-only decode APIs.

**Rationale:** Shared schema targets keep comparator fairness while preserving the existing public API boundary.
**Source:** 07-04-SUMMARY.md

---

### Label Tier 3 As A DOM-Era Placeholder
Tier 3 selective benchmarks remain explicitly scoped as a DOM-era placeholder and do not imply a shipped On-Demand or path-query API.

**Rationale:** The benchmark harness can measure selective traversal without overpromising an API surface planned for later work.
**Source:** 07-04-SUMMARY.md

---

### Close Phase 7 Without A New Tag
Phase 7 was closed as a benchmark/docs/legal baseline without creating a new release tag; Phase 8 owns low-overhead traversal/materialization ABI work and Phase 9 owns benchmark recalibration plus any later release decision.

**Rationale:** The current Tier 1 full-`any` result is not a supported public headline, so tagging a patch release from this baseline would have overstated the benchmark story.
**Source:** 07-06-SUMMARY.md

---

## Lessons

### Tier 1 Is Not The Current Headline
Tier 1 full `any` materialization on `darwin/arm64` remains slower than `encoding/json + any` on the current DOM ABI.

**Context:** The measured results were `0.21x` on `twitter.json`, `0.20x` on `citm_catalog.json`, and `0.17x` on `canada.json`.
**Source:** 07-05-SUMMARY.md

---

### Materialization Dominates The Tier 1 Cost
Tier 1 diagnostics showed the remaining cost is dominated by materialization rather than parse.

**Context:** The diagnostic split showed `parse-only` as small relative to `pure-simdjson-full` and `materialize-only` across the Tier 1 corpora.
**Source:** 07-05-SUMMARY.md

---

### Tier 2 Typed Extraction Is The Strength Story
Tier 2 typed extraction is the current performance strength for the library.

**Context:** Published medians were `14.52x`, `11.45x`, and `10.08x` faster than `encoding/json` struct decoding on `twitter.json`, `citm_catalog.json`, and `canada.json`.
**Source:** 07-05-SUMMARY.md

---

### Tier 3 Selective Traversal Is Also Strong
The DOM-era Tier 3 selective placeholder rows produced strong results despite not being a future On-Demand API.

**Context:** The measured rows were `15.19x` faster than `encoding/json` struct decoding on `twitter.json` and `20.05x` faster on `citm_catalog.json`.
**Source:** 07-05-SUMMARY.md

---

### Local Minio Parity Could Not Be Measured
Local x86_64 parity with `minio/simdjson-go` remained unavailable in this phase.

**Context:** The Rosetta-backed amd64 run aborted with `Host CPU does not meet target specs`.
**Source:** 07-05-SUMMARY.md

---

### Sequential Git Operations Matter In Autonomous Execution
Overlapping git operations caused a transient `.git/index.lock` during Plan 07-02.

**Context:** Retrying the commit sequentially resolved the issue without changing plan scope.
**Source:** 07-02-SUMMARY.md

---

### Public And Private ABI Symbols Need Explicit Separation
The generated header briefly leaked private `psimdjson_*` bridge hooks.

**Context:** The fix was to add the bridge symbols to the cbindgen exclude list and regenerate the public header.
**Source:** 07-03-SUMMARY.md

---

### Verification Commands Must Match Tool Defaults
The direct header-audit command in the plan did not match the existing CLI behavior.

**Context:** `tests/abi/check_header.py` had to default to all registered rules when no `--rule` flags are supplied so the planned command could run directly.
**Source:** 07-03-SUMMARY.md

---

### Shared Schemas Need To Cover The Actual Workload
The initial Twitter benchmark schema did not include all fields required by the Tier 2 and Tier 3 workloads.

**Context:** `benchTwitterRow` was expanded with the missing boolean and string fields before the typed/selective benchmark families could remain honest.
**Source:** 07-04-SUMMARY.md

---

### Grep-Based Acceptance Gates Need Stable Text Anchors
Centralizing native metric reporting into one helper removed the literal metric names from Tier 1 files and broke the plan's grep-based acceptance gate.

**Context:** Explicit comments restored `native-bytes/op`, `native-allocs/op`, and `native-live-bytes` in the Tier 1 and cold/warm benchmark files while keeping the helper centralized.
**Source:** 07-04-SUMMARY.md

---

## Patterns

### Sync.Once Fixture Caching
Benchmark and oracle helpers memoize committed testdata with `sync.Once`.

**When to use:** Use this for benchmark fixtures and oracle manifests that are immutable during a test run but expensive or noisy to reload for every iteration.
**Source:** 07-01-SUMMARY.md

---

### Manifest-To-Filesystem Symmetry Check
The oracle validates manifest-to-filesystem symmetry before parsing cases.

**When to use:** Use this when a committed manifest is the source of truth and silent extra or missing corpus files would invalidate correctness coverage.
**Source:** 07-01-SUMMARY.md

---

### Registry-Driven Comparator Selection
Benchmark comparators are keyed and selected through one registry surface.

**When to use:** Use this when several benchmark families need the same availability rules, omission reasons, and comparator names.
**Source:** 07-02-SUMMARY.md

---

### Adapter And Stub Build-Tag Pairs
Target-constrained comparators use real adapter files on supported targets and stub files with omission reasons elsewhere.

**When to use:** Use this for optional dependencies whose imports or native paths do not compile across every supported platform.
**Source:** 07-02-SUMMARY.md

---

### Stable Benchstat Names By Fixture Family
Tier 1 benchmarks use per-fixture top-level benchmark functions with comparator sub-benchmarks.

**When to use:** Use this when benchmark rows need stable names for benchstat comparisons and public documentation.
**Source:** 07-02-SUMMARY.md

---

### Reset-Snapshot Native Metric Helper
Benchmark-side native allocator metrics reset immediately before measured work and snapshot immediately after it, then report custom metrics through `b.ReportMetric`.

**When to use:** Use this for benchmark-only diagnostic counters that must align with Go `-benchmem` rows.
**Source:** 07-04-SUMMARY.md

---

### ABI Contract Update Bundle
Public ABI additions are updated across generated headers, contract docs, audit scripts, Rust tests, C smoke tests, and Go bindings in one change.

**When to use:** Use this whenever a public FFI symbol, struct, or diagnostic surface is added.
**Source:** 07-03-SUMMARY.md

---

### Shared-Schema Comparator Fairness
Typed benchmarks decode every comparator into the same schema structs.

**When to use:** Use this when benchmark results could otherwise be biased by comparator-specific struct shapes or extraction shortcuts.
**Source:** 07-04-SUMMARY.md

---

### Placeholder Naming For Future API Boundaries
The selective benchmark family uses `SelectivePlaceholder` in names and comments to avoid implying a shipped On-Demand API.

**When to use:** Use this when measuring a behavior that is useful now but adjacent to an intentionally deferred public API.
**Source:** 07-04-SUMMARY.md

---

### Evidence-Backed Public Benchmark Docs
README and benchmark methodology claims link committed result snapshots and raw benchmark outputs.

**When to use:** Use this for public performance claims that must remain auditable after the measurement environment changes.
**Source:** 07-05-PLAN.md

---

## Surprises

### Materialization Was The Main Tier 1 Bottleneck
The diagnostic split showed materialization dominated Tier 1 cost rather than parsing itself.

**Impact:** The follow-up performance work was routed to low-overhead traversal/materialization ABI work in Phase 8.
**Source:** 07-05-SUMMARY.md

---

### The Phase 7 Release Premise Changed
The phase did not produce a new public patch tag even though the original phase name included release framing.

**Impact:** Phase 7 closed as a benchmark/docs/legal baseline, with release recalibration moved to Phase 9 after Phase 8 performance work.
**Source:** 07-06-SUMMARY.md

---

### Rosetta Could Not Provide The x86_64 Comparator Check
The local amd64 `minio/simdjson-go` parity run failed under Rosetta.

**Impact:** The results snapshot had to mark x86_64 minio parity as unavailable on this machine.
**Source:** 07-05-SUMMARY.md

---

### Private Bridge Symbols Reached The Public Header
`cbindgen` initially emitted private bridge declarations into `include/pure_simdjson.h`.

**Impact:** The public/private ABI split needed an explicit exclude-list update before verification could pass.
**Source:** 07-03-SUMMARY.md

---

### Header Audit Defaults Were Too Narrow
The header-audit script required explicit `--rule` flags while the plan expected a direct no-flag command.

**Impact:** The CLI default was changed to run all registered rules, making both Makefile and direct planner verification paths work.
**Source:** 07-03-SUMMARY.md

---

### Benchmark Schema Was Missing Workload Fields
The shared Twitter schema lacked fields required by the actual Tier 2 and Tier 3 extraction workloads.

**Impact:** The schema had to expand before the benchmark families could preserve shared-schema fairness.
**Source:** 07-04-SUMMARY.md

---

### Centralizing Metrics Broke Textual Verification
Moving metric reporting into a helper removed grep-visible metric strings from Tier 1 benchmark files.

**Impact:** The implementation kept centralized metric reporting but added explicit metric-name comments to satisfy the documented verification contract.
**Source:** 07-04-SUMMARY.md

---

### Git Index Locking Surfaced During Parallel Work
Two git operations overlapped during Plan 07-02 and temporarily blocked a commit.

**Impact:** The work completed after retrying the commit sequentially, with no scope change.
**Source:** 07-02-SUMMARY.md
