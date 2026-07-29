---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
verified: 2026-07-29T14:20:07Z
status: gaps_found
score: 86/88 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 81/84
  gaps_closed:
    - "Accepted configured depths are bounded safely in the current source: zero selects 1024, 1 through 1024 are accepted, values above 1024 fail before allocation/traversal, and malformed input at the accepted ceiling returns a typed status from a subprocess."
  gaps_remaining: []
  regressions:
    - "Plan 11-02 D-02 was previously counted as verified; fresh malformed-token probing proves that non-integer syntax can be accepted as kind 9."
    - "The roadmap contract/correctness gate truth was previously counted as verified; the bad_alloc status mismatch proves that the green suite does not cover the full normative exception contract."
gaps:
  - truth: "Only valid oversized integer tokens enter the BigInt path; malformed JSON is rejected without silently dropping token suffixes."
    status: failed
    reason: "The built ABI returns success for oversized integers followed by x, underscore, plus, slash, or NUL. Positive and negative roots become kind 9 with only the digit prefix copied, and a nested malformed token is accepted as an array. The generated simdjson BigInt branch returns before the normal structural-or-whitespace delimiter check, then visit_number scans only sign plus digits and returns success."
    artifacts:
      - path: "patches/simdjson-v4.6.4-positive-bigint.patch"
        issue: "All nine architecture hunks reroute positive overflow from INVALID_NUMBER to BIGINT_NUMBER without preserving the trailing-token validity check."
      - path: "tests/rust_shim_bigint.rs"
        issue: "Valid boundary and exact-text cases exist, but there is no positive, negative, root, or nested malformed-suffix regression."
      - path: "testdata/jsontestsuite/expectations.tsv"
        issue: "The oracle covers ordinary malformed numbers and valid oversized integers, but not malformed oversized integers that enter BIGINT_ERROR/number_as_string."
    missing:
      - "Validate the character after the complete BigInt token as structural or JSON whitespace before returning success on every generated architecture path."
      - "Add positive, negative, root, nested, and multiple-suffix regressions proving malformed oversized integers return PURE_SIMDJSON_ERR_INVALID_JSON and never expose truncated text."
  - truth: "Every C++ exception trapped at the Rust/C++ seam maps to the normative PURE_SIMDJSON_ERR_CPP_EXCEPTION status 97."
    status: failed
    reason: "docs/ffi-contract.md is normative and says C++ exceptions map to status 97, but the first catch branch routes std::bad_alloc through an overload that returns PURE_SIMDJSON_ERR_INTERNAL status 127. The existing forced-exception test throws std::runtime_error only, so it passes without exercising the conflicting branch."
    artifacts:
      - path: "src/native/simdjson_bridge.cpp"
        issue: "map_cpp_exception(const std::bad_alloc&) returns PURE_SIMDJSON_ERR_INTERNAL while the other C++ exception overloads return PURE_SIMDJSON_ERR_CPP_EXCEPTION."
      - path: "docs/ffi-contract.md"
        issue: "The normative exception policy and status table require trapped C++ exceptions to use status 97."
      - path: "tests/rust_shim_minimal.rs"
        issue: "The exception seam covers std::runtime_error but has no deterministic std::bad_alloc path."
    missing:
      - "Make the bad_alloc catch behavior agree with the normative public status contract, or record an explicit accepted contract override before changing the documentation."
      - "Add a deterministic bad_alloc exception seam and assert the selected public status."
---

# Phase 11: Upstream simdjson Refresh, BigInt, and Diagnostics Verification Report

**Phase Goal:** Establish the compatibility foundation for v0.2 by moving to the current audited simdjson 4.6 patch release, preserving oversized integer literals as exact decimal text, and exposing parser controls/diagnostics.

**Verified:** 2026-07-29T14:20:07Z

**Status:** gaps_found

**Re-verification:** Yes — after Plan 11-15 attempted to close the prior maximum-depth gap.

## Verification Scope

This report verifies code and behavior, not SUMMARY claims.

- The Plan 11-15 closure was verified at current source HEAD `7253804522596343b6964afaa532d2a28ac2727d`.
- Historical publication evidence remains tied to immutable annotated tag `v0.1.7`: tag object `cd153ae770745dad124750ec8dd765eb1afdb83e`, target `ab86c2e1e666c6c313d1dd951c37a8c43538c407`, and an ancestry check against local `origin/main` at `7c1051b8758139645ce437a8ae38ca75fc8f2174`.
- The checked-out simdjson gitlink is exactly `1bcf71bd85059ab6574ea1159de9298dcc1212c5`, and that commit identifies itself as `v4.6.4`.
- The BigInt patch and the conflicting `std::bad_alloc` mapping are present in both the current source and the immutable `v0.1.7` source. The Plan 11-15 depth ceiling is newer source-only work and must not be attributed to `v0.1.7`.
- Existing user changes in `.planning/config.json` and the untracked Phase 10 learnings file were not modified.

## Goal Achievement

### Roadmap Contract

| # | Observable truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | simdjson v4.6.4 is integrated and the Go/Rust/C++ contract, correctness, benchmark, and five-target gates are green. | ✗ FAILED | The exact v4.6.4 base, historical five-target publication, and fresh local commands are green, but independent behavior falsifies contract/correctness: malformed oversized numeric tokens are accepted and `std::bad_alloc` maps to the wrong normative status. A green but incomplete gate is not goal evidence. |
| 2 | Valid integers above `uint64` become `TypeBigInt`, and `GetBigInt` returns exact decimal text without automatic `math/big` allocation. | ✓ VERIFIED | Fresh ABI probing and Rust/Go tests preserve positive and negative exact text as kind 9; no production Go file imports `math/big`. |
| 3 | Kernel report/override plus bounded capacity/depth controls are available and safe. | ✓ VERIFIED | Kernel selection is wired through mandatory native bindings; capacity is bounded; depth now normalizes to 1024 and rejects values above 1024 before native allocation or replay. |
| 4 | Syntax/UTF-8 failures expose a proven upstream byte offset when available and explicit unknown otherwise, without fabricating one or terminating the process. | ✓ VERIFIED | The nine-case corpus passed, pointer-derived offsets remain range-checked, unavailable locations remain `UINT64_MAX` plus false, and the depth-1024 malformed subprocess passed without a signal. |

**Roadmap score:** 3/4 criteria verified.

### PLAN Must-Have Resolution

The 15 PLAN files contain 84 detailed truths. Every plan's frontmatter truths were checked. Plans 11-01 through 11-14 received quick regression checks after the previous report; Plan 11-15 and newly failed surfaces received full existence, substance, wiring, and behavior checks.

| Plan | Resolution | Evidence and exceptions |
| --- | --- | --- |
| 11-01 | 4/4 VERIFIED | v4.6.4 provenance, audited build-output patch application, branch-count guard, and upstream C++17 singleheader path remain present. |
| 11-02 | 5/6 VERIFIED | **D-02 FAILED:** malformed oversized tokens such as `18446744073709551616x` classify as kind 9, so it is false that only integer syntax enters kind 9. Valid boundary, frame, accessor, and race/oracle paths otherwise work. |
| 11-03 | 6/6 VERIFIED | Copied BigInt ownership, lifetime, strict wrong-type behavior, kind hints, and `ffi_wrap` wiring remain substantive. |
| 11-04 | 6/6 VERIFIED | Configured capacity/depth transport, immutable options, pre-copy capacity rejection, and legacy defaults remain wired. |
| 11-05 | 8/8 VERIFIED | Diagnostic replay is input-derived, two-pass, bounded by parser limits, range-proven, and explicitly unknown when upstream cannot supply a location. |
| 11-06 | 5/5 VERIFIED | Kernel reporting/selection is process-global, exact, serialized, and locked by parser creation. |
| 11-07 | 7/7 VERIFIED | ABI/version/bootstrap mirrors remain coherent at `0x00010002`; staged artifact policy checks pass. |
| 11-08 | 4/4 VERIFIED | Loader probes ABI before binding mandatory symbols and surfaces compatibility failures without partial publication. |
| 11-09 | 5/5 VERIFIED | Generated header, C smoke, append-only ABI values/layouts, and normative documentation exist. The separately reported bad_alloc behavior contradicts that normative contract and blocks the roadmap contract truth even though this plan's document-existence truth is present. |
| 11-10 | 5/5 VERIFIED | BigInt and diagnostics cross the Go materializers and public error model with copied ownership and explicit known/unknown state. |
| 11-11 | 5/5 VERIFIED | Parser pools preserve homogeneous immutable capacity/depth configurations. |
| 11-12 | 6/6 VERIFIED | Kernel/diagnostic integration and race/contract checks remain substantive and green. |
| 11-13 | 7/7 VERIFIED | Benchmark signal, readiness scripts, packaged smoke, and release-facing source gates exist and pass. |
| 11-14 | 6/6 VERIFIED | Annotated main-ancestor tag and historical five-target/fallback publication evidence remain coherent; no tag was moved. |
| 11-15 | 4/4 VERIFIED | One 1024 ceiling is enforced across Go/native construction, both replay passes, recursive consumption, and materialization; the largest accepted malformed case is subprocess-safe; all requested regression gates passed. |

**PLAN score:** 83/84 truths verified.

**Merged score:** 86/88 must-haves verified.

## Re-verification of the Previous Gap

The previous depth-control gap is closed in the current source.

| Check | Evidence | Status |
| --- | --- | --- |
| Go option normalization | `parser_options.go` defines `maxSupportedDepth = 1024`, derives the default from it, accepts zero or 1..1024, and rejects larger values during option normalization before library loading. | ✓ VERIFIED |
| Native constructor ordering | `parser_new_configured_with_selection_lock` rejects `max_depth > MAX_SUPPORTED_DEPTH` before acquiring the implementation-selection mutex and before parser allocation. | ✓ VERIFIED |
| Replay bounds | Both `replay_raw_json_location` and `replay_recursive_location` reject a limit above the shared constant before allocating an On-Demand parser. | ✓ VERIFIED |
| Recursive traversal | `consume_ondemand_value` is bounded by the same effective maximum passed to the replay parser. | ✓ VERIFIED |
| Materializer bound | `append_materialize_frame` uses `MAX_SUPPORTED_DEPTH`; the former independent materializer depth literal is gone. | ✓ VERIFIED |
| Process isolation | `deep_malformed_at_max_supported_depth_is_process_safe` launches the exact child test, requires normal success plus `1 passed`, and therefore catches signal termination. | ✓ VERIFIED |
| Fresh behavior | `cargo test --locked --test rust_shim_limits --test rust_shim_diagnostics -- --test-threads=1` passed 14/14 tests. | ✓ PASS |

The immutable `v0.1.7` tag predates Plan 11-15 and therefore still contains the old unbounded-depth behavior. Plan 11-15 explicitly limited itself to source/readiness closure and did not move or replace that tag. Phase 16 owns the final v0.2 publication; `v0.1.7` is not used here as evidence that the new depth closure has shipped.

## Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `third_party/simdjson` | Exact official v4.6.4 gitlink | ✓ VERIFIED | Gitlink and checkout are `1bcf71bd...`; exact tag lookup returns `v4.6.4`; tracked/staged submodule state is clean. |
| `patches/simdjson-v4.6.4-positive-bigint.patch`, `build.rs` | Audited, fail-closed positive-BigInt patch | ⚠ PARTIAL | Base, clean-tree, apply-check, and exactly-nine-hunk guards are real. The behavior is incomplete because the rerouted overflow exits before delimiter validation. |
| `src/native/simdjson_bridge.cpp` | BigInt, limits, kernel, diagnostics, and exception boundary | ⚠ PARTIAL | BigInt spans and controls are wired; depth closure is substantive. `std::bad_alloc` returns status 127 contrary to the normative status-97 contract. |
| `src/lib.rs`, `src/runtime/registry.rs`, `src/runtime/mod.rs` | Rust ABI exports and owned copy/diagnostic transport | ✓ VERIFIED | Exports delegate through the registry and copy string/BigInt/error state into owned storage. |
| `include/pure_simdjson.h`, `internal/ffi/bindings.go` | ABI 1.2 C/Go mirrors | ✓ VERIFIED | ABI `0x00010002`, kind 9, statuses, signatures, symbols, and layouts agree; generated-header and C layout checks pass. |
| `element.go`, `materializer_fastpath.go` | Public and fast-path BigInt handling | ✓ VERIFIED | Kind 9 maps to `TypeBigInt`; both paths copy real native bytes; valid exact text flows. |
| `parser_options.go`, `parser.go`, `pool.go` | Safe immutable capacity/depth controls | ✓ VERIFIED | Configuration is validated before library use and preserved homogeneously in pools. |
| `kernel.go`, `library_loading.go` | Kernel report/override and lifecycle lock | ✓ VERIFIED | Real native names flow through mandatory binding; the cache is not a hardcoded result. |
| `errors.go` and diagnostic exports | Known/unknown offset model | ✓ VERIFIED | Offset is exposed only with a true known flag; unknown remains sentinel plus false. |
| `tests/rust_shim_*.rs`, Go tests, JSONTestSuite oracle | Cross-language regression coverage | ⚠ PARTIAL | Existing suites are substantive and green, but omit malformed oversized-number suffixes and a deterministic bad_alloc seam. |
| `.github/workflows/release.yml`, public-bootstrap workflow, `11-ARTIFACT-READINESS.md` | Historical artifact/public validation | ✓ VERIFIED | Immutable `v0.1.7` identity and previously audited hosted evidence remain coherent; current source-only Plan 11-15 changes are not misrepresented as part of that tag. |

The SDK artifact heuristic reported false negatives for the submodule directory and some prose/pattern links. Manual checks resolved those cases with `git ls-tree`, exact source inspection, generated-header diffing, and executable tests. The two PARTIAL artifacts above are behavioral failures, not heuristic misses.

## Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| Official simdjson v4.6.4 | Built native source | clean exact-base check → copied build output → nine patch hunks | ✓ WIRED | Provenance and fail-closed patch application are real. |
| Oversized numeric token | DOM BigInt tape value | `BIGINT_NUMBER`/`BIGINT_ERROR` → `number_as_string(true)` | ✗ NOT CORRECTLY WIRED | The branch bypasses the later structural-or-whitespace check; `visit_number` copies only sign/digits and returns success, dropping a malformed suffix. |
| Native kind 9 | Go `GetBigInt()` and materializers | native span → Rust-owned copy → Go binding/string | ✓ WIRED | Valid positive/negative exact text survives document lifetime and both materialization paths. |
| `WithMaxDepth` | primary parser, both replay parsers, recursive replay, materializer | shared effective value and `MAX_SUPPORTED_DEPTH` | ✓ WIRED | All paths use the 1024 ceiling and reject larger values before unsafe work. |
| Upstream location pointer | Go `Error.Offset` / `HasOffset` | checked in-range pointer → Rust transport → Go typed error | ✓ WIRED | Known zero is preserved; unavailable/out-of-range values remain explicitly unknown. |
| Go kernel API | process-global simdjson implementation selection | loader binding → native setter/report → selection lock | ✓ WIRED | Kernel tests are process-isolated and use the real implementation registry. |
| C++ catch boundary | ABI status code | catch macro → `map_cpp_exception` | ✗ PARTIAL | `runtime_error` and unknown exceptions map to 97; `std::bad_alloc` maps to 127. |
| Rust ABI source | generated header → Go binding → C smoke | cbindgen plus mandatory symbol table | ✓ WIRED | `make verify-contract` passes header diff, rules, and C layout compilation. |
| Annotated `v0.1.7` tag | historical five-target/public bootstrap evidence | tag-driven release and separate public validation | ✓ WIRED | Tag target remains an ancestor of local `origin/main`; Plan 11-15 is correctly treated as later source work. |

## Data-Flow Trace (Level 4)

| Artifact | Data variable | Source | Produces real data | Status |
| --- | --- | --- | --- | --- |
| BigInt public/materializer path | decimal text | parsed oversized token → native span → copied Rust bytes → Go string | Yes for valid input | ⚠ FLOWING BUT VALIDATION-HOLLOW: a malformed suffix is discarded before the same path returns the digit prefix. |
| Parser limits | `maxCapacity`, `maxDepth` | public Go options or C constructor values | Yes | ✓ FLOWING: exact normalized values reach primary parse, replay, and materialization, bounded at 1024 depth. |
| Diagnostics | offset plus known flag | upstream `current_location()` followed by pointer-range proof | Yes when upstream supplies a valid pointer | ✓ FLOWING: otherwise exact sentinel/false, never a guessed offset. |
| Kernel | active/requested implementation name | simdjson runtime implementation registry | Yes | ✓ FLOWING: report and override are not static placeholders. |
| Exception status | ABI return code | C++ catch overload | Yes | ✗ WRONG VALUE for `std::bad_alloc`: 127 instead of normative 97. |
| Release/bootstrap | artifact identity and checksum metadata | immutable tag and hosted workflows | Yes for historical `v0.1.7` | ✓ FLOWING, with Plan 11-15 correctly excluded from the old tag. |

## Behavioral Spot-Checks

| Behavior | Command / invocation | Result | Status |
| --- | --- | --- | --- |
| Fresh production library | `cargo build --release --locked` | Release dylib built successfully. | ✓ PASS |
| Maximum-depth and diagnostics closure | `cargo test --locked --test rust_shim_limits --test rust_shim_diagnostics -- --test-threads=1` | 14/14 passed, including the exact child-process crash regression. | ✓ PASS |
| Rust/C/header contract | `make verify-contract` | Rust suites, 25 header audits, generated-header diff, header rules, and C layout compile passed. | ✓ PASS |
| Existing exception seam | exact `psimdjson_test_force_cpp_exception_returns_err_cpp_exception` test | Forced `std::runtime_error` returned 97 and passed; it does not exercise `std::bad_alloc`. | ✓ PASS, INCOMPLETE COVERAGE |
| Go API race gate | fresh exact dylib + `go test ./... -race -count=1 -timeout=180s` | All four packages passed under the race detector. | ✓ PASS |
| Benchmark regression signal | `scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir <temp>` | Exit 0; non-empty `.bench.txt`, `summary.json`, and `markdown.md` produced. | ✓ PASS |
| Non-strict release readiness | `bash scripts/release/check_readiness.sh` | `basic release readiness checks passed`. | ✓ PASS |
| Valid BigInt control | ABI parse of `18446744073709551616` | `parse_rc=0 kind=9 text=18446744073709551616`. | ✓ PASS |
| Malformed positive BigInt | ABI parse of `18446744073709551616x` | `parse_rc=0 kind=9 text=18446744073709551616`; suffix silently discarded. | ✗ FAIL |
| Malformed negative BigInt | ABI parse of `-9223372036854775809x` | `parse_rc=0 kind=9 text=-9223372036854775809`; suffix silently discarded. | ✗ FAIL |
| Malformed nested BigInt | ABI parse of `[123456789012345678901x]` | `parse_rc=0 kind=7`; malformed document accepted. | ✗ FAIL |
| Invalid UTF-8 control | ABI parse of oversized digits plus byte `0xff` | `parse_rc=32`; invalid UTF-8 remains rejected. | ✓ PASS |

The malformed-input probe also reproduced success/truncation for `_`, `+`, `/`, and NUL suffixes. This is not a generic “the parser accepts trailing content” claim: ordinary small-number JSONTestSuite cases remain rejected. The failure is specific to the oversized-integer BigInt branch.

## Probe Execution

No conventional `scripts/*/tests/probe-*.sh` files or Phase 11 declared probe scripts exist.

**Step 7c:** SKIPPED — no probe contract was declared. The runnable ABI and subprocess checks above cover the phase behaviors directly.

## Requirements Coverage

| Requirement | Source plans | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| UP-01 | 11-01, 11-02, 11-07, 11-13, 11-14 | Exact audited simdjson v4.6.4 plus compatibility gates | ✗ BLOCKED | Exact provenance and automated gates pass, but the patch accepts malformed JSON and the exception boundary contradicts the normative ABI contract. |
| NUM-01 | 11-02, 11-03, 11-10, 11-13, 11-14 | Preserve valid oversized integers as exact text instead of invalid JSON | ✗ BLOCKED | Valid literals work, but the preservation mechanism also turns malformed oversized tokens into valid/truncated BigInts; the feature is not correctness-safe. |
| NUM-02 | 11-02, 11-03, 11-07 through 11-10, 11-13, 11-14 | `TypeBigInt`/`GetBigInt` without automatic arbitrary precision dependency | ✓ SATISFIED | Kind 9 and exact copied text cross native/Rust/Go paths; no production `math/big` import. |
| DIAG-01 | 11-06 through 11-09, 11-12 through 11-14 | Kernel report and exact override | ✓ SATISFIED | Mandatory bindings, process-global lock semantics, real report, and process-isolated tests pass. |
| DIAG-02 | 11-05, 11-07 through 11-10, 11-12 through 11-15 | Real offsets when upstream proves them; explicit unknown otherwise | ✓ SATISFIED | Corpus, known-zero, range proof, stale-state clearing, limit boundaries, and depth-1024 subprocess all pass. |
| LIMIT-01 | 11-04, 11-07 through 11-09, 11-11 through 11-15 | Safe immutable capacity/depth options and homogeneous pools | ✓ SATISFIED | Capacity checks precede copy; depth is capped at 1024 across all recursive consumers; pool identity includes both values. |

No Phase 11 requirement mapped in `REQUIREMENTS.md` is absent from all plans. There are no orphaned Phase 11 requirement IDs.

## Anti-Patterns Found

| File | Line / area | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `patches/simdjson-v4.6.4-positive-bigint.patch` | 18-75 | Nine early overflow returns changed from `INVALID_NUMBER` to `BIGINT_NUMBER` without retaining delimiter validation | 🛑 BLOCKER | Malformed oversized numeric tokens reach the string-preservation path. |
| generated patched `simdjson.cpp` | `parse_number` / `tape_builder::visit_number` | BigInt error returns before the normal delimiter check; visitor scans sign/digits only and returns success | 🛑 BLOCKER | Suffix bytes are silently discarded and malformed JSON becomes a valid BigInt/document. |
| `src/native/simdjson_bridge.cpp` | 253-258, 286-309 | Special `std::bad_alloc` overload returns a different public status from the normative all-C++-exception rule | 🛑 BLOCKER | Callers receive status 127 where ABI documentation promises 97. |
| `tests/rust_shim_bigint.rs`, JSONTestSuite oracle | coverage gap | No malformed oversized-token suffix matrix | ⚠ WARNING | Full Rust/Go/oracle gates remain green while the parser accepts malformed JSON. |
| `tests/rust_shim_minimal.rs` | exception seam | Only `std::runtime_error` is forced | ⚠ WARNING | The conflicting `std::bad_alloc` catch branch is never executed by the contract suite. |

No unreferenced `TBD`, `FIXME`, or `XXX` debt marker was found in the Phase 11 closure files. The word “placeholder” in benchmark documentation names an intentionally scoped DOM-era benchmark and does not flow to product behavior.

## Deferred-Phase Filter

Phases 12-16 cover DOM navigation, batched On-Demand extraction, borrowed views, streaming, and final v0.2 stabilization/release. None names malformed BigInt token validation or reconciliation of C++ exception status 97 versus 127. Both gaps remain Phase 11 compatibility-foundation defects and are **not deferred**.

Phase 16 does own publication of the final v0.2 artifact. That future publication does not fix either source defect and is not used to suppress them.

## Human Verification Required

None. The relevant behavior is ABI-, source-, subprocess-, and command-observable. There are no deferred `<human-check>` blocks in the Phase 11 plans.

## Gaps Summary

Plan 11-15 successfully closes the previous recursive diagnostic stack-overflow gap in the current source. ABI identity, v4.6.4 provenance, valid BigInt text flow, kernel controls, bounded parser limits, known/unknown diagnostic offsets, contract compilation, Go race tests, benchmark signal, and readiness checks are all substantive.

The phase goal is still not achieved:

1. The BigInt preservation path changes parser validity. It accepts malformed oversized numeric tokens and exposes a truncated decimal prefix as if the JSON were valid.
2. The C++ exception boundary disagrees with its normative ABI contract for `std::bad_alloc`.

Both issues affect the compatibility foundation, neither is covered by a later roadmap phase, and neither has an accepted override. Phase 12 should not proceed on the assumption that Phase 11 is complete.

---

_Verified: 2026-07-29T14:20:07Z_

_Verifier: the agent (gsd-verifier)_
