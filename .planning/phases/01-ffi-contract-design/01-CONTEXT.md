# Phase 1: FFI Contract Design - Context

**Gathered:** 2026-04-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Commit a reviewable FFI contract for `pure-simdjson` that locks the C ABI shape before implementation: error-code space, handle format, ownership rules, ABI-version handshake, and the iteration/accessor contract that all later phases must follow.

This phase defines how the boundary behaves. It does not add product features beyond the already-scoped `v0.1` DOM API.

</domain>

<decisions>
## Implementation Decisions

### Already locked from project and research context
- **D-01:** `v0.1` stays DOM-based. On-Demand remains deferred to `v0.2`.
- **D-02:** Every `Parse` copies input into Rust-owned padded memory. No zero-copy-in path in `v0.1`.
- **D-03:** Iteration stays cursor/pull only. No Go callbacks across FFI.
- **D-04:** Number access stays split across distinct `int64`, `uint64`, and `float64` accessors with explicit overflow / precision-loss errors.
- **D-05:** Unsupported CPUs fail loudly instead of silently falling back to scalar behavior.
- **D-06:** Generation-stamped handles remain mandatory for safe close/reuse semantics.

### Parser lifecycle and reuse
- **D-07:** `parser_parse` must hard-fail with `ERR_PARSER_BUSY` when the parser already owns a live `Doc`. Re-parse does not implicitly invalidate the prior `Doc`.
- **D-08:** `Doc.Close()` is the explicit release point that clears the parser-busy state. Old parser/doc handles still rely on generation checks to turn stale use into a clean error rather than UB.

### Error and diagnostics contract
- **D-09:** The primary ABI contract is stable `int32` error codes on every export. No struct-by-value returns and no stringly-typed primary failure path.
- **D-10:** The ABI should include a small diagnostics surface in addition to numeric codes: ABI version handshake, active implementation name, and bounded diagnostic helpers for parse context such as last-error text and byte offset.
- **D-11:** Diagnostic helpers are advisory only. Control flow must remain driven by numeric error codes so bindings stay portable across all five targets.

### Handle and value model
- **D-12:** `Parser` and `Doc` remain opaque generation-stamped `u64` handles.
- **D-13:** `Element`, `Array`, and `Object` are lightweight views tied to a `Doc` plus native node/index state. They are not independently allocated/freeable opaque handles.
- **D-14:** Value-view results cross the ABI through out-params / iterator state, not struct-by-value returns.

### Read-side ABI shape
- **D-15:** String access in `v0.1` is copy-out only. The contract should use explicit `ptr + len` results with a matching free path rather than borrowed zero-copy string views.
- **D-16:** Array and object traversal use stateful iterator/cursor entry points driven from Go. `next()` returns key/value or node references through out-params.
- **D-17:** Direct field lookup remains part of the ABI via explicit lookup helpers such as `object_get_field`, returning value-view state through out-params rather than materializing Go-native structures.

### the agent's Discretion
- Exact numeric allocation of the error-code ranges, as long as the space remains stable and documented.
- Final naming of helper functions and iterator structs, as long as they preserve the decisions above.
- Whether bounded diagnostics use thread-local scratch space or explicit copy-out buffers, as long as they remain advisory rather than primary control flow.

</decisions>

<specifics>
## Specific Ideas

- Fast path selected: use the recommended explicit contract choices rather than adding magical auto-invalidation or richer-but-looser ABI behavior.
- Favor explicit failure over silent state mutation when parser/doc lifetime invariants are violated.
- Keep the ABI boring and portable first; richer ergonomics can live in the Go wrapper or a later phase.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` — Phase 1 goal, must-haves, success criteria, and research flag for FFI contract design.
- `.planning/PROJECT.md` — core value, constraints, pure-* family pattern, and already-locked product-level decisions.
- `.planning/REQUIREMENTS.md` — `FFI-01` through `FFI-08` plus `DOC-02`, which define the required ABI rules this phase must lock.

### Research conclusions that already constrain the contract
- `.planning/research/SUMMARY.md` — resolved product-level decisions from 2026-04-14 and the roadmap implications for Phase 1.
- `.planning/research/ARCHITECTURE.md` — recommended handle/value patterns, parser busy model, and Go/Rust boundary shape.
- `.planning/research/PITFALLS.md` — P0 failure modes the contract must prevent, especially parser reuse, panic/exception containment, struct returns, mixed float/int args, invalid UTF-8 timing, numeric precision, and handle double-free.
- `.planning/research/STACK.md` — toolchain and ABI portability constraints around purego, cbindgen, C++ build strategy, and target support.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- No implementation code exists yet. This phase starts from planning and research artifacts only.

### Established Patterns
- The repo is following the amikos `pure-*` pattern: Rust cdylib shim, purego loader, cbindgen-generated header, committed ABI contract, and generation-stamped handle lifecycle.
- Phase 1 must translate those family patterns into a simdjson-specific contract before any source files are written.

### Integration Points
- `include/pure_simdjson.h` will become the contract artifact downstream phases compile against.
- The Rust shim crate and root Go package will both consume the decisions in this file as their ABI source of truth.

</code_context>

<deferred>
## Deferred Ideas

- Zero-copy string views tied to `Doc` lifetime remain a `v0.2` concern, not part of the `v0.1` FFI contract.
- Richer diagnostics and introspection beyond a minimal helper surface should be revisited only if post-MVP demand justifies the added ABI weight.

</deferred>

---

*Phase: 01-ffi-contract-design*
*Context gathered: 2026-04-14*
