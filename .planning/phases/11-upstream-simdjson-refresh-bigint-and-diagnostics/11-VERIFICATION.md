---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
verified: 2026-07-29T16:08:28Z
status: gaps_found
score: 96/99 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 86/88
  gaps_closed:
    - "Malformed oversized integer roots and nested values followed by x, underscore, plus, slash, or NUL now return invalid JSON without exposing a truncated decimal prefix; valid positive and negative BigInts remain exact kind-9 text."
    - "The direct std::bad_alloc status mapper now returns PURE_SIMDJSON_ERR_CPP_EXCEPTION (97), and the parser-independent hidden exception seam covers runtime_error, bad_alloc, and unsupported selectors."
  gaps_remaining:
    - "The production parser-aware bad_alloc catch can still terminate while constructing diagnostic text before the status-97 mapper is reached."
  regressions: []
gaps:
  - truth: "Every C++ exception caught at the Rust/C++ boundary, including parser-side std::bad_alloc, returns PURE_SIMDJSON_ERR_CPP_EXCEPTION (97) without terminating the process."
    status: failed
    reason: "capture_parser_exception(parser, const std::bad_alloc&) is noexcept, but it constructs std::string(\"std::bad_alloc: \") + error.what() as a function argument before try_set_last_error_message enters its try block. A second allocation failure therefore escapes the noexcept capture helper and invokes std::terminate. The parser catch macro calls this helper before map_cpp_exception, so status 97 is not guaranteed. Existing forced-exception tests exercise only PSIMDJSON_CATCH_CPP_EXCEPTIONS, not the parser-aware macro."
    artifacts:
      - path: "src/native/simdjson_bridge.cpp"
        issue: "Lines 274-300 allocate while forming parser diagnostic text inside a noexcept bad_alloc catch path, before the guarded assignment and before the status mapper."
      - path: "tests/rust_shim_minimal.rs"
        issue: "The selector tests prove the parser-independent catch macro only; no deterministic parser-aware bad_alloc path proves process survival, untouched outputs, and status 97."
    missing:
      - "Make bad_alloc diagnostic capture allocation-free before entering the guarded assignment, or wrap the entire temporary-string construction and assignment in a local try/catch that cannot escape noexcept."
      - "Add a deterministic internal parser-aware fault seam and a subprocess regression proving the process stays alive, output sentinels remain untouched, and status 97 reaches Rust."
---

# Phase 11: Upstream simdjson Refresh, BigInt, and Diagnostics Verification Report

**Phase Goal:** Establish the compatibility foundation for v0.2 by moving to the current audited simdjson 4.6 patch release, preserving oversized integer literals as exact decimal text, and exposing the small set of parser controls and diagnostics required for production operation.

**Verified:** 2026-07-29T16:08:28Z

**Status:** gaps_found

**Re-verification:** Yes — after gap-closure Plans 11-16 and 11-17.

## Verification Scope

This report verifies current source and executable behavior rather than accepting SUMMARY claims.

- Current source HEAD inspected: `50f22409a76b25a8d6acb9f42ee4820f4067a4dc`.
- All 17 PLAN frontmatters contribute 95 detailed truths. The four ROADMAP success criteria remain non-negotiable, for 99 merged must-haves.
- Previous passed items received quick regression checks through source inspection and the complete current contract/race/oracle gates. The two previous gaps and the advisory review's CR-01 received full existence, substance, wiring, and behavior checks.
- No verification override exists.
- The unrelated modified `.planning/config.json` and untracked Phase 10 learnings file were preserved.

## Goal Achievement

### Roadmap Contract

| # | Observable truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | simdjson v4.6.4 is integrated and the Go/Rust/C++ contract, correctness, benchmark, and five-target gates remain green. | ✗ FAILED | The exact pin, local gates, workflow contracts, and prior independently audited five-target evidence remain coherent. However, the normative exception contract is not actually preserved: parser-side `bad_alloc` can terminate before returning status 97. A green but incomplete test gate does not prove the contract. |
| 2 | Valid integers above `uint64` become `TypeBigInt`, and `GetBigInt` returns exact decimal text without automatic `math/big` allocation. | ✓ VERIFIED | Valid positive/negative roots and nested values remain kind 9 with byte-exact copied text; malformed suffixes now reject; no production Go file directly imports `math/big`. |
| 3 | Kernel report/override plus bounded capacity/depth controls are available and safe. | ✓ VERIFIED | Real native implementation names flow through mandatory bindings; selection is process-global and locked; capacity is checked before copy; depth is uniformly bounded at 1024. |
| 4 | Syntax/UTF-8 failures expose a proven upstream byte offset when available and explicit unknown otherwise, without fabrication. | ✓ VERIFIED | The nine-case v4.6.4 corpus, known-zero state, pointer-range proof, unknown sentinel, stale-state clearing, and maximum-depth subprocess all pass. |

**Roadmap score:** 3/4 criteria verified.

### PLAN Must-Have Resolution

| Plan | Resolution | Current evidence and exceptions |
| --- | --- | --- |
| 11-01 | 4/4 VERIFIED | The historical operator decision, intermediate-artifact boundary, and publication sequencing records remain coherent. |
| 11-02 | 6/6 VERIFIED | D-02 is now restored: only complete valid oversized integer tokens reach kind 9. v4.6.4 provenance, kinds, strict numeric getters, frames, fuzzing, and oracle behavior regress cleanly. |
| 11-03 | 6/6 VERIFIED | Strict copied BigInt ownership, lifetime, wrong-type behavior, kind hints, write-on-success, and Rust `ffi_wrap` wiring remain substantive. |
| 11-04 | 6/6 VERIFIED | Immutable configured construction, pre-copy capacity enforcement, exact defaults, depth boundaries, and diagnostic clearing remain wired. |
| 11-05 | 8/8 VERIFIED | Upstream-only two-pass replay, exact parser limits, range-proven offsets, explicit unknown state, and bounded recursion remain substantive. |
| 11-06 | 5/5 VERIFIED | Kernel selection is exact, serialized, process-global, runtime-validated, and irreversibly locked. |
| 11-07 | 7/7 VERIFIED | ABI/version/bootstrap mirrors remain coherent at ABI `0x00010002`; enum growth remains append-only. |
| 11-08 | 4/4 VERIFIED | ABI-only probing precedes complete mandatory binding and cache installation; ABI 1.1 has no fallback path. |
| 11-09 | 5/5 VERIFIED | Generated ABI 1.2 header, C smoke, internal-symbol exclusion, layouts, and normative documentation remain coherent. |
| 11-10 | 5/5 VERIFIED | Public and materializer BigInt paths use copied exact native data across root and descendant flows. |
| 11-11 | 5/5 VERIFIED | Go options and pools preserve one validated immutable capacity/depth identity. |
| 11-12 | 6/6 VERIFIED | Known/unknown offsets and kernel lifecycle remain wired through real native state and pass race checks. |
| 11-13 | 7/7 VERIFIED | Contract, correctness, release-source, explicit-library, benchmark-signal, and source-readiness gates pass on current source. |
| 11-14 | 6/6 VERIFIED | Prior independently audited hosted evidence is regression-checked by the unchanged annotated `v0.1.7` object/commit/ancestry and current five-target workflow contracts. Later Phase 11 source fixes are not attributed to that old tag. |
| 11-15 | 4/4 VERIFIED | One 1024 ceiling governs Go validation, native construction, both replay passes, recursive replay, and materialization; maximum-depth malformed input is subprocess-safe. |
| 11-16 | 6/6 VERIFIED | All malformed suffix classes reject through the real ABI, valid exact text remains kind 9, 9/9/9 architecture guards pass at build time, and four persistent oracle fixtures reject. |
| 11-17 | 3/5 VERIFIED | The direct mapper, hidden selectors, returned-MEMALLOC distinction, ABI stability, and requested command gates pass. **FAILED:** every caught exception is not guaranteed to return 97, and parser diagnostic capture can prevent the shared mapper from returning at all. |

**PLAN score:** 93/95 truths verified.

**Merged score:** 96/99 must-haves verified.

## Re-verification of Previous Gaps

### Gap 1 — malformed BigInt suffixes

**Status: CLOSED**

| Check | Evidence | Status |
| --- | --- | --- |
| Complete-token validation | The one repository patch calls `is_not_structural_or_whitespace(*p)` before too-many-digits, negative-overflow, and positive-overflow BigInt returns. | ✓ VERIFIED |
| All generated architectures | `build.rs` requires each guarded branch exactly nine times and its legacy unguarded form zero times. A fresh release build passed these fail-closed assertions. | ✓ VERIFIED |
| Invalid roots and nested values | The Rust ABI matrix covers both signs, root/array/object contexts, and `x`, `_`, `+`, `/`, and NUL. All return status 32 and leave the document sentinel untouched. | ✓ VERIFIED |
| Valid controls | Positive and negative roots plus structural/whitespace-delimited array/object values remain kind 9 with exact signed decimal text. | ✓ VERIFIED |
| Persistent oracle | Four exact-byte project fixtures are sorted `reject` rows; the complete 322-case oracle passes. | ✓ VERIFIED |
| Provenance | Gitlink is clean at official v4.6.4 commit `1bcf71bd...`; exactly one `simdjson*.patch` exists and is applied only to the Cargo output copy. | ✓ VERIFIED |

### Gap 2 — trapped `std::bad_alloc`

**Status: REMAINS OPEN**

The direct status mismatch is fixed: `map_cpp_exception(const std::bad_alloc&)` returns 97, selector 0 returns 97 for `runtime_error`, selector 1 returns 97 for `bad_alloc`, and unsupported selectors return status 1. Returned `simdjson::MEMALLOC` remains status 127.

That does not close the production parser path:

1. `psimdjson_parser_parse` uses `PSIMDJSON_CATCH_PARSER_CPP_EXCEPTIONS`.
2. Its `bad_alloc` branch calls `capture_parser_exception` before `map_cpp_exception`.
3. The `noexcept` capture helper forms `std::string("std::bad_alloc: ") + error.what()` before entering `try_set_last_error_message`.
4. C++ evaluates that allocating argument before control enters the callee's `try`.
5. If the new allocation also fails during exhaustion, it escapes `noexcept`, invokes `std::terminate`, and status 97 is never returned.

`LastErrorBuffer::assign` itself can throw on failed `malloc`, but that allocation is safely inside `try_set_last_error_message` and is caught. The blocker is the earlier temporary-string construction. The existing selector seam ends in `PSIMDJSON_CATCH_CPP_EXCEPTIONS`, so its passing test cannot exercise this path.

## Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `third_party/simdjson` | Exact official v4.6.4 gitlink | ✓ VERIFIED | Gitlink and checkout are `1bcf71bd85059ab6574ea1159de9298dcc1212c5`; tag lookup returns `v4.6.4`; tracked/staged state is clean. |
| `patches/simdjson-v4.6.4-positive-bigint.patch`, `build.rs` | One audited output-copy patch plus fail-closed architecture parity | ✓ VERIFIED | Exactly one patch; exact base and clean tree checks; guarded branches 9/9/9; zero legacy shapes; C++17 singleheader build retained. |
| `tests/rust_shim_bigint.rs`, JSONTestSuite fixtures | Valid exact-text and invalid-suffix coverage | ✓ VERIFIED | 20 malformed ABI cases, valid structural/whitespace controls, untouched output sentinels, and four persistent oracle rejects. |
| `src/native/simdjson_bridge.cpp` | BigInt, limits, diagnostics, kernel, and exception boundary | ✗ PARTIAL / BLOCKER | BigInt and controls are substantive. Parser-aware `bad_alloc` capture may terminate before the status mapper. |
| `src/native/simdjson_bridge.h`, `src/runtime/mod.rs`, `src/lib.rs` | Internal selector seam and stable Rust ABI | ✓ VERIFIED | One fixed-selector seam is threaded internally; public ABI remains `0x00010002`; no new public fault switch exists. |
| `tests/rust_shim_minimal.rs` | Deterministic exception contract | ⚠ PARTIAL | Exact 97/97/1 selector outcomes pass, but only through the parser-independent catch macro. |
| `include/pure_simdjson.h`, `internal/ffi/types.go`, `docs/ffi-contract.md` | Stable append-only ABI 1.2 contract | ✓ VERIFIED | Header and normative document are byte-identical to pre-Plan-17 hashes; kind 9 and statuses 97/127 remain unchanged. |
| Go public/options/diagnostic/kernel files | Real user-facing data and controls | ✓ VERIFIED | `GetBigInt`, frame materialization, parser options/pools, `HasOffset`, `Kernel`, and `SetKernel` all consume real native state/data. |
| Release and public-validation workflows | Five-target build and bootstrap gates | ✓ VERIFIED | Current contract tests pin five release targets, native/packaged smokes, five R2 targets, three fallback targets, CI-only publication, and origin/main ancestry. |

## Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| Official simdjson v4.6.4 | Built native source | exact-base check → output copy → one patch → 9/9/9 guards | ✓ WIRED | Provenance and fail-closed parity are executable during every Cargo build. |
| Oversized numeric token | DOM kind 9 | guarded BigInt returns → `number_as_string(true)` | ✓ WIRED | Valid tokens retain exact text; dirty delimiters return invalid JSON before a document exists. |
| Native kind 9 | Go `GetBigInt` and materializers | native span → Rust-owned allocation → Go copy/string | ✓ WIRED | Exact positive/negative text survives document lifetime and both materializer paths. |
| Parser limits | primary parse, replay, and materializer | one effective capacity/depth identity and shared ceiling | ✓ WIRED | Capacity gates before copy; all recursive paths are bounded at 1024. |
| Upstream location pointer | Go `Error.Offset` / `HasOffset` | checked in-range pointer → Rust transport → Go typed error | ✓ WIRED | Known zero and explicit unknown remain distinct. |
| Go kernel API | simdjson implementation state | mandatory binding → native selection mutex/lock | ✓ WIRED | Report and override use real process-global implementation state. |
| Generic exception selector | direct exception mapper | hidden selector → `PSIMDJSON_CATCH_CPP_EXCEPTIONS` → `map_cpp_exception` | ✓ WIRED, LIMITED | Runtime error and bad allocation return 97; invalid selector returns 1. It does not cover parser-aware capture. |
| Parser parse catch | status-97 mapper | parser-aware catch → diagnostic capture → mapper | ✗ NOT SAFELY WIRED | An allocating diagnostic expression can terminate before the mapper executes. |
| Returned `simdjson::MEMALLOC` | explicit internal status | `map_error` switch | ✓ WIRED | Returned engine errors remain distinct at status 127. |
| Rust ABI source | generated header → Go binding → C smoke | cbindgen plus complete mandatory binding | ✓ WIRED | Current header diff, 25 audits, layout compile, and Go binding tests pass. |

## Data-Flow Trace (Level 4)

| Artifact | Data variable | Source | Produces real data | Status |
| --- | --- | --- | --- | --- |
| BigInt public/materializer path | exact decimal text | parsed token → native span/frame → Rust-owned bytes → Go string | Yes | ✓ FLOWING |
| Malformed BigInt rejection | parser status/document output | delimiter guard on complete token | Yes | ✓ FLOWING: status 32, untouched document sentinel |
| Parser limits | capacity/depth | public options or C constructor | Yes | ✓ FLOWING: normalized once and enforced before unsafe work |
| Diagnostics | offset and known bit | upstream `current_location()` plus pointer proof | Yes when upstream supplies a usable pointer | ✓ FLOWING; otherwise explicit unknown |
| Kernel | active/requested implementation | simdjson runtime implementation registry | Yes | ✓ FLOWING |
| Parser exception status | ABI return code | parser catch → best-effort diagnostic → mapper | Not reliably | ✗ INTERRUPTED: a second bad allocation can terminate before status 97 |

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| BigInt, limits, diagnostics, and exception selectors | `cargo test --locked --test rust_shim_bigint --test rust_shim_minimal --test rust_shim_limits --test rust_shim_diagnostics -- --test-threads=1` | 46/46 passed; malformed suffixes reject and selectors return 97/97/1. | ✓ PASS, exception coverage incomplete |
| Full Rust/header/C contract | `make verify-contract && make verify-docs` | 90 Rust tests, deterministic header diff, 25 audits, header rules, C layout compile, and docs gate passed. | ✓ PASS |
| Fresh optimized library | `cargo build --release --locked` | Release dylib built; patch/base/parity assertions passed. | ✓ PASS |
| Go API race gate | explicit fresh dylib + `go test ./... -race -count=1 -timeout=180s` | All four packages passed under the race detector. | ✓ PASS |
| JSON correctness oracle | explicit fresh dylib + `go test . -run '^TestJSONTestSuiteOracle$' -count=1` | All 322 manifest rows/cases passed. | ✓ PASS |
| Release-source contracts | Four release/public/bootstrap Python test files | 10 + 12 + 11 + 10 tests passed. | ✓ PASS |
| Non-strict source readiness | `bash scripts/release/check_readiness.sh` | `basic release readiness checks passed`. | ✓ PASS |
| Benchmark execution signal | `scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir <temp>` | Non-empty 72-line bench output, JSON summary, and Markdown; no flagged rows. | ✓ PASS, advisory no-baseline signal |
| Parser-aware allocation exhaustion | Source-level catch trace | An allocating argument precedes the guarded helper and mapper. | ✗ FAIL |

## Probe Execution

No conventional `scripts/*/tests/probe-*.sh` files or Phase 11 declared probes exist.

**Step 7c:** SKIPPED — no probe contract was declared.

## Requirements Coverage

| Requirement | Source plans | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| UP-01 | 11-01, 11-02, 11-07, 11-13, 11-14, 11-16, 11-17 | Exact audited v4.6.4 plus reproducible compatibility gates | ✗ BLOCKED | Pinning, build, oracle, race, benchmark, and release contracts pass, but the normative C++ exception containment contract is not actually preserved on parser-side `bad_alloc`. |
| NUM-01 | 11-02, 11-03, 11-10, 11-13, 11-14, 11-16 | Preserve valid oversized integers as exact text | ✓ SATISFIED | Valid signed/unsigned values are kind 9 and exact; malformed suffixes reject without truncation. |
| NUM-02 | 11-02, 11-03, 11-07 through 11-10, 11-13, 11-14 | `TypeBigInt` / `GetBigInt` without automatic arbitrary precision | ✓ SATISFIED | Kind 9 and exact copied text cross native/Rust/Go; no production direct `math/big` import. |
| DIAG-01 | 11-06 through 11-09, 11-12 through 11-14 | Kernel report and exact diagnostic override | ✓ SATISFIED | Mandatory bindings, real implementation state, process-global locks, and isolated tests pass. |
| DIAG-02 | 11-05, 11-07 through 11-10, 11-12 through 11-15 | Proven offsets where available; explicit unknown otherwise | ✓ SATISFIED | Corpus, pointer proof, known zero, stale-state clearing, limits, and subprocess checks pass. |
| LIMIT-01 | 11-04, 11-07 through 11-09, 11-11 through 11-15 | Safe immutable capacity/depth options and homogeneous pools | ✓ SATISFIED | Capacity precedes copy; depth is uniformly capped at 1024; pool identity includes both values. |

All six Phase 11 requirement IDs mapped by `REQUIREMENTS.md` appear in PLAN frontmatter. No Phase 11 requirement is orphaned.

## Threat-Mitigation Verification

| Threat | Expected mitigation | Status | Evidence |
| --- | --- | --- | --- |
| T-11-16-01 | Reject malformed BigInt delimiters through the real ABI | ✓ MITIGATED | Both signs, roots/nested contexts, five suffix classes, and untouched outputs pass. |
| T-11-16-02 | Fail closed on generated architecture drift | ✓ MITIGATED | Three guarded branch shapes must each occur nine times; legacy forms must occur zero times. |
| T-11-16-03 | Persist malformed cases in the correctness oracle | ✓ MITIGATED | Four exact-byte reject fixtures are manifest-complete and the oracle passes. |
| T-11-17-01 | Contain every C++ exception and return 97 | ✗ NOT MITIGATED | Parser-aware bad_alloc diagnostic capture may invoke `std::terminate`. |
| T-11-17-02 | Keep thrown exceptions distinct from returned engine failures | ✓ MITIGATED | Thrown generic-seam errors map to 97; returned `MEMALLOC` and explicit internal errors remain 127. |
| T-11-17-03 | Keep the deterministic seam internal and fail closed on selectors | ✓ MITIGATED | Existing `psimdjson_*` symbol remains excluded from cbindgen/public bindings; only selectors 0/1 throw and others return invalid argument. |
| T-11-SC | Preserve audited source and dependency supply chain | ✓ MITIGATED | Exact clean v4.6.4 gitlink, one output-copy patch, locked dependencies, no added package/source tree. |

## Known-Risk Audit

### CR-01 — confirmed blocker

The review finding is independently confirmed from source. It is not a speculative style concern: C++ argument evaluation occurs before the called function begins, `capture_parser_exception` is `noexcept`, and the bad-allocation message concatenation can allocate during an allocation-exhaustion catch. The mapper's correct numeric return is unreachable if that second allocation throws.

### WR-01 — non-blocking Phase-adjacent warning

`internal/ffi/bindings.go::copyElementBytes` trusts successful native pointer/length pairs. A compatible but faulty artifact could return nil/nonzero, nonnil/zero, or an unrepresentable length and cause a Go panic or leaked allocation.

This does not block the current Phase 11 goal because the current Rust producer validates borrowed spans, returns only `(nil,0)` for empty values, registers non-empty owned allocations, and all real BigInt/string copy paths pass. It remains worthwhile fail-closed boundary hardening for corrupt or mismatched native producers.

### WR-02 — non-blocking release-tooling warning

`SEMVER_RE` in `check_bootstrap_abi_state.py` accepts malformed suffix/leading-zero forms and rejects valid build metadata. Current source uses valid `0.1.7`, current readiness/source-policy tests pass, and Phase 11 does not invent or publish another version. The validator should be hardened before Phase 16 release work, but it is not the root cause of the Phase 11 goal failure.

## Anti-Patterns Found

| File | Line / area | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `src/native/simdjson_bridge.cpp` | 274-300 | Allocation while handling `bad_alloc` inside `noexcept`, before guarded assignment/status mapping | 🛑 BLOCKER | Real parser allocation failure can terminate instead of returning 97. |
| `tests/rust_shim_minimal.rs` | 346-366 | Passing seam covers only parser-independent macro | ⚠ WARNING | Green status tests do not exercise production parser diagnostic capture. |
| `internal/ffi/bindings.go` | 385-408 | Successful native span is not shape/range-validated before `unsafe.Slice` | ⚠ WARNING | Faulty compatible artifact could panic or leak; current producer does not emit those spans. |
| `scripts/release/check_bootstrap_abi_state.py` | 23-27 | Permissive/noncanonical SemVer grammar | ⚠ WARNING | Future malformed release identity could pass this source checker. |

No unreferenced `TBD`, `FIXME`, or `XXX` marker exists in the phase implementation files. No placeholder, empty handler, or hardcoded empty value feeds a Phase 11 public behavior.

## Deferred-Phase Filter

Phases 12-15 cover DOM navigation, On-Demand extraction, zero-copy views, and streaming. Phase 16 covers final v0.2 stabilization, evidence, and publication. None specifically owns making the parser-aware C++ exception catch allocation-safe, and future safety validation is not an implementation fix. CR-01 remains a Phase 11 compatibility-foundation gap and is not deferred.

The current Plans 11-15 through 11-17 source is newer than the immutable historical `v0.1.7` publication. Phase 16 owns publication of final matching v0.2 artifacts; this expected publication boundary does not suppress CR-01.

## Human Verification Required

None. The phase's remaining failure is determinable from C++ evaluation order, `noexcept` semantics, macro wiring, and test-seam coverage. No PLAN contains a deferred `<human-check>` block.

## Gaps Summary

Plan 11-16 fully closes malformed BigInt token validation: valid positive and negative values retain exact kind-9 text, malformed roots/nested values reject, all nine generated architecture paths share the guard, provenance stays fail-closed, and the persistent oracle passes.

Plan 11-17 closes the direct status-code mismatch but not the production parser exception contract. Parser-aware `bad_alloc` handling may allocate before its best-effort guard and before `map_cpp_exception`, allowing `std::terminate` instead of status 97. Because this violates the normative ABI and the phase's production compatibility foundation, Phase 11 remains `gaps_found`.

---

_Verified: 2026-07-29T16:08:28Z_

_Verifier: the agent (gsd-verifier)_
