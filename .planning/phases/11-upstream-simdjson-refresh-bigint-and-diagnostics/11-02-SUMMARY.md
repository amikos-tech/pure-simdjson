---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 02
subsystem: native-parser
tags: [simdjson, bigint, ffi, rust, cpp, go]

# Dependency graph
requires:
  - phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
    provides: Stable 72-byte preorder materializer frame contract and copied Go value boundary
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-01 intermediate ABI 1.2 artifact decision
provides:
  - Exact official simdjson v4.6.4 gitlink with a single provenance-recorded downstream overflow patch
  - Fail-closed build-output patch application guarded by exact-base and clean-submodule checks
  - Native BigInt kind 9 propagation with exact signed and unsigned decimal frame spans
  - Go TypeBigInt classification and oversized-integer fuzz/oracle reconciliation
affects: [11-03, 11-07, 11-10, native-abi, go-dom, release-builds]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Patch audited upstream only in Cargo build output after exact provenance checks
    - Transport additive native kinds across Rust as raw u32 until public ABI mirrors synchronize
    - Preserve BigInt text in the existing doc-owned materializer string span

key-files:
  created:
    - patches/simdjson-v4.6.4-positive-bigint.patch
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-02-SUMMARY.md
  modified:
    - third_party/simdjson
    - build.rs
    - src/native/simdjson_bridge.cpp
    - src/runtime/mod.rs
    - tests/rust_shim_minimal.rs
    - tests/rust_shim_fast_materializer.rs
    - element.go
    - element_fuzz_test.go
    - element_scalar_test.go
    - materializer_fastpath_test.go
    - testdata/jsontestsuite/expectations.tsv

key-decisions:
  - "D-01 amendment: keep official simdjson v4.6.4 as the audited base and permit exactly one provenance-recorded positive-overflow patch applied only to a verified build-output copy."
  - "Additive native kind 9 crosses the interim Rust bridge as raw u32; Plan 11-07 remains the owner of synchronized public ABI enum mirrors."
  - "Wave 1 exposes TypeBigInt and exact frame text but does not add GetBigInt; Plan 11-10 remains the accessor and Go materialization owner."

patterns-established:
  - "Fail-closed upstream patching: exact commit, clean tracked submodule, git apply --check, and semantic postcondition before compilation."
  - "BigInt preservation: classify only out-of-range integer syntax as kind 9 while keeping existing numeric accessors strict."

requirements-completed: [UP-01, NUM-01, NUM-02]

# Metrics
duration: 27min
completed: 2026-07-23
---

# Phase 11 Plan 02: Native BigInt Boundary Summary

**Official simdjson v4.6.4 pinned with a provenance-locked, fail-closed overflow patch that propagates exact signed and unsigned BigInt text as kind 9 through native frames and Go `TypeBigInt`**

## Performance

- **Duration:** 27 min
- **Started:** 2026-07-23T11:11:37Z
- **Completed:** 2026-07-23T11:38:03Z
- **Tasks:** 3
- **Files modified:** 16

## Accomplishments

- Pinned `third_party/simdjson` to official commit `1bcf71bd85059ab6574ea1159de9298dcc1212c5`, tagged `v4.6.4`, without changing the singleheader, `cc`, or C++17 build mechanism.
- Added one narrowly scoped patch for the positive 20-digit unsigned-overflow gap and made Cargo apply it only to an output-directory copy after exact-base, clean-worktree, hunk, and semantic postcondition checks.
- Enabled `number_as_string(true)` across production DOM parsers, propagated BigInt as native kind 9, kept numeric getters strict, and preserved exact signed/unsigned decimal text in the unchanged 72-byte frame layout.
- Exposed Go `TypeBigInt = 9`, added deterministic positive/negative oversized-integer traversal coverage, and changed only the three valid integer-only JSONTestSuite rows to acceptance.
- Proved a clean isolated clone builds while a clone moved off the locked upstream commit fails closed.

## Task Commits

Each task was committed atomically:

1. **Task 1: Pin the audited v4.6.4 upstream without changing the build mechanism** - `2bc0295` (chore)
2. **Task 2 RED: Add failing native BigInt boundary and exact-frame coverage** - `d95789d` (test)
3. **D-01 decision amendment: Authorize the minimal pinned-base patch** - `8df0e37` (docs)
4. **Task 2 GREEN: Enable native BigInt classification and frame propagation** - `6ee2371` (feat)
5. **Task 3: Reconcile Go and correctness-oracle expectations** - `74e6de3` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `third_party/simdjson` - Advances the gitlink from v4.6.1 to official v4.6.4.
- `patches/simdjson-v4.6.4-positive-bigint.patch` - Records the sole downstream semantic delta and its exact upstream provenance.
- `build.rs` - Verifies the upstream checkout, patches a copied singleheader source, and fails on provenance or patch drift.
- `src/native/simdjson_bridge.cpp` - Enables BigInt parsing, kind 9 classification, strict getters, and exact frame text.
- `src/runtime/mod.rs` - Carries additive native kinds through raw `u32` out-parameters without materializing an undeclared Rust enum discriminant.
- `tests/rust_shim_minimal.rs` - Locks signed/unsigned boundaries, float syntax, kind numbering, and wrong-type getter behavior.
- `tests/rust_shim_fast_materializer.rs` - Locks exact root and nested BigInt frame spans without frame growth.
- `element.go` - Adds public Go `TypeBigInt = 9` classification and accurate accessor contracts.
- `element_fuzz_test.go` - Adds positive/negative BigInt seeds, deterministic traversal, and parse-error tightening.
- `element_scalar_test.go` - Replaces the obsolete oversized-uint rejection assertion with parse preservation.
- `materializer_fastpath_test.go` - Requires nested 23-digit integer parsing and classification without prematurely adding Go materialization.
- `testdata/jsontestsuite/expectations.tsv` - Flips exactly three valid integer-only fixtures to v4.6.4 BigInt acceptance.
- `11-CONTEXT.md`, `11-RESEARCH.md`, and `11-02-PLAN.md` - Durably record the approved D-01 amendment and its threat boundary.

## Decisions Made

- Official simdjson v4.6.4 remains the immutable audited base. The downstream delta is one reviewable unified patch, never a dirty submodule, unpublished submodule commit, fork, second source tree, or parser replacement.
- The patch is applied to `$OUT_DIR/simdjson.cpp`; the checked-in submodule stays clean and byte-for-byte official.
- Native kind 9 is transported as raw `u32` in the interim Rust boundary. Public Rust/C ABI enum synchronization remains dependency-ordered to Plan 11-07.
- Existing int64, uint64, and float64 getters retain strict semantics and return wrong-type for BigInt. Exact `GetBigInt` access remains Plan 11-10 work.

## Patch Provenance and Reproducibility

- **Upstream base:** simdjson v4.6.4 at `1bcf71bd85059ab6574ea1159de9298dcc1212c5`.
- **Patch:** `patches/simdjson-v4.6.4-positive-bigint.patch`.
- **Semantic delta:** in each generated architecture copy, route an already detected positive 20-digit unsigned overflow from `INVALID_NUMBER` to the existing `BIGINT_NUMBER` path.
- **Build gate:** require the exact commit, a clean tracked submodule, successful `git apply --check`, successful application to the copied source, and exactly nine patched generated branches.
- **Isolation proof:** a local clean clone built successfully; moving only its submodule checkout off the locked commit caused the Cargo build to fail with the expected base identity.

## TDD Gate Compliance

- **RED:** `d95789d` added native boundary, strict-getter, and exact-frame tests and demonstrated failure against the pinned but unchanged bridge.
- **GREEN:** `6ee2371` implemented the minimal native behavior and passed all 30 focused Rust integration tests.
- **Go reconciliation:** `74e6de3` replaced the obsolete rejection contract, added deterministic BigInt traversal coverage, and passed the complete race-enabled Go suite.

## Deviations from Plan

### Approved Plan Amendment

Official simdjson v4.6.4 and upstream HEAD both returned `NUMBER_ERROR` for positive `uint64 + 1`, contradicting the original assumption that the audited pin alone exposed the required BigInt path. The user approved a minimal patch, and D-01 was amended in context, research, plan, and threat-model artifacts before implementation.

### Auto-fixed Issues

**1. [Rule 1 - Bug] Prevented `git apply` from discovering the outer repository**

- **Found during:** Task 2 (native BigInt propagation)
- **Issue:** The initial output-directory patch command could discover the enclosing Git worktree and return success without changing the copied source.
- **Fix:** Set `GIT_CEILING_DIRECTORIES` at the project root and added an exact nine-occurrence semantic postcondition before compilation.
- **Files modified:** `build.rs`
- **Verification:** The positive BigInt regression passes, the patched output contains all nine expected branches, and the isolated-clone base-drift check fails closed.
- **Committed in:** `6ee2371`

---

**Total deviations:** 1 auto-fixed bug plus 1 user-approved plan amendment.
**Impact on plan:** The amendment preserves the official upstream pin and existing build architecture; the auto-fix strengthens correctness and tamper resistance without adding dependencies or scope.

## Issues Encountered

- The audited upstream behavior did not match the positive-overflow assumption. This was resolved through the explicit D-01 amendment rather than an untracked fork or parser workaround.
- Output-directory patching needed a repository-discovery boundary. The final build now verifies both application mechanics and the resulting semantic edit.

## Verification

- `cargo test --locked --test rust_shim_minimal --test rust_shim_fast_materializer -- --test-threads=1` — 30/30 passed.
- `make verify-contract` — Rust unit/integration tests, generated-header diff, ABI audits, and C layout compile passed.
- `cargo build --release --locked` — passed against the pinned base and approved output-copy patch.
- `PURE_SIMDJSON_LIB_PATH=target/release/libpure_simdjson.dylib go test ./... -race -count=1` — all four Go packages passed.
- `TestJSONTestSuiteOracle` — passed with exactly three integer-only expectation flips; exponent/real overflow and invalid non-string-key fixtures remain rejected.
- Isolated clone reproduction — clean build passed and wrong-base build failed closed.

## Threat and Security Impact

- **T-11-SC mitigated:** upstream trust is bounded by an exact commit, clean tracked checkout, one provenance-recorded patch, hunk validation, and a semantic postcondition. No package install, dirty submodule, unpublished commit, or second vendored source was introduced.
- **T-11-04 preserved:** BigInt bytes remain borrowed only inside the existing document-owned frame lifetime; root and nested exact-span tests cover the unchanged ownership boundary.
- No unplanned network endpoint, authentication path, schema boundary, or file-access surface was introduced beyond the amended and documented build-time patch boundary.

## Known Stubs

None. `GetBigInt` and Go BigInt materialization are intentionally absent because Plan 11-10 owns those dependency-ordered public behaviors; this plan leaves no placeholder API or mock data path.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 11-03 can remove the now-dead registry precision-loss fallback.
- Plan 11-07 can add synchronized public Rust/C/Go ABI enum mirrors for kind 9.
- Plan 11-10 can add `GetBigInt` and complete Go materialization using the exact native frame text established here.
- No blockers remain for the dependent Phase 11 plans.

## Self-Check: PASSED

- All created and modified implementation artifacts exist.
- Task, RED, decision-amendment, and GREEN commits are present in repository history.
- The submodule remains clean at the exact official v4.6.4 commit.
- Required requirement IDs and stub tracking are present in this summary.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
