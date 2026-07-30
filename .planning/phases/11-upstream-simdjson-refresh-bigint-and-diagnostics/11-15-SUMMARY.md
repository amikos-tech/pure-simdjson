---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 15
subsystem: parser-safety
tags: [go-api, cpp, diagnostics, depth-limits, subprocess, ffi-contract, tdd]

# Dependency graph
requires:
  - phase: 11-05
    provides: Two-pass upstream diagnostic replay with proven known/unknown offsets
  - phase: 11-11
    provides: Immutable Go parser options and configured native construction
  - phase: 11-14
    provides: Published and validated ABI 1.2 compatibility baseline
provides:
  - One supported maximum depth of 1024 across Go validation and native parser-owned traversal
  - Pre-allocation native rejection for configured depths above 1024
  - Child-process proof that malformed input at the largest accepted nesting returns typed diagnostics without a signal
  - Generated C and FFI documentation for the zero-or-1-through-1024 contract
affects: [12-dom-navigation-and-utilities, 16-v0.2-release, diagnostics, parser-options]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Reject unsupported work bounds before native selection, allocation, or recursive traversal
    - Isolate signal-sensitive native regressions in a child test process
    - Generate public C documentation from authoritative Rust declarations

key-files:
  created:
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-15-SUMMARY.md
  modified:
    - parser_options.go
    - parser_options_test.go
    - src/native/simdjson_bridge.cpp
    - tests/rust_shim_limits.rs
    - tests/rust_shim_diagnostics.rs
    - src/lib.rs
    - include/pure_simdjson.h
    - docs/ffi-contract.md

key-decisions:
  - "Define 1024 as both the default and the supported maximum parser depth; zero still normalizes to 1024."
  - "Reject native depth above 1024 before the irreversible implementation-selection lock and before parser allocation."
  - "Use the same native maximum for both replay passes, the recursive consumer, and the internal materializer."

patterns-established:
  - "Depth safety: Go rejects unsupported options before library resolution, while native construction independently rejects them before process-global state changes."
  - "Signal regression: the parent test requires a normal zero exit from a child that performs the maximum accepted malformed parse."

requirements-completed: [DIAG-02, LIMIT-01]

# Metrics
duration: 14min
completed: 2026-07-29
---

# Phase 11 Plan 15: Maximum-Depth Diagnostic Safety Summary

**One 1024-depth ceiling now protects configured construction, diagnostic replay, and internal materialization, with child-process proof that the largest accepted malformed input returns normally**

## Performance

- **Duration:** 14 min
- **Started:** 2026-07-29T13:15:21Z
- **Completed:** 2026-07-29T13:29:24Z
- **Tasks:** 2
- **Files modified:** 8 task files

## Accomplishments

- Added Go validation that accepts zero or depths `1..1024` and returns `ErrInvalidOption` for 1025 before native library resolution.
- Added native validation before implementation-selection locking or parser allocation, while preserving an untouched output handle on rejection.
- Replaced the separate materializer depth literal with the shared native ceiling and aligned its N-1-success/N-failure boundary with parsing.
- Added a subprocess regression for 1023 nested containers under configured depth 1024; malformed JSON returns `PURE_SIMDJSON_ERR_INVALID_JSON` with either a proven in-range offset or exact unknown state.
- Preserved the nine-case v4.6.4 diagnostic corpus, ABI `0x00010002`, all public symbols and layouts, the full race suite, and the existing benchmark/readiness gates.

## Task Commits

Task work was committed atomically:

1. **Task 1 RED: Add failing maximum-depth safety contracts** - `a5b3d77` (test)
2. **Task 1 GREEN: Enforce maximum supported parser depth** - `ae1a2c1` (feat)
3. **Task 2: Synchronize the supported depth contract** - `5d44d97` (docs)

Plan metadata is committed with this summary.

## Files Created/Modified

- `parser_options.go` - Defines the supported Go depth ceiling and rejects larger options before library resolution.
- `parser_options_test.go` - Covers depth 1025 in normalization and missing-library preflight tests.
- `src/native/simdjson_bridge.cpp` - Shares one depth ceiling across configured construction, replay, recursive consumption, and materialization.
- `tests/rust_shim_limits.rs` - Proves zero and 1024 construct successfully while 1025 fails without changing the output handle.
- `tests/rust_shim_diagnostics.rs` - Runs the maximum accepted malformed replay in a child process and validates truthful diagnostic state.
- `src/lib.rs` - Documents the configured constructor's zero-or-`1..1024` contract.
- `include/pure_simdjson.h` - Regenerated public C documentation from the Rust declaration.
- `docs/ffi-contract.md` - States the shared parse, replay, and materialization depth boundary in plain language.

## Decisions Made

- A safe fixed ceiling is smaller and clearer than adding a second traversal implementation solely for diagnostic replay.
- Native validation remains independent of Go validation so direct C callers receive the same safety contract.
- The child accepts both valid diagnostic outcomes: a proven offset inside the input or the exact `(UINT64_MAX, false)` unknown pair. It never guesses an offset.

## TDD Gate Compliance

- **RED:** `a5b3d77` failed because Go proceeded to library loading for depth 1025 and native configured construction returned success.
- **GREEN:** `ae1a2c1` made the focused Go suite and all 14 Rust limit/diagnostic tests pass.
- No refactor commit was needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Corrected stale plan position before state advancement**

- **Found during:** Plan closeout
- **Issue:** Phase-start tracking showed Plan 1 of 15 even though summaries 11-01 through 11-14 already existed.
- **Fix:** Used the registered state handler to restore Plan 15 of 15 before the normal advance, progress, metric, and session handlers.
- **Files modified:** `.planning/STATE.md`
- **Verification:** Final state reports Phase 11 complete and the roadmap reports 15/15 plans.
- **Committed in:** Plan metadata commit

**Total deviations:** 1 auto-fixed blocking workflow-state issue.
**Impact on plan:** Implementation scope did not change; the correction prevents completed Phase 11 work from being recorded as Plan 2 of 15.

## Issues Encountered

None. The initial Go and Rust failures were the expected TDD RED evidence.

## Verification

- `cargo test --locked --test rust_shim_limits --test rust_shim_diagnostics -- --test-threads=1` - passed 14/14 focused tests, including the child-process regression.
- `make verify-contract` - passed `cargo check`, all 86 Rust tests, 25 generated-header audits, header regeneration diff, and the C layout compile.
- `cargo build --release --locked` - built the fresh platform library successfully.
- `PURE_SIMDJSON_LIB_PATH=<fresh release library> go test ./... -race -count=1` - passed all four Go packages under the race detector.
- `scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir <fresh temporary directory>` - exited zero and produced non-empty benchmark, JSON summary, and Markdown files. This is a regression signal, not a comparative performance claim.
- `bash scripts/release/check_readiness.sh` - passed the non-strict source-readiness gate.
- A second `make generate-header` was byte-stable; the committed header diff contains documentation only.
- The exact diagnostic corpus remains known at offsets `3, 8, 16, 0, 15, 9`, with empty input, invalid UTF-8, and unclosed string explicitly unknown.
- Plans and summaries 11-01 through 11-14 remain unchanged.

## Threat and Security Impact

- **T-11-07 mitigated:** caller-controlled depth above 1024 is rejected before native state changes, allocation, or recursive replay.
- **T-11-05 preserved:** malformed-input diagnostics still report only upstream-proven offsets or explicit unknown.
- **T-11-SC preserved:** no dependency, ABI value, public symbol, layout, release identity, tag, remote ref, or publication state changed.
- No security-relevant surface outside the plan threat model was introduced.

## Known Stubs

None.

## User Setup Required

None - parser depth safety requires no external configuration.

## Next Phase Readiness

- The Phase 11 verification gap is closed locally with fresh contract, race, benchmark-signal, and readiness evidence.
- Phase 11 is ready for verification and Phase 12 planning; Phase 16 still owns the final v0.2 release.

## Self-Check: PASSED

- All eight task files and this summary exist.
- Task commits `a5b3d77`, `ae1a2c1`, and `5d44d97` are present in repository history.
- Requirements `DIAG-02` and `LIMIT-01` are recorded in summary frontmatter.
- The unrelated config and Phase 10 learnings artifacts remain unstaged and unmodified by this plan.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-29*
