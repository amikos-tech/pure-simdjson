# Phase 11: Upstream simdjson refresh, exact big integers, and production diagnostics - Context

**Gathered:** 2026-07-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish the compatibility foundation for v0.2 by upgrading the vendored upstream base from simdjson v4.6.1 to the audited v4.6.4 patch release, preserving oversized integer literals as exact decimal text, and exposing truthful parse locations, diagnostic kernel control, and bounded parser capacity/depth.

This phase does not add DOM navigation, On-Demand extraction, zero-copy views, or streaming. Existing safe copied DOM behavior remains the default. C ABI changes must be deliberate and additive, except for the explicitly accepted Go `NewParserPool` source break below. ABI 1.2 native artifacts and the default bootstrap state must move together so a Phase 11 parser never silently runs with only part of the Phase 11 contract.

</domain>

<decisions>
## Implementation Decisions

### Upstream and BigInt Contract
- **D-01:** Upgrade the reproducibly pinned simdjson source from v4.6.1 to the audited v4.6.4 patch release and rerun the existing Go, Rust, C++, correctness, benchmark, contract, and five-target build gates.
- **D-02:** Enable upstream DOM BigInt preservation. Only integer-syntax literals below `-9223372036854775808` or above `18446744073709551615` become BigInt. The two boundary values remain `TypeInt64` and `TypeUint64`; decimal/exponent forms such as `1.0` and `1e20` remain `TypeFloat64`.
- **D-03:** Append `TypeBigInt` / native `ValueKindBigInt` after the existing kinds (expected numeric value `9`). Do not renumber kinds `0` through `8`, change an existing layout, or change behavior for JSON values the current parser already accepts.
- **D-04:** `Element.GetBigInt() (string, error)` is strict: it accepts only `TypeBigInt` and returns the exact decimal spelling as copied Go text. It does not accept `TypeInt64` or `TypeUint64`, and the package does not acquire an automatic `math/big` dependency.
- **D-05:** Existing numeric getters called on `TypeBigInt` return `ErrWrongType`, matching upstream's typed accessor contract. Update the currently anticipatory comments that mention `ErrPrecisionLoss`; those comments describe an unreachable kind today and do not override the selected Phase 11 behavior.

### Truthful Error Locations
- **D-06:** Preserve the existing `Error.Offset() uint64` signature and add `Error.HasOffset() bool`. This is additive and keeps existing callers compiling.
- **D-07:** A known failure at byte zero is represented by `Offset() == 0` and `HasOffset() == true`. An unknown location is `Offset() == 0` and `HasOffset() == false`.
- **D-08:** Populate an offset only when upstream supplies a concrete, reliable, in-bounds location. Do not run `encoding/json`, a second scanner, or any other estimator to manufacture a location after simdjson fails.
- **D-09:** Failures for which upstream cannot provide a trustworthy location—including applicable stage-one syntax/UTF-8, resource, or internal failures—remain explicitly unknown. Exact message wording is flexible, but the programmatic known/unknown state must never be fabricated.

### Parser Controls and Lifecycle
- **D-10:** Use immutable functional options: `NewParser(opts ...ParserOption) (*Parser, error)`. Existing zero-argument calls keep working; options are normalized and validated before native allocation.
- **D-11:** Deliberately change the pool constructor to `NewParserPool(opts ...ParserOption) (*ParserPool, error)`. The changed return count is an explicit, user-approved source break despite the general preference to preserve existing contracts.
- **D-12:** `Kernel()` and diagnostic-only `SetKernel(name string) error` are package-level because upstream kernel selection is process-wide. An empty name restores automatic selection; explicit names are validated for availability/runtime support. Kernel selection locks once the first parser or parser pool is created.
- **D-13:** Maximum input capacity and maximum depth are immutable per-parser options. Omitted/zero values mean the current defaults; invalid positive values fail instead of being silently clamped. Capacity must be rejected before the input is copied into Rust-owned padded memory.
- **D-14:** A parser pool stores one normalized option set, uses it for every miss, and rejects parsers whose configuration does not match the pool. Pools must not become heterogeneous through `Put`.

### ABI and Bootstrap Compatibility
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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and prior decisions
- `.planning/ROADMAP.md` — Phase 11 goal, requirements, success criteria, dependency, and explicit ABI/bootstrap boundary.
- `.planning/REQUIREMENTS.md` — `UP-01`, `NUM-01`, `NUM-02`, `DIAG-01`, `DIAG-02`, and `LIMIT-01` are the complete Phase 11 requirement set.
- `.planning/PROJECT.md` — copied-DOM default, precision-preservation goal, five-platform requirement, no-cgo constraint, and FFI safety rules.
- `.planning/phases/04-full-typed-accessor-surface/04-CONTEXT.md` — existing typed number boundaries, exact-float policy, and the prior oversized-integer rejection behavior that Phase 11 intentionally supersedes.
- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-CONTEXT.md` — current fast-materializer frame ABI and oversized-literal normalization that must be updated without reintroducing partial frame output.
- `.planning/phases/09.1-bootstrap-artifact-and-abi-alignment-for-default-installs/09.1-CONTEXT.md` — bootstrap/ABI coordination rules, release-readiness guard, and the failure mode caused by a stale artifact pin.

### Public Go and C contracts
- `docs/ffi-contract.md` — normative FFI ownership, error, versioning, panic-safety, and compatibility rules.
- `include/pure_simdjson.h` — generated public ABI, error codes, value kinds, parser exports, implementation-name diagnostics, and current last-error-offset contract.
- `element.go` — public `ElementType` mapping and typed accessor semantics to extend with strict `TypeBigInt` / `GetBigInt` behavior.
- `errors.go` — current `Error.Offset()` normalization, native detail wrapping, and sentinel mapping; add the known-offset state here without changing `Offset()`.
- `parser.go` — existing zero-option parser lifecycle and ABI gate; integration point for immutable parser options.
- `pool.go` — current pool constructor/Get/Put behavior; Phase 11 deliberately changes the constructor and adds homogeneous configuration enforcement.
- `internal/ffi/types.go` — Go ABI constants, error codes, value-kind numbers, and frame layouts that must remain synchronized.
- `internal/ffi/bindings.go` — mandatory purego symbol binding and existing implementation-name/last-error APIs.

### Native implementation and upstream pin
- `build.rs` — vendored simdjson amalgamation build and the reproducible upstream integration point.
- `src/lib.rs` — Rust C exports, ABI constant, `ffi_wrap`, parser entry points, typed accessors, and header-generation source.
- `src/runtime/registry.rs` — parser/document registry, Rust-owned input arena, parser construction, and last-error state.
- `src/native/simdjson_bridge.h` — C++ bridge surface and the materializer frame layout shared across C++, Rust, and Go.
- `src/native/simdjson_bridge.cpp` — upstream DOM parsing, kind/accessor mapping, implementation selection, parser diagnostics, and capacity/depth integration.

### Bootstrap and release guards
- `internal/bootstrap/version.go` — default native artifact pin that must identify an ABI 1.2-capable publication.
- `internal/bootstrap/abi_assertion.go` — compile-time bootstrap/ABI compatibility canary to extend for ABI 1.2.
- `scripts/release/check_bootstrap_abi_state.py` — release-readiness ABI minimum-version policy and source-state validation.
- `scripts/release/check_readiness.sh` — strict pre-tag/pre-publication gate that invokes the ABI/bootstrap checks.
- `.github/workflows/release.yml` — five-platform publication workflow that must build the matching ABI 1.2 artifacts.
- `.github/workflows/public-bootstrap-validation.yml` — fresh-runner R2/GitHub fallback validation for the updated bootstrap pin.

### Authoritative upstream behavior
- `https://github.com/simdjson/simdjson/releases/tag/v4.6.4` — audited upstream patch release selected by the roadmap.
- `https://github.com/simdjson/simdjson/blob/v4.6.4/tests/dom/big_integer_tests.cpp` — upstream DOM BigInt classification and strict accessor behavior.
- `https://github.com/simdjson/simdjson/blob/v4.6.4/doc/dom.md` — upstream DOM BigInt exact-text contract.
- `https://simdjson.github.io/simdjson/md_doc_2basics.html` — upstream error-location guarantees and cases where `current_location()` is unavailable.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pure_simdjson_get_implementation_name_len` / `pure_simdjson_copy_implementation_name` and their purego bindings already provide the read side needed for `Kernel()`; Phase 11 adds the diagnostic override rather than inventing a second kernel-reporting path.
- `pure_simdjson_parser_get_last_error_offset`, `ffi.LastErrorOffsetUnknown`, `wrapParserStatus`, and `Error` already carry parse diagnostic details end to end. The new work is truthful native population plus a separate known bit at the Go boundary.
- The existing copied-string ABI pattern used by `Element.GetString()` can guide exact copied BigInt text ownership without exposing borrowed memory.
- `Parser`, `ParserPool`, and the native parser registry already enforce one live document per parser and provide the locks needed to freeze configuration at construction.
- The ABI canary and release-readiness scripts from Phase 09.1 already encode the rule that wrapper and published native artifacts move together.

### Established Patterns
- Public `ElementType` values numerically mirror native `ffi.ValueKind`; append new kinds and keep all existing values stable.
- Every C export returns an error code with out-parameters, passes through the panic/exception boundary, and is reflected in the generated header.
- Input is copied into a Rust-owned padded arena before native parsing; a maximum-capacity policy must run before this allocation/copy to be a real safety bound.
- The default path is safe copied DOM access. BigInt returns copied text, while zero-copy and borrowed views remain Phase 14 work.
- Native frame layouts are pinned across C++, Rust, and Go. Adding BigInt to the internal materializer must preserve layout parity and exact-text lifetime/copy rules.
- Existing platform, header-diff, correctness-oracle, and benchmark gates are regression fences for the upstream upgrade, not optional cleanup.

### Integration Points
- Update the simdjson gitlink/amalgamation input and build integration, then propagate new behavior through C++ bridge -> Rust registry/exports -> generated C header -> Go FFI types/bindings -> public Go methods.
- Extend parser construction across `parser.go`, `pool.go`, the Rust registry, and the C++ parser wrapper so one immutable option set reaches the native parser before its first parse.
- Extend error detail capture at the native failure site and preserve the known/unknown bit through Rust and Go without changing the existing offset symbol's sentinel contract unnecessarily.
- Update fast materialization, scalar tests, fuzz/oracle expectations, ABI/header audits, bootstrap compatibility checks, and the five-platform release/smoke workflows as one coordinated compatibility change.

</code_context>

<specifics>
## Specific Ideas

- The user chose upstream's strict typed BigInt model: `GetBigInt` is not a generic integer-to-text helper.
- “Without breaking existing contracts” means existing C symbols, layouts, kind numbers, accepted-value behavior, and `Error.Offset()` stay intact. The one deliberate exception is the Go pool constructor, whose return count changes so functional options can be validated immediately.
- The user preferred uniform Phase 11 behavior over legacy native compatibility: ABI 1.1 is rejected rather than supported with feature probing.
- Truthful diagnostics take priority over broad coverage: an explicit unknown is better than an offset inferred by a different parser.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Context gathered: 2026-07-22*
