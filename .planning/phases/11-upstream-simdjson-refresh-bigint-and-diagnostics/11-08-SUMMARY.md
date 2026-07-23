---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 08
subsystem: ffi-loader
tags: [abi, purego, staged-binding, bigint, diagnostics, cache]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-07 established the coherent ABI 0x00010002 Go/Rust source identity and append-only enum mirrors
provides:
  - Minimal ABI getter-only probe before any Phase 11 symbol lookup
  - Complete fail-closed ABI 1.2 binding for configured construction, BigInt, known-offset, and kernel controls
  - Loader-owned compatible-ABI classification with cache installation after full binding and implementation-name validation
affects: [11-09, 11-10, 11-13, 11-14, 16-v0.2-release]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Probe -> classify -> bind -> read implementation name -> cache
    - Injectable loader operations for deterministic ABI-state fixtures without compiler or package dependencies

key-files:
  created:
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-08-SUMMARY.md
  modified:
    - internal/ffi/bindings.go
    - internal/ffi/bindings_test.go
    - library_loading.go
    - library_loading_test.go
    - parser.go
    - parser_test.go
    - helpers_test.go

key-decisions:
  - "Accept ABI major 1 values at or above 0x00010002 only when every wrapper-required symbol binds successfully."
  - "Compatibility classification belongs in the loader; parser construction no longer re-queries ABI or carries a test override."
  - "All five Phase 11 public exports are mandatory while native allocator telemetry and the internal materializer retain their existing optional policy."

patterns-established:
  - "Staged native loading: the ABI getter is the only pre-classification symbol, and incomplete compatible artifacts fail before cache installation."
  - "Copied scalar ownership: string and BigInt bindings share one copy/free path and never expose native storage."

requirements-completed: [NUM-02, DIAG-01, DIAG-02, LIMIT-01]

# Metrics
duration: 14min
completed: 2026-07-23
---

# Phase 11 Plan 08: ABI-First Loader and Complete Binding Summary

**ABI compatibility is now classified through a one-symbol probe before a complete fail-closed ABI 1.2 bind, with cache installation delayed until the native implementation name is successfully read**

## Performance

- **Duration:** 14 min
- **Started:** 2026-07-23T13:51:30Z
- **Completed:** 2026-07-23T14:05:01Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Added `ffi.ProbeABI`, which resolves and calls only `pure_simdjson_get_abi_version`, preserving a typed ABI mismatch for old artifacts before any ABI 1.2-only lookup.
- Made configured parser construction, known-offset state, exact BigInt copy-out, implementation selection, and implementation locking mandatory bindings with focused width and ownership tests.
- Moved ABI compatibility into the loader, accepting later additive ABI 1.x values only after the wrapper's complete required surface binds.
- Ensured incomplete or wrongly versioned artifacts never enter `cachedLibrary`, including failures while reading the implementation name.
- Removed the parser-level exact-version query and its atomic test override seam.

## Task Commits

Each TDD gate was committed atomically:

1. **Task 1 RED: Add failing staged binding contract** - `dd41349` (test)
2. **Task 1 GREEN: Stage complete ABI 1.2 bindings** - `a8e6a72` (feat)
3. **Task 2 RED: Add failing ABI-first loader fixtures** - `931870d` (test)
4. **Task 2 GREEN: Enforce ABI-first loader compatibility** - `da1ef8a` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `internal/ffi/bindings.go` - Adds the minimal ABI probe, mandatory Phase 11 function table, and configured parser/offset/BigInt/kernel wrappers.
- `internal/ffi/bindings_test.go` - Pins probe lookup order, missing-symbol diagnostics, argument widths, copied BigInt ownership, and kernel wrapper calls.
- `library_loading.go` - Owns ABI classification and the probe/bind/name/cache sequence while preserving double-checked cache locking.
- `library_loading_test.go` - Exercises ABI 1.1, complete and incomplete 1.2, later 1.x, wrong-major, and name-read failure fixtures.
- `parser.go` - Removes the redundant exact ABI query and parser-level version override state.
- `parser_test.go` - Removes the obsolete constructor-level mismatch test in favor of loader fixtures.
- `helpers_test.go` - Removes the obsolete expected-ABI override helper.

## Decisions Made

- ABI compatibility is a minimum within one major: ABI 1.1 and other majors are mismatches; ABI 1.2 or later ABI 1.x values proceed only if the complete required binding succeeds.
- Full `Bind` remains fail-closed for every public Phase 11 export. Only the already-optional allocator telemetry and internal materializer use optional registration.
- The implementation-name status must succeed before cache installation; a successfully opened handle alone is never cacheable.
- Loader fixtures use injected operations rather than compiling temporary shared libraries, keeping focused Go tests deterministic and dependency-free while Task 1 directly pins the production FFI lookup behavior.

## TDD Gate Compliance

- **Task 1 RED:** `dd41349` failed on the missing `ProbeABI`, mandatory binding fields, and Phase 11 wrappers.
- **Task 1 GREEN:** `a8e6a72` made the complete focused FFI suite and all existing optional-binding tests pass.
- **Task 2 RED:** `931870d` failed on the absent staged-loader operation path after replacing the parser-level mismatch seam.
- **Task 2 GREEN:** `da1ef8a` made all six ABI loader fixtures pass and removed every old override identifier.
- No refactor commits were needed.

## Deviations from Plan

None - plan execution stayed within the specified binding, loader, parser, and focused test surfaces.

## Issues Encountered

- `make verify-contract` ran every Rust and ABI audit successfully but printed the expected generated-header diff for the Phase 11 enums and exports. Plan 11-09 owns regeneration and audit synchronization.
- The pre-existing `verify-contract` recipe continues after the header `diff` and exits successfully; this behavior was already recorded by Plan 11-07 and was not widened into this plan.

## Verification

- `go test ./internal/ffi -run '^Test(ProbeABI|Bind|ElementGetBigInt|ParserNewConfigured|ParserLastErrorHasOffset|ImplementationSelection)' -count=1` - passed.
- `go test . -run '^TestABI' -count=1` - passed all ABI 1.1, complete/incomplete 1.2, later 1.x, wrong-major, and implementation-name failure fixtures.
- `go test . -run '^Test(ABI|ActiveLibraryLockScope)' -count=1` - passed, preserving double-checked cache lock scope.
- The obsolete seam scan for `abiVersionOverride`, `expectedABIVersion`, `setExpectedABIVersionForTest`, and `TestABIMismatchAtNewParser` returned no matches.
- `cargo build --release --locked` followed by `PURE_SIMDJSON_LIB_PATH=<fresh target/release library> go test ./... -race` - all four Go packages passed against the explicit ABI 1.2 artifact.
- `make verify-contract` - all 17 Rust unit tests, all integration suites, 20 ABI header-checker tests, and the committed-header audits passed; only the known Plan 11-09 generated-header diff remains.

## Threat and Security Impact

- **T-11-03 mitigated:** old and wrong-major ABIs stop before Phase 11 lookups, missing compatible-ABI symbols are named, and cache installation follows full binding plus implementation-name success.
- **T-11-04 mitigated at the binding boundary:** BigInt uses the same copied byte buffer and mandatory `bytes_free` lifecycle as strings; no native pointer escapes.
- **T-11-SC preserved:** no dependency, package install, alternate source, network endpoint, or publication path was added.

## User Setup Required

None - no external service configuration or publication action is part of this plan.

## Next Phase Readiness

- Plan 11-09 can regenerate the public header and synchronize ABI/export audits against the now-complete Go binding contract.
- Plans 11-10 and 11-13 can consume the mandatory BigInt, configured-parser, known-offset, and kernel methods without capability checks.
- ABI 1.2 source identity remains `0x00010002`; no tag or artifact publication was attempted.

## Self-Check: PASSED

- All seven task files and this summary exist.
- Task commits `dd41349`, `a8e6a72`, `931870d`, and `da1ef8a` are present in repository history.
- The committed Go ABI identity remains exactly `0x00010002`.
- The obsolete parser-level ABI override identifiers are absent.
- No unrelated dirty planning path is staged or modified by this plan.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
