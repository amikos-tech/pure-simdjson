---
phase: 01-ffi-contract-design
plan: "03"
subsystem: api
tags: [ffi, abi, docs, verification, cbindgen]
requires:
  - phase: "01-02"
    provides: "Finalized ABI symbols, error codes, and fixed handle/view layouts in Rust and the generated header"
provides:
  - Normative FFI contract document covering lifecycle, ownership, diagnostics, and panic/exception policy
  - Repeatable `make verify-contract` and `make verify-docs` entrypoints
  - Static header and layout checks that catch ABI drift before later phases implement the shim
affects: [phase-2-shim-implementation, go-bindings, header-regeneration, contract-verification]
tech-stack:
  added: []
  patterns: [normative-ffi-contract, static-abi-drift-checks]
key-files:
  created: [docs/ffi-contract.md, Makefile, tests/abi/check_header.py, tests/abi/handle_layout.c, tests/abi/README.md]
  modified: []
key-decisions:
  - "Treat docs/ffi-contract.md as normative for semantic policy while include/pure_simdjson.h remains normative for names and types."
  - "Enforce ABI drift with three static gates together: temp header regeneration diff, signature lint rules, and compile-time layout assertions."
  - "Keep panic/exception safety as an explicit contract requirement now so Phase 2 cannot soften ffi_fn!, catch_unwind, or .get(err) behavior."
patterns-established:
  - "Contract prose and generated header change together under make verify-contract plus make verify-docs."
  - "ABI policy claims are enforced by literal grep and lint gates instead of reviewer memory."
requirements-completed: [FFI-05, FFI-06, DOC-02]
duration: 8m
completed: 2026-04-14
---

# Phase 1 Plan 3: FFI Contract Design Summary

**Normative ABI contract with static header, layout, and documentation gates for the pure-simdjson FFI surface**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-14T12:09:00Z
- **Completed:** 2026-04-14T12:17:07Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Wrote `docs/ffi-contract.md` as the single normative statement of ABI scope, lifecycle, ownership, diagnostics, split-number access, and panic/exception policy.
- Added `make verify-contract` and `make verify-docs` so the repository can mechanically reject header drift and missing contract clauses.
- Added a header linter, fixed-layout compile assertions, and a requirement traceability note under `tests/abi/`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Author `docs/ffi-contract.md` as the normative ABI contract** - `4c13846` (`feat`)
2. **Task 2: Add static verification scaffolding for header drift, signature rules, and layout invariants** - `924bcfa` (`feat`)

## Files Created/Modified

- `docs/ffi-contract.md` - Defines the public ABI semantics, lifecycle rules, error-code meanings, ownership model, diagnostics surface, and unwind/exception policy.
- `Makefile` - Adds repeatable contract verification entrypoints for header regeneration, ABI linting, layout checks, and contract grep gates.
- `tests/abi/check_header.py` - Encodes the public signature rules and required-symbol checks for the generated header.
- `tests/abi/handle_layout.c` - Compiles fixed-size and fixed-offset assertions for handles, views, and iterators.
- `tests/abi/README.md` - Maps each static rule to `FFI-01` through `FFI-08` and `DOC-02`.

## Decisions Made

- The generated header and the Markdown contract are both authoritative, but in different ways: the header fixes names and C types, while the Markdown fixes lifecycle and safety semantics.
- The static suite must fail on both ABI drift and policy drift, so documentation grep gates ship alongside header and layout checks instead of relying on review alone.
- Parser busy-state, Rust-owned padded copy-in, and split number access are documented as fixed contract rules rather than implementation options for later phases.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `cargo check` generated `Cargo.lock` and `target/`; both were removed before committing so the task commits stayed limited to the plan-owned artifacts.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 2 now has a fixed contract document and a mechanical verification suite to run before any Rust/C++ shim behavior lands.
- Later phases can extend implementation behind the committed ABI without renegotiating lifecycle, diagnostics, or panic/exception policy.

## Self-Check: PASSED

- Verified summary file exists: `.planning/phases/01-ffi-contract-design/01-03-SUMMARY.md`
- Verified created files exist: `docs/ffi-contract.md`, `Makefile`, `tests/abi/check_header.py`, `tests/abi/handle_layout.c`, `tests/abi/README.md`
- Verified task commits exist in git history: `4c13846`, `924bcfa`

---
*Phase: 01-ffi-contract-design*
*Completed: 2026-04-14*
