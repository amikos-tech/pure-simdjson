# Phase 11: Upstream simdjson refresh, exact big integers, and production diagnostics - Research

**Researched:** 2026-07-22
**Domain:** Go/Rust/C++ DOM ABI evolution, parser controls, diagnostics, and bootstrap distribution
**Confidence:** HIGH overall; MEDIUM for the exact set of malformed inputs that yield an upstream-proven byte location

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Upstream and BigInt Contract
- **D-01:** Upgrade the reproducibly pinned simdjson source from v4.6.1 to the audited v4.6.4 patch release and rerun the existing Go, Rust, C++, correctness, benchmark, contract, and five-target build gates.
- **D-02:** Enable upstream DOM BigInt preservation. Only integer-syntax literals below `-9223372036854775808` or above `18446744073709551615` become BigInt. The two boundary values remain `TypeInt64` and `TypeUint64`; decimal/exponent forms such as `1.0` and `1e20` remain `TypeFloat64`.
- **D-03:** Append `TypeBigInt` / native `ValueKindBigInt` after the existing kinds (expected numeric value `9`). Do not renumber kinds `0` through `8`, change an existing layout, or change behavior for JSON values the current parser already accepts.
- **D-04:** `Element.GetBigInt() (string, error)` is strict: it accepts only `TypeBigInt` and returns the exact decimal spelling as copied Go text. It does not accept `TypeInt64` or `TypeUint64`, and the package does not acquire an automatic `math/big` dependency.
- **D-05:** Existing numeric getters called on `TypeBigInt` return `ErrWrongType`, matching upstream's typed accessor contract. Update the currently anticipatory comments that mention `ErrPrecisionLoss`; those comments describe an unreachable kind today and do not override the selected Phase 11 behavior.

#### Truthful Error Locations
- **D-06:** Preserve the existing `Error.Offset() uint64` signature and add `Error.HasOffset() bool`. This is additive and keeps existing callers compiling.
- **D-07:** A known failure at byte zero is represented by `Offset() == 0` and `HasOffset() == true`. An unknown location is `Offset() == 0` and `HasOffset() == false`.
- **D-08:** Populate an offset only when upstream supplies a concrete, reliable, in-bounds location. Do not run `encoding/json`, a second scanner, or any other estimator to manufacture a location after simdjson fails.
- **D-09:** Failures for which upstream cannot provide a trustworthy location—including applicable stage-one syntax/UTF-8, resource, or internal failures—remain explicitly unknown. Exact message wording is flexible, but the programmatic known/unknown state must never be fabricated.

#### Parser Controls and Lifecycle
- **D-10:** Use immutable functional options: `NewParser(opts ...ParserOption) (*Parser, error)`. Existing zero-argument calls keep working; options are normalized and validated before native allocation.
- **D-11:** Deliberately change the pool constructor to `NewParserPool(opts ...ParserOption) (*ParserPool, error)`. The changed return count is an explicit, user-approved source break despite the general preference to preserve existing contracts.
- **D-12:** `Kernel()` and diagnostic-only `SetKernel(name string) error` are package-level because upstream kernel selection is process-wide. An empty name restores automatic selection; explicit names are validated for availability/runtime support. Kernel selection locks once the first parser or parser pool is created.
- **D-13:** Maximum input capacity and maximum depth are immutable per-parser options. Omitted/zero values mean the current defaults; invalid positive values fail instead of being silently clamped. Capacity must be rejected before the input is copied into Rust-owned padded memory.
- **D-14:** A parser pool stores one normalized option set, uses it for every miss, and rejects parsers whose configuration does not match the pool. Pools must not become heterogeneous through `Put`.

#### ABI and Bootstrap Compatibility
- **D-15:** Set the Phase 11 ABI to `0x00010002` and require it as the strict native minimum. The Phase 11 Go wrapper rejects ABI 1.1 artifacts even if they could still perform baseline parsing; no capability-gated legacy mode is supported.
- **D-16:** Treat the native change as additive ABI growth, not an ABI 2.0 reset: retain existing C symbols/signatures/layouts, append the BigInt kind, and add new exports for the new surface. An artifact claiming ABI 1.2 but missing any mandatory Phase 11 symbol fails as mismatched/corrupt rather than degrading.
- **D-17:** Coordinate the Go/Rust/header ABI constants, bootstrap version pin, compile-time ABI canary, release-readiness policy, and native artifacts. A matching ABI 1.2 artifact must be available through the default bootstrap path before the Phase 11 implementation is considered merge-ready.
- **D-18:** Require the ABI 1.2 artifact/build contract to pass the existing five-platform matrix. Explicit-path users with ABI 1.1 binaries receive `ErrABIVersionMismatch`; they are not silently kept on the old feature set.

### the agent's Discretion
- Exact `ParserOption` type and option function names, provided the public semantics above remain immutable and validated.
- Whether duplicate capacity/depth options are rejected or deterministically resolved, provided the behavior is documented and tested.
- Exact typed error used when kernel selection is locked or an option/kernel name is invalid; reuse existing sentinels when they describe the condition honestly and add no sentinel without need.
- Exact native diagnostic replay/current-location technique and error-message wording, provided only upstream-proven offsets set `HasOffset()`.
- Internal file organization and test decomposition across Go, Rust, C++, ABI, release, and platform checks.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| UP-01 | Upgrade vendored simdjson v4.6.1 to audited v4.6.4 without losing existing gates. | Pin tag `v4.6.4` at commit `1bcf71bd85059ab6574ea1159de9298dcc1212c5`, retain the single-header `build.rs` path, and run the contract/correctness/benchmark/five-target matrix. [VERIFIED: official git tag] [CITED: https://github.com/simdjson/simdjson/releases/tag/v4.6.4] |
| NUM-01 | Preserve oversized integer literals as exact text. | Set DOM `number_as_string(true)` for the Phase 11 constructor path and carry `BIGINT` through the bridge, Rust registry, and fast materializer. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/dom.md] |
| NUM-02 | Expose strict `TypeBigInt` and `GetBigInt`. | Append kind `9`; use upstream `get_bigint()` borrowed text and the repository's existing copy-out/free pattern to return an owned Go string. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/element.h] |
| DIAG-01 | Expose active/forced kernel diagnostics. | Reuse the implementation-name read API; add a process-global setter that checks the compiled implementation list and runtime support, with a creation-time lock. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md] |
| DIAG-02 | Expose truthful known/unknown byte offsets. | Preserve the native `UINT64_MAX` unknown sentinel, retain known zero with a separate Go boolean, and only accept an in-range pointer returned by upstream `current_location()`. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md] |
| LIMIT-01 | Add immutable maximum-capacity/depth options. | Normalize zero to upstream defaults, reject values that upstream would clamp, check capacity in Rust before `Vec::resize`/copy, and configure DOM depth before parsing. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/parser.h] |
</phase_requirements>

## Summary

The native update itself is narrow: the repository is pinned to simdjson `v4.6.1`, while official `v4.6.4` is commit `1bcf71bd85059ab6574ea1159de9298dcc1212c5`, published 2026-05-06. The v4.6.4 release is a patch protecting a 32-bit string-builder overflow; the official v4.6.1...v4.6.4 comparison does not change the DOM BigInt API used here. [VERIFIED: codebase git submodule status and official git tag] [CITED: https://github.com/simdjson/simdjson/releases/tag/v4.6.4] [CITED: https://github.com/simdjson/simdjson/compare/v4.6.1...v4.6.4]

The largest implementation risks are in this repository's glue, not in the upstream patch. Today the Rust registry resizes and copies into its padded `Vec` before simdjson can return `CAPACITY`; the loader binds every required symbol before it reads the ABI; known byte zero is collapsed to unknown; and BigInt is explicitly converted to precision loss or invalid kind in the bridge, registry, comments, and frame materializer. Each of those paths must change together. [VERIFIED: codebase grep]

The ABI/bootstrap constraint creates a real sequencing dependency: `release.yml` only publishes a tag whose commit is already anchored on `origin/main`, while the Phase 11 wrapper may not be declared ready until its default bootstrap path can fetch ABI 1.2. Plan an artifact-enabling source state that is squash-merged to `main`, followed by an operator-only checkpoint with the exact order **`main` -> strict readiness -> annotated tag -> `release.yml` -> Phase 06.1 public bootstrap validation**. Only after that proof may Phase 11 close or any remaining public integration be treated as merge-ready. No tag or publication happens during research or ordinary plan execution. This compatibility artifact is an intermediate foundation, not the final v0.2 release; Phase 16 still owns final API stabilization, release evidence, and the v0.2 publication. [VERIFIED: codebase release workflow, docs/releases.md, and ROADMAP.md]

**Primary recommendation:** Implement one normalized parser configuration end to end, add a two-stage ABI loader, copy BigInt text at the existing Rust ownership boundary, and make artifact availability a hard human checkpoint rather than hiding publication inside an implementation task.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| BigInt classification and raw digits | C++ native engine | Rust FFI / Go API | simdjson owns the DOM type and `string_view`; Rust performs the ownership copy and Go exposes the strict method. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/element-inl.h] |
| Maximum input capacity | Rust registry | Go options / C++ parser | Rust owns the first allocation and copy, so only a Rust preflight can guarantee rejection happens before copy. [VERIFIED: codebase grep] |
| Maximum depth | C++ native engine | Go/Rust configuration | DOM parser allocation owns the depth limit and emits `DEPTH_ERROR`; Go supplies immutable normalized configuration. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/parser-inl.h] |
| Kernel selection | Process-global C++ state | Go lifecycle lock | Upstream active implementation is process-wide; Go additionally locks selection when an empty pool is created. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md] |
| Error location proof | C++ failure path | Rust/Go error transport | The pointer is meaningful only while native holds the padded input; Rust transports the numeric sentinel and Go preserves known/unknown. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md] |
| ABI compatibility | Go loader / FFI binder | Rust/header/release guards | ABI must be read before new symbols are required, then every ABI 1.2 symbol becomes mandatory. [VERIFIED: codebase grep] |
| Default artifact availability | CI distribution | Go bootstrap | Existing CI publishes five artifacts; the bootstrap pin and canary choose which compatible artifact a default install loads. [VERIFIED: docs/releases.md and release.yml] |

## Project Constraints (from AGENTS.md)

- Do not include prohibited internal repository or domain information in commit messages, pull requests, or related artifacts; notify the user if a change would require it. [VERIFIED: session AGENTS.md]
- Use squash merge for both the artifact-producing integration and the final Phase 11 integration. [VERIFIED: session AGENTS.md]
- Prefer extending existing ownership, binding, bootstrap, and release mechanisms over adding parallel abstractions. [VERIFIED: session AGENTS.md]
- Explain new public behavior with plain language and small examples. [VERIFIED: session AGENTS.md]

## Standard Stack

### Core

| Component | Version / Pin | Purpose | Why Standard |
|-----------|---------------|---------|--------------|
| simdjson | `v4.6.4`, `1bcf71bd85059ab6574ea1159de9298dcc1212c5` | DOM parse, BigInt text, kernel dispatch, depth/capacity, diagnostic replay | Locked audited upstream; the project already builds its release single-header through `build.rs`. [CITED: https://github.com/simdjson/simdjson/releases/tag/v4.6.4] |
| Go | module baseline `1.24` | Public API, lifecycle, options, error semantics | Existing project constraint and public surface. [VERIFIED: go.mod] |
| Rust | stable `1.85+` project baseline | Handle registry, padded owned input, panic-safe C exports | Existing safety and ownership boundary. [VERIFIED: PROJECT.md and Cargo.toml] |
| C++ | C++17 | Thin simdjson bridge | Existing `cc`-driven amalgamation build; no CMake dependency is needed for the library. [VERIFIED: build.rs] |
| purego | `v0.10.0` | Runtime binding without cgo | Existing loader/binding mechanism; no replacement is needed. [VERIFIED: go.mod] |

### Supporting

| Component | Version | Purpose | When to Use |
|-----------|---------|---------|-------------|
| `cc` crate | `1.2.60` locked | Build the C++ single-header and bridge from Cargo | Keep the current `build.rs` path. [VERIFIED: Cargo.lock] |
| cbindgen | local `0.29.2` | Regenerate the committed public C header | Run after Rust export/type changes, then diff-check. [VERIFIED: local environment and Makefile] |
| Existing release workflows | repository-local | Five-platform build, signing, publication, public bootstrap validation | Reuse unchanged publication mechanics; Phase 11 only extends their ABI assertions/smokes. [VERIFIED: docs/releases.md] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Upstream BigInt tape text | Parse decimal digits in Go/Rust | Rejected: duplicates validated upstream behavior and creates more code and failure modes. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/dom.md] |
| Upstream `current_location()` proof | A Go scanner or `encoding/json` replay | Prohibited: a second parser would estimate a location rather than report simdjson's own state. |
| Existing tag-driven CI | Manual or branch artifact upload | Prohibited by the repository release runbook. [VERIFIED: docs/releases.md] |

**Installation:** No new external package is required. Update the existing simdjson gitlink and keep all current Go/Rust dependencies locked. [VERIFIED: codebase dependency manifests]

## Package Legitimacy Audit

Not triggered: Phase 11 recommends no new npm, PyPI, crates.io, or Go module dependency. simdjson is an already-vendored upstream git submodule pinned to an official release commit. [VERIFIED: codebase git submodule status] [CITED: https://github.com/simdjson/simdjson/releases/tag/v4.6.4]

## Architecture Patterns

### System Architecture Diagram

```text
NewParser(options) / NewParserPool(options)
        |
        +--> normalize zero/defaults + validate once
        |          |
        |          +--> pool stores comparable normalized config
        |
        +--> resolve native artifact
                   |
                   +--> bind ABI getter --> ABI < 1.2 / major mismatch --> ErrABIVersionMismatch
                   |
                   +--> ABI compatible --> bind every mandatory 1.2 symbol
                                             |
Parse(input) ---------------------------------+
        |
        v
Rust registry: handle/busy check --> len > configured capacity? --> typed capacity error
        |                                                     (no resize/copy)
        v
resize reusable arena + copy input + padding
        |
        v
C++ DOM parser (BigInt enabled, configured depth, selected process kernel)
        |                         |
        | success                 | failure
        v                         v
document/tape          upstream-only diagnostic replay
        |                         |
        | BigInt                   +--> concrete in-range pointer? offset : unknown
        v
C++ string_view --> Rust-owned copy --> Go string --> native copy freed
```

The diagram reflects the current ownership boundary: Rust retains the padded input for document lifetime and the C++ DOM borrows it; public string results are copied before they escape. [VERIFIED: codebase grep]

### Recommended Project Structure

```text
parser_options.go                 # ParserOption, normalized comparable config, validation
kernel.go                         # package-global Kernel/SetKernel lifecycle state
parser.go / pool.go               # constructors, config storage, pool homogeneity
element.go / errors.go            # TypeBigInt/GetBigInt and HasOffset
internal/ffi/types.go             # ABI 1.2, kind 9, config/error mirrors
internal/ffi/bindings.go          # ABI-first bind, mandatory Phase 11 symbols
src/lib.rs                        # additive C exports and ABI constant
src/runtime/{mod,registry}.rs     # config storage, pre-copy capacity gate, copied BigInt
src/native/simdjson_bridge.{h,cpp}# upstream options, kernel, location proof, frame kind
include/pure_simdjson.h           # regenerated public ABI
tests/                            # Rust/C/ABI/smoke coverage
scripts/release/                  # ABI-minimum and bootstrap readiness policy
```

This split follows existing files; only `parser_options.go` and `kernel.go` are suggested additions, not a new subsystem. [VERIFIED: codebase layout]

### Component Responsibilities and Exact Seams

| File / boundary | Required Phase 11 change | Planning dependency |
|-----------------|--------------------------|---------------------|
| `third_party/simdjson` + `build.rs` | Move the gitlink from v4.6.1 to the official v4.6.4 commit while retaining the existing single-header, `cc`, and C++17 build path. [VERIFIED: codebase gitlink/build.rs] [CITED: https://github.com/simdjson/simdjson/releases/tag/v4.6.4] | Pin first, then make all native behavior and characterization tests run against the new source. |
| `src/native/simdjson_bridge.{h,cpp}` | Add a configured parser constructor path, call `number_as_string(true)`, establish max capacity/depth before first parse, map `BIGINT` to kind `9`, expose BigInt text views, write BigInt frame text into the existing `string_ptr/string_len` fields, add synchronized implementation selection, and capture only proven error locations. Retain every existing bridge signature. [VERIFIED: codebase bridge] [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/dom.md] | Native characterization must precede the Rust/Go contract work for offsets and depth boundaries. |
| `src/runtime/mod.rs` + `src/runtime/registry.rs` | Thread one normalized `{max_capacity,max_depth}` configuration into parser creation; store it on `ParserEntry`; reject `input.len() > max_capacity` before `Vec::resize` or `copy_from_slice`; copy BigInt bytes through the existing registered allocation/free mechanism; and reset stale native diagnostic state on every pre-copy rejection. [VERIFIED: codebase registry/copy-out paths] | The pre-copy gate and diagnostic reset must land together or a capacity failure can inherit the prior parse's message/offset. |
| `src/lib.rs` | Bump ABI to `0x00010002`, append `PURE_SIMDJSON_VALUE_KIND_BIGINT = 9`, retain `pure_simdjson_parser_new`, and add mandatory additive exports equivalent to `pure_simdjson_parser_new_with_limits`, `pure_simdjson_element_get_bigint`, and `pure_simdjson_set_implementation_name`; every export stays inside `ffi_wrap`. [VERIFIED: existing export pattern and FFI contract] | Generate the header only after Rust signatures and numeric values are final. Names may follow the repository prefix exactly, but the three capabilities must be mandatory for an ABI 1.2 artifact. |
| `include/pure_simdjson.h` | Regenerate with cbindgen; preserve all existing signatures/layouts and append kind `9`, any new honest capacity-limit status, and the new functions. [VERIFIED: Makefile/cbindgen flow] | `make verify-contract` must show a clean generated-header diff and C layout compile. |
| `internal/ffi/types.go` | Set `ABIVersion = 0x00010002`, append `ValueKindBigInt = 9`, mirror any new status code, and leave `ValueView` and `InternalFrame` layouts unchanged. [VERIFIED: current mirrored ABI types] | Cross-language numeric assertions must be updated in the same task. |
| `internal/ffi/bindings.go` + `library_loading.go` | Probe and validate the ABI getter before binding the ABI 1.2 table; reject ABI 1.1 as `ErrABIVersionMismatch`; after a compatible 1.x/1.2+ handshake, require every Phase 11 symbol and treat absence as a corrupt/mismatched artifact. Refresh the cached implementation name after an override. [VERIFIED: current bind-before-check ordering] | This loader refactor must precede tests that use an ABI 1.1 fixture; otherwise they fail as generic missing-symbol errors. |
| `parser_options.go` + `parser.go` | Define immutable `ParserOption` closures over a private comparable config; use `WithMaxCapacity(int)` and `WithMaxDepth(int)`; normalize/validate once; make `NewParser(opts ...ParserOption)` call an internal constructor that accepts normalized config; store that config on `Parser`. [VERIFIED: current constructor/lifecycle] | Public option tests come before pool work so the pool can reuse the same normalizer and constructor. |
| `pool.go` | Change to `NewParserPool(opts ...ParserOption) (*ParserPool, error)`, store the normalized config, use it for every miss, and compare a returned parser's effective config during `Put`. [VERIFIED: current sync.Pool wrapper] | Update every call site in one compile-fixing task; do not allow a temporary heterogeneous pool. |
| `kernel.go` + native setter | Serialize `Kernel`, `SetKernel`, `NewParser`, and `NewParserPool` around one Go lifecycle state; independently serialize native setter/parser construction so non-Go callers are safe. Empty selection restores auto-detection; an explicit supported `fallback` is distinguishable from silent automatic fallback. [VERIFIED: current fallback gate and loader cache] [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md] | Kernel tests must be subprocess-isolated because creation locks state for the rest of the process. |
| `element.go` + `materializer_fastpath.go` | Append `TypeBigInt`; add strict copied `GetBigInt`; route all existing numeric getters on kind `9` to `ErrWrongType`; copy frame text before the doc lock/lifetime ends; remove anticipatory precision-loss comments. [VERIFIED: current element/materializer switches] | Root, descendant, iterator, lookup, and frame paths must be tested together to prevent partial propagation. |
| `errors.go` | Add `hasOffset bool` to `Error`/native details, preserve `Offset() uint64`, add `HasOffset() bool`, and derive known state from `raw != LastErrorOffsetUnknown` before normalizing. Add an honest capacity-limit sentinel/status because `ErrInternal` does not describe an input exceeding a caller-selected bound. [VERIFIED: current normalization and status mapping] | Known zero, unknown, and stale-detail-after-capacity-failure tests are required before changing formatting. |
| `internal/bootstrap/{version.go,abi_assertion.go}` + `scripts/release/check_bootstrap_abi_state.py` | Move the selected intermediate artifact version, compile-time canary, and ABI minimum policy to 1.2 in one source state. [VERIFIED: current ABI/bootstrap guard] | Version selection is a human decision and must precede the artifact-enabling merge. |
| `.github/workflows/release.yml` + smoke/header/release tests | Require the complete ABI 1.2 export/smoke surface on all five artifacts and keep tag ancestry on `origin/main`. [VERIFIED: release.yml] | This source must be on `main` before strict readiness and the annotated tag; CI remains the only publisher. |

### Pattern 1: Normalize Once, Store Comparable Configuration

Use `WithMaxCapacity(int)` and `WithMaxDepth(int)` as the public option constructors. Resolve zero to `SIMDJSON_MAXSIZE_BYTES` (`0xFFFFFFFF`) and `DEFAULT_MAX_DEPTH` (`1024`) during construction, then store the effective values on both `Parser` and `ParserPool`. Explicit default and omitted configuration should compare equal. Let later duplicate options win deterministically; document and test that rule. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/base.h]

Reject negative values, positive capacity below 32 (upstream would clamp it), capacity above `0xFFFFFFFF`, nonzero depth values that cannot cross the ABI safely, and any native-width conversion overflow. Capacity `32` is valid; only `1..31` would be changed by upstream. Return a small Go-side invalid-option sentinel rather than letting a caller error become `ErrInternal`. Use a dedicated ABI/Go capacity-limit status for a valid parse input that exceeds the configured maximum; the current `ErrInternal` mapping is not truthful for that condition. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/base.h] [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/parser-inl.h] [VERIFIED: current status mapping]

### Pattern 2: ABI-First, Symbols-Second Binding

Split binding into a minimal ABI probe and the complete ABI 1.2 registration. This is required because the current binder looks up all symbols before `NewParser` checks the version; adding mandatory symbols there would turn an ABI 1.1 explicit-path artifact into a generic load failure instead of `ErrABIVersionMismatch`. After a compatible 1.2 handshake, a missing BigInt/config/kernel symbol must fail binding with the missing symbol named. [VERIFIED: internal/ffi/bindings.go and parser.go]

Treat `0x00010002` as the minimum within ABI major 1: reject 1.1 and all other majors, accept 1.2 or a later additive 1.x only if every symbol this wrapper requires binds successfully. Keep the bootstrap canary exact for the artifact version it pins. This reconciles “strict minimum” with additive minor growth. [VERIFIED: docs/ffi-contract.md version encoding]

### Pattern 3: Reuse the Existing Copy-Out Ownership Path

Add a C++ `get_bigint` view helper parallel to `get_string_view`; copy its `std::string_view` in Rust; register/free the allocation through the existing byte-allocation registry; and convert it to Go text in the binding. Do not return a borrowed pointer to Go. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/element-inl.h] [VERIFIED: codebase `element_get_string` path]

The internal materializer can reuse `InternalFrame.StringPtr/StringLen` for kind `9`, so the pinned frame layout need not grow. Add kind `9` cases in both native frame production and Go frame consumption; clear partial frames on any error as today. [VERIFIED: src/native/simdjson_bridge.h and materializer_fastpath.go]

### Pattern 4: Prove, Do Not Guess, Error Locations

Keep `UINT64_MAX` as the C/Rust unknown sentinel. On a DOM failure, an internal failure-only replay may use upstream On-Demand traversal (`raw_json()` consumes the document) and then `current_location()`. Accept the result only when upstream returns success and the pointer is within `[input, input+len)`; immediately convert it to an integer offset. If `iterate()` itself fails, `current_location()` fails, the pointer is at/end/outside the input, or the primary failure is resource/internal, retain unknown. This is an inference-backed use of public upstream diagnostics, not a second-parser estimate. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md] [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/generic/ondemand/document-inl.h]

Upstream explicitly says no location is available when `iterate()` fails for cases including `EMPTY`, `UTF8_ERROR`, `UNESCAPED_CHARS`, and `UNCLOSED_STRING`; tests must expect unknown for those cases rather than promise broad UTF-8 coverage. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md]

At the Go boundary, store `offset uint64` and `hasOffset bool` separately. `wrapParserStatus` sets the boolean for every native value other than `LastErrorOffsetUnknown`, including zero; `Offset()` stays zero for unknown, and error formatting consults the boolean rather than `offset != 0`. [VERIFIED: errors.go]

### Pattern 5: Lock Process-Global Kernel State at Creation

Add a native setter that looks up the exact, case-sensitive name, rejects null/not-compiled implementations, checks `supported_by_runtime_system()`, and assigns `get_active_implementation()`. Empty input assigns `detect_best_supported()`. Protect setter/parser creation with native process-global synchronization; protect Go `SetKernel`, `NewParser`, and `NewParserPool` with a Go mutex so even creating an empty pool freezes selection. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md]

An explicitly requested `fallback` is diagnostic opt-in and is not the silent automatic fallback prohibited by project policy; track explicit selection so it can be used for reproducibility, while automatic fallback still returns `ErrCPUUnsupported`. Kernel mutation tests must run in subprocesses because the first parser/pool permanently locks state for that process. [VERIFIED: PROJECT.md and existing fallback gate tests]

`Kernel()` should be read-only and avoid initiating a download: return the active cached native implementation after the library has been loaded, and `""` before it is available. `SetKernel` may resolve/load the library because validation requires the compiled implementation registry; document that distinction. [VERIFIED: library_loading.go]

### Pattern 6: Treat Artifact Publication as a Phase Gate

The plan must split the work around the repository's publication invariant:

1. Produce and fully test a source state containing ABI `0x00010002`, every mandatory Phase 11 native symbol, the chosen bootstrap pin, updated canary/policy, compatible Go source, ABI 1.2 smoke assertions, and the matching changelog entry; integrate that artifact-enabling state to `main` by squash merge. Do not mark Phase 11 complete at this point. [VERIFIED: docs/releases.md and release.yml]
2. Pause at a human/operator checkpoint. The normal executor must not create a tag or publish. From the `main`-anchored commit, the operator runs `bash scripts/release/check_readiness.sh --strict --version <semver-without-v>`. [VERIFIED: repo-local release skill and docs/releases.md]
3. Only after strict readiness passes, create and push an annotated `v<version>` tag on that same `main` commit. `release.yml` verifies that the tag commit is an ancestor of `origin/main`, then builds, smokes, signs, and publishes all five artifacts. [VERIFIED: docs/releases.md and release.yml]
4. After `release.yml` succeeds, dispatch `.github/workflows/public-bootstrap-validation.yml` with the published version: full five-target R2 validation plus the existing representative GitHub-fallback subset. This is Phase 06.1's responsibility. [VERIFIED: docs/releases.md and public-bootstrap-validation.yml]
5. Only after the default bootstrap fetch loads ABI 1.2 and passes smoke may Phase 11 close or any remaining public integration be squash-merged. If post-publication fixes change native bytes, they require a new version and a fresh annotated tag; never move or replace the published tag. Phase 16 later repeats release-grade stabilization for the complete v0.2 surface. [VERIFIED: ROADMAP.md and immutable release layout in docs/releases.md]

This staged integration is logically necessary because the supported workflow refuses to publish an unmerged branch commit. It is not permission to tag or publish during planning/execution. [VERIFIED: release.yml]

### Anti-Patterns to Avoid

- **Bind new symbols before reading ABI:** converts the required ABI mismatch into an opaque lookup failure. [VERIFIED: current binding order]
- **Check capacity in C++ only:** Rust has already allocated and copied by then. [VERIFIED: current registry order]
- **Infer `HasOffset` from a nonzero offset:** loses the valid byte-zero state. [VERIFIED: current `hasOffset` helper]
- **Return upstream BigInt `string_view` directly:** it is document-owned and must not escape to Go. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/element.h]
- **Make kernel selection per parser:** upstream's active implementation is process-wide. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md]
- **Publish ABI 1.2 with optional Phase 11 symbols:** contradicts the strict complete-contract decision.
- **Treat the Phase 11 bootstrap artifact as the v0.2 release:** Phase 16 owns the final release boundary. [VERIFIED: ROADMAP.md]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Big integer recognition | Decimal parser/range classifier | `number_as_string(true)`, `element_type::BIGINT`, `get_bigint()` | Upstream already preserves raw digits and enforces strict typed accessors. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/tests/dom/big_integer_tests.cpp] |
| Kernel registry/CPU detection | CPUID tables in Go/Rust | `get_available_implementations()` and `supported_by_runtime_system()` | Matches the exact implementations compiled into the artifact. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md] |
| Error offset estimator | Go scanner or `encoding/json` replay | Upstream `current_location()` when it returns a valid pointer; otherwise unknown | Different parsers can disagree, so estimates would violate the truthfulness contract. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md] |
| BigInt memory ownership | New allocator/free family | Existing Rust byte-copy registry and `pure_simdjson_bytes_free` | It already handles copied strings across the ABI. [VERIFIED: codebase grep] |
| Pool implementation | Custom concurrent queue | Existing `sync.Pool` plus normalized config checks | Existing lifecycle/finalizer behavior remains valid. [VERIFIED: pool.go] |
| Artifact publication | Local upload script or prep-branch checksum rewrite | Existing `release.yml`, published `SHA256SUMS`, and Phase 06.1 workflow | This is the only supported publication/validation path. [VERIFIED: docs/releases.md] |

**Key insight:** Phase 11 is mostly careful propagation of upstream state through existing ownership and compatibility boundaries; parallel mechanisms would add ambiguity without adding capability.

## Runtime State Inventory

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | Existing downloaded ABI 1.0/1.1 libraries live under version-qualified cache directories. The new bootstrap version resolves to a different `v<version>/<goos>-<goarch>` path, so old bytes are not overwritten or mistaken for ABI 1.2. [VERIFIED: internal/bootstrap/cache.go] | No data migration and no deletion. Keep prior cache entries recoverable; validate the newly versioned path through Phase 06.1. |
| Live service config | Release publication depends on the existing CI release workflow and its configured signing/storage credentials; Phase 11 introduces no new secret or variable name. [VERIFIED: docs/releases.md and release.yml] | No configuration rename. An operator must confirm the existing release environment is available at the post-`main` checkpoint. |
| OS-registered state | None: the repository does not register launch agents, services, scheduled tasks, or package-manager daemons for parser operation. [VERIFIED: codebase inspection] | None. |
| Secrets/env vars | `PURE_SIMDJSON_LIB_PATH` remains the explicit-library escape hatch, and `PURE_SIMDJSON_CACHE_DIR` remains the cache-root override. A path pinned to ABI 1.1 must intentionally fail with `ErrABIVersionMismatch`; names do not change. [VERIFIED: library_loading.go and internal/bootstrap/cache.go] | No env migration. Update the binary named by explicit-path test/deployment configuration when adopting Phase 11. |
| Build artifacts / loaded process state | `target/release` may contain an ABI 1.1 library, and `cachedLibrary` plus upstream kernel selection remain process-global after first use. [VERIFIED: testmain_test.go, library_loading.go, and upstream implementation-selection contract] | Rebuild the release library before Go tests, use an isolated explicit path/cache for compatibility fixtures, and start a fresh subprocess for each kernel-lock scenario. No on-disk source migration. |

**Canonical result:** Updating every tracked file is not sufficient by itself: old explicit-path artifacts and already-running processes remain old/locked until the binary is replaced and the process restarts. Versioned default caches need no destructive cleanup. [VERIFIED: codebase loader/cache behavior]

## Common Pitfalls

### Pitfall 1: Capacity Limit After Allocation
**What goes wrong:** A huge input is copied into the Rust arena before returning a limit error. [VERIFIED: src/runtime/registry.rs]
**How to avoid:** Compare `input.len()` to the normalized parser capacity before taking/resizing `reusable_input`; keep the native capacity as defense in depth.
**Warning signs:** Allocation telemetry changes or the reusable buffer grows after a rejected parse.

### Pitfall 2: Upstream Silently Clamps Small Capacity
**What goes wrong:** `set_max_capacity` maps values below 32 to 32, hiding invalid configuration. A requested capacity of exactly 32 is already valid. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/parser-inl.h]
**How to avoid:** Reject positive values below 32 before native construction; zero alone means default.
**Warning signs:** `WithMaxCapacity(1)` succeeds.

### Pitfall 3: Depth Boundary Is Off by One
**What goes wrong:** Existing evidence shows default `1024` accepts 1023 nested arrays and rejects 1024. [VERIFIED: materializer_fastpath_test.go]
**How to avoid:** Define the option in upstream max-depth terms and add `N-1` accepted / `N` rejected tests for a small N and for the default.
**Warning signs:** Documentation says “N nested containers allowed” without an executable boundary test.

### Pitfall 4: Partial BigInt Propagation
**What goes wrong:** Parse succeeds but `Type`, iteration, direct lookup, or fast materialization returns invalid/precision loss because one kind switch was missed. [VERIFIED: codebase grep]
**How to avoid:** Audit every `ValueKind`/`element_type` switch across C++, Rust, Go, frame materialization, tests, and header checks.
**Warning signs:** Root BigInt works but nested BigInt or Tier 1 materialization fails.

### Pitfall 5: Losing Known Offset Zero
**What goes wrong:** Current normalization treats zero as unknown. [VERIFIED: errors.go]
**How to avoid:** Carry an explicit boolean from the raw unknown sentinel and test known-zero formatting/accessors separately.
**Warning signs:** `HasOffset()` is implemented as `Offset() != 0`.

### Pitfall 6: Overpromising Error Coverage
**What goes wrong:** Stage-one UTF-8/string failures have no valid document and therefore no `current_location`. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md]
**How to avoid:** Golden-test a small known/unknown corpus and describe coverage by upstream evidence, not error category names.
**Warning signs:** Every `ErrInvalidJSON` reports an offset.

### Pitfall 7: Kernel Race or Test Pollution
**What goes wrong:** Concurrent setter/constructor calls select inconsistent state, or an earlier test permanently locks selection. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md]
**How to avoid:** Synchronize natively and in Go; use subprocess tests for pre-lock, post-lock, reset-auto, invalid-name, unsupported-name, and explicit fallback cases.
**Warning signs:** Kernel tests pass only in a particular test order.

### Pitfall 8: Wrong Error for ABI 1.1
**What goes wrong:** Mandatory symbol lookup runs first and explicit-path users receive a generic load error. [VERIFIED: current binding order]
**How to avoid:** Probe ABI first, return `ErrABIVersionMismatch` for 1.1, then require the full 1.2 surface.
**Warning signs:** The ABI 1.1 fixture error mentions a missing BigInt symbol.

### Pitfall 9: Heterogeneous Pool
**What goes wrong:** A parser with different bounds enters a pool and future callers receive inconsistent policy.
**How to avoid:** Store effective config on parser/pool and compare under the parser mutex before `Put`.
**Warning signs:** Options are stored only as non-comparable closures or rerun on every miss.

### Pitfall 10: Treating Publication as an Ordinary Code Task
**What goes wrong:** A plan attempts to tag an unmerged branch, bypass CI, or calls Phase 11 complete before public bootstrap validation. [VERIFIED: docs/releases.md]
**How to avoid:** Use an explicit human checkpoint and preserve `main -> strict readiness -> annotated tag -> release.yml -> Phase 06.1 validation` sequencing.
**Warning signs:** A plan contains `git tag`, a manual upload, or a merge-ready claim before the artifact URL is proven.

### Pitfall 11: Stale Diagnostics on a Rust-Side Capacity Rejection
**What goes wrong:** The registry rejects an oversized input before entering C++, but `wrapParserStatus` still asks the C++ parser for its last message and offset, so details from the preceding parse can leak into the new capacity error. [VERIFIED: current registry/error-detail flow]
**How to avoid:** Clear native parser diagnostics at parse-attempt start or maintain authoritative last-error state in Rust; test a known-offset failure followed by an oversized input and require the second error to have no inherited offset/message.
**Warning signs:** A capacity-limit error reports syntax text or an offset from an earlier buffer.

## Code Examples

### Upstream BigInt Preservation

```cpp
// Source: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/dom.md
simdjson::dom::parser parser;
parser.number_as_string(true);
simdjson::dom::element doc;
auto error = parser.parse("[123456789012345678901]"_padded).get(doc);
std::string_view digits;
error = doc.at(0).get_bigint().get(digits);
```

Upstream returns `INCORRECT_TYPE` when `get_int64`, `get_uint64`, or `get_double` is called on this value; map that to `ErrWrongType`. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/tests/dom/big_integer_tests.cpp]

### Recommended Go Option Shape

```go
parser, err := purejson.NewParser(
	purejson.WithMaxCapacity(8 << 20),
	purejson.WithMaxDepth(128),
)

pool, err := purejson.NewParserPool(purejson.WithMaxCapacity(8 << 20))
```

Both constructors validate before native allocation; the pool stores the effective configuration and applies it on every miss.

### Truthful Offset Use

```go
var parseErr *purejson.Error
if errors.As(err, &parseErr) && parseErr.HasOffset() {
	log.Printf("invalid JSON at byte %d", parseErr.Offset())
} else {
	log.Printf("invalid JSON at an unknown byte")
}
```

This example deliberately checks `HasOffset()` rather than treating zero as unknown.

### Official Kernel Validation Pattern

```cpp
// Source: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md
auto implementation = simdjson::get_available_implementations()[name];
if (implementation == nullptr) { /* invalid name */ }
if (!implementation->supported_by_runtime_system()) { /* unsafe on this CPU */ }
simdjson::get_active_implementation() = implementation;
```

## State of the Art

| Old Approach | Current Phase 11 Approach | When Changed | Impact |
|--------------|---------------------------|--------------|--------|
| simdjson v4.6.1 gitlink | Official v4.6.4 commit | v4.6.4 released 2026-05-06 | Audited patch base with the same DOM BigInt surface. [CITED: https://github.com/simdjson/simdjson/releases/tag/v4.6.4] |
| Oversized integer rejects parse | Opt-in DOM BigInt tape text | Available in upstream 4.6.x | Valid integer syntax is preserved exactly. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/dom.md] |
| `Offset()==0` means unknown | `HasOffset()` carries truth independently | Phase 11 contract | Byte zero becomes representable without breaking `Offset()`. |
| No parser construction options | Normalized immutable functional options | Phase 11 contract | Bounds are stable for parser and pool lifetime. |
| Auto kernel only | Diagnostic process-global override before creation | Phase 11 contract | Reproducible tests/benchmarks without pretending kernel choice is per parser. |
| Bind everything, then compare exact ABI | ABI probe, compatibility check, mandatory bind | Phase 11 | Correct mismatch semantics and corruption detection. [VERIFIED: current loader gap] |

**Deprecated/outdated:**
- BigInt comments promising `ErrPrecisionLoss` are anticipatory and must be replaced with the locked strict `ErrWrongType` contract. [VERIFIED: element.go]
- A source-prep checksum rewrite is not part of the current release path; runtime resolves published `SHA256SUMS`. [VERIFIED: docs/releases.md]
- `allocate_capacity` is deprecated upstream; use `allocate`/constructor configuration. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/parser.h]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| — | No `[ASSUMED]` factual claims are used. Recommendations are derived from locked decisions, repository inspection, and official v4.6.4 sources. | All | — |

## Open Questions

1. **Which semantic version will publish the intermediate ABI 1.2 bootstrap artifact?**
   - What we know: source currently pins bootstrap version `0.1.4`, and the readiness policy has entries only for ABI 1.0/1.1. [VERIFIED: internal/bootstrap/version.go and check_bootstrap_abi_state.py]
   - What's unclear: the user did not lock the next artifact version, and this research does not authorize a tag.
   - Recommendation: make version selection a human checkpoint, then add that exact version to the ABI 1.2 policy/canary; do not consume the Phase 16 `v0.2` release label.

2. **Which malformed inputs produce a stable upstream location through diagnostic replay?**
   - What we know: upstream documents locations after errors on a valid On-Demand document and explicitly documents no location for several `iterate()` failures. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md]
   - What's unclear: the exact covered subset for this repository's DOM-error corpus is runtime-path dependent.
   - Recommendation: start DIAG-02 with a small native characterization test and lock only observed, upstream-proven known/unknown cases; never make all syntax/UTF-8 errors known.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go | Public API/tests | ✓ | local `1.26.5`; module baseline `1.24` | CI uses `go.mod`. [VERIFIED: local probe and go.mod] |
| Rust/Cargo | ABI shim/tests | ✓ | local `1.89.0`; project baseline `1.85+` | CI toolchains. [VERIFIED: local probe and PROJECT.md] |
| Apple clang++ | Local C++ bridge | ✓ | Apple clang 21 | Five-platform CI compilers. [VERIFIED: local probe] |
| cbindgen | Header generation | ✓ | `0.29.2` | CI installs cbindgen. [VERIFIED: local probe and phase2 workflow] |
| Python | ABI/release tests | ✓ | `3.14.5` | Hosted CI Python. [VERIFIED: local probe] |
| Docker | Linux/release approximation | ✓ | client/server `29.6.1` | Hosted release runners. [VERIFIED: local probe] |
| Five target OS/architectures | Artifact contract | CI only | linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64 | Existing release matrix. [VERIFIED: release.yml] |

**Missing dependencies with no fallback:** None for local planning/implementation.

**Missing dependencies with fallback:** Four non-local target environments are covered by the existing hosted matrix; no local machine can replace the final five-target evidence. [VERIFIED: release.yml]

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Frameworks | Go `testing`/fuzz, Rust unit+integration tests, Python `unittest`, C compile/smoke, GitHub Actions five-platform matrix. [VERIFIED: codebase test inventory] |
| Config file | `Makefile`, `Cargo.toml`, `go.mod`, workflow YAML; no separate test config. [VERIFIED: codebase] |
| Quick run command | `cargo test --locked --test rust_shim_accessors --test rust_shim_minimal -- --test-threads=1 && cargo build --release --locked && go test . -run 'Test(BigInt|ParserOption|ParserPoolOption|Kernel|ErrorOffset|Capacity|Depth)' -count=1` |
| Full suite command | `make verify-contract && cargo build --release --locked && go test ./... -race && python3 scripts/release/test_check_bootstrap_abi_state.py && python3 scripts/release/test_release_workflow_contracts.py && python3 scripts/release/test_public_bootstrap_validation_contracts.py && python3 scripts/release/test_render_release_notes.py` |

### Test Layers

| Layer | Purpose | Command / gate |
|-------|---------|----------------|
| C++ characterization | Lock v4.6.4 BigInt boundaries, configured depth/capacity behavior, exact implementation selection, and the malformed-input subset with proven `current_location()`. | Add focused cases to a Rust integration binary or native test target, then run `cargo test --locked --test rust_shim_bigint --test rust_shim_diagnostics --test rust_shim_kernel -- --test-threads=1`. Wave 0 creates these targets. |
| Rust registry/FFI | Prove pre-copy capacity rejection, copied BigInt ownership, panic-safe exports, kind/status numbers, and no stale diagnostic carry-over. | `cargo test --locked -- --test-threads=1` (already owned by `make verify-contract`). |
| Generated C ABI | Prove ABI `0x00010002`, old symbol retention, new mandatory symbols, enum append-only behavior, out-param rules, and unchanged struct layouts. | `make verify-contract`; extend `tests/abi/check_header.py`, `test_check_header.py`, `handle_layout.c`, and `tests/smoke/ffi_export_surface.c`. |
| Go public contract | Prove strict getters, known/unknown offsets, immutable options, homogeneous pools, kernel lifecycle, and deliberate pool-constructor source update. | `cargo build --release --locked && go test ./... -race`. Use subprocess helpers for kernel cases. |
| Correctness + performance fence | Preserve JSONTestSuite behavior for the pre-existing corpus and detect material Tier 1/2/3 movement after the upstream/native changes. | `go test . -run '^TestJSONTestSuiteOracle$' -count=1` and `bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir /tmp/phase11-pr-bench`. |
| Release source-state contract | Prove bootstrap pin/canary/policy, tag ancestry policy, changelog rendering, and Phase 06.1 workflow structure before any operator action. | Run the four Python release test files in the full-suite command; do **not** run a tag or publication command during implementation. |
| Hosted artifact proof | Prove the exact ABI 1.2 artifact on linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, and windows/amd64, then prove public bootstrap paths. | Operator gate only: strict readiness on `main`, annotated tag, `release.yml`, then Phase 06.1 `public-bootstrap-validation.yml`. [VERIFIED: docs/releases.md] |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UP-01 | Pin is exactly v4.6.4; header/ABI/Go/Rust/C++ and correctness remain green | contract + integration | `git -C third_party/simdjson rev-parse HEAD && make verify-contract && cargo build --release && go test ./... -race` | Partial; new pin assertion is Wave 0 |
| UP-01 | Existing JSONTestSuite oracle still agrees | correctness | `go test . -run '^TestJSONTestSuiteOracle$' -count=1` | Existing |
| UP-01 | Representative Tier 1/2/3 paths do not regress unexpectedly | benchmark smoke | `bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir /tmp/phase11-pr-bench` | Existing |
| NUM-01 | Below/above bounds become BigInt; boundaries and float syntax keep existing kinds | unit + cross-ABI | `go test . -run '^TestBigInt(Classification|Boundaries)$' -count=1 && cargo test --test rust_shim_accessors -- --test-threads=1` | Wave 0 |
| NUM-02 | Strict copied `GetBigInt`; other numeric getters return `ErrWrongType`; nested/root/materializer paths work | unit + lifetime | `go test . -run '^TestBigInt(Getter|WrongType|CopyLifetime|Materializer)$' -count=1` | Wave 0 |
| DIAG-01 | Read, set, auto-reset, invalid/unsupported, explicit fallback, and post-create lock behavior | subprocess integration | `go test . -run '^TestKernel' -count=1 && cargo test --test rust_shim_kernel -- --test-threads=1` | Wave 0 |
| DIAG-02 | Known nonzero, known zero, and explicit unknown survive C++ -> Rust -> Go | native + Go unit | `go test . -run '^TestError(HasOffset|OffsetKnownZero|OffsetUnknown|OffsetCorpus)$' -count=1 && cargo test --test rust_shim_diagnostics -- --test-threads=1` | Wave 0 |
| LIMIT-01 | Invalid options fail before native allocation; zero preserves defaults | unit | `go test . -run '^TestParserOption' -count=1` | Wave 0 |
| LIMIT-01 | Oversize input is rejected before Rust buffer resize/copy | Rust white-box + Go integration | `cargo test registry::tests::capacity -- --test-threads=1 && go test . -run '^TestParserCapacity' -count=1` | Wave 0 |
| LIMIT-01 | Configured depth N boundary and default boundary are stable | integration | `go test . -run '^TestParserDepth' -count=1` | Partial; existing default tests, configured tests Wave 0 |
| LIMIT-01 | Pool misses reuse config and mismatched `Put` is rejected | unit/race | `go test . -run '^TestParserPool.*(Option|Config)' -race -count=1` | Wave 0 |
| D-15..D-18 | ABI 1.1 gives mismatch; ABI 1.2 missing symbol gives corrupt/load error; 1.2 complete binds | loader contract | `go test . -run '^TestABI' -count=1 && go test ./internal/ffi -run '^TestBind' -count=1` | Partial; Wave 0 fixtures required |
| D-17..D-18 | Bootstrap policy, canary, release source/changelog, and five artifacts agree | release contract | `python3 scripts/release/test_check_bootstrap_abi_state.py && python3 scripts/release/test_release_workflow_contracts.py && python3 scripts/release/test_public_bootstrap_validation_contracts.py && python3 scripts/release/test_render_release_notes.py` | Existing tests require Phase 11 cases |

### Sampling Rate

- **Per task commit:** Run the narrow Go/Rust command for the touched layer plus `make verify-contract` whenever ABI/header code changes.
- **Per wave merge:** `cargo build --release && go test ./... -race`; run the correctness oracle after upstream/native changes.
- **Artifact-producing merge gate:** full local suite, generated-header diff clean, benchmark smoke, strict bootstrap/ABI source tests, and existing PR CI green.
- **Phase gate:** five-platform release build/smoke green and Phase 06.1 public bootstrap validation green before Phase 11 is declared merge-ready; no local substitute.

### Wave 0 Gaps

- [ ] Add BigInt Go tests covering exact boundaries, decimal/exponent non-BigInt, strict wrong-type behavior, copied lifetime, nested traversal, and fast materialization.
- [ ] Add Rust/C++ integration coverage for configured parser creation, kind `9`, copied BigInt bytes, and numeric accessor status.
- [ ] Add kernel tests in isolated subprocess/test binaries so process-global lock state cannot leak between cases.
- [ ] Add native diagnostic characterization tests and Go `HasOffset` representation/formatting tests, including known zero and unknown.
- [ ] Add parser option/pool configuration tests, including duplicate last-wins, explicit-default equivalence, invalid/clamped values, the dedicated capacity-limit status, pre-copy rejection, depth boundaries, and no stale error details after a Rust-side rejection.
- [ ] Add two-stage binding fixtures for ABI 1.1, complete ABI 1.2, and ABI 1.2 missing a mandatory symbol.
- [ ] Extend `tests/abi/check_header.py`, `test_check_header.py`, `handle_layout.c`, native smoke/export surface, bootstrap policy tests, and public smoke for ABI `0x00010002` and new symbols.
- [ ] Update every deliberate `NewParserPool` source-break call site in `pool_test.go`, `example_test.go`, and `docs/concurrency.md`. [VERIFIED: codebase grep]

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | This parser library has no authentication boundary. [VERIFIED: codebase] |
| V3 Session Management | No | No session state is handled. [VERIFIED: codebase] |
| V4 Access Control | No | No authorization decisions are made. [VERIFIED: codebase] |
| V5 Input Validation | Yes | simdjson validation plus early capacity/depth limits and strict type/status mapping. [CITED: https://owasp.org/www-project-application-security-verification-standard/] |
| V6 Cryptography | No new control | Artifact checksum/signature handling stays in the existing release/bootstrap pipeline; Phase 11 adds no cryptographic primitive. [VERIFIED: docs/releases.md] |

### Known Threat Patterns for Go/Rust/C++ Parser ABI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Oversized input causes memory exhaustion before policy check | Denial of Service | Rust length gate before arena resize/copy; typed capacity error; boundary tests. [VERIFIED: current gap in registry.rs] |
| Deep nesting exhausts parser/materializer resources | Denial of Service | Immutable DOM depth limit plus retained materializer defense-in-depth guard. [VERIFIED: codebase] |
| Forced unsupported SIMD kernel executes illegal instructions | Denial of Service | Require compiled-name lookup and `supported_by_runtime_system()` before assignment. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md] |
| ABI 1.2 label hides missing functions | Tampering | ABI-first handshake followed by mandatory symbol binding; fail closed. |
| Borrowed BigInt text outlives document | Information disclosure / Tampering | Copy through Rust-owned allocation before returning Go text; free native copy after conversion. [VERIFIED: existing string ownership pattern] |
| Fabricated offset misdirects incident response | Repudiation | Only upstream-success, in-range locations set `HasOffset`; all other cases remain unknown. [CITED: https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md] |
| Kernel setter races parser creation | Tampering / Denial of Service | Native and Go synchronization with irreversible creation lock. |

## Sources

### Primary (HIGH confidence)

- [simdjson v4.6.4 release](https://github.com/simdjson/simdjson/releases/tag/v4.6.4) — tag commit, date, and patch scope.
- [simdjson v4.6.4 DOM BigInt documentation](https://github.com/simdjson/simdjson/blob/v4.6.4/doc/dom.md) — opt-in storage, exact text, strict accessor behavior.
- [simdjson v4.6.4 BigInt tests](https://github.com/simdjson/simdjson/blob/v4.6.4/tests/dom/big_integer_tests.cpp) — positive/negative text, max uint64, and wrong-type assertions.
- [DOM element API](https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/element.h) and [implementation](https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/element-inl.h) — `BIGINT`, `get_bigint`, and `string_view` behavior.
- [DOM parser API](https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/parser.h), [implementation](https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/dom/parser-inl.h), and [base constants](https://github.com/simdjson/simdjson/blob/v4.6.4/include/simdjson/base.h) — capacity/depth defaults, allocation, and clamp behavior.
- [Implementation selection](https://github.com/simdjson/simdjson/blob/v4.6.4/doc/implementation-selection.md) — active implementation, name lookup, and runtime support check.
- [Error-location documentation](https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md) — `current_location` guarantees and documented unavailable cases.
- Repository `CONTEXT.md`, source, tests, `docs/ffi-contract.md`, `docs/releases.md`, and workflow YAML — current cross-language ABI, ownership, loader, bootstrap, and release behavior.

### Secondary (MEDIUM confidence)

- The recommended failure-only On-Demand diagnostic replay is an implementation inference from official `raw_json()` consumption and `current_location()` behavior. Its exact known-input corpus must be characterized in Wave 0 before being promised.

### Tertiary (LOW confidence)

- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — fixed by locked decisions, manifests, git pin, and official v4.6.4 sources.
- Architecture: HIGH — derived from the repository's existing ownership and ABI boundaries.
- BigInt contract: HIGH — directly covered by upstream docs/tests and locked user decisions.
- Capacity/depth: HIGH — current copy order and upstream constants/allocation behavior are explicit.
- Kernel control: HIGH — official process-global selection APIs are explicit; the lock is a project lifecycle rule.
- Error-location coverage: MEDIUM — upstream guarantees known and unavailable cases, but the exact DOM-failure replay corpus requires characterization.
- Release sequencing: HIGH — current runbook, workflow ancestry check, and Phase 06.1 boundary are explicit.

**Research date:** 2026-07-22
**Valid until:** 2026-08-21 (repository state may change; v4.6.4 source claims are immutable)
