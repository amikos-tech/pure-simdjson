# Phase 7: Benchmarks + v0.1 Release - Pattern Map

**Mapped:** 2026-04-22
**Files analyzed:** 20
**Analogs found:** 16 / 20

---

## File Classification

| New / Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---------------------|------|-----------|----------------|---------------|
| `testdata/bench/*.json` | vendored fixture data | repo-local benchmark input | `third_party/simdjson/jsonexamples/*.json` | direct-source copy |
| `testdata/bench/README.md` | provenance doc | source snapshot -> local corpus | `docs/bootstrap.md` tables + `docs/releases.md` artifact layout sections | docs-structure adapt |
| `testdata/jsontestsuite/expectations.tsv` | oracle manifest | case file -> expected accept/reject | no in-repo analog | new |
| `testdata/jsontestsuite/cases/*` | oracle corpus | repo-local correctness input | no in-repo analog | new |
| `benchmark_fixtures_test.go` | fixture loader helpers | testdata -> benchmark/oracle code | `helpers_test.go`, `parser_test.go` | strong analog |
| `benchmark_schema_test.go` | typed benchmark schemas | fixture bytes -> normalized struct targets | `example_test.go` typed extraction examples | role-match |
| `benchmark_comparators_test.go` | comparator adapters | fixture bytes -> comparator result | `library_loading_test.go` env/availability gating | role-match |
| `benchmark_fullparse_test.go` | Tier 1 benchmark | fixture -> full materialized Go tree | `parser_test.go` DOM walk + conversion checks | role-match |
| `benchmark_coldstart_test.go` | cold/warm benchmark | parser lifecycle -> timing families | `pool_test.go`, `parser_test.go` lifecycle patterns | role-match |
| `benchmark_typed_test.go` | Tier 2 benchmark | fixture -> typed extraction results | `iterator_test.go`, `example_test.go` | strong analog |
| `benchmark_selective_test.go` | Tier 3 placeholder benchmark | fixture -> selected DOM fields only | `iterator_test.go` object traversal + field access | strong analog |
| `benchmark_native_alloc_test.go` | telemetry wrapper | FFI stats -> benchmark metrics | `internal/bootstrap/bootstrap_test.go` custom counter assertions | role-match |
| `benchmark_oracle_test.go` | correctness oracle | manifest + cases -> pass/fail test | `element_fuzz_test.go`, `parser_test.go` negative/positive JSON coverage | strong analog |
| `scripts/bench/run_benchstat.sh` | report helper | benchmark output -> comparison table | `scripts/release/*.sh` argument parsing + strict shell style | exact shell pattern |
| `README.md` | public entrypoint doc | benchmark results + install path -> user docs | `docs/bootstrap.md`, `docs/releases.md` | docs synthesis |
| `docs/benchmarks.md` | methodology doc | harness details -> public benchmark narrative | `docs/bootstrap.md` contract-style explanation | docs-structure adapt |
| `LICENSE` | repo-root license text | project legal baseline | no in-repo analog | standard |
| `NOTICE` | third-party attribution | vendored simdjson -> user-facing notice | `third_party/simdjson/LICENSE` | direct-source adapt |
| `src/lib.rs`, `include/pure_simdjson.h`, `internal/ffi/bindings.go` | FFI telemetry surface | Rust counters -> Go benchmark helpers | Phase 1/2/4 FFI export additions | exact adapt |
| `internal/bootstrap/version.go`, `CHANGELOG.md` | release-close state | source tree -> patch release tag | Phase 6 release-prep contract | exact adapt |

---

## Pattern Assignments

### `benchmark_*_test.go` - CREATE

**Analogs**

- `parser_test.go`
- `iterator_test.go`
- `pool_test.go`
- `example_test.go`

**What to keep**

- Root-package test layout (`package purejson`)
- Table-driven fixture iteration
- Explicit `NewParser()` / `Parse()` / `Close()` lifecycle handling
- Precise error assertions instead of loose string matching

**What to adapt**

- Use benchmark naming families instead of unit-test naming
- Cache fixture bytes in helpers so benchmark code is readable
- Treat unsupported comparators as skipped/omitted, not failing tests

### `benchmark_oracle_test.go` - CREATE

**Analogs**

- `parser_test.go`
- `element_fuzz_test.go`

**Pattern**

- Deterministic repo-local inputs only
- Explicit positive and negative JSON parsing expectations
- Failure output identifies the exact case path

### `scripts/bench/run_benchstat.sh` - CREATE

**Analogs**

- `scripts/release/check_readiness.sh`
- `scripts/release/run_public_bootstrap_smoke.sh`

**Pattern**

- `#!/usr/bin/env bash`
- `set -euo pipefail`
- explicit usage function
- validate required inputs before doing work

### `testdata/bench/README.md` - CREATE

**Analogs**

- `docs/releases.md` published-layout tables
- `docs/bootstrap.md` env-var / platform tables

**Pattern**

- one row per vendored file
- concrete provenance fields
- explain why runtime reads `testdata/bench/` instead of `third_party/`

### `docs/benchmarks.md` - CREATE

**Analogs**

- `docs/bootstrap.md`
- `docs/releases.md`

**Pattern**

- use crisp sections instead of narrative prose
- include exact command lines
- separate "what this number means" from "how to rerun it"

### `README.md` - CREATE

**Analogs**

- `docs/bootstrap.md` for installation/bootstrap contract
- `tests/smoke/go_bootstrap_smoke.go` for the minimal end-to-end code path
- `example_test.go` for public API usage style

**Pattern**

- consumer-facing and short
- link to deeper docs instead of duplicating them
- keep platform list consistent with `internal/bootstrap/url.go::SupportedPlatforms`

### FFI telemetry additions - MODIFY

**Analogs**

- Phase 1 ABI additions in `include/pure_simdjson.h`
- Phase 2/4 bridge and bindings growth in `src/lib.rs`, `src/runtime/mod.rs`, `internal/ffi/bindings.go`
- `tests/smoke/ffi_export_surface.c`

**Pattern**

- add explicit C-compatible structs and out-param exports
- update committed header and smoke/export tests in the same change
- keep names `pure_simdjson_*` and diagnostic-only behavior explicit

### Release-close files - MODIFY

**Analogs**

- `docs/releases.md`
- `internal/bootstrap/version.go`
- `CHANGELOG.md`

**Pattern**

- do not invent a new publish path
- version constant and changelog entry move together
- tag creation is non-autonomous and always anchored on `origin/main`

---

## Existing Code Anchors To Reuse

| Existing File | Why It Matters | Pattern To Reuse |
|---------------|----------------|------------------|
| `tests/smoke/go_bootstrap_smoke.go` | Minimal public API example already exists | README quick-start should mirror this flow, not invent a different API story |
| `example_test.go` | Current public accessor style and naming | Typed benchmark extractors should follow the same field-access patterns |
| `parser_test.go` | Parse lifecycle, error handling, cleanup discipline | Benchmark helpers and oracle tests should reuse the same parser/doc ownership rules |
| `iterator_test.go` | Object and array traversal using current DOM API | Tier 2 and Tier 3 benchmark code should follow this traversal style |
| `helpers_test.go` | Test helper organization in the root package | Fixture caching and benchmark helper functions should live in a similar lightweight helper file |
| `docs/bootstrap.md` | Consumer/operator doc tone and table style | README and `docs/benchmarks.md` should align with its concise contract-first structure |
| `docs/releases.md` | Release-close sequence is already locked | Final Phase 7 release plan should call into it instead of redefining release steps |
| `internal/bootstrap/url.go` | Canonical supported-platform list | README platform matrix must match this file exactly |
| `third_party/simdjson/jsonexamples/` | Current in-repo source for two canonical corpus files | Copy source bytes from here into `testdata/bench/` where available |
| `third_party/simdjson/LICENSE` and `LICENSE-MIT` | Upstream licensing facts | `NOTICE` should cite these exact upstream files |

---

## Planner Guidance

- Prefer root-package benchmark files over a separate benchmark package.
- Keep benchmark helper write scopes narrow so plans can stay mostly non-overlapping.
- Treat `testdata/bench/` and `testdata/jsontestsuite/` as committed assets, not generated-at-runtime directories.
- If allocator telemetry adds new public FFI exports, always update `include/pure_simdjson.h` and `tests/smoke/ffi_export_surface.c` in the same task.
- Use `README.md` for the short benchmark snapshot and `docs/benchmarks.md` for the full methodology and fairness note.
