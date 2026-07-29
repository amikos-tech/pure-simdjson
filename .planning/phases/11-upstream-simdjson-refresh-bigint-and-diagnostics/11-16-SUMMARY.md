---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 16
subsystem: native-parser
tags: [simdjson, bigint, token-validation, rust-ffi, cpp, jsontestsuite, tdd]

# Dependency graph
requires:
  - phase: 11-02
    provides: Official v4.6.4 gitlink, one audited output-copy patch, and native kind-9 BigInt preservation
  - phase: 11-10
    provides: Copied Rust ABI BigInt accessor with exact signed decimal ownership
  - phase: 11-15
    provides: Latest verified parser-safety baseline before gap closure
provides:
  - Delimiter validation before every generated BigInt early return
  - Fail-closed nine-architecture parity guards for all three BigInt branch shapes
  - NUL-safe public Rust ABI regressions for malformed oversized integer suffixes
  - Manifest-complete JSONTestSuite rejects for positive, negative, array, and object BigInt corruption
affects: [11-17, 12-dom-navigation-and-utilities, native-parser, correctness-oracle]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Validate token boundaries before any exact-text BigInt early return
    - Count every guarded generated branch and reject every legacy unguarded shape before compilation
    - Keep project-owned malformed cases in the existing manifest-complete correctness oracle

key-files:
  created:
    - testdata/jsontestsuite/cases/n_array_bigint_suffix_plus.json
    - testdata/jsontestsuite/cases/n_number_bigint_negative_suffix_underscore.json
    - testdata/jsontestsuite/cases/n_number_bigint_positive_suffix_x.json
    - testdata/jsontestsuite/cases/n_object_bigint_suffix_slash.json
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-16-SUMMARY.md
  modified:
    - patches/simdjson-v4.6.4-positive-bigint.patch
    - build.rs
    - tests/rust_shim_bigint.rs
    - testdata/jsontestsuite/expectations.tsv

key-decisions:
  - "Validate delimiters inside all three BigInt early-return branches in the single audited output-copy patch."
  - "Treat nine guarded copies and zero unguarded copies as a build-time architecture-parity contract."
  - "Extend the existing manifest-driven JSONTestSuite oracle instead of adding another fixture loader."

patterns-established:
  - "BigInt validity: exact text is preserved only after the next byte is proven to be JSON structural punctuation or JSON whitespace."
  - "Generated-source parity: each protected branch must occur exactly nine times and its unguarded predecessor must occur zero times."

requirements-completed: [UP-01, NUM-01]

# Metrics
duration: 8min
completed: 2026-07-29
---

# Phase 11 Plan 16: BigInt Token-Boundary Validation Summary

**All nine generated parsers now reject dirty oversized-integer suffixes before kind-9 success while preserving valid signed decimal text exactly**

## Performance

- **Duration:** 8 min
- **Started:** 2026-07-29T15:11:56Z
- **Completed:** 2026-07-29T15:20:44Z
- **Tasks:** 2
- **Files modified:** 8 task files

## Accomplishments

- Strengthened the one D-01 patch so too-many-digits, negative-overflow, and positive-overflow BigInt exits validate their delimiter in all nine generated architecture implementations.
- Replaced the former single positive-branch count with build guards requiring 9/9/9 guarded branches and zero retained unguarded branch shapes before C++ compilation.
- Added a NUL-safe Rust ABI matrix covering both signs, roots, arrays, objects, five malformed suffix classes, an untouched document sentinel, and exact valid text across structural and whitespace delimiters.
- Added four exact-byte JSONTestSuite fixtures and sorted `reject` expectations while preserving every existing expectation and all three valid oversized-integer `accept` controls.

## Task Commits

Task work was committed through two complete RED/GREEN cycles:

1. **Task 1 RED: Add failing BigInt token-boundary regressions** - `5cdb755` (test)
2. **Task 1 GREEN: Guard every BigInt early return** - `941e3a7` (feat)
3. **Task 2 RED: Add failing malformed BigInt oracle fixtures** - `dfaad41` (test)
4. **Task 2 GREEN: Register malformed BigInt oracle rejects** - `633e90d` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `patches/simdjson-v4.6.4-positive-bigint.patch` - Validates the next byte before each BigInt early return in every generated parser copy.
- `build.rs` - Enforces exact guarded and unguarded branch counts before compiling the patched output copy.
- `tests/rust_shim_bigint.rs` - Covers malformed byte slices, embedded NUL, untouched document outputs, valid delimiters, kind 9, and exact copied text.
- `testdata/jsontestsuite/cases/n_array_bigint_suffix_plus.json` - Rejects a nested positive BigInt followed by `+`.
- `testdata/jsontestsuite/cases/n_number_bigint_negative_suffix_underscore.json` - Rejects a negative BigInt root followed by `_`.
- `testdata/jsontestsuite/cases/n_number_bigint_positive_suffix_x.json` - Rejects a positive BigInt root followed by `x`.
- `testdata/jsontestsuite/cases/n_object_bigint_suffix_slash.json` - Rejects a nested negative BigInt followed by `/`.
- `testdata/jsontestsuite/expectations.tsv` - Registers the four project-owned fixtures as sorted rejects without changing existing rows.

## Decisions Made

- Delimiter validation stays inside the same upstream `parse_number` branches that return `BIGINT_NUMBER`; no secondary tokenizer, parser, source copy, dependency, or ABI surface was added.
- The build checks complete branch shapes, not just a shared substring, so partial edits or architecture drift fail before native compilation.
- Persistent oracle coverage continues through `TestJSONTestSuiteOracle`, preserving its one-to-one manifest completeness check.

## TDD Gate Compliance

- **Task 1 RED:** `5cdb755` failed because `18446744073709551616x` returned `PURE_SIMDJSON_OK` instead of status 32.
- **Task 1 GREEN:** `941e3a7` made all 8 focused Rust tests pass after guarding the three BigInt exits across all nine architecture copies.
- **Task 2 RED:** `dfaad41` failed because the oracle correctly reported the new fixture files as unlisted.
- **Task 2 GREEN:** `633e90d` added the four sorted reject rows and made the complete 322-case oracle pass.
- No refactor commit was needed for either task.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Corrected stale phase-start plan position before state advancement**

- **Found during:** Plan closeout
- **Issue:** Phase-start tracking reported Plan 1 of 17 even though summaries 11-01 through 11-15 already existed.
- **Fix:** Restored the registered `Plan` field to 16 of 17 before the normal advance, progress, metric, and session handlers.
- **Files modified:** `.planning/STATE.md`
- **Verification:** Final state reports Plan 17 of 17 and ROADMAP reports 16/17 plans executed.
- **Committed in:** Plan metadata commit

**Total deviations:** 1 auto-fixed blocking workflow-state issue.
**Impact on plan:** Implementation scope did not change; the correction prevents plan 11-16 from being recorded as plan 2.

## Issues Encountered

- Repository-wide `cargo fmt --all -- --check` remains red on unrelated pre-existing formatting drift already tracked in `deferred-items.md`. The plan-owned Rust files were formatted individually, and all plan diffs pass `git diff --check`.
- Both behavioral failures were expected TDD RED evidence and were resolved by their corresponding GREEN commits.

## Verification

- `cargo test --locked --test rust_shim_bigint -- --test-threads=1` - passed 8/8 tests.
- `cargo build --release --locked` - passed exact-base, clean-submodule, patch-apply, C++17, and 9/9/9 guarded-branch checks with zero unguarded shapes.
- `PURE_SIMDJSON_LIB_PATH=<fresh release library> go test . -run '^TestJSONTestSuiteOracle$' -count=1` - passed all 322 manifest rows and case files.
- Temporary-copy patch audit - reported exactly nine guarded too-many-digits, nine guarded negative-overflow, and nine guarded positive-overflow branches; all three legacy unguarded shapes were absent.
- `git -C third_party/simdjson rev-parse HEAD` - remained `1bcf71bd85059ab6574ea1159de9298dcc1212c5`.
- `git -C third_party/simdjson status --short` - empty.
- Exactly one `simdjson*.patch` remains under `patches/`.
- Manifest lexical ordering, one exact row per new fixture, unchanged prior rows, exact fixture bytes, and the three valid oversized-integer `accept` controls all passed focused shell assertions.
- `git diff --check` - passed.

## Threat and Security Impact

- **T-11-16-01 mitigated:** malformed `x`, `_`, `+`, `/`, and NUL suffixes cannot cross from untrusted JSON bytes into successful kind-9 values.
- **T-11-16-02 mitigated:** all nine generated implementations must contain every guarded branch shape, with no unguarded predecessor retained.
- **T-11-16-03 mitigated:** four manifest-complete reject fixtures persist the positive-root, negative-root, array, and object regressions.
- **T-11-SC preserved:** the exact official gitlink, one output-copy patch, locked dependencies, and clean submodule remain unchanged.
- No unmodeled network endpoint, authentication path, schema boundary, file-access surface, dependency, or ABI change was introduced.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 11-17 can close the remaining normative C++ exception-status gap.
- The malformed BigInt verification blocker is closed locally without changing the official v4.6.4 source checkout or public ABI.
- No implementation blocker remains from this plan.

## Self-Check: PASSED

- All four created oracle fixtures and this summary exist.
- Task commits `5cdb755`, `941e3a7`, `dfaad41`, and `633e90d` are present in repository history.
- The official simdjson gitlink remains clean at the exact v4.6.4 commit.
- Requirements `UP-01` and `NUM-01`, verification evidence, threat coverage, and stub tracking are present in this summary.
- The unrelated config and Phase 10 learnings changes remain unstaged and unmodified by this plan.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-29*
