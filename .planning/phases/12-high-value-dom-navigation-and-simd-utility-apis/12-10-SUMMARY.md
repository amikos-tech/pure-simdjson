---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 10
subsystem: ffi-ci-contract
tags: [cpp, sanitizers, github-actions, ffi, abi]

# Dependency graph
requires:
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plans 12-01 through 12-04's complete native ABI 1.3 navigation and utility exports
  - phase: 06-ci-release-matrix-platform-coverage
    provides: Shared five-target native release smoke and Go wrapper workflow paths
provides:
  - Durable three-run ASan/UBSan minify alias gate with dynamic supported-kernel accounting
  - Semantic C smoke coverage for all nine ABI 1.3 navigation and utility exports
  - Automatic five-platform Go race coverage on Phase 12 branch pushes
  - Normative ABI 1.3 loader, ownership, utility safety, and compatibility rules
affects: [12-11-loader-bindings, phase-16-cross-platform-release]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Promote safety probes into durable source and self-triggering CI paths outside planning artifacts
    - Keep public export smoke closed-world through resolution, semantic invocation, and call coverage

key-files:
  created:
    - tests/native/minify_buffer_safety_probe.cpp
    - scripts/ci/verify_minify_buffer_safety.sh
    - .planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/12-10-SUMMARY.md
  modified:
    - .github/workflows/phase2-rust-shim-smoke.yml
    - tests/smoke/ffi_export_surface.c
    - .github/workflows/phase3-go-wrapper-smoke.yml
    - scripts/release/test_release_workflow_contracts.py
    - docs/ffi-contract.md

key-decisions:
  - "Derive D-14 totals from 12 cases times the ordered host-supported kernel set, while requiring fallback plus haswell or westmere on Linux x86-64."
  - "Treat all nine navigation and utility additions as mandatory ABI 1.3 symbols; ABI 1.2 is historical and rejected by the current loader contract."
  - "Extend the existing shared native release smoke and five-platform Go workflow instead of creating parallel release machinery."

patterns-established:
  - "Durable native safety gate: sanitizer build in a temporary directory, three byte-identical runs, machine-readable dynamic summary, and architecture-specific kernel assertions."
  - "ABI smoke closure: every public export participates in enum/name/typedef/table/resolve/mark/call coverage and carries a semantic assertion."

requirements-completed: [DOM-01, DOM-02, DOM-03, DOM-04, UTIL-01, UTIL-02]

# Metrics
duration: 15min
completed: 2026-07-31
---

# Phase 12 Plan 10: ABI 1.3 Cross-Platform Contract Summary

**A durable sanitizer gate now proves host-supported minify alias safety, while the shared release smoke semantically invokes every ABI 1.3 addition and Phase 12 pushes schedule all five Go race jobs**

## Performance

- **Duration:** 15 min
- **Started:** 2026-07-31T11:08:40Z
- **Completed:** 2026-07-31T11:24:00Z
- **Tasks:** 2
- **Files modified:** 7 task files

## Accomplishments

- Promoted the 12-case minify alias probe into durable test source and a three-run ASan/UBSan verifier that derives totals from the ordered supported-kernel set.
- Made the Phase 2 native workflow self-trigger when either the verifier or `tests/native/**` changes, with Linux x86-64 requiring fallback plus haswell or westmere.
- Expanded the existing cross-platform C smoke to resolve and semantically exercise all nine ABI 1.3 exports, including wildcard allocation/free and exact-alias minification.
- Added the Phase 12 branch to the unchanged five-job Go race workflow and made ABI 1.3 the current strict loader/build contract.

## Task Commits

Each task was committed atomically:

1. **Task 1: Promote D-14 into a durable, path-triggered native CI gate** - `42d4e03` (test)
2. **Task 2: Expand the ABI smoke, current-branch five-platform gate, and ABI 1.3 docs** - `e869e46` (test)

Plan summary metadata is committed separately from the task commits.

## Files Created/Modified

- `tests/native/minify_buffer_safety_probe.cpp` - Runs 12 exact-size separate/aliased cases per supported kernel and emits an ordered machine-readable summary.
- `scripts/ci/verify_minify_buffer_safety.sh` - Builds with ASan/UBSan, validates three deterministic runs, computes dynamic totals, and enforces Linux x86 kernels.
- `.github/workflows/phase2-rust-shim-smoke.yml` - Adds durable gate paths and invokes the verifier in `linux-smoke`.
- `tests/smoke/ffi_export_surface.c` - Resolves and calls all nine ABI 1.3 additions with exact C signatures and semantic assertions.
- `.github/workflows/phase3-go-wrapper-smoke.yml` - Runs its existing five Go race jobs on Phase 12 pushes.
- `scripts/release/test_release_workflow_contracts.py` - Pins both workflow triggers, all five jobs/runners, all nine smoke stages, durable paths, and normative docs.
- `docs/ffi-contract.md` - Defines ABI 1.3 as current, documents wildcard ownership and utility safety, and rejects historical ABI 1.2 before new lookup.

## Decisions Made

- Kept D-14 host-adaptive: the probe sorts the supported kernels, and the script derives `12 * kernel_count` instead of freezing one machine's total.
- Kept release verification on `scripts/release/run_native_smoke.sh`; every release platform automatically receives the expanded C smoke without duplicate workflow logic.
- Made ABI 1.3 the current strict minimum and allowed later additive ABI 1.x artifacts only after the complete wrapper-required surface binds.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed an unsupported leak-sanitizer runtime option on macOS**
- **Found during:** Task 1 sanitizer execution
- **Issue:** Apple ASan aborted before the probe because `detect_leaks=1` is unsupported on this platform.
- **Fix:** Retained ASan/UBSan fail-fast behavior with portable `halt_on_error=1` and `-fno-sanitize-recover=all`, without requesting unsupported leak detection.
- **Files modified:** `scripts/ci/verify_minify_buffer_safety.sh`
- **Verification:** The verifier completed three deterministic arm64/fallback runs with 24 dynamically derived cases and zero failures.
- **Committed in:** `42d4e03`

---

**Total deviations:** 1 auto-fixed bug.
**Impact on plan:** The portability fix preserves the required address/undefined-behavior sanitizer coverage and avoids a host-specific false failure.

## Issues Encountered

None beyond the auto-fixed sanitizer option above.

## Verification

- `bash scripts/ci/verify_minify_buffer_safety.sh` - passed three deterministic runs on `arm64,fallback`, 24/24 dynamically derived cases.
- `cc -std=c11 -Wall -Wextra -Iinclude -c tests/smoke/ffi_export_surface.c ...` - passed without warnings.
- `python3 scripts/release/test_release_workflow_contracts.py` - passed 17/17 workflow, smoke, and documentation contract tests.
- `make verify-docs` - passed.
- `cargo build --release` - passed.
- `bash scripts/release/run_native_smoke.sh target/release/libpure_simdjson.dylib darwin-arm64` - passed the shared export-surface and minimal-parse smoke path.
- Hosted Linux x86-64 and five-platform Go execution are now scheduled automatically by the exact branch/path triggers on the next matching push.

## Threat and Security Impact

- **T-12-ALIAS mitigated:** exact-size separate and in-place writes run under ASan/UBSan for every supported host kernel, with Linux x86 requirements enforced by the verifier.
- **T-12-TRIGGER mitigated:** changes to the verifier or any durable native probe schedule the Phase 2 workflow, and static contracts reject planning-local dependencies.
- **T-12-ABI-SMOKE mitigated:** all nine ABI 1.3 functions are resolved through exact typedefs, invoked semantically, and required by closed call coverage.
- **T-12-SC unchanged:** no action, package, external dependency, network endpoint, authentication path, schema, or publication mechanism was added.

## Known Stubs

None.

## Publication Boundary

- No merge, push, tag, workflow dispatch, artifact upload, or release-state rewrite was performed.
- The supported release path remains CI-only; `release.yml` continues to call the shared native smoke on all five targets.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 12-11 can make all nine ABI 1.3 symbols mandatory in the Go loader and release policy using the completed smoke/docs contract.
- Later Go API plans can rely on automatic five-platform race coverage when changes land on the Phase 12 branch.
- Hosted-runner results will supply the Linux x86-64 and full five-platform evidence on the next matching push; no source-level blocker remains.

## Self-Check: PASSED

- Verified all seven task files and this summary exist.
- Verified task commits `42d4e03` and `e869e46` exist in repository history.
- Re-ran every task acceptance criterion, focused contract test, C compile, documentation gate, release build, and shared native smoke successfully.
- Scanned all task files for placeholder/TODO/FIXME and hardcoded-empty stub patterns; none block the plan goal.
- Confirmed no tracked files were deleted and no unmodeled trust boundary was introduced.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
