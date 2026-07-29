---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 17
subsystem: ffi-exception-boundary
tags: [cpp, rust-ffi, exceptions, bad-alloc, abi-contract, tdd]

# Dependency graph
requires:
  - phase: 11-09
    provides: Normative ABI 1.2 status-97 exception contract and generated public-header audits
  - phase: 11-16
    provides: BigInt token-boundary closure and the latest green combined Phase 11 source baseline
provides:
  - Selector-based runtime_error and bad_alloc coverage through the existing hidden native exception seam
  - Normative status-97 mapping for every C++ exception trapped by the shared production catch mapper
  - Combined Rust, contract, Go race/oracle, benchmark-signal, and readiness closure evidence
affects: [12-dom-navigation-and-utilities, native-bridge, ffi-contract, diagnostics]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Extend hidden test seams with fixed selectors instead of adding parallel fault-injection exports
    - Keep thrown exceptions distinct from returned engine error codes at the ABI boundary

key-files:
  created:
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-17-SUMMARY.md
  modified:
    - src/native/simdjson_bridge.h
    - src/native/simdjson_bridge.cpp
    - src/runtime/mod.rs
    - src/lib.rs
    - tests/rust_shim_minimal.rs

key-decisions:
  - "Extend the existing hidden exception seam with fixed selectors rather than add a second symbol or production fault switch."
  - "Map trapped bad_alloc to status 97 while leaving returned simdjson MEMALLOC and explicit internal failures on their existing status-127 path."
  - "Treat the implementation as a correction to the locked contract, keeping the public header, normative document, ABI number, and public symbol surface unchanged."

patterns-established:
  - "Exception containment: every catch category delegates to one status mapper, while parser diagnostics remain best-effort side data."
  - "Fault selection: selector 0 forces runtime_error, selector 1 forces bad_alloc, and unsupported selectors return invalid argument without throwing."

requirements-completed: [UP-01]

# Metrics
duration: 8min
completed: 2026-07-29
---

# Phase 11 Plan 17: C++ Exception Status Contract Summary

**Runtime errors and allocation exceptions now deterministically traverse the existing native catch boundary and return normative public status 97 without changing returned engine-error semantics**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-29T15:27:23Z
- **Completed:** 2026-07-29T15:34:55Z
- **Tasks:** 1
- **Files modified:** 5 task files

## Accomplishments

- Extended the existing non-public exception seam with fixed runtime-error and bad-allocation selectors plus a fail-closed unsupported-selector path.
- Corrected the shared `std::bad_alloc` mapper from internal status 127 to C++ exception status 97 while preserving stderr logging, `noexcept`, catch ordering, and parser diagnostic capture.
- Kept returned `simdjson::MEMALLOC` and explicit internal failures distinct on status 127.
- Passed the combined Phase 11 Rust, generated contract, documentation, Go race/oracle, benchmark-signal, and readiness gates without changing the public ABI.

## Task Commits

The task was committed through one complete RED/GREEN cycle:

1. **Task 1 RED: Add failing bad_alloc exception regression** - `5fec86f` (test)
2. **Task 1 GREEN: Map bad_alloc exceptions to status 97** - `d0bbeb4` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/native/simdjson_bridge.h` - Adds the fixed selector argument to the existing internal exception seam declaration.
- `src/native/simdjson_bridge.cpp` - Selects runtime error, bad allocation, or invalid argument and maps every trapped exception category to status 97.
- `src/runtime/mod.rs` - Threads the selector through the existing native declaration and Rust runtime wrapper.
- `src/lib.rs` - Threads the selector through the hidden Rust test helper without adding a public export.
- `tests/rust_shim_minimal.rs` - Asserts exact statuses for runtime error, bad allocation, and an unsupported selector.

## Decisions Made

- Reused the existing cbindgen-excluded symbol so deterministic coverage did not widen the generated or production-facing API.
- Preserved both production catch macros as delegates to `map_cpp_exception`; parser-aware diagnostic capture remains best effort and cannot change the returned status.
- Left `map_error(simdjson::MEMALLOC)` unchanged because a returned engine code is not a trapped C++ exception.

## TDD Gate Compliance

- **Baseline:** The original runtime-error-only test passed with status 97.
- **RED (`5fec86f`):** The selector seam compiled and ran all three cases; runtime error returned 97, the unsupported selector returned 1, and bad allocation failed for the intended reason with actual status 127 versus expected status 97.
- **GREEN (`d0bbeb4`):** A one-line mapper correction made all three focused tests pass.
- No refactor commit was needed.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The only failure was the required TDD RED result. It was resolved by the planned shared-mapper correction.

## Verification

- `cargo test --locked --test rust_shim_minimal psimdjson_test_force_ -- --test-threads=1` - passed 3/3 selector tests with exact statuses 97, 97, and 1.
- `cargo test --locked --test rust_shim_bigint --test rust_shim_minimal -- --test-threads=1` - passed 8/8 BigInt tests and 24/24 minimal-shim tests on the combined gap-closure source.
- `make verify-contract` - passed `cargo check`, the complete 90-test Rust unit/integration suite, deterministic header generation, 25 header audits, contract rules, and C layout compilation.
- `make verify-docs` - passed the normative documentation checks.
- `cargo build --release --locked` plus fresh-library `go test ./... -race -count=1 -timeout=180s` - passed all four Go packages, including the manifest-complete JSON oracle.
- `bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir <temp>` - exited zero and produced non-empty `head.bench.txt`, `summary.json`, and `markdown.md`.
- `bash scripts/release/check_readiness.sh` - reported `basic release readiness checks passed`.
- `git diff --exit-code 5fec86f^ -- include/pure_simdjson.h docs/ffi-contract.md` - passed; both files retained their pre-plan SHA-256 digests.
- Static acceptance inspection confirmed both catch macros still call `map_cpp_exception`, `simdjson::MEMALLOC` still maps to internal status 127, the existing symbol remains excluded from cbindgen, and no ABI constant or public enum changed.

## Threat and Security Impact

- **T-11-17-01 mitigated:** runtime error, bad allocation, and unknown catches share the normative status-97 mapper and remain contained by `noexcept`.
- **T-11-17-02 mitigated:** thrown allocation failure stays distinct from returned `MEMALLOC`, matching the unchanged status table and caller classification.
- **T-11-17-03 mitigated:** deterministic testing reuses one hidden symbol with two accepted selectors and a fail-closed invalid-selector result.
- **T-11-SC preserved:** no dependency, package install, alternate source, network surface, or publication path was added.
- No unmodeled endpoint, authentication path, schema boundary, file-access surface, or public ABI surface was introduced.

## Known Stubs

None.

## Publication Boundary

- No merge, tag, publication, workflow dispatch, artifact upload, or release-state mutation occurred.
- This plan changes source behavior only; release publication remains outside its scope.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The final local Phase 11 verification gap is closed on the combined source.
- Phase 11 is ready for phase-level verification before Phase 12 proceeds.
- No implementation blocker remains from this plan.

## Self-Check: PASSED

- All five task files and this summary exist.
- RED commit `5fec86f` and GREEN commit `d0bbeb4` are present in repository history.
- Requirement `UP-01`, verification evidence, threat coverage, publication boundaries, and stub tracking are present in this summary.
- The public header and normative contract retain their pre-plan SHA-256 digests.
- The unrelated config and Phase 10 learnings changes remain unstaged and unmodified by this plan.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-29*
