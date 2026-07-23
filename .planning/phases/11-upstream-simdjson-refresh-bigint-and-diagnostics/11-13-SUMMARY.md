---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 13
subsystem: release-readiness
tags: [abi-1.2, bigint, bootstrap, release-contracts, nyquist]

# Dependency graph
requires:
  - phase: 11-12
    provides: Completed Phase 11 ABI 1.2, BigInt, parser-limit, error-offset, and kernel source surface
provides:
  - Packaged and public smoke contracts that require configured ABI 1.2 BigInt behavior
  - Exact serial source-gate evidence labeled SOURCE READY — NOT RELEASED
  - Evidence-backed source-gate approval and completed local Nyquist map
affects: [11-14, 06.1-public-bootstrap-validation, 16-v0.2-release]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Tag-owned Go smoke reached through existing packaged/public wrapper indirection
    - Fresh explicit release-library path on every pre-publication Go runtime gate
    - Planning-only readiness evidence separated from hosted publication proof

key-files:
  created:
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-ARTIFACT-READINESS.md
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-13-SUMMARY.md
  modified:
    - tests/smoke/go_bootstrap_smoke.go
    - scripts/release/test_release_workflow_contracts.py
    - scripts/release/test_public_bootstrap_validation_contracts.py
    - docs/bootstrap.md
    - purejson.go
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-VALIDATION.md

key-decisions:
  - "Keep the existing release/public wrapper indirection and make the tag-owned Go smoke the configured ABI 1.2 behavior contract."
  - "Stop Plan 11-13 at SOURCE READY — NOT RELEASED; strict origin/main ancestry and hosted proof remain the 11-14-T1 operator gate."
  - "Use the freshly built release library through PURE_SIMDJSON_LIB_PATH for every local Go runtime gate before publication."

patterns-established:
  - "Artifact smoke contract: configured parser + active kernel + exact BigInt + normal scalar + clean close."
  - "Evidence boundary: local source results never substitute for tag-driven five-platform and default-bootstrap proof."

requirements-completed: [UP-01, NUM-01, NUM-02, DIAG-01, DIAG-02, LIMIT-01]

# Metrics
duration: 19min
completed: 2026-07-23
---

# Phase 11 Plan 13: Source Readiness and Artifact Smoke Summary

**Configured ABI 1.2 BigInt smoke now flows through the existing five-target release/public validation path, backed by a green serial source gate and an explicit not-released boundary**

## Performance

- **Duration:** 19 min
- **Started:** 2026-07-23T15:25:11Z
- **Completed:** 2026-07-23T15:43:51Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Strengthened the tag-owned Go smoke to require nondefault capacity/depth,
  active implementation selection, exact `TypeBigInt`/`GetBigInt`, normal
  scalar compatibility, and clean document/parser closure.
- Added executable contracts that retain all five release targets, both
  native and packaged smoke layers, the five-target R2 band, and the
  three-target GitHub fallback band.
- Ran the exact serial source gate against a fresh release library, including
  the race suite, release-source policies, JSONTestSuite oracle, no-baseline
  Tier 1/2/3 signal, non-strict readiness, pool-call-site audit, and
  cross-language stale-contract scan.
- Recorded the tested source/upstream/version/ABI identities and advanced the
  validation contract to source-gate approval while leaving hosted proof with
  Plan 11-14.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: packaged/public ABI 1.2 contracts** - `d94cc89` (test)
2. **Task 1 GREEN: configured packaged ABI 1.2 smoke and docs** - `f7019bc` (feat)
3. **Task 2: source-ready evidence and validation closeout** - `cf9ec90` (docs)

## Files Created/Modified

- `tests/smoke/go_bootstrap_smoke.go` - Proves configured ABI 1.2 BigInt,
  scalar, kernel, and cleanup behavior in the tag-owned runnable smoke.
- `scripts/release/test_release_workflow_contracts.py` - Locks the five release
  targets, native smoke, and packaged-wrapper-to-Go-smoke chain.
- `scripts/release/test_public_bootstrap_validation_contracts.py` - Locks both
  published-tag validation bands and their path to the tag-owned smoke.
- `docs/bootstrap.md` - Labels `v0.1.5` as source-prepared and documents ABI
  1.1 explicit-path mismatch plus the Phase 06.1 hosted boundary.
- `purejson.go` - Adds small parser-option, copied-BigInt, truthful-offset, and
  process-global kernel-lock package examples.
- `11-VALIDATION.md` - Replaces placeholder ownership with concrete task,
  wave, command, and evidence mappings and approves the local source gate.
- `11-ARTIFACT-READINESS.md` - Records exact zero exits, identities, oracle,
  benchmark signal, and the `SOURCE READY — NOT RELEASED` boundary.

## Decisions Made

- Preserved `release.yml -> run_go_packaged_smoke.sh -> go_bootstrap_smoke.go`
  and the corresponding public-validation wrapper path instead of adding a
  duplicate smoke mechanism.
- Initially treated `f7019bcf9262418bc96b70204fa893e8a7542d33` as the tested
  artifact-enabling source commit. Before merge, the complete source gate was
  rerun successfully at `99f7937c78d739cc79299929fea62146a86a2f24`
  after the build-only line-ending compatibility fix. Planning-only readiness
  commits do not change the product source under test.
- Reserved strict readiness for the squash-merged `origin/main`-anchored
  commit in Plan 11-14. CI remains the only publisher, and Phase 06.1 remains
  the post-publish fresh-runner boundary.

## TDD Gate Compliance

- **RED:** Both release/public contract suites failed because the Go smoke did
  not yet contain configured parser or BigInt behavior and the bootstrap docs
  still named the old example version.
- **GREEN:** The focused smoke plus both Python suites passed against the fresh
  release library after the minimal smoke/docs implementation.
- Commit order is `d94cc89` then `f7019bc`; no refactor commit was needed.

## Verification

The exact source gate passed twice, including the final automated verifier with
the evidence files present:

- `make verify-contract`
- `cargo build --release --locked`
- `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test ./... -race -count=1`
- all four release-source Python test files
- explicit-library JSONTestSuite oracle
- explicit-library no-baseline Tier 1/2/3 benchmark capture
- non-strict `scripts/release/check_readiness.sh`
- exact v4.6.4 commit check
- `NewParserPool` two-return call-site audit
- stale BigInt contract scan across Go, C++, Rust, and generated C header

Observed benchmark evidence contained 60 rows across 12 benchmark cases with
no flagged rows. Its no-baseline result is an execution signal, not a
comparative regression claim. Full command/output evidence is in
`11-ARTIFACT-READINESS.md`.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None. The closeout scan found only intentional empty-state assertions and
evidence fields; no placeholder feeds public behavior or prevents the source
readiness goal.

## Next Phase Readiness

- Plan 11-14 can take the artifact-enabling source through the required squash
  merge to `main`.
- From the approved `origin/main`-anchored commit, Plan 11-14 owns strict
  readiness, annotated-tag publication through `release.yml`, and the Phase
  06.1 hosted public bootstrap validation.
- `v0.1.5` and `v0.1.6` failed before publication and remain immutable.
  Recovery source now targets `v0.1.7`; no default-bootstrap proof is claimed,
  and Phase 11 is not complete.

## Self-Check: PASSED

- Created readiness and summary files exist.
- Task commits `d94cc89`, `f7019bc`, and `cf9ec90` resolve as commits.
- Readiness/validation status lines and summary formatting checks pass.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
