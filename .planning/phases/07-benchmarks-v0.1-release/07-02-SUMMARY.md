---
phase: 07-benchmarks-v0.1-release
plan: "02"
subsystem: testing
tags: [benchmarks, benchstat, sonic, simdjson-go, go-json, go-test]
requires:
  - phase: 07-01
    provides: vendored benchmark fixtures, oracle manifest, and fixture-loading helpers
provides:
  - shared benchmark schema types for twitter, citm_catalog, and canada
  - comparator registry with build-tag-safe omission reporting
  - Tier 1 full-materialization benchmark families
  - cold-start and warm parser benchmark families plus benchstat helper commands
affects: [07-03, 07-04, phase-07-benchmark-docs, benchmark-reporting]
tech-stack:
  added: [github.com/bytedance/sonic, github.com/goccy/go-json, github.com/minio/simdjson-go, golang.org/x/perf/cmd/benchstat]
  patterns:
    - registry-driven benchmark comparator selection
    - build-tag-safe comparator omission files
    - per-fixture benchmark family wrappers for stable benchstat names
key-files:
  created:
    - benchmark_schema_test.go
    - benchmark_comparators_test.go
    - benchmark_comparators_minio_amd64_test.go
    - benchmark_comparators_minio_stub_test.go
    - benchmark_comparators_sonic_supported_test.go
    - benchmark_comparators_sonic_stub_test.go
    - benchmark_fullparse_test.go
    - benchmark_coldstart_test.go
    - scripts/bench/run_benchstat.sh
  modified:
    - go.mod
    - go.sum
    - Makefile
key-decisions:
  - "Comparator availability is registered once and split by build tags so unsupported libraries are omitted structurally with human-readable reasons."
  - "Tier 1 benchmarks use per-fixture top-level benchmark functions with comparator sub-benchmarks to keep names stable for benchstat and README reporting."
  - "Cold-start means first Parse after NewParser inside an already loaded process; bootstrap and download time stay out of this benchmark family."
patterns-established:
  - "Benchmark registry: every comparator key is canonicalized in one file and later benchmark plans can consume availability and omission reasons from the same surface."
  - "Target-aware comparators: architecture- or toolchain-constrained adapters live in dedicated build-tagged files paired with explicit stub registrations."
requirements-completed: [BENCH-01, BENCH-03, BENCH-04]
duration: 15min
completed: 2026-04-22
---

# Phase 7 Plan 02: Tier 1 harness and cold-warm benchmark surface Summary

**Shared benchmark comparator registry with target-aware omissions, Tier 1 full-materialization benchmark families, and cold/warm benchstat command surfaces for the phase 7 fixtures**

## Performance

- **Duration:** 15 min
- **Started:** 2026-04-22T19:19:00Z
- **Completed:** 2026-04-22T19:33:46Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments

- Added the shared schema and comparator surface for `pure-simdjson`, `encoding/json` any/struct, `minio/simdjson-go`, `bytedance/sonic`, and `goccy/go-json`.
- Split `minio` and `sonic` adapters behind compile-time-safe build tags so unsupported targets omit comparators with explicit reason strings instead of breaking `go test ./...`.
- Added runnable Tier 1 full-materialization benchmarks plus separate `BenchmarkColdStart_*` and `BenchmarkWarm_*` families.
- Added repo-local benchmark command entrypoints in `Makefile` and a strict `scripts/bench/run_benchstat.sh` helper.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add comparator dependencies, shared schema definitions, and compile-time-safe availability gates** - `52d6f32` (feat)
2. **Task 2: Add Tier 1 full-materialization benchmarks, cold/warm families, and benchstat commands** - `709e971` (feat)

## Files Created/Modified

- `benchmark_schema_test.go` - Shared typed benchmark targets for twitter, citm_catalog, and canada fixtures.
- `benchmark_comparators_test.go` - Canonical comparator keys, registry, omission surface, and full-materialization helpers.
- `benchmark_comparators_minio_amd64_test.go`, `benchmark_comparators_minio_stub_test.go` - `minio/simdjson-go` adapter and non-`amd64` omission path.
- `benchmark_comparators_sonic_supported_test.go`, `benchmark_comparators_sonic_stub_test.go` - `sonic` native-path adapter and unsupported-toolchain omission path.
- `benchmark_fullparse_test.go` - `BenchmarkTier1FullParse_*` families with comparator sub-benchmarks.
- `benchmark_coldstart_test.go` - `BenchmarkColdStart_*` and `BenchmarkWarm_*` parser lifecycle benchmarks.
- `scripts/bench/run_benchstat.sh` - strict `benchstat` wrapper with argument validation and install guidance.
- `Makefile` - `bench-phase7`, `bench-phase7-cold`, and `bench-phase7-compare` targets.
- `go.mod`, `go.sum` - comparator dependency set reconciled to the new benchmark imports.

## Decisions Made

- Comparator omission is data-driven from one registry so later benchmark plans can reuse the same availability and reason strings.
- `sonic` is treated as available only on the dependency's native-supported build tags, even though its compatibility fallback exists, so published comparisons do not accidentally benchmark stdlib fallback behavior under the `sonic` label.
- Full-materialization fairness for `pure-simdjson` is enforced by timing DOM parse plus recursive conversion to ordinary Go values inside the comparator function itself.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- A transient `.git/index.lock` blocked the first Task 2 commit attempt because two git operations overlapped. Retrying the commit sequentially resolved it without changing plan scope.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan `07-03` can consume the shared comparator registry and benchmark naming surface directly.
- Tier 1 and cold/warm benchmark commands now exist, so later plans can add Tier 2/Tier 3 families and allocator telemetry without revisiting command layout.

## Self-Check: PASSED
