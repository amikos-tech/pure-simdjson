---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 09
subsystem: ffi-abi-contract
tags: [rust, cbindgen, c-abi, contract-tests, dom-navigation]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Additive ABI 1.2 precedent, stable public layouts, and the Phase 16 release boundary
provides:
  - Native and generated-C ABI version 0x00010003
  - Additive invalid-path and index-out-of-range statuses at values 11 and 12
  - Independent Rust, C, Python, and documentation pins for the ABI 1.3 numeric foundation
affects: [12-01-native-navigation, 12-02-container-helpers, 12-03-wildcard-navigation, 12-04-simd-utilities, 12-10-abi-smoke, 12-11-go-bindings]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Keep the Rust constant and hand-written cbindgen macro synchronized through independent tests
    - Reserve additive status values across Rust, generated C, static assertions, Python fixtures, and normative docs

key-files:
  created:
    - .planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/12-09-SUMMARY.md
  modified:
    - src/lib.rs
    - cbindgen.toml
    - include/pure_simdjson.h
    - tests/abi/handle_layout.c
    - tests/abi/check_header.py
    - tests/abi/test_check_header.py
    - docs/ffi-contract.md

key-decisions:
  - "Assign INVALID_PATH to 11 and INDEX_OUT_OF_RANGE to 12, leaving 13-31 as the remaining additive status band."
  - "Keep ABI 1.2 loader requirements as the clearly identified Phase 11 baseline until Plan 12-10 defines the complete ABI 1.3 mandatory symbol surface."
  - "Regenerate the public header from Rust and cbindgen sources instead of editing it by hand."

patterns-established:
  - "Numeric ABI growth: every new value is pinned independently in Rust behavior tests, generated C, C static assertions, Python contract fixtures, and normative documentation."
  - "Historical compatibility text remains intact while current-version statements advance explicitly."

requirements-completed: [DOM-01, DOM-02, DOM-03, DOM-04, UTIL-01, UTIL-02]

# Metrics
duration: 5min
completed: 2026-07-31
---

# Phase 12 Plan 09: ABI 1.3 Numeric Foundation Summary

**ABI 1.3 is synchronized across Rust, generated C, static assertions, Python fixtures, and the normative contract, with navigation statuses pinned at 11 and 12**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-31T09:55:55Z
- **Completed:** 2026-07-31T10:01:16Z
- **Tasks:** 2
- **Files modified:** 7 task files

## Accomplishments

- Advanced the authoritative Rust constant and generated C macro from ABI 1.2 to ABI 1.3 without changing any existing enum value, signature, or public layout.
- Added `PURE_SIMDJSON_ERR_INVALID_PATH = 11` and `PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE = 12` as append-only navigation statuses.
- Pinned the new numbers through Rust unit tests, C static assertions, Python header-contract fixtures, and the normative error table.
- Preserved the historical ABI 1.2 loader baseline while reserving the complete ABI 1.3 mandatory-symbol statement for Plan 12-10.

## Task Commits

Each task was committed atomically:

1. **Task 1: Pin the Rust/generated-C ABI 1.3 numeric foundation** - `0b7c8e0` (feat)
2. **Task 2: Synchronize Python contract fixtures and normative status documentation** - `20ec063` (docs)

Plan metadata is committed with this summary.

## Files Created/Modified

- `src/lib.rs` - Advances the ABI constant and appends statuses 11/12 with exact unit-test pins.
- `cbindgen.toml` - Advances the hand-written generated-header ABI macro source.
- `include/pure_simdjson.h` - Regenerated ABI 1.3 public enum and macro declarations.
- `tests/abi/handle_layout.c` - Statically pins ABI 1.3 and both new statuses while retaining all prior layout assertions.
- `tests/abi/check_header.py` - Requires the ABI 1.3 macro in generated-header validation.
- `tests/abi/test_check_header.py` - Updates the independent diagnostic-surface fixture to ABI 1.3.
- `docs/ffi-contract.md` - Defines the current packed ABI, status meanings, and remaining reserved numeric bands.
- `.planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/12-09-SUMMARY.md` - Records execution and verification evidence.

## Decisions Made

- Used the research-selected reserved-band values 11 and 12 so navigation failures remain distinguishable without renumbering any existing status.
- Updated the remaining reserved band to 13-31 so the normative document cannot simultaneously assign and reserve values 11/12.
- Left the Phase 11 ABI 1.2 mandatory-symbol text intact; Plan 12-10 owns the complete ABI 1.3 loader statement after all Phase 12 exports exist.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Verification

- `cargo test --lib abi_version_getter_returns_the_pinned_constant` - passed the exact constant/getter ABI 1.3 check.
- `cargo test --lib public_enum_values_are_append_only` - passed every existing numeric pin plus new values 11/12.
- `make generate-header` followed by `cc -Iinclude tests/abi/handle_layout.c -c -o target/phase12-handle-layout.o` - regenerated the header and compiled all ABI/layout assertions.
- `python3 tests/abi/test_check_header.py` - passed 25/25 contract-fixture tests.
- `python3 tests/abi/check_header.py --rule diag-surface include/pure_simdjson.h` - accepted the ABI 1.3 generated header.
- `make verify-docs` - passed.
- `make verify-contract` - passed `cargo check`, 91 Rust tests, deterministic header regeneration, 25 Python header audits, every header rule, and C layout compilation.

## Threat and Security Impact

- **T-12-ABI-NUM mitigated:** independent Rust, cbindgen, generated-C, C-static, Python, and documentation checks agree on ABI 1.3 and statuses 11/12.
- **T-12-ABI-WAVE preserved:** the ABI foundation remains branch-local and unpublished; later Phase 12 plans complete the symbol surface before the project squash merge and release work.
- **T-12-SC unchanged:** no package, dependency, vendored source, or toolchain input changed.
- No unmodeled network endpoint, authentication path, schema boundary, file-access pattern, or trust boundary was introduced.

## Known Stubs

None.

## Publication Boundary

- No merge, tag, publication, workflow dispatch, artifact upload, or release-state mutation occurred.
- This plan establishes contract numbers only; it does not claim that the full ABI 1.3 symbol surface is complete or published.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 12-01 can implement native `AtPointer`/`AtPath` against the committed ABI 1.3 numeric foundation.
- Plans 12-02 through 12-04 can reuse statuses 11/12 without reopening the error-number decision.
- Plan 12-10 remains responsible for the complete mandatory-symbol smoke and ABI 1.3 loader documentation.

## Self-Check: PASSED

- All seven task files and this summary exist.
- Task commits `0b7c8e0` and `20ec063` exist in repository history in the required order.
- All task acceptance criteria and the plan-level `make verify-contract` gate passed.
- No tracked files were deleted, no dependency changed, and the stub scan found no goal-blocking placeholder.
- The orchestrator-owned `.planning/STATE.md` start marker remained preserved for the prescribed close-out update.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
