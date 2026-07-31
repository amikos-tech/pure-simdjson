---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
plan: 11
subsystem: ffi-loader-release-policy
tags: [go, purego, ffi, abi, bootstrap, loader]

# Dependency graph
requires:
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plan 12-05's unpublished 0.2.0-dev/ABI 1.3 source identity
  - phase: 12-high-value-dom-navigation-and-simd-utility-apis
    provides: Plan 12-10's complete native export, smoke, and ABI 1.3 contract
provides:
  - Mandatory purego bindings and typed wrappers for all nine Phase 12 native exports
  - Copy-before-free ownership for wildcard value-view arrays with malformed pair rejection
  - ABI 1.3 bootstrap policy and loader fixtures that reject old or incomplete artifacts
affects: [12-06-public-navigation, 12-07-public-utilities, 12-08-public-container-helpers, phase-16-release]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Bind the complete public ABI before caching a loaded native library
    - Copy native-owned arrays into Go memory before exactly one matching free

key-files:
  created:
    - .planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/12-11-SUMMARY.md
    - .planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/deferred-items.md
  modified:
    - internal/ffi/bindings.go
    - internal/ffi/bindings_test.go
    - scripts/release/check_bootstrap_abi_state.py
    - scripts/release/test_check_bootstrap_abi_state.py
    - library_loading_test.go

key-decisions:
  - "Require all nine Phase 12 exports before an ABI 1.3 library can be cached."
  - "Copy wildcard ValueView arrays into Go-owned memory before one native free, with a view-specific cleanup warning."
  - "Map ABI 1.3 to minimum version 0.2.0 while accepting the exact unpublished 0.2.0-dev source identity."

patterns-established:
  - "Required ABI growth: version compatibility is necessary but insufficient; complete symbol binding must also succeed before cache installation."
  - "Bulk native ownership: validate pointer/count pairs before unsafe slicing, copy, free once, and return the copied result even if cleanup only emits a warning."

requirements-completed: [DOM-01, DOM-02, DOM-03, DOM-04, UTIL-01, UTIL-02]

# Metrics
duration: 9min
completed: 2026-07-31
---

# Phase 12 Plan 11: ABI 1.3 Go Binding and Loader Policy Summary

**All nine DOM-navigation and SIMD-utility exports now bind as one mandatory ABI 1.3 surface, with ownership-safe wildcard transport and fail-closed loader policy**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-31T11:30:28Z
- **Completed:** 2026-07-31T11:39:36Z
- **Tasks:** 2
- **Files modified:** 5 task files plus 1 deferred-item record

## Accomplishments

- Added required binding fields, registrations, and typed wrappers for pointer/path navigation, wildcard arrays, indexed/container helpers, minify, and UTF-8 validation.
- Validated wildcard pointer/count pairs before `unsafe.Slice`, returned a non-nil empty slice for `(nil, 0)`, copied native views into Go memory, and freed valid arrays exactly once.
- Extended bootstrap policy so `0.2.0-dev` is the accepted ABI 1.3 source identity while published `0.1.7`/ABI 1.2 is stale for this source tree.
- Pinned loader behavior so ABI 1.2 stops before new lookups, complete ABI 1.3 caches, incomplete ABI 1.3 fails closed, and complete ABI 1.4 remains additive-compatible.

## Task Commits

Each task was committed atomically:

1. **Task 1: Bind all 9 ABI 1.3 symbols with typed wrapper methods** - `2216cee` (feat)
2. **Task 2: Enforce ABI 1.3 release policy, binding behavior, and loader compatibility** - `476ed4a` (feat)

Plan summary metadata is committed separately from the task commits.

## Files Created/Modified

- `internal/ffi/bindings.go` - Requires all nine ABI 1.3 exports and exposes typed, lifetime-safe wrappers.
- `internal/ffi/bindings_test.go` - Covers required lookups, marshaling, wildcard ownership/null states, cleanup warnings, and utility results.
- `scripts/release/check_bootstrap_abi_state.py` - Maps ABI 1.3 to the 0.2.0 minimum source version.
- `scripts/release/test_check_bootstrap_abi_state.py` - Accepts 0.2.0-dev/ABI 1.3 and rejects stale or unknown policy combinations.
- `library_loading_test.go` - Models the complete ABI 1.3 surface and old, incomplete, and later-additive artifacts.
- `.planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/deferred-items.md` - Records one unrelated pre-existing vet warning without changing its source.

## Decisions Made

- Kept every Phase 12 public export on the mandatory registration path; optional binding remains limited to internal diagnostics.
- Used one shared string-marshaling helper for pointer and path lookup because both native signatures have the same bounded pointer/length contract.
- Kept release work source-only: policy recognizes `0.2.0-dev`, but no tag, artifact, upload, or final 0.2.0 claim was created.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `go vet ./...` emitted the pre-existing `materializer_fastpath.go:217:37: possible misuse of unsafe.Pointer` warning while exiting successfully. The file is outside Plan 12-11, so it was left unchanged and recorded in `deferred-items.md`.

## Verification

- `go build ./... && go vet ./...` - exited 0; only the unrelated deferred warning above was emitted.
- `python3 -m unittest scripts/release/test_check_bootstrap_abi_state.py` - passed 14/14 policy tests.
- `go test ./internal/ffi` - passed all required-symbol and wrapper behavior tests.
- `go test . -run 'Test.*(ABI|Bind|Binding|Library|Load)' -count=1` - passed focused loader compatibility tests.
- `python3 scripts/release/check_bootstrap_abi_state.py --version 0.2.0-dev` - accepted version 0.2.0-dev with ABI 0x00010003.
- `make verify-contract && make verify-docs && cargo build --release && go test ./... -race` - passed the full Wave 7 contract, documentation, native build, and race gate.

## Threat and Security Impact

- **T-12-12 mitigated:** an old ABI is rejected before new lookups, and a claimed ABI 1.3 artifact missing any required Phase 12 symbol cannot be cached.
- **T-12-13 mitigated:** wildcard results validate all pointer/count states, copy before free, free once, and use a distinct cleanup-warning path.
- **T-12-BOOT mitigated:** ABI 1.3 requires the 0.2.0 source line while retaining exact requested-version checks and the Phase 16 publication boundary.
- **T-12-SC unchanged:** no dependency, package install, network endpoint, authentication path, schema, artifact, tag, or publication operation was added.
- No unmodeled trust boundary was introduced.

## Known Stubs

None.

## Publication Boundary

- No merge, push, tag, workflow dispatch, artifact upload, release-state rewrite, or final `Version = "0.2.0"` change was performed.
- The supported release path remains `main` -> annotated tag -> CI publication; `release.yml` expects the tag commit to be anchored on `origin/main`.
- Phase 06.1 remains the post-publish fresh-runner validation boundary, and Phase 16 retains final v0.2 publication.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 12-06, 12-07, and 12-08 can consume the complete cached binding for public navigation, utilities, and container helpers.
- ABI 1.3 source and loader policy now agree across native exports, Go bindings, bootstrap identity, and release checks.
- No Plan 12-11 blocker remains.

## Self-Check: PASSED

- Verified all five task files, the deferred-item record, and this summary exist.
- Verified task commits `2216cee` and `476ed4a` exist in repository history.
- Re-ran every task acceptance criterion and the complete Wave 7 verification gate successfully.
- Confirmed no tracked file was deleted, no goal-blocking stub exists, and no unmodeled threat surface was introduced.

---
*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Completed: 2026-07-31*
