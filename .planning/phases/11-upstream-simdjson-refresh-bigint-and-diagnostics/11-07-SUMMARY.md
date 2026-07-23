---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 07
subsystem: release-governance
tags: [abi, semver, bootstrap, bigint, ffi, release-policy]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-01 approved version 0.1.5 and Plan 11-06 completed the native status/symbol prerequisites
provides:
  - Coherent bootstrap 0.1.5 and ABI 0x00010002 source identity
  - Append-only Rust/Go BigInt and diagnostic enum mirrors
  - Bidirectional compile-time bootstrap/ABI canary
  - Fail-closed ABI minimum-version policy and synthetic regression coverage
affects: [11-08, 11-09, 11-10, 11-13, 11-14, 16-v0.2-release]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - One approved semantic version shared by bootstrap pin, checksum examples, canary, policy, and changelog
    - Bidirectional compile-time ABI equality plus runtime source-state validation

key-files:
  created:
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-07-SUMMARY.md
  modified:
    - internal/bootstrap/version.go
    - internal/bootstrap/checksums.go
    - internal/bootstrap/abi_assertion.go
    - scripts/release/check_bootstrap_abi_state.py
    - scripts/release/test_check_bootstrap_abi_state.py
    - src/lib.rs
    - internal/ffi/types.go
    - internal/ffi/types_test.go
    - CHANGELOG.md
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/deferred-items.md

key-decisions:
  - "Bootstrap 0.1.5 requires ABI 0x00010002; an older ABI cannot claim the newer source identity."
  - "BigInt kind 9 and statuses 9/10 are append-only; all existing numeric values and layouts remain unchanged."
  - "The 0.1.5 pin is source preparation only; tag-driven CI and Phase 06.1 validation remain future gates."

patterns-established:
  - "Forward and inverse version policy: each ABI has a minimum version, and each prepared version requires the newest ABI whose policy threshold it reaches."
  - "Source identity moves atomically across Rust, Go, bootstrap, policy, examples, and release notes."

requirements-completed: [UP-01, NUM-02, DIAG-01, DIAG-02, LIMIT-01]

# Metrics
duration: 11min
completed: 2026-07-23
---

# Phase 11 Plan 07: ABI 1.2 Source Identity Summary

**Bootstrap 0.1.5 now binds exactly to ABI 1.2 across Rust, Go, release policy, compile-time canary, checksum examples, and intermediate-artifact release notes**

## Performance

- **Duration:** 11 min
- **Started:** 2026-07-23T13:37:51Z
- **Completed:** 2026-07-23T13:49:43Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments

- Pinned the approved intermediate compatibility identity to bootstrap version `0.1.5` and ABI `0x00010002` without creating or publishing a tag.
- Added fail-closed policy coverage for valid ABI 1.2 state, stale 0.1.4 state, inverse ABI 1.1/version 0.1.5 state, Go/Rust drift, unknown ABI, and requested-version mismatch.
- Appended native/Go BigInt kind `9`, capacity status `9`, and kernel-locked status `10` while pinning every prior value and all existing FFI layouts.
- Preserved an empty checksum override map while coordinating all five commented platform examples on `v0.1.5`.
- Added a dated changelog entry that names the intermediate artifact, audited v4.6.4 base, BigInt/options/diagnostics surface, deliberate pool-constructor source break, ABI 1.1 rejection, and rebuild requirement without claiming publication.

## Task Commits

Each TDD gate was committed atomically:

1. **Task 1 RED: Add failing ABI 1.2 policy cases** - `64d42e0` (test)
2. **Task 1 GREEN: Bind bootstrap 0.1.5 to ABI 1.2** - `dd18255` (feat)
3. **Task 2 RED: Add failing ABI mirror contract** - `8cfc50b` (test)
4. **Task 2 GREEN: Synchronize ABI 1.2 mirrors** - `7113c8f` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `internal/bootstrap/version.go` - Pins the approved intermediate artifact version `0.1.5`.
- `internal/bootstrap/checksums.go` - Retains five commented `<sha256>` examples under `v0.1.5` while leaving the runtime override map empty.
- `internal/bootstrap/abi_assertion.go` - Binds bootstrap 0.1.5 to Go ABI 1.2 in both compile-time subtraction directions.
- `scripts/release/check_bootstrap_abi_state.py` - Adds ABI 1.2 policy and rejects older ABI values for newer prepared versions.
- `scripts/release/test_check_bootstrap_abi_state.py` - Covers approved, stale, mismatched, unknown, and requested-version source states.
- `src/lib.rs` - Sets Rust ABI 1.2, appends BigInt kind 9, and pins every public enum discriminant in tests.
- `internal/ffi/types.go` - Mirrors ABI 1.2, BigInt kind 9, capacity status 9, and kernel-locked status 10 in Go.
- `internal/ffi/types_test.go` - Pins all Go numeric values and unchanged value-view, iterator, allocator-stats, and internal-frame layouts.
- `CHANGELOG.md` - Documents the source-prepared intermediate compatibility artifact and its rebuild/source-break boundary.
- `.planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/deferred-items.md` - Records the pre-existing fail-open contract recipe behavior for later hardening.

## Decisions Made

- The version policy is checked in both directions: ABI 1.2 requires at least 0.1.5, and bootstrap 0.1.5 cannot be paired with ABI 1.1.
- ABI evolution remains additive. Kinds `0..8`, statuses `0..8`, and every existing FFI field size/offset remain unchanged.
- `0.1.5` is a prepared source identity only. CI remains the only publication path; a future tag commit must be anchored on `origin/main`, and Phase 06.1 remains the post-publication fresh-runner boundary.

## TDD Gate Compliance

- **Task 1 RED:** `64d42e0` failed on the missing ABI 1.2 policy, stale-version diagnostic, and inverse ABI 1.1 rejection.
- **Task 1 GREEN:** `dd18255` made all ten synthetic policy tests pass.
- **Task 2 RED:** `8cfc50b` failed because Go lacked statuses 9/10 and BigInt kind 9, while Rust lacked its BigInt enum variant.
- **Task 2 GREEN:** `7113c8f` synchronized the mirrors and made all numeric/layout, canary, policy, and Rust ABI tests pass.
- No refactor commits were needed.

## Deviations from Plan

None - plan execution changed only the specified source identity, policy, enum, test, and changelog contracts.

## Issues Encountered

- The plan's checksum verification loop uses `path` as its loop variable. In zsh, `path` is tied to `PATH`, so the first iteration hides `rg`; running the same assertion under Bash completed successfully.
- `make verify-contract` printed the expected pending generated-header diff but returned success because its multi-command recipe continued past the failed `diff`. This pre-existing Makefile behavior is recorded in `deferred-items.md`; Plan 11-07 did not widen scope to change it.

## Verification

- `python3 scripts/release/test_check_bootstrap_abi_state.py` - all 10 policy tests passed.
- Five exact `v0.1.5/<platform>/<library>: <sha256>` comment assertions passed; no `v0.1.4/` example or live checksum-map entry remains.
- `go test ./internal/ffi ./internal/bootstrap -count=1` - passed, including the bidirectional canary and every numeric/layout assertion.
- `python3 scripts/release/check_bootstrap_abi_state.py --version 0.1.5` - reported version `0.1.5`, ABI `0x00010002`.
- `cargo test --locked --lib -- --test-threads=1` - all 17 library tests passed.
- `cargo build --release --locked` followed by `PURE_SIMDJSON_LIB_PATH=<fresh target/release dylib> go test ./... -race` - all four Go packages passed against the explicit local ABI 1.2 library.
- `make verify-contract` - all Rust tests passed and the generated-header diff contained the planned pending enum/function synchronization owned by Plan 11-09; no header was hand-edited.
- Refreshed-tag preflight still finds no `v0.1.5` tag.

## Threat and Security Impact

- **T-11-03 mitigated:** exact policy mapping, inverse stale-ABI rejection, bidirectional canary, numeric/layout tests, and the real source-state checker fail closed on drift.
- **T-11-SC preserved:** the version came only from the operator-approved Plan 11-01 record; no tag, publication, upload, dependency, remote push, or alternate release path was introduced.
- No unplanned network, authentication, schema, or file-access trust boundary was added.

## Publication Boundary

- No tag was created or pushed, no artifact was uploaded, and no release/publication API was called.
- Strict readiness was not represented as passing on this phase branch and no tag push is recommended from it.
- Any future `v0.1.5` tag commit must first be integrated to `origin/main` by squash merge and pass strict readiness there.
- Plan 11-14 retains publication/default-bootstrap proof, and Phase 06.1 retains fresh-runner public validation.

## Known Stubs

- `internal/bootstrap/checksums.go:10` - `internal/bootstrap/checksums.go:14` intentionally retain commented `<sha256>` examples. The runtime override map stays empty by contract and resolves published `SHA256SUMS`; no digest was invented.

## Deferred Issues

- Harden the pre-existing `make verify-contract` recipe to stop when its generated-header `diff` fails, after Plan 11-09 performs the planned generated ABI synchronization.
- Repository-wide Rust formatting drift from earlier plans remains recorded in `deferred-items.md` and was not touched.

## User Setup Required

None - no external service configuration or manual publication action is part of this plan.

## Next Phase Readiness

- Plan 11-08 can build staged ABI-first binding on the now-coherent Go/Rust ABI 1.2 identity.
- Plan 11-09 can update cbindgen, the generated public header, and ABI/native smoke contracts without guessing version or enum values.
- Plans 11-13 and 11-14 must preserve exact version `0.1.5`, CI-only publication, `origin/main` ancestry, and the Phase 06.1 validation boundary.

## Self-Check: PASSED

- All ten planned implementation/policy/tracking files and this summary exist.
- Task commits `64d42e0`, `dd18255`, `8cfc50b`, and `7113c8f` are present in repository history.
- The actual source-state checker reports bootstrap `0.1.5` with ABI `0x00010002`.
- The refreshed tag set still contains no `v0.1.5`.
- No prohibited repository information appears in the changed artifacts.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
