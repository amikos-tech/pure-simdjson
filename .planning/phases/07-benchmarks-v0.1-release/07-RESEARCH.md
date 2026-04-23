# Phase 7: Benchmarks + v0.1 Release - Research

**Researched:** 2026-04-22
**Domain:** Reproducible Go benchmark harness design, correctness-oracle packaging, public-facing release documentation, and release-close sequencing against an already-published `v0.1.0`
**Confidence:** HIGH on benchmark/docs shape, MEDIUM on release-close sequencing because the repo already has a published `v0.1.0` tag

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Tier 1 is strict full-materialization parity. `pure-simdjson` must materialize an equivalent Go tree before timing is counted.
- **D-02:** Tier 2 is schema-shaped end-to-end typed extraction through the current public API.
- **D-03:** Tier 3 is a runnable selective-field benchmark built on the current DOM API and labeled as a `v0.2` placeholder.
- **D-04:** Public benchmark tables only list comparators that actually run on that target/toolchain combination.

### Repo State That Changes Planning

- `git tag --list` already contains `v0.1.0`.
- `internal/bootstrap/version.go` is currently pinned to `0.1.0`.
- `docs/releases.md` and Phase 6 verification both describe `v0.1.0` as a real published release.
- As of 2026-04-22, Phase `06.1` is also shipped on `main`.

### Implication

Phase 7 must not assume it is cutting the first `v0.1.0` tag. If new benchmark/docs/legal artifacts must ship as part of a public tagged release, the safe path is a patch release (`v0.1.1`), not moving the existing `v0.1.0` tag.

### the agent's Discretion

- Exact benchmark file/package layout, as long as `go test -bench` remains the primary entrypoint.
- Exact benchmark snapshot format, as long as cold-start and warm numbers stay separate and reproducible.
- Exact patch-release sequencing after Phase 7 lands, as long as CI remains the only publish path and `v0.1.0` remains immutable.

</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BENCH-01 | Three-tier benchmark harness | Use root-package `*_test.go` benchmarks with explicit `BenchmarkTier1*`, `BenchmarkTier2*`, and `BenchmarkTier3*` families |
| BENCH-02 | Canonical corpus: `twitter`, `canada`, `citm_catalog`, `mesh`, `numbers` | Vendor a repo-local snapshot under `testdata/bench/`; do not read directly from `third_party/` at benchmark runtime |
| BENCH-03 | Baselines: stdlib any, stdlib struct, `simdjson-go`, `sonic`, `goccy/go-json` | Add explicit comparator adapters with per-target availability gates and omission-by-absence reporting |
| BENCH-04 | `benchstat`; cold-start separated from warm | Add dedicated cold-start benchmark families plus a benchstat helper script / make target |
| BENCH-05 | Native allocator stats beside Go alloc counts | Add Rust-side allocation counters with FFI snapshot/reset hooks consumed only by benchmark helpers |
| BENCH-06 | Correctness oracle against vendored JSON test corpus | Vendor a pinned local expectation manifest and run it through a normal Go test |
| BENCH-07 | README documents the public benchmark claim | Generate a benchmark snapshot plus methodology notes, then summarize the claim in `README.md` |
| DOC-01 | `README.md` with install, quick start, platform matrix, benchmark snapshot | Build a consumer-facing README that links to `docs/bootstrap.md`, `docs/releases.md`, and a detailed benchmark methodology doc |
| DOC-06 | Keep-a-Changelog changelog | Keep `## [Unreleased]` during implementation, then roll it into a concrete patch-release entry at release-close |
| DOC-07 | `LICENSE` + `NOTICE` | Add repo-root MIT license and an Apache-2.0 attribution notice for vendored simdjson |

</phase_requirements>

---

## Summary

Phase 7 should be planned as **five sequential workstreams**:

1. Vendor the benchmark and oracle inputs so the phase does not depend on missing or mutable upstream paths.
2. Build the Tier 1 harness, comparator adapters, and cold-start vs warm reporting surface.
3. Add Tier 2 and Tier 3 benches plus native allocator telemetry.
4. Turn the measured results into public docs, legal artifacts, and changelog content.
5. Close the release through the existing CI-only release path, using `v0.1.1` if the new public artifacts must ship in a tagged release.

That split keeps the benchmark code grounded in repo-local inputs first, delays public claims until real numbers exist, and handles the current-state mismatch between the roadmap wording and the already-published `v0.1.0`.

---

## Recommended File Layout

### Benchmark Assets

- `testdata/bench/README.md`
- `testdata/bench/twitter.json`
- `testdata/bench/citm_catalog.json`
- `testdata/bench/canada.json`
- `testdata/bench/mesh.json`
- `testdata/bench/numbers.json`
- `testdata/jsontestsuite/README.md`
- `testdata/jsontestsuite/expectations.tsv`
- `testdata/jsontestsuite/cases/...`

### Benchmark Harness

- `benchmark_fixtures_test.go`
- `benchmark_schema_test.go`
- `benchmark_comparators_test.go`
- `benchmark_fullparse_test.go`
- `benchmark_coldstart_test.go`
- `benchmark_typed_test.go`
- `benchmark_selective_test.go`
- `benchmark_native_alloc_test.go`
- `benchmark_oracle_test.go`
- `scripts/bench/run_benchstat.sh`
- `Makefile`

### Release-Facing Docs

- `README.md`
- `docs/benchmarks.md`
- `CHANGELOG.md`
- `LICENSE`
- `NOTICE`

### FFI/Runtime Touchpoints for BENCH-05

- `src/lib.rs`
- `src/runtime/mod.rs`
- `include/pure_simdjson.h`
- `internal/ffi/bindings.go`
- `tests/rust_shim_minimal.rs`
- `tests/smoke/ffi_export_surface.c`

---

## Findings

### 1. The current submodule does not contain the full Phase 7 corpus

`third_party/simdjson/jsonexamples/` currently contains `twitter.json` and `citm_catalog.json`, but not `canada.json`, `mesh.json`, or `numbers.json`.

**Recommendation:** Phase 7 must vendor the full benchmark corpus into `testdata/bench/` with provenance and checksums written down in `testdata/bench/README.md`. Do not make benchmark execution depend on the exact shape of the vendored submodule.

### 2. The correctness oracle should use a pinned local manifest, not live upstream assumptions

The repo does not currently contain a JSON test suite snapshot. Because BENCH-06 is a normal test requirement, the oracle needs:

- local JSON case files,
- a local expectation manifest, and
- a deterministic Go test entrypoint.

**Recommendation:** Vendor a pinned snapshot into `testdata/jsontestsuite/cases/` and record one tab-separated manifest `expectations.tsv` with exact `accept` / `reject` outcomes. The Go test should treat that manifest as the only runtime source of truth.

### 3. Tier 1 fairness requires explicit tree materialization for `pure-simdjson`

The context already locks this, and the pitfall research reinforces it. The purejson path must not stop at DOM parse plus object walk; it must copy into an equivalent Go representation while the timer is running.

**Recommendation:** Use a single normalization helper that converts DOM values into `map[string]any`, `[]any`, `string`, `bool`, `nil`, `int64`, `uint64`, and `float64`, then compare that workload against stdlib `any` and other DOM-ish baselines.

### 4. Comparator omission must be structural, not cosmetic

`minio/simdjson-go` is not universally available across targets. Even when a comparator is unsupported or temporarily incompatible, the result surface should omit it instead of emitting dead rows or fake `N/A` cells.

**Recommendation:** Build explicit availability gates in `benchmark_comparators_test.go` and carry comparator availability through to any README or `docs/benchmarks.md` table generation.

### 5. BENCH-05 needs first-class allocator instrumentation

Go's `-benchmem` will under-report `pure-simdjson` cost because the real parse work happens in Rust/C++. A doc-only caveat is not enough to satisfy the requirement.

**Recommendation:** Add diagnostic-only FFI exports:

- `pure_simdjson_native_alloc_stats_reset(void)`
- `pure_simdjson_native_alloc_stats_snapshot(pure_simdjson_native_alloc_stats_t *out_stats)`

with a committed header struct:

- `live_bytes`
- `total_alloc_bytes`
- `alloc_count`
- `free_count`

The benchmark helpers can then report `native-bytes/op`, `native-live-bytes`, and `native-allocs/op` as custom metrics.

### 6. Cold-start and warm results should be separate benchmark families

The repo already has a real bootstrap/download story and a reusable `Parser`, so mixing `NewParser()` / first parse cost into steady-state numbers will blur the story.

**Recommendation:** Use distinct benchmark names:

- `BenchmarkColdStart_<Fixture>`
- `BenchmarkWarm_<Fixture>`
- `BenchmarkTier1FullParse_<Fixture>/<Comparator>`
- `BenchmarkTier2Typed_<Fixture>/<Comparator>`
- `BenchmarkTier3SelectivePlaceholder_<Fixture>/<Comparator>`

and keep `benchstat` comparisons on like-for-like families only.

### 7. README claims should be backed by a fuller methodology doc

The README should stay concise. The fairness note, comparator omissions, native allocator caveat, and exact benchmark commands belong in a dedicated `docs/benchmarks.md`.

**Recommendation:** Put the full tier definitions, comparator rules, corpus provenance, and benchstat commands in `docs/benchmarks.md`; keep the README snapshot to a short table plus two explicit caveats.

### 8. Release-close must respect the already-published `v0.1.0`

`v0.1.0` already exists locally and is described as published in the checked-in Phase 6 artifacts. Reusing that tag would be history-rewriting and would also make the benchmark/docs/legal additions ambiguous for consumers.

**Recommendation:** Phase 7 should prepare and publish `v0.1.1` if the new artifacts need a public tag. The existing release flow is already correct for that:

1. update `internal/bootstrap/version.go` to `0.1.1`,
2. move the Phase 7 changelog bullets from `Unreleased` to `[0.1.1]`,
3. run `bash scripts/release/check_readiness.sh --strict --version 0.1.1`,
4. create/push annotated tag `v0.1.1` from `origin/main`,
5. wait for `release.yml`,
6. dispatch `public-bootstrap-validation.yml` for `0.1.1`.

---

## Planner Guidance

- Keep the benchmark harness in the root Go package so it can reuse existing unexported helpers and keep `go test -bench` simple.
- Do not make benchmark execution read from `third_party/simdjson/` directly.
- Keep Tier 3 clearly labeled as a DOM-era placeholder, not an on-demand API claim.
- Use `@latest` or version-agnostic install guidance in the README so the docs are not needlessly rewritten during the patch release.
- Treat the release-close step as non-autonomous because it pushes tags and watches GitHub Actions.
- Do not reintroduce a prep-branch checksum rewrite flow; the checked-in release runbook already forbids that.

---

## Sources

### Primary

- `.planning/ROADMAP.md`
- `.planning/PROJECT.md`
- `.planning/REQUIREMENTS.md`
- `.planning/STATE.md`
- `.planning/phases/07-benchmarks-v0.1-release/07-CONTEXT.md`
- `.planning/research/SUMMARY.md`
- `.planning/research/STACK.md`
- `.planning/research/PITFALLS.md`
- `.planning/research/FEATURES.md`
- `.planning/research/ARCHITECTURE.md`
- `.planning/phases/06-ci-release-matrix-platform-coverage/06-VERIFICATION.md`
- `.planning/phases/06.1-fresh-machine-end-to-end-bootstrap-uat-against-live-r2-githu/06.1-03-SUMMARY.md`
- `docs/releases.md`
- `docs/bootstrap.md`
- `internal/bootstrap/version.go`
- `internal/bootstrap/url.go`
- `CHANGELOG.md`
- `third_party/simdjson/jsonexamples/`
- `third_party/simdjson/LICENSE`
- `third_party/simdjson/LICENSE-MIT`

### Direct Repo Checks

- `git tag --list --sort=creatordate` shows `v0.1.0`
- `git log --oneline --decorate -n 12` shows `main` already includes Phase `06.1`

---
*Research completed: 2026-04-22*
