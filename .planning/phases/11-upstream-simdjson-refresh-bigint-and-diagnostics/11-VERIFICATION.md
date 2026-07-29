---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
verified: 2026-07-29T12:28:41Z
status: gaps_found
score: 81/84 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Accepted configured depths remain safe on malformed input, and syntax failures return either a proven byte offset or explicit unknown instead of terminating the process."
    status: failed
    reason: "On the immutable v0.1.7 release, a parser configured with max_capacity=1000000 and max_depth=256002 parses a valid 256000-level document successfully, but the same-depth document with an innermost trailing comma exits with SIGSEGV (139) during recursive diagnostic replay. The option accepts the depth, the primary parser can process it, and only the error-location replay exhausts the native call stack."
    artifacts:
      - path: "src/native/simdjson_bridge.cpp"
        issue: "consume_ondemand_value recursively calls itself once per array/object level while using caller-controlled max_depth; the malformed-input replay can overflow the process stack."
      - path: "parser_options.go"
        issue: "WithMaxDepth accepts every positive uint32 value without imposing a bound that is safe for the recursive diagnostic implementation."
      - path: "tests/rust_shim_diagnostics.rs"
        issue: "The replay depth-boundary regression uses MAX_DEPTH=4 and does not exercise a large accepted depth in an isolated subprocess."
    missing:
      - "Replace call-stack-recursive diagnostic traversal with bounded iterative traversal, or define and enforce one demonstrably safe maximum depth across option validation and every parser-owned traversal."
      - "Add a subprocess regression at a large accepted depth proving malformed syntax returns ErrInvalidJSON with a known/unknown offset state and never terminates by signal."
      - "Reconcile the internal fast materializer's separate hard-coded depth 1024 with the chosen parser-depth contract."
---

# Phase 11: Upstream simdjson refresh, exact big integers, and production diagnostics Verification Report

**Phase Goal:** Establish the compatibility foundation for v0.2 by moving to the current audited simdjson 4.6 patch release, preserving oversized integer literals as exact decimal text, and exposing the small set of parser controls and diagnostics required for production operation.

**Verified:** 2026-07-29T12:28:41Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Verification Scope and Source Identity

The checked-out branch predates the published release, so it was not used as evidence for shipped behavior. Verification used the user-designated immutable production source:

| Property | Independently observed value |
| --- | --- |
| Annotated tag | `v0.1.7` |
| Tag object | `cd153ae770745dad124750ec8dd765eb1afdb83e` |
| Tag target | `ab86c2e1e666c6c313d1dd951c37a8c43538c407` |
| Audited `origin/main` | `7c1051b8758139645ce437a8ae38ca75fc8f2174` |
| Main ancestry | `git merge-base --is-ancestor v0.1.7^{commit} origin/main` returned 0 |
| simdjson gitlink | `1bcf71bd85059ab6574ea1159de9298dcc1212c5` (v4.6.4) |
| Bootstrap / ABI | `0.1.7` / `0x00010002` |

No previous `11-VERIFICATION.md` existed. SUMMARY claims were not treated as evidence.

## Goal Achievement

### Roadmap Contract

| # | Observable truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | simdjson v4.6.4 is integrated and Go/Rust/C++ contract, correctness, benchmark, and five-target gates are green. | ✓ VERIFIED | Exact audited gitlink and patch gate in `build.rs`; fresh contract, Rust, Go race, and no-baseline benchmark runs passed; release run `30030435051` succeeded for all five targets. |
| 2 | Integers above `uint64` become `TypeBigInt` and `GetBigInt` returns exact decimal text without automatic `math/big` allocation. | ✓ VERIFIED | Kind 9 flows from native materialization through Rust/FFI to `element.go`; fresh BigInt tests passed; no Go production import of `math/big`. |
| 3 | Kernel report/override and safe bounded capacity/depth controls are available. | ✗ FAILED | Kernel and capacity paths are wired, but an accepted large `WithMaxDepth` value permits malformed input to crash in diagnostic replay. |
| 4 | Syntax/UTF-8 failures report a real offset when available and explicit unknown otherwise, never fabricating one. | ✗ FAILED | Covered corpus behavior is correct, but the accepted large-depth syntax failure terminates with SIGSEGV and reports neither a known offset nor explicit unknown. |

**Roadmap score:** 2/4 criteria verified

### PLAN Must-Have Resolution

The 14 PLAN files contain 80 detailed truths. Every truth was checked against tagged source, wiring, tests, or hosted/public evidence.

| Plan | Resolution | Notes |
| --- | --- | --- |
| 11-01 | 4/4 VERIFIED | Immutable v4.6.4 base/patch contract and build-time audit are substantive. |
| 11-02 | 6/6 VERIFIED | ABI 1.2 and exact BigInt kind/span contract are present and tested. |
| 11-03 | 6/6 VERIFIED | Go `TypeBigInt`/`GetBigInt` path is copied, exact, and wired. |
| 11-04 | 6/6 VERIFIED | Configured parser creation stores and applies exact capacity/depth values. |
| 11-05 | 7/8 VERIFIED, 1 FAILED | The truth requiring resource exhaustion to remain unknown and never trigger an unbounded attempt fails at large accepted depth. |
| 11-06 | 5/5 VERIFIED | Go error offset/known transport and reset behavior are wired. |
| 11-07 | 7/7 VERIFIED | Kernel reporting/override and post-creation lock behavior are present. |
| 11-08 | 4/4 VERIFIED | ParserPool options and lock behavior are wired. |
| 11-09 | 5/5 VERIFIED | Contract/header validation covers the additive ABI. |
| 11-10 | 5/5 VERIFIED | BigInt and diagnostics cross-language tests are substantive. |
| 11-11 | 5/5 VERIFIED | Limits and kernel integration tests are substantive. |
| 11-12 | 6/6 VERIFIED | Correctness/contract integration is present. |
| 11-13 | 7/7 VERIFIED | Readiness, benchmark, and release-facing gates are present. |
| 11-14 | 6/6 VERIFIED | Annotated main-ancestor tag, CI release, five R2 jobs, and three fallback jobs were independently observed. |

Merged with the four non-negotiable roadmap truths, **81/84 must-haves are verified**. The three failed truth rows have one root cause and are grouped as one actionable gap.

## Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `third_party/simdjson` + `patches/simdjson-v4.6.4-*.patch` + `build.rs` | Exact audited upstream and deterministic patch gate | ✓ VERIFIED | Base commit, clean-submodule checks, OUT_DIR copy, patch apply-check, and nine semantic postconditions are enforced. |
| `src/native/simdjson_bridge.{h,cpp}` | ABI 1.2 bridge, BigInt spans, configured parser, kernel, diagnostics | ⚠ PARTIAL | Substantive and exported; diagnostic replay is unsafe at a large accepted depth. |
| `src/lib.rs`, `src/runtime/registry.rs`, `src/runtime/mod.rs` | Rust ABI exports and runtime ownership/copy paths | ✓ VERIFIED | Mandatory symbols delegate to registry; copied byte/error state and configured construction are present. |
| `include/pure_simdjson.h`, `internal/ffi/bindings.go` | C/Go ABI mirrors | ✓ VERIFIED | ABI/version/status/kind values and function layouts agree; contract audit passed. |
| `element.go`, `parser_options.go`, `parser.go`, `pool.go`, `kernel.go`, `errors.go` | Public Go surface | ⚠ PARTIAL | BigInt, kernel, capacity, and normal diagnostics work; max-depth validation does not protect recursive replay. |
| `tests/rust_shim_*.rs`, Go `*_test.go`, contract scripts | Cross-language regression coverage | ⚠ PARTIAL | Fresh suites pass, but no high-depth crash regression exists. |
| `.github/workflows/release.yml`, `.github/workflows/public-bootstrap-validation.yml` | CI-only publication and fresh-runner validation | ✓ VERIFIED | Hosted runs and public artifacts were checked independently. |

## Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| Audited simdjson gitlink | Compiled native bridge | `build.rs` commit/cleanliness/patch/postcondition gate | ✓ WIRED | Build fails closed on base or patch drift. |
| JSON oversized integer | Go `GetBigInt()` | native kind 9 span → Rust copied bytes → FFI binding → Go string | ✓ WIRED | Exact boundary corpus passed. |
| `WithMaxCapacity` | pre-copy parser rejection | Go normalization → configured FFI → registry comparison | ✓ WIRED | Capacity is checked before `mem::take`, resize, or input copy. |
| `WithMaxDepth` | primary parser and both replay parsers | stored config → `DiagnosticReplayLimits` | ⚠ UNSAFE | Exact values are forwarded, but recursive replay turns a legal value into stack exhaustion. |
| Upstream error pointer | Go `Error.Offset` / `HasOffset` | in-range pointer proof → Rust transport → Go typed error | ✓ WIRED | Zero is distinguishable from unknown in covered cases. |
| Parser/kernel setup | process-wide override lock | Go mutex + mandatory native support/select exports | ✓ WIRED | Reporting is cache-only and selection locks after parser creation. |
| `v0.1.7` tag | public bootstrap | release workflow → signed assets/checksums → validation workflow | ✓ WIRED | Publication precedes successful five-target R2 and three-target fallback validation. |

## Data-Flow Trace (Level 4)

| Artifact | Data variable | Source | Produces real data | Status |
| --- | --- | --- | --- | --- |
| `element.go` | BigInt decimal bytes | Real parsed token from patched simdjson | Yes; copied across native/Rust/Go lifetime boundaries | ✓ FLOWING |
| `parser_options.go` / registry | capacity/depth config | Public options | Yes; exact values reach all parser constructors | ⚠ FLOWING BUT UNSAFE |
| `errors.go` | offset + known flag | Checked upstream `current_location()` pointer | Yes for covered failures | ⚠ FLOWING UNTIL LARGE-DEPTH CRASH |
| `kernel.go` | active/requested kernel | Native runtime support/selection | Yes; real runtime result, not hardcoded | ✓ FLOWING |
| release/bootstrap workflows | artifacts and checksum metadata | Immutable tag build and public object storage | Yes; 18 release assets and live metadata observed | ✓ FLOWING |

## Behavioral Spot-Checks

All local commands below ran in a fresh detached checkout of `v0.1.7` with the exact submodule initialized.

| Behavior | Command / invocation | Result | Status |
| --- | --- | --- | --- |
| Cross-language contract and Rust behavior | `make verify-contract` | 17 Rust unit tests, 67 integration tests, 25 header audits, and C layout compile passed | ✓ PASS |
| Go API under race detector | fresh release dylib + `go test ./... -race -count=1` | All four Go packages passed | ✓ PASS |
| PR benchmark harness | `scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir <temp>` | Exit 0; benchmark output and JSON summary produced | ✓ PASS |
| Five-target release | Hosted run `30030435051` on tag target | Eight jobs successful, including five platform builds, Alpine smoke, and publish | ✓ PASS |
| Public bootstrap | Hosted run `30448288140` for `v0.1.7` | Five R2 targets and linux/darwin/windows fallback jobs successful | ✓ PASS |
| Public metadata | anonymous GET of `v0.1.7/SHA256SUMS` and `latest.json` | HTTP 200; five real checksum rows; latest version `v0.1.7` | ✓ PASS |
| Deep valid input control | ABI parser: capacity `1,000,000`, depth `256,002`, input `"[" × 256000 + "0" + "]" × 256000` | create 0, parse 0, free 0; process exit 0 | ✓ PASS |
| Deep malformed syntax | Same config, input `"[" × 256000 + "0," + "]" × 256000` | Process terminated with exit 139 / SIGSEGV | ✗ FAIL |

### Exact Failure Reproduction

Against the fresh `target/release/libpure_simdjson` built from `v0.1.7`:

1. Call `pure_simdjson_parser_new_configured(1_000_000, 256_002, &parser)`.
2. Build 512,002 bytes: `b"[" * 256_000 + b"0," + b"]" * 256_000`.
3. Call `pure_simdjson_parser_parse(parser, input, len(input), &doc)`.
4. The process exits 139 instead of returning a typed status and known/unknown diagnostic state.

The control differs only by removing the comma. It returns success from create, parse, document free, and parser free. Smaller malformed probes at depths 1,025 through 128,000 returned ordinary `ErrInvalidJSON` with diagnostic state; 256,000 exposed the stack limit.

The source explains the transition: `consume_ondemand_value` at `src/native/simdjson_bridge.cpp:426` recursively calls itself at lines 458 and 490, while `parser_options.go:75-92` permits depth values through `uint32` maximum.

## Probe Execution

No Phase 11 PLAN/SUMMARY declares a `probe-*.sh`, and no conventional `scripts/*/tests/probe-*.sh` exists. Step 7c is therefore **SKIPPED (no declared probe scripts)**. The direct ABI crash reproduction above is recorded as a behavioral spot-check.

## Requirements Coverage

| Requirement | Source plans | Status | Evidence |
| --- | --- | --- | --- |
| UP-01 | 11-01, 11-09, 11-12, 11-13, 11-14 | ✓ SATISFIED | Exact v4.6.4 integration, patch gate, contract tests, and cross-platform release evidence. |
| NUM-01 | 11-02, 11-03, 11-09, 11-10, 11-12, 11-14 | ✓ SATISFIED | `TypeBigInt` kind 9 is stable and exact decimal text crosses every layer. |
| NUM-02 | 11-02, 11-03, 11-10, 11-12, 11-14 | ✓ SATISFIED | Exact accessor is opt-in; no automatic Go `math/big` allocation. |
| DIAG-01 | 11-07, 11-08, 11-11, 11-12, 11-14 | ✓ SATISFIED | Kernel report/override is real, validated, and locked after parser creation. |
| DIAG-02 | 11-05, 11-06, 11-09, 11-10, 11-12, 11-14 | ✗ BLOCKED | Normal known/unknown behavior works, but an accepted large-depth syntax failure crashes before returning either state. |
| LIMIT-01 | 11-04, 11-05, 11-08, 11-09, 11-11, 11-12, 11-14 | ✗ BLOCKED | Capacity and normal depth boundaries work; the advertised accepted depth range is not safe for diagnostic replay. |

All six requirements mapped to Phase 11 appear in PLAN frontmatter; no Phase 11 requirement is orphaned.

## Anti-Patterns and Advisory Findings

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `src/native/simdjson_bridge.cpp` | 426, 458, 490 | Caller-bounded but call-stack-recursive error traversal | 🛑 BLOCKER | Large accepted malformed input can terminate the host process. |
| `src/native/simdjson_bridge.cpp` | 845-849 | Internal fast materializer hard-codes depth 1024 | ⚠ WARNING | A valid document accepted above depth 1024 can return depth-limit status from the internal benchmark/test materializer. No exported Go API currently uses this helper. |
| `README.md` at `v0.1.7` | 67 | Says the published default-install version is still `v0.1.4` | ⚠ WARNING | Consumer-facing release status is stale after successful `v0.1.7` publication. |
| `docs/releases.md` at `v0.1.7` | 21, 60, 141, 160 | Operator examples remain pinned to `0.1.4` | ℹ INFO | Examples are usable only after manual version substitution; the live release itself is correctly `v0.1.7`. |

No unreferenced `TBD`, `FIXME`, or `XXX` debt marker was found in the Phase 11 production-file delta.

## Deferred-Phase Filter

Phases 12-16 cover DOM navigation, batched On-Demand extraction, borrowed views, streaming, and final v0.2 stabilization/release. None specifically promises to make the existing Phase 11 configured-depth diagnostic replay safe. The gap is therefore **not deferred**.

## Human Verification Required

None. Visual behavior is not in scope, and the tag identity, hosted jobs, public endpoints, ABI behavior, and crash were all directly observable.

## Gaps Summary

Phase 11 substantially delivers the v4.6.4 refresh, exact BigInt path, kernel controls, release artifacts, and ordinary diagnostics. It does not yet deliver a safe production depth-control/diagnostic contract.

The smallest coherent closure is:

1. Make error-location traversal iterative, or cap accepted depth to a value proven safe for every parser-owned traversal.
2. Apply that contract consistently to primary parsing, both diagnostic replay passes, and the internal materializer.
3. Add a subprocess regression so a signal is reported as a test failure rather than killing the test runner.
4. Re-run the contract suite, Go race suite, large-depth valid/malformed controls, benchmark harness, and release readiness checks.

---

_Verified: 2026-07-29T12:28:41Z_
_Verifier: the agent (gsd-verifier)_
