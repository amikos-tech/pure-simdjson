---
phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
plan: 01
subsystem: testing
tags: [go, python, abi, materializer, benchmarks]

requires:
  - phase: 07-benchmarks-v0.1-release
    provides: Phase 7 benchmark baseline and Tier 1 materialization handoff
provides:
  - Public-header audit guard rejecting Phase 8 internal traversal prefixes
  - Wave 0 FastMaterializer test names with public-accessor baseline assertions
  - Phase 8 benchmark artifact destination
affects: [phase-08, phase-09, abi-validation, tier1-benchmarks]

tech-stack:
  added: []
  patterns:
    - Python unittest coverage for ABI audit rules
    - Go Wave 0 tests with public-accessor baseline assertions before skip guards

key-files:
  created:
    - materializer_fastpath_test.go
    - testdata/benchmark-results/phase8/.gitkeep
  modified:
    - Makefile
    - tests/abi/check_header.py
    - tests/abi/test_check_header.py

key-decisions:
  - "Makefile verify-contract must pass --rule no-internal-symbols because its explicit rule list bypasses default rules."
  - "FastMaterializer oversized-literal guard uses 18446744073709551616, the current public ErrInvalidJSON parse-rejection fixture."

patterns-established:
  - "Internal ABI leakage guard: parse psdj_internal_ and psimdjson_ prototypes, then fail public-header audits with a dedicated no-internal-symbols rule."
  - "Wave 0 materializer tests: assert current accessor behavior before skipping the not-yet-linked fast materializer."

requirements-completed: [D-01, D-02, D-03, D-04, D-05, D-06, D-07, D-08, D-09, D-10, D-11, D-12, D-13, D-14, D-15, D-16, D-17]

duration: 8min
completed: 2026-04-23
---

# Phase 8 Plan 01: Wave 0 Guardrails Summary

**Header-audit protection and assertive FastMaterializer Wave 0 tests for Phase 8 internal traversal work**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-23T17:05:49Z
- **Completed:** 2026-04-23T17:13:39Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added `FORBIDDEN_INTERNAL_SYMBOL_PREFIXES` plus `rule_no_internal_symbols` so public header audits reject `psdj_internal_` and `psimdjson_` symbols.
- Added eight named `TestFastMaterializer*` tests with public-accessor baseline assertions for parity, numeric kinds, duplicate keys, string ownership, closed docs, subtree materialization, and busy/closed behavior.
- Created the Phase 8 raw benchmark destination under `testdata/benchmark-results/phase8/`.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Header guard tests** - `d5401ba` (test)
2. **Task 1 GREEN: Header audit rule** - `ff3b3e4` (feat)
3. **Task 2 RED: FastMaterializer guardrail tests** - `cfd275c` (test)
4. **Task 2 GREEN: Guarded scaffold and benchmark path** - `cddcbcc` (test)

**Plan metadata:** committed separately after summary/state/roadmap updates.

## Files Created/Modified

- `materializer_fastpath_test.go` - Wave 0 fast-materializer behavior tests and accessor baseline helper.
- `testdata/benchmark-results/phase8/.gitkeep` - Committed Phase 8 benchmark artifact directory.
- `tests/abi/check_header.py` - Internal-prefix parsing and `no-internal-symbols` rule.
- `tests/abi/test_check_header.py` - Unittest coverage for internal symbol rejection.
- `Makefile` - `verify-contract` now includes the explicit `no-internal-symbols` rule.

## Verification

All plan-level checks passed:

```sh
python3 tests/abi/test_check_header.py
python3 tests/abi/check_header.py include/pure_simdjson.h
make verify-contract
go test ./... -run 'TestFastMaterializer' -count=1
```

## Decisions Made

- `make verify-contract` now names `no-internal-symbols` explicitly because the target passes an explicit `--rule` list.
- The oversized-literal parse-rejection test uses `18446744073709551616`; the larger BIGINT-style literal tried during execution currently maps to `ErrPrecisionLoss`, not `ErrInvalidJSON`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added the internal-symbol rule to the Makefile contract gate**
- **Found during:** Task 1 (public-header guards)
- **Issue:** `check_header.py` default rules included `no-internal-symbols`, but `make verify-contract` invoked an explicit rule list that would skip it.
- **Fix:** Added `--rule no-internal-symbols` to the Makefile invocation.
- **Files modified:** `Makefile`
- **Verification:** `make verify-contract` output includes `no-internal-symbols` and passes.
- **Committed in:** `ff3b3e4`

**2. [Rule 1 - Bug] Corrected the oversized-literal fixture to match current public parser behavior**
- **Found during:** Task 2 (FastMaterializer behavior tests)
- **Issue:** The initial 23-digit fixture returned `ErrPrecisionLoss`; the plan requires parse-time `ErrInvalidJSON` coverage for oversized literals that current public parsing rejects.
- **Fix:** Used the existing public parse-rejection boundary `18446744073709551616`.
- **Files modified:** `materializer_fastpath_test.go`
- **Verification:** `go test ./... -run 'TestFastMaterializer' -count=1` passes.
- **Committed in:** `cddcbcc`

---

**Total deviations:** 2 auto-fixed (1 Rule 2, 1 Rule 1)
**Impact on plan:** Both fixes strengthened the planned guardrails without widening the public API or changing benchmark claims.

## Known Stubs

- `materializer_fastpath_test.go` - `requireFastMaterializerLinkedForTest` intentionally skips with `fast materializer implementation not linked`. This is the Wave 0 guard that Plan 08-03 removes when routing tests to the production fast materializer.

## Issues Encountered

- TDD RED commits intentionally introduced failing tests before the GREEN commits made the relevant task slice pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 08-02 can add the internal native frame-stream ABI with the public header guard active. Plan 08-03 can remove the fast-materializer skip guard and route the already-named tests to the internal Go materializer.

## Self-Check: PASSED

- Created/modified key files exist on disk.
- Task commits `d5401ba`, `ff3b3e4`, `cfd275c`, and `cddcbcc` are reachable in git history.
- Stub scan found no untracked TODO/FIXME/placeholder strings; the intentional Wave 0 skip guard is documented under Known Stubs.

---
*Phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi*
*Completed: 2026-04-23*
