---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
verified: 2026-07-29T18:10:22Z
status: passed
score: "104/104 must-haves verified"
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "96/99"
  gaps_closed:
    - "Every C++ exception caught at the Rust/C++ boundary, including parser-side std::bad_alloc, now returns PURE_SIMDJSON_ERR_CPP_EXCEPTION (97) without an escaping diagnostic allocation or process termination."
  gaps_remaining: []
  regressions: []
---

# Phase 11: Upstream simdjson Refresh, BigInt, and Diagnostics Verification Report

**Phase Goal:** Establish the compatibility foundation for v0.2 by moving to the current audited simdjson 4.6 patch release, preserving oversized integer literals as exact decimal text, and exposing the small set of parser controls and diagnostics required for production operation.

**Verified:** 2026-07-29T18:10:22Z

**Status:** passed

**Re-verification:** Yes — after gap-closure Plan 11-18.

## Verification Scope

This report verifies current code and independently executed behavior. SUMMARY claims were used only to locate work.

- Current source HEAD inspected: `03d13eff18933a9a70b1568e59eabf51dc7d1caa`.
- Plans 11-01 through 11-18 contribute 100 detailed truths. The four ROADMAP success criteria remain non-negotiable, for 104 merged must-haves.
- The previous parser-aware `std::bad_alloc` gap received full existence, substance, wiring, data-flow, and subprocess checks. Previously passed items received quick regression checks through current source and the combined Rust, Go, C/header, documentation, oracle, release-source, and benchmark gates.
- No verification override exists.
- The unrelated modified `.planning/config.json` and untracked Phase 10 learnings file were preserved.

## Goal Achievement

### Observable Truths — Roadmap Contract

| # | Observable truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | simdjson v4.6.4 is integrated and the Go/Rust/C++ contract, correctness, benchmark, and five-target gates remain green. | ✓ VERIFIED | The exact clean v4.6.4 gitlink and one audited output-copy patch remain intact. Current Rust/header/C, Go race, 322-case oracle, release-source, readiness, and benchmark gates pass. Five-target workflow contracts and previously accepted immutable hosted evidence are unchanged by Plan 11-18. |
| 2 | Valid integers above `uint64` become `TypeBigInt`, and `GetBigInt` returns exact decimal text without automatic `math/big` allocation. | ✓ VERIFIED | Valid positive/negative roots and nested values remain kind 9 with byte-exact copied text; malformed suffixes reject; no production Go file directly imports `math/big`. |
| 3 | Kernel report/override plus bounded capacity/depth controls are available and safe. | ✓ VERIFIED | Real native implementation names flow through mandatory bindings; selection is process-global and locked; capacity is checked before copy; depth is uniformly bounded at 1024. |
| 4 | Syntax/UTF-8 failures expose a proven upstream byte offset when available and explicit unknown otherwise, without fabrication. | ✓ VERIFIED | The v4.6.4 diagnostic corpus, known-zero state, pointer-range proof, unknown sentinel, stale-state clearing, and maximum-depth subprocess regressions pass. |

**Roadmap score:** 4/4 criteria verified.

### PLAN Must-Have Resolution

| Plan | Resolution | Current evidence |
| --- | --- | --- |
| 11-01 | 4/4 VERIFIED | The operator decision, intermediate-artifact boundary, and publication sequencing records remain coherent. |
| 11-02 | 6/6 VERIFIED | Exact v4.6.4 provenance, complete-token BigInt classification, strict numeric getters, frames, fuzzing, and oracle behavior regress cleanly. |
| 11-03 | 6/6 VERIFIED | Copied BigInt ownership, lifetime, wrong-type behavior, kind hints, write-on-success, and Rust `ffi_wrap` wiring remain substantive. |
| 11-04 | 6/6 VERIFIED | Immutable configured construction, pre-copy capacity enforcement, exact defaults, depth boundaries, and diagnostic clearing remain wired. |
| 11-05 | 8/8 VERIFIED | Upstream-only two-pass replay, exact parser limits, range-proven offsets, explicit unknown state, and bounded recursion remain substantive. |
| 11-06 | 5/5 VERIFIED | Kernel selection is exact, serialized, process-global, runtime-validated, and irreversibly locked. |
| 11-07 | 7/7 VERIFIED | ABI/bootstrap mirrors remain coherent at ABI `0x00010002`; enum growth remains append-only. |
| 11-08 | 4/4 VERIFIED | ABI-only probing precedes complete mandatory binding and cache installation; ABI 1.1 has no fallback path. |
| 11-09 | 5/5 VERIFIED | Generated ABI 1.2 header, C smoke, internal-symbol exclusion, layouts, and normative documentation remain coherent. |
| 11-10 | 5/5 VERIFIED | Public and materializer BigInt paths consume copied exact native data across root and descendant flows. |
| 11-11 | 5/5 VERIFIED | Go options and pools preserve one validated immutable capacity/depth identity. |
| 11-12 | 6/6 VERIFIED | Known/unknown offsets and kernel lifecycle remain wired through real native state and pass race checks. |
| 11-13 | 7/7 VERIFIED | Contract, correctness, release-source, explicit-library, benchmark-signal, and source-readiness gates pass on current source. |
| 11-14 | 6/6 VERIFIED | Previously audited hosted evidence remains immutable and coherent; later Phase 11 source changes are not attributed to the old tag. |
| 11-15 | 4/4 VERIFIED | One 1024 ceiling governs Go validation, native construction, both replay passes, recursive replay, and materialization; maximum-depth malformed input remains subprocess-safe. |
| 11-16 | 6/6 VERIFIED | Malformed suffix classes reject through the real ABI, valid exact text remains kind 9, 9/9/9 architecture guards pass at build time, and four persistent oracle fixtures reject. |
| 11-17 | 5/5 VERIFIED | Direct exception mapping, hidden selectors, returned-`MEMALLOC` distinction, ABI stability, and regression gates remain intact. |
| 11-18 | 5/5 VERIFIED | Parser-aware bad-allocation capture is allocation-safe before the guard; selector 3 uses the production macro; the exact child proves status 97, sentinel integrity, and survival; all preserved contracts regress cleanly. |

**PLAN score:** 100/100 truths verified.

**Merged score:** 104/104 must-haves verified.

## Re-verification of Previous Gap

### Parser-aware `std::bad_alloc` containment

**Status: CLOSED**

| Check | Evidence | Status |
| --- | --- | --- |
| No allocation before the diagnostic guard | `capture_parser_exception(psimdjson_parser *, const std::bad_alloc &)` passes the fixed literal `"std::bad_alloc"` directly as a `std::string_view`; it creates no `std::string`, concatenation, or other allocating temporary. | ✓ VERIFIED |
| Any diagnostic-buffer allocation is caught | `LastErrorBuffer::assign` may allocate with `malloc`, but it is invoked only inside `try_set_last_error_message`'s `try`/`catch (...)`. A second allocation failure is swallowed and cannot escape either `noexcept` helper. | ✓ VERIFIED |
| Production mapper is reached | `PSIMDJSON_CATCH_PARSER_CPP_EXCEPTIONS` performs best-effort capture and then returns the shared `map_cpp_exception`; `psimdjson_parser_parse` uses this exact macro. All mapper overloads return status 97. | ✓ VERIFIED |
| Deterministic production-path seam | Private `force_parser_bad_alloc_for_tests` creates a valid stack parser, throws `std::bad_alloc`, and exits through the same parser-aware macro as production parsing. Selector 3 routes to that helper without adding a public symbol or wrapper. | ✓ VERIFIED |
| Output remains success-only | Selector 3 initializes `0xA5A5A5A5A5A5A5A5`, returns status 127 if it changes, and otherwise returns the helper's exact status. The child accepts only 97, so a sentinel mutation cannot pass. | ✓ VERIFIED |
| Process survival | The independently run exact-filter child exited 0, ran exactly one passing test, and printed `PARSER_BAD_ALLOC_CHILD_OK` only after the status-97 assertion. A signal, abort, `std::terminate`, wrong status, or sentinel mutation would fail. | ✓ VERIFIED |
| Existing distinctions retained | Direct selectors still return exactly 97, 97, and 1. Returned `simdjson::MEMALLOC` and explicit internal failures remain status 127. | ✓ VERIFIED |

The former blocker was the allocating temporary evaluated before the guarded callee. That expression no longer exists. The fixed-literal-to-`string_view` conversion is non-allocating, and the only diagnostic allocation left is inside the catch-all guard.

## Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `third_party/simdjson` | Exact official v4.6.4 gitlink | ✓ VERIFIED | Gitlink and checkout are `1bcf71bd85059ab6574ea1159de9298dcc1212c5`; tag lookup returns `v4.6.4`; tracked and staged submodule state is clean. |
| `patches/simdjson-v4.6.4-positive-bigint.patch`, `build.rs` | One audited output-copy patch plus fail-closed architecture parity | ✓ VERIFIED | Exactly one patch; exact-base and clean-tree checks; guarded branches 9/9/9; zero legacy shapes; C++17 singleheader build retained. |
| `tests/rust_shim_bigint.rs`, JSONTestSuite fixtures | Valid exact-text and invalid-suffix coverage | ✓ VERIFIED | Both signs, roots/nested contexts, dirty delimiter classes, untouched output sentinels, valid delimiter controls, and four persistent oracle rejects pass. |
| `src/native/simdjson_bridge.cpp` | BigInt, limits, diagnostics, kernel, and exception boundary | ✓ VERIFIED | Substantive production implementation; parser-aware bad-allocation capture is now allocation-safe and reaches the common status-97 mapper. |
| `src/native/simdjson_bridge.h`, `src/runtime/mod.rs`, `src/lib.rs` | Stable internal seam and Rust ABI | ✓ VERIFIED | The existing hidden selector symbol and wrapper are unchanged; the private parser helper stays translation-unit-local; ABI remains `0x00010002`. |
| `tests/rust_shim_minimal.rs` | Deterministic exception contract | ✓ VERIFIED | Direct selectors prove 97/97/1; the exact child proves parser-aware status 97, sentinel integrity, one-test isolation, and normal process exit. |
| `include/pure_simdjson.h`, `internal/ffi/types.go`, `docs/ffi-contract.md` | Stable append-only ABI 1.2 contract | ✓ VERIFIED | Generated header and normative contract pass; kind 9 and statuses 97/127 remain unchanged; the fault seam is excluded from the public header. |
| Go public/options/diagnostic/kernel files | Real user-facing data and controls | ✓ VERIFIED | `GetBigInt`, frame materialization, parser options/pools, `HasOffset`, `Kernel`, and `SetKernel` consume real native state/data. |
| Release and public-validation workflows | Five-target build and bootstrap gates | ✓ VERIFIED | Current contract tests preserve five release targets, native/packaged smokes, five hosted targets, three fallback targets, CI-only publication, and origin/main ancestry requirements. |

The automatic artifact checker reported false negatives for the gitlink, conceptual C++ method paths, and patterns split over multiple lines. Manual inspection and executable gates resolved those checks; no artifact is missing, stubbed, or orphaned.

## Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| Official simdjson v4.6.4 | Built native source | exact-base check → output copy → one patch → 9/9/9 guards | ✓ WIRED | Provenance and fail-closed parity execute during every Cargo build. |
| Oversized numeric token | DOM kind 9 | complete-token guards → `number_as_string(true)` | ✓ WIRED | Valid tokens retain exact text; dirty delimiters return invalid JSON before a document is exposed. |
| Native kind 9 | Go `GetBigInt` and materializers | native span/frame → Rust-owned allocation → Go copy/string | ✓ WIRED | Exact positive/negative text survives document lifetime and both materializer paths. |
| Parser limits | primary parse, replay, and materializer | one effective capacity/depth identity and shared ceiling | ✓ WIRED | Capacity gates before copy; recursive paths are bounded at 1024. |
| Upstream location pointer | Go `Error.Offset` / `HasOffset` | checked in-range pointer → Rust transport → Go typed error | ✓ WIRED | Known zero and explicit unknown remain distinct. |
| Go kernel API | simdjson implementation state | mandatory binding → native selection mutex/lock | ✓ WIRED | Report and override use real process-global implementation state. |
| Fixed bad-allocation diagnostic | guarded parser error buffer | allocation-free `string_view` → `try_set_last_error_message` | ✓ WIRED | Any buffer allocation and thrown `bad_alloc` occur inside `catch (...)`. |
| Selector 3 | shared exception mapper | private stack-parser helper → production parser-aware catch macro | ✓ WIRED | The helper returns 97; selector returns 127 if its output sentinel changes. |
| Rust subprocess | selector 3 | exact-filter child → exact status assertion → post-assertion marker | ✓ WIRED | Independent child run exited normally with one passing test and the marker. |
| Returned `simdjson::MEMALLOC` | explicit internal status | `map_error` switch | ✓ WIRED | Returned engine failures remain distinct at status 127. |
| Rust ABI source | generated header → Go binding → C smoke | cbindgen plus complete mandatory binding | ✓ WIRED | Current header diff, 25 audits, C layout compile, and Go binding tests pass. |

## Data-Flow Trace (Level 4)

| Artifact | Data variable | Source | Produces real data | Status |
| --- | --- | --- | --- | --- |
| BigInt public/materializer path | exact decimal text | parsed token → native span/frame → Rust-owned bytes → Go string | Yes | ✓ FLOWING |
| Malformed BigInt rejection | parser status/document output | complete-token delimiter guards | Yes | ✓ FLOWING: status 32 and untouched document sentinel |
| Parser limits | capacity/depth | public options or C constructor | Yes | ✓ FLOWING: normalized once and enforced before unsafe work |
| Diagnostics | offset and known bit | upstream `current_location()` plus pointer proof | Yes when upstream supplies a usable pointer | ✓ FLOWING; otherwise explicit unknown |
| Kernel | active/requested implementation | simdjson runtime implementation registry | Yes | ✓ FLOWING |
| Parser exception status | ABI return code | parser catch → guarded best-effort diagnostic → shared mapper | Yes | ✓ FLOWING: exact status 97 reaches Rust |
| Parser exception output | selector sentinel | initialized local value, writable only on impossible success path | Yes | ✓ FLOWING: child status 97 excludes the sentinel-change status 127 |

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Focused exception boundary | `cargo test --locked --test rust_shim_minimal psimdjson_test_force_ -- --nocapture --test-threads=1` | 4/4 passed, including the parser-aware exact child. | ✓ PASS |
| Exact parser-aware child | `PURE_SIMDJSON_PARSER_BAD_ALLOC_CHILD=1 <rust_shim_minimal> --exact psimdjson_test_force_parser_bad_alloc_is_process_safe --nocapture --test-threads=1` | Exit 0; exactly 1 passed, 0 failed; `PARSER_BAD_ALLOC_CHILD_OK` emitted after exact status-97 assertion. | ✓ PASS |
| BigInt, limits, diagnostics, and exception regressions | `cargo test --locked --test rust_shim_bigint --test rust_shim_minimal --test rust_shim_limits --test rust_shim_diagnostics -- --test-threads=1` | 47/47 passed. | ✓ PASS |
| Full Rust/header/C contract | `make verify-contract && make verify-docs` | 91 Rust tests, deterministic header diff, 25 header audits, C layout compilation, and documentation gate passed. | ✓ PASS |
| Fresh optimized library | `cargo build --release --locked` | Release dylib built; exact-base, clean-tree, patch, and architecture-parity assertions passed. | ✓ PASS |
| Go API race gate | Fresh explicit dylib + `go test ./... -race -count=1 -timeout=180s` | All four Go packages passed under the race detector. | ✓ PASS |
| JSON correctness oracle | Fresh explicit dylib + `go test . -run '^TestJSONTestSuiteOracle$' -count=1` | Complete 322-case oracle passed. | ✓ PASS |
| Release-source contracts | Four release/public/bootstrap Python test files | 10 + 12 + 11 + 10 = 43/43 passed. | ✓ PASS |
| Source readiness | `bash scripts/release/check_readiness.sh` | `basic release readiness checks passed`. | ✓ PASS |
| Benchmark execution signal | `scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir <temp>` | Exit 0; non-empty 72-line benchmark output, 8-line JSON summary, and 5-line Markdown report. | ✓ PASS |

All spot-checks were read-only with respect to repository source and release state. No service, tag, publication, or external mutation was started.

## Probe Execution

No conventional `scripts/*/tests/probe-*.sh` file or Phase 11 declared probe exists.

**Step 7c:** SKIPPED — no probe contract was declared.

## Requirements Coverage

| Requirement | Source plans | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| UP-01 | 11-01, 11-02, 11-07, 11-13, 11-14, 11-16 through 11-18 | Exact audited v4.6.4 plus reproducible compatibility gates | ✓ SATISFIED | Pinning, build, oracle, race, benchmark, release contracts, and the corrected parser exception boundary pass. |
| NUM-01 | 11-02, 11-03, 11-10, 11-13, 11-14, 11-16 | Preserve valid oversized integers as exact text | ✓ SATISFIED | Valid signed/unsigned values are kind 9 and exact; malformed suffixes reject without truncation. |
| NUM-02 | 11-02, 11-03, 11-07 through 11-10, 11-13, 11-14 | `TypeBigInt` / `GetBigInt` without automatic arbitrary precision | ✓ SATISFIED | Kind 9 and exact copied text cross native/Rust/Go; no production direct `math/big` import. |
| DIAG-01 | 11-06 through 11-09, 11-12 through 11-14 | Kernel report and exact diagnostic override | ✓ SATISFIED | Mandatory bindings, real implementation state, process-global locks, and isolated tests pass. |
| DIAG-02 | 11-05, 11-07 through 11-10, 11-12 through 11-15 | Proven offsets where available; explicit unknown otherwise | ✓ SATISFIED | Corpus, pointer proof, known zero, stale-state clearing, limits, and subprocess checks pass. |
| LIMIT-01 | 11-04, 11-07 through 11-09, 11-11 through 11-15 | Safe immutable capacity/depth options and homogeneous pools | ✓ SATISFIED | Capacity precedes copy; depth is uniformly capped at 1024; pool identity includes both values. |

All six requirement IDs mapped to Phase 11 in `REQUIREMENTS.md` appear in PLAN frontmatter. No requirement is blocked, human-only, or orphaned.

## Threat-Mitigation Verification

| Threat | Expected mitigation | Status | Evidence |
| --- | --- | --- | --- |
| T-11-16-01 | Reject malformed BigInt delimiters through the real ABI | ✓ MITIGATED | Both signs, roots/nested contexts, suffix classes, and untouched outputs pass. |
| T-11-16-02 | Fail closed on generated architecture drift | ✓ MITIGATED | Three guarded branch shapes must each occur nine times; legacy forms must occur zero times. |
| T-11-16-03 | Persist malformed cases in the correctness oracle | ✓ MITIGATED | Four exact-byte reject fixtures are manifest-complete and the oracle passes. |
| T-11-17-01 / T-11-18-01..03 | Contain every caught C++ exception and return 97 | ✓ MITIGATED | No allocation precedes the bad-allocation diagnostic guard; selector 3 uses the production catch and the child reaches exact status 97 without termination. |
| T-11-17-02 / T-11-18-07 | Keep thrown exceptions distinct from returned engine failures | ✓ MITIGATED | Thrown seam errors map to 97; returned `MEMALLOC` and explicit internal failures remain 127. |
| T-11-18-04 | Preserve success-only output state | ✓ MITIGATED | Sentinel mutation downgrades selector 3 to 127; the child accepts only 97. |
| T-11-17-03 / T-11-18-05 | Keep the deterministic seam internal and fail closed on selectors | ✓ MITIGATED | The existing symbol remains cbindgen-excluded; the new helper is private; unsupported selector 2 returns invalid argument. |
| T-11-18-06 | Detect process termination or incomplete child execution | ✓ MITIGATED | Parent requires successful exit, exactly one passing child test, and a marker emitted after the status assertion. |
| T-11-SC | Preserve audited source and dependency supply chain | ✓ MITIGATED | Exact clean v4.6.4 gitlink, one output-copy patch, locked dependencies, and no added package/source tree. |

## Anti-Patterns and Known Risks

| File | Line / area | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `internal/ffi/bindings.go` | `copyElementBytes` | Successful native pointer/length pairs are trusted before `unsafe.Slice`. | ⚠ WARNING (WR-01) | A corrupt compatible native artifact could panic or leak. The current Rust producer supplies valid owned spans, and all real copy paths pass. This does not falsify a Phase 11 truth. |
| `scripts/release/check_bootstrap_abi_state.py` | `SEMVER_RE` | SemVer grammar accepts some malformed forms and rejects build metadata. | ⚠ WARNING (WR-02) | Future release identity validation should be hardened. Current valid source identity and readiness checks pass; no Phase 11 behavior depends on an invalid form. |
| `.planning/ROADMAP.md` | Plan 11-18 checklist | Plan entry remains unchecked although its PLAN, SUMMARY, commits, and implementation exist and `roadmap.analyze` reports Phase 11 complete. | ℹ INFO | Planning-state drift only; it does not affect code behavior or goal achievement. |

No unreferenced `TBD`, `FIXME`, or `XXX` marker exists in the phase implementation files. No placeholder, empty handler, hardcoded empty public data, or console-only implementation feeds a Phase 11 behavior. The Plan 11-18 implementation diff changes only `src/native/simdjson_bridge.cpp` and `tests/rust_shim_minimal.rs`; public headers, wrappers, bindings, dependencies, bootstrap state, and release workflows are unchanged.

## Adversarial Disconfirmation Pass

- **Partially hardened requirement:** WR-01 leaves the Go copy-out boundary less defensive against a deliberately corrupt but ABI-compatible native artifact. This path is not covered for malformed pointer/length combinations. It does not defeat NUM-02 because the audited current Rust producer validates borrowed spans and returns registered owned allocations; current public and materializer BigInt flows use real valid data and pass.
- **Potentially misleading green test:** The selector-3 subprocess throws an initial `std::bad_alloc`; it does not inject a second allocator failure inside diagnostic assignment. The process test alone therefore would not prove the former gap closed. The conclusion also depends on direct source proof: the literal conversion cannot allocate, and the only remaining buffer allocation is lexically inside `try_set_last_error_message`'s catch-all guard.
- **Historical platform evidence:** The five-target hosted run is immutable historical evidence, not a new run of current HEAD. Plan 11-18 changed only the native exception capture and its Rust test; current local cross-language/contract gates and the unchanged five-target workflow contract rule out a relevant regression. Final matching v0.2 publication remains Phase 16 work and is not claimed here.
- **Alternative failure theory tested:** A sentinel mutation would cause C++ selector 3 to return 127 before Rust sees the result. Because the exact child asserts 97 before printing its marker, its successful exit jointly excludes termination, wrong mapping, and sentinel mutation.

These limits are recorded as evidence boundaries, not hidden as passing assertions. None makes a must-have uncertain or failed.

## Deferred-Phase Filter

Phases 12-15 cover DOM navigation, On-Demand extraction, zero-copy views, and streaming. Phase 16 covers final v0.2 stabilization, matching release evidence, and publication. No current Phase 11 truth is deferred to them, and no implementation gap remains to filter.

## Human Verification Required

None. The phase behaviors are determinable from source structure, ABI checks, exact native/Rust/Go tests, subprocess termination semantics, and generated-contract gates. No PLAN contains a deferred `<human-check>` block, and the phase requires no visual, real-time, or external-service judgment.

## Gaps Summary

No blocking or partial goal gaps remain.

Plan 11-18 closes the sole previous blocker: parser-aware `std::bad_alloc` diagnostic capture performs no allocating work before the guard, any diagnostic allocation failure is swallowed, the production catch reaches the shared status-97 mapper, and an exact-filter subprocess proves normal process survival and success-only output behavior. Previously closed BigInt, parser-limit, diagnostic, ABI, source-provenance, and contract behaviors show no regression.

---

_Verified: 2026-07-29T18:10:22Z_

_Verifier: the agent (gsd-verifier)_
