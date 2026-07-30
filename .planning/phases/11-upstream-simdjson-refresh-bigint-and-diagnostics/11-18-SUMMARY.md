---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 18
subsystem: ffi-exception-boundary
tags: [cpp, rust-ffi, bad-alloc, noexcept, subprocess, abi-contract, tdd]

# Dependency graph
requires:
  - phase: 11-17
    provides: Existing hidden selector seam and normative status-97 C++ exception mapper
provides:
  - Allocation-safe parser-aware bad-allocation diagnostic capture
  - Selector 3 coverage through the production parser-aware catch macro
  - Exact-filter subprocess proof of status 97, sentinel integrity, and process survival
affects: [12-dom-navigation-and-utilities, native-bridge, ffi-contract, diagnostics]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Keep memory-pressure diagnostics allocation-free until inside a catch-all guard
    - Prove noexcept exception containment in an exact-filter child process

key-files:
  created:
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-18-SUMMARY.md
  modified:
    - src/native/simdjson_bridge.cpp
    - tests/rust_shim_minimal.rs

key-decisions:
  - "Record parser-aware bad allocation as the fixed `std::bad_alloc` view so diagnostic allocation can occur only inside the existing catch-all guard."
  - "Route selector 3 through a private stack-parser helper and the production parser-aware catch macro while retaining the existing hidden symbol and Rust wrapper."
  - "Treat output-sentinel mutation as internal status 127 while preserving exact status 97 for the contained exception."

patterns-established:
  - "Best-effort diagnostics: a diagnostic failure is swallowed and cannot prevent the shared exception status mapper from running."
  - "Process-survival proof: the parent requires a successful exact-filter child, one passed test, and a marker emitted only after the status assertion."

requirements-completed: [UP-01]

# Metrics
duration: 8min
completed: 2026-07-29
---

# Phase 11 Plan 18: Parser-Aware bad_alloc Containment Summary

**Parser-aware allocation failures now reach status 97 without an escaping diagnostic allocation, with child-process proof that the output sentinel stays unchanged and execution survives**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-29T17:36:07Z
- **Completed:** 2026-07-29T17:44:33Z
- **Tasks:** 1
- **Files modified:** 2 task files

## Accomplishments

- Removed the allocating temporary from the parser-aware `std::bad_alloc` diagnostic overload and passed the fixed diagnostic directly into the existing guarded assignment path.
- Added selector 3 to the existing hidden seam through a private helper that uses a valid stack-local parser and the same parser-aware catch macro as production parsing.
- Added an exact-filter subprocess regression that proves status 97, unchanged output sentinel, one child test, process survival, and arrival at a post-assertion success marker.
- Preserved selectors 0/1/2, returned `simdjson::MEMALLOC` and explicit internal status 127, BigInt behavior, ABI/header/docs, dependencies, bootstrap state, and release behavior.

## Task Commits

The task was committed through a genuine RED/GREEN sequence:

1. **Task 1 RED: Add failing parser bad_alloc survival regression** - `ba168e7` (test)
2. **Task 1 GREEN: Contain parser bad_alloc diagnostics** - `c18103e` (fix)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/native/simdjson_bridge.cpp` - Uses an allocation-free bad-allocation diagnostic and routes selector 3 through the production parser-aware catch macro with a fail-closed sentinel check.
- `tests/rust_shim_minimal.rs` - Runs selector 3 in an exact-filter child and requires exact status, one-test, survival, and post-assertion evidence.
- `.planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-18-SUMMARY.md` - Records implementation, verification, scope preservation, and threat evidence.

## Decisions Made

- Used the fixed `std::bad_alloc` diagnostic instead of concatenating `error.what()` because even constructing the diagnostic argument must be safe during allocation exhaustion.
- Kept the new path inside the existing hidden selector symbol; no header declaration, Rust wrapper, public binding, cbindgen entry, or production fault switch was added.
- Kept thrown exceptions distinct from returned engine failures: contained C++ exceptions return 97, while returned `MEMALLOC` and sentinel-contract failures remain 127.

## TDD Gate Compliance

- **Baseline:** The three existing selector tests passed with exact statuses 97, 97, and 1.
- **RED (`ba168e7`):** The new parent test failed for the intended reason. Its exact-filter child ran one test, selector 3 returned invalid-argument status 1 instead of 97, the child exited 101, and the post-assertion marker was not reached.
- **GREEN (`c18103e`):** The same focused command passed 4/4 tests. Direct child-mode evidence reported exactly one passed test and printed `PARSER_BAD_ALLOC_CHILD_OK` after the exact status-97 assertion.
- No refactor commit was needed.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- An optional repository-wide `cargo fmt --check` exposed pre-existing formatting drift in files outside this plan. No unrelated formatting rewrite was made, and the scoped changes passed diff checks and all required build/test gates.
- REVIEW.md warnings WR-01 and WR-02 were intentionally not absorbed.

## Verification

- `cargo test --locked --test rust_shim_minimal psimdjson_test_force_ -- --nocapture --test-threads=1` - passed 4/4 direct and subprocess exception regressions.
- Exact child-mode run - passed exactly one test, returned status 97, and emitted `PARSER_BAD_ALLOC_CHILD_OK`.
- `cargo test --locked --test rust_shim_bigint --test rust_shim_minimal --test rust_shim_limits --test rust_shim_diagnostics -- --test-threads=1` - passed 47/47 tests.
- `make verify-contract && make verify-docs` - passed 91 Rust tests, deterministic header generation, 25 header audits, C layout compilation, and documentation checks.
- Fresh `cargo build --release --locked` library plus `go test ./... -race -count=1 -timeout=180s` - passed all four Go packages.
- Fresh-library `TestJSONTestSuiteOracle` - passed the complete correctness oracle.
- Release-source contract suites - passed 43/43 Python tests without release-state changes.
- PR benchmark signal - exited zero with non-empty 72-line benchmark output, 8-line JSON summary, and 5-line Markdown report.
- `bash scripts/release/check_readiness.sh` - reported basic release readiness checks passed.
- Static source checks confirmed direct fixed diagnostic capture, selector 3's production macro path, sentinel fail-closed behavior, and unchanged status-127 `MEMALLOC` mapping.
- Pre-plan-to-GREEN diffs were clean for the public header, normative contract, Rust/native wrappers, Go FFI, dependency locks, bootstrap state, release scripts, and workflows.

## Threat and Security Impact

- **T-11-18-01 through T-11-18-03 mitigated:** parser-aware bad allocation performs no throwing diagnostic work before the guard, reaches the shared mapper, and survives in a child process.
- **T-11-18-04 mitigated:** the private selector returns internal status 127 if its success-only output sentinel changes; the child accepts only status 97.
- **T-11-18-05 mitigated:** selector 3 reuses the cbindgen-excluded hidden seam and introduces no public symbol or production control.
- **T-11-18-06 mitigated:** the parent rejects signal/non-zero termination, missing one-test evidence, and a missing post-assertion marker.
- **T-11-18-07 and T-11-SC preserved:** returned engine errors, ABI values, dependencies, audited source, and release behavior remain unchanged.
- No unmodeled network endpoint, authentication path, schema boundary, file-access surface, or public ABI surface was introduced.

## Known Stubs

None.

## Publication Boundary

- No branch creation or switch, merge, tag, publication, workflow dispatch, artifact upload, or release-state mutation occurred.
- The repository release guidance remained a constraint only; this plan performs no release action.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The parser-aware allocation-exception containment gap is closed on the combined Phase 11 source.
- Phase 11 is ready for phase-level re-verification before Phase 12 proceeds.
- WR-01 and WR-02 remain outside this plan exactly as directed.

## Self-Check: PASSED

- Both task files and this summary exist.
- RED commit `ba168e7` precedes GREEN commit `c18103e` in repository history.
- The combined task diff contains only `src/native/simdjson_bridge.cpp` and `tests/rust_shim_minimal.rs`; no tracked files were deleted.
- All acceptance criteria, plan-level verification commands, threat mitigations, publication boundaries, and stub checks are recorded above.
- Unrelated `.planning/config.json` and Phase 10 learnings changes remain unstaged and unmodified by this plan.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-29*
