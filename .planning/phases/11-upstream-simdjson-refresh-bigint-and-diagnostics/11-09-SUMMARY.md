---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 09
subsystem: ffi-contract
tags: [abi, cbindgen, bigint, diagnostics, native-smoke, release]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-07 established the coherent ABI 0x00010002 source identity and append-only enum values
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-08 established the complete mandatory ABI 1.2 binding surface
provides:
  - Deterministic generated public ABI 1.2 header with explicit internal exclusions
  - Fail-closed header audits for every Phase 11 symbol, signature, ownership rule, and BigInt contract comment
  - One cross-platform C smoke that actively executes configured BigInt, known-offset, and kernel-lock behavior
  - Normative append-only ABI 1.2 lifecycle, ownership, limits, diagnostics, and compatibility contract
affects: [11-10, 11-13, 11-14, 16-v0.2-release, ffi, bootstrap]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Public declarations are generated from Rust through cbindgen and audited against synthetic negative fixtures
    - One dynamically resolved C source executes the complete mandatory export surface on every release target

key-files:
  created:
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-09-SUMMARY.md
  modified:
    - cbindgen.toml
    - include/pure_simdjson.h
    - tests/abi/check_header.py
    - tests/abi/test_check_header.py
    - tests/abi/handle_layout.c
    - tests/smoke/ffi_export_surface.c
    - docs/ffi-contract.md

key-decisions:
  - "The public header remains generated from src/lib.rs; new private psimdjson bridge declarations are excluded explicitly rather than deleted or hand-filtered afterward."
  - "The shared C smoke proves kernel order in one process: automatic selection and name read, configured construction and parse, post-construction locked setter, then legacy construction."
  - "ABI 1.2 is the strict minimum; later additive ABI 1.x artifacts are usable only when the complete wrapper-required symbol surface binds."

patterns-established:
  - "Generated ABI fence: exact macro, symbols, signatures, comments, enum numbers, and layouts are checked independently."
  - "Release smoke call coverage: every resolved export must be invoked, including ownership release and failure-state paths."

requirements-completed: [NUM-02, DIAG-01, DIAG-02, LIMIT-01]

# Metrics
duration: 13min
completed: 2026-07-23
---

# Phase 11 Plan 09: ABI 1.2 Header and C Contract Summary

**Generated ABI 1.2 declarations, fail-closed audits, and one five-target C smoke now freeze exact BigInt, parser-limit, diagnostic-offset, and kernel-lock behavior**

## Performance

- **Duration:** 13 min
- **Started:** 2026-07-23T14:09:02Z
- **Completed:** 2026-07-23T14:21:48Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Regenerated `include/pure_simdjson.h` from the corrected Rust source at ABI `0x00010002`, with kind 9, statuses 9/10, all five Phase 11 exports, and no internal bridge/test declarations.
- Extended synthetic and real-header audits to reject each missing Phase 11 symbol, every wrong new signature, missing BigInt copy/free ownership, stale BigInt precision-loss comments, and numeric/layout drift.
- Extended the single release-consumed C smoke to set automatic implementation selection, read its name, use a configured parser, copy/free exact BigInt text, reject all three numeric getters as wrong type, distinguish an unknown offset, and prove post-construction status 10.
- Rewrote the normative FFI document around additive ABI 1.2 ownership, immutable limits, process-global kernel locking, bounded failure-only diagnostic replay, and fail-closed loader compatibility.

## Task Commits

Each TDD gate was committed atomically:

1. **Task 1 RED: Add failing ABI 1.2 header audits** - `6c3e866` (test)
2. **Task 1 GREEN: Generate complete ABI 1.2 header** - `4719b5f` (feat)
3. **Task 2 RED: Add failing ABI 1.2 C smoke** - `efbacfc` (test)
4. **Task 2 GREEN: Exercise and document ABI 1.2** - `cb69982` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `cbindgen.toml` - Sets ABI 1.2 and excludes the new private kernel, configured-parser, diagnostic, and BigInt bridge declarations.
- `include/pure_simdjson.h` - Generated public header with the complete additive ABI 1.2 contract.
- `tests/abi/check_header.py` - Requires the exact ABI 1.2 surface, copy/free ownership, and corrected BigInt documentation.
- `tests/abi/test_check_header.py` - Adds synthetic failures for missing symbols, wrong signatures, missing ownership, and stale comments.
- `tests/abi/handle_layout.c` - Pins statuses 0 through 10, kind 9, ABI 1.2, and all prior public sizes/offsets.
- `tests/smoke/ffi_export_surface.c` - Resolves and calls every new export in the release-shared C smoke while retaining legacy paths.
- `docs/ffi-contract.md` - Defines the normative ABI 1.2 ownership, limits, kernel, diagnostic, and compatibility rules.

## Decisions Made

- Header generation remains one-way from `src/lib.rs`; `include/pure_simdjson.h` is never hand-edited.
- The ownership audit covers both copied strings and copied BigInts through the one existing `pure_simdjson_bytes_free` allocator boundary.
- A proven unknown diagnostic fixture uses empty input. The earlier `{"bad":}` case is now truthfully known at byte 7 after Plan 11-05 and cannot serve as an unknown-location assertion.
- The smoke calls the post-construction setter before the explicit idempotent lock call, so status 10 proves parser construction itself locked selection.

## TDD Gate Compliance

- **Task 1 RED:** `6c3e866` failed on the ABI 1.1 checker/header, absent Phase 11 symbol/signature/ownership rules, and missing appended C constants.
- **Task 1 GREEN:** `4719b5f` made all 25 Python audits pass, compiled every static layout assertion, and produced a deterministic internal-symbol-free header.
- **Task 2 RED:** `efbacfc` failed C compilation because the new call contract referenced five not-yet-wired export-table fields.
- **Task 2 GREEN:** `cb69982` wired all five typedef/table/resolver paths and made the shared dynamic C smoke pass.
- No refactor commits were needed.

## Deviations from Plan

None - plan execution stayed within the specified generated-header, ABI-audit, C-smoke, and normative-document surfaces.

## Issues Encountered

- The pre-existing smoke expected `{"bad":}` to have no offset, but the Phase 11 upstream replay now correctly proves byte 7. The planned explicitly unknown check now uses empty input and validates both `UINT64_MAX` and `has_offset == 0`.

## Verification

- `python3 tests/abi/test_check_header.py` - all 25 synthetic and real-header audits passed.
- `make generate-header && make verify-contract` - deterministic header diff was clean; 84 Rust unit/integration tests passed; the header checker passed; and the C layout object compiled.
- The stale-contract scan found no BigInt-unclassifiable or BigInt-precision-loss wording in `src/lib.rs` or the generated header.
- `make verify-docs` - all normative contract markers passed.
- `cargo build --release --locked` - completed successfully.
- `bash scripts/release/run_native_smoke.sh target/release/libpure_simdjson.dylib darwin-arm64` - reported `ffi export surface smoke passed` and `phase2 smoke passed`.
- Static source inspection confirmed all five new symbols in typedef, table, resolver, and active call sections, with setter/name read before configured construction and the legacy constructor after the post-lock assertion.

## Threat and Security Impact

- **T-11-03 mitigated:** exact required-symbol/signature audits, deterministic generation, static layout assertions, and active C calls fail closed on header/export drift.
- **T-11-04 mitigated:** the public BigInt contract is copied bytes plus mandatory `pure_simdjson_bytes_free`; no borrowed pointer escapes.
- **T-11-05 mitigated:** the smoke and normative document transport the known bit independently from the numeric offset and forbid inference.
- **T-11-SC preserved:** no dependency, package install, alternate source, publication path, or network surface was introduced.

## Publication Boundary

- No tag was created or pushed, no artifact was uploaded, and no publication API was called.
- CI remains the only publish path. A future tag commit must first reach `main` through squash merge, be anchored on `origin/main`, and pass strict readiness before `release.yml` publishes it.
- Phase 06.1 remains the post-publication fresh-runner validation boundary.

## Deferred Issues

- The pre-existing multi-command `make verify-contract` recipe can continue after a failed generated-header `diff`; the direct deterministic checks were clean here, and the recipe-hardening item remains recorded in the phase deferred-items file.

## User Setup Required

None - no external service configuration or manual publication action is part of this plan.

## Next Phase Readiness

- Plan 11-10 can expose the Go BigInt API against a generated, mandatory ABI 1.2 contract.
- Plans 11-13 and 11-14 can rely on configured construction, explicit offset-known state, and process-global kernel controls being present on every compatible artifact.
- Later release work must preserve the generated-header audit and single shared C smoke across all five platform jobs.

## Self-Check: PASSED

- All seven task files and this summary exist.
- Task commits `6c3e866`, `4719b5f`, `efbacfc`, and `cb69982` are present in repository history.
- The generated header still reports ABI `0x00010002`, and no prohibited repository information appears in the changed artifacts.
- The unrelated dirty planning paths remain unstaged and unmodified by this plan.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
