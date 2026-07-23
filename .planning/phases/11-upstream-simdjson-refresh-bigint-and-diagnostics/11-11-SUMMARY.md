---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 11
subsystem: parser-lifecycle
tags: [go-api, parser-options, capacity, depth, pooling, concurrency, tdd]

# Dependency graph
requires:
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-08 made configured parser construction mandatory in the ABI 1.2 Go binding
  - phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
    provides: Plan 11-09 froze configured construction and statuses 9/10 in the generated public contract
provides:
  - Opaque immutable ParserOption values with validated capacity and depth normalization
  - Configured NewParser construction with exact effective limits stored on every Parser
  - Fallible pure-Go ParserPool construction with homogeneous misses and mismatch rejection
  - Dedicated public sentinels for invalid options and configured capacity failures
affects: [11-12, 11-13, 16-v0.2-release, parser-api, concurrency]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Normalize public options once into a private comparable configuration before native resolution
    - Store one effective configuration on parsers and pools and compare it under the parser lock
    - Keep pool construction pure Go while deferring native bootstrap to the first cache miss

key-files:
  created:
    - parser_options.go
    - parser_options_test.go
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-11-SUMMARY.md
  modified:
    - parser.go
    - errors.go
    - errors_test.go
    - library_loading_test.go
    - pool.go
    - pool_test.go
    - example_test.go
    - docs/concurrency.md

key-decisions:
  - "Normalize omitted and explicit-zero limits to 0xFFFFFFFF/1024 before library resolution, with later duplicate options winning."
  - "Keep NewParserPool construction pure Go; the first Get miss is the first native-library touch."
  - "Reject mismatched pool insertion under the parser mutex after preserving closed and busy error precedence."

patterns-established:
  - "Opaque option values: exported constructors create immutable values while all configuration fields and normalization remain private."
  - "Homogeneous pools: every miss receives the stored comparable config and Put cannot admit a parser from another limit policy."

requirements-completed: [LIMIT-01]

# Metrics
duration: 13min
completed: 2026-07-23
---

# Phase 11 Plan 11: Immutable Parser Limits and Homogeneous Pools Summary

**Validated capacity/depth options now flow once through configured ABI construction, while parser pools preserve one immutable limit policy without loading native code until the first miss**

## Performance

- **Duration:** 13 min
- **Started:** 2026-07-23T14:43:34Z
- **Completed:** 2026-07-23T14:56:35Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments

- Added opaque `ParserOption`, `WithMaxCapacity`, and `WithMaxDepth` values with default normalization, strict ABI-width validation, and deterministic last-option-wins behavior.
- Changed `NewParser` to validate before library resolution, call `ParserNewConfigured`, and retain one comparable effective configuration on the parser.
- Added truthful `ErrInvalidOption` and `ErrCapacityLimitExceeded` sentinels, including native status-9 mapping and stale-diagnostic regression coverage.
- Changed `NewParserPool` to the approved `(*ParserPool, error)` signature without introducing construction-time bootstrap or native loading.
- Made every pool miss use its stored normalized limits and made `Put` reject capacity or depth mismatches under the parser lock.
- Updated every constructor call site, the executable example, and concurrency guidance for the deliberate source break.

## Task Commits

Each TDD gate was committed atomically:

1. **Task 1 RED: Add failing parser option contract** - `45251ae` (test)
2. **Task 1 GREEN: Add immutable parser limits** - `f40e7b5` (feat)
3. **Task 2 RED: Add failing homogeneous pool contract** - `0812776` (test)
4. **Task 2 GREEN: Enforce homogeneous parser pools** - `66a29db` (feat)

Plan metadata is committed with this summary.

## Files Created/Modified

- `parser_options.go` - Defines opaque public option values, private comparable config, defaults, validation, and duplicate resolution.
- `parser_options_test.go` - Covers invalid options, pre-load rejection, exact capacity, stale detail reset, and configured/default depth boundaries.
- `parser.go` - Normalizes options before loading and routes construction through `ParserNewConfigured`.
- `errors.go` - Adds invalid-option and capacity-limit sentinels and maps native status 9.
- `errors_test.go` - Pins centralized status mapping for capacity, depth, and invalid native arguments.
- `library_loading_test.go` - Updates the compile-time constructor contract from the obsolete exact zero-argument function type to the approved variadic option signature.
- `pool.go` - Stores normalized config, defers native work to `Get`, and rejects mismatched `Put` values.
- `pool_test.go` - Covers constructor purity, equivalent defaults, configured misses, homogeneous rejection, reuse, lifecycle, finalizers, and races.
- `example_test.go` - Handles the pool constructor's error return in the executable example.
- `docs/concurrency.md` - Documents immutable options, last-wins behavior, bootstrap timing, and mismatch rejection.

## Decisions Made

- Zero capacity/depth means the documented defaults only during normalization; every constructed parser and pool stores the resulting effective values.
- Capacity values `1..31`, negative values, and values beyond the ABI width fail in Go before any native work; capacity `32` is valid.
- Pool construction validates and stores config but intentionally does not resolve or download a library. A first `Get` miss owns that work.
- `Put` preserves the existing closed and busy error precedence, then compares the immutable parser and pool configurations before insertion.

## TDD Gate Compliance

- **Task 1 RED:** `45251ae` failed on missing option constructors, normalizer/config storage, variadic construction, and capacity sentinel mapping.
- **Task 1 GREEN:** `f40e7b5` made the option, capacity, depth, status, documentation, and existing zero-argument constructor suites pass.
- **Task 2 RED:** `0812776` failed on the old pool return count, absent stored config, and missing mismatch enforcement.
- **Task 2 GREEN:** `66a29db` made the focused race suite, executable example, full root suite, and all updated call sites pass.
- No refactor commits were needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated the obsolete exact NewParser function-type assertion**

- **Found during:** Task 1 RED
- **Issue:** `library_loading_test.go` asserted `NewParser` had the exact non-variadic type `func() (*Parser, error)`, which cannot coexist with the approved `func(...ParserOption) (*Parser, error)` API even though zero-argument calls remain source-compatible.
- **Fix:** Replaced the obsolete assertion with a compile-time variadic signature assertion while retaining full-package coverage for existing `NewParser()` calls.
- **Files modified:** `library_loading_test.go`
- **Verification:** `go test . -run '^TestNewParserVariadicSignature$' -count=1` and `go test . -count=1`
- **Committed in:** `45251ae`

**Total deviations:** 1 auto-fixed blocking issue.
**Impact on plan:** The change was required to express the approved constructor contract; no production scope was added.

## Issues Encountered

None.

## Verification

- `cargo build --release --locked` - passed against the current ABI 1.2 native library.
- `go test . -run '^Test(ParserOption|ParserCapacity|ParserDepth)' -count=1` - passed option normalization and capacity/depth boundaries.
- `go test . -run '^TestSentinelMapping$' -count=1` - passed dedicated capacity-limit mapping.
- `go test . -run '^TestParserPool.*(Option|Config|ConstructionDoesNotLoad|Reuse|Concurrent|Busy|Closed|Nil)' -race -count=1` - passed constructor, homogeneity, lifecycle, reuse, and concurrency coverage.
- `go test . -run '^ExampleParserPool_Get$' -count=1` - passed the migrated executable example.
- `go test ./... -race -count=1` - passed all four Go packages under the race detector.
- `make verify-docs` - passed repository documentation checks.
- `go doc ParserOption`, `go doc NewParserPool`, and an exported-package scan confirmed opaque fields and no exported `parserConfig` signature.
- Repository call-site scan found no stale one-return `NewParserPool` usage outside historical planning artifacts.

## Threat and Security Impact

- **T-11-01 mitigated:** caller-controlled limits are rejected before native resolution/allocation; capacity is enforced before native input copy; exact depth boundaries are executable.
- **T-11-06 mitigated:** pool configuration is comparable and immutable, and mismatched insertion is rejected while holding the parser lock.
- **T-11-SC preserved:** no package, dependency, alternate source, network endpoint, or publication path was introduced.
- No security-relevant surface outside the plan threat model was added.

## User Setup Required

None - parser options and pools require no external service configuration.

## Next Phase Readiness

- Plan 11-12 can serialize kernel selection with `NewParser` and `NewParserPool` while preserving this plan's no-native-load pool construction guarantee.
- Plan 11-12 can add known-offset state and kernel-lock sentinels on top of the centralized status mapping updated here.
- Every parser and pool now exposes one stable lifecycle point for the remaining Phase 11 diagnostic controls.

## Self-Check: PASSED

- All ten task files and this summary exist.
- Task commits `45251ae`, `f40e7b5`, `0812776`, and `66a29db` are present in repository history.
- The complete Go race suite, executable example, release build, and documentation checks passed.
- The unrelated dirty planning paths remain unstaged and unmodified by this plan.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-23*
