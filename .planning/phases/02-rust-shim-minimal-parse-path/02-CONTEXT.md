# Phase 2: Rust Shim + Minimal Parse Path - Context

**Gathered:** 2026-04-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Build the first real Rust-native shim for `pure-simdjson`: vendor and compile simdjson through the Rust build, replace the contract-only stubs with a real parser/doc implementation for the minimal happy path, and prove the contract with a native smoke test.

This phase is limited to the smallest honest end-to-end path: `parser_new -> parser_parse -> doc_root -> element_get_int64`, plus the minimum supporting runtime state and diagnostics needed to make that path trustworthy and debuggable. It does not add the full typed accessor surface, Go bindings, bootstrap/download behavior, or the full release matrix.

</domain>

<decisions>
## Implementation Decisions

### Surface depth
- **D-01:** Phase 2 stays narrow: the only fully real data path is the minimal `42` smoke path (`parser_new -> parser_parse -> doc_root -> element_get_int64`).
- **D-02:** Phase 2 should also make a thin diagnostics spine real so native bring-up is inspectable: `element_type` plus parser last-error helpers become real in this phase.
- **D-03:** The remaining accessor and iterator exports stay explicit stubs in Phase 2 rather than pulling Phase 4 scope forward.

### Safety foundation
- **D-04:** Phase 2 must implement the real generation-checked parser/doc safety model now rather than using a temporary thin shim.
- **D-05:** Use a small custom registry that maps directly to the locked packed handle ABI (`slot:u32 | generation:u32`) instead of introducing a helper crate such as `slotmap`.
- **D-06:** Enforce the locked contract semantics immediately: stale handles return `ERR_INVALID_HANDLE`, a parser with a live document returns `ERR_PARSER_BUSY`, and `doc_free` is the release point that clears the busy state.

### Diagnostics and CPU gate
- **D-07:** `get_implementation_name` becomes real in Phase 2 and reports the selected simdjson kernel/implementation.
- **D-08:** `parser_new` must hard-fail with `ERR_CPU_UNSUPPORTED` when simdjson selects the `fallback` implementation.
- **D-09:** Any bypass of the fallback CPU gate is hidden to tests/CI only; Phase 2 does not add a public kernel override or public bypass control.
- **D-10:** Parser last-error text/offset helpers become real for parse-time failures, but remain advisory only. Offset may use a pinned "unknown" sentinel when no precise location exists.

### Cross-platform proof
- **D-11:** The second smoke-test target for Phase 2, alongside `linux/amd64`, is `windows/amd64 (MSVC)`.
- **D-12:** Phase 2 should spend its non-Linux proof budget on the Windows/MSVC path because that is the explicitly identified native build risk; full multi-platform release coverage remains later-phase work.

### the agent's Discretion
- Exact internal registry representation, locking, and ownership bookkeeping, as long as the packed handle format and failure semantics remain identical to the locked contract.
- Exact implementation of parser last-error storage/copy-out, as long as diagnostics remain advisory and stable enough for smoke/debug use.
- Exact hidden test/CI bypass mechanism for fallback-kernel coverage, as long as it is not exposed as a public Phase 2 feature.
- Whether Darwin gets a compile-only check or remains deferred, as long as the Phase 2 smoke gate proves `linux/amd64` and `windows/amd64`.

</decisions>

<specifics>
## Specific Ideas

- Keep Phase 2 honest but narrow: prove the real contract in code, not a temporary shim that will be rewritten in Phase 3.
- Add enough native diagnostics to make simdjson/build/MSVC bring-up debuggable, but stop short of a full diagnostics subsystem.
- Burn down the Windows MSVC + `cc` risk early instead of spending the single non-Linux smoke slot on macOS.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` — Phase 2 goal, must-haves, success criteria, and the explicit smoke-gate/platform expectations.
- `.planning/PROJECT.md` — core value, platform constraints, pure-* family expectations, and product-level non-negotiables.
- `.planning/REQUIREMENTS.md` — `SHIM-01` through `SHIM-07` and the phase traceability that Phase 2 must satisfy.

### Locked contract from prior work
- `.planning/phases/01-ffi-contract-design/01-CONTEXT.md` — Phase 1 decisions that are already locked and must be implemented rather than revisited.
- `docs/ffi-contract.md` — normative lifecycle, ownership, diagnostics, panic/exception, and ABI rules for the shim.
- `include/pure_simdjson.h` — committed public header that the Phase 2 implementation must continue matching byte-for-byte.

### Research that constrains the implementation
- `.planning/research/SUMMARY.md` — resolved scope decisions, phase ordering, and the Phase 2 bring-up expectations.
- `.planning/research/ARCHITECTURE.md` — recommended shim layering, runtime state shape, build approach, and platform-risk notes.
- `.planning/research/PITFALLS.md` — parser lifetime, ownership, panic/exception, and Windows/MSVC failure modes that Phase 2 must address.
- `.planning/research/STACK.md` — recommended `cc`/cbindgen/simdjson toolchain and platform/toolchain guidance.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `src/lib.rs`: already contains the ABI enums/structs, `ffi_wrap`, metadata helpers, `write_out`, `copy_out_bytes`, and the full exported symbol surface to convert from contract stubs into real behavior.
- `Cargo.toml`: already pins the crate as `cdylib` + `staticlib` and sets `panic = "abort"` in dev/release profiles.
- `cbindgen.toml`: existing header-generation path should remain the source for the committed public header.

### Established Patterns
- `Makefile`: `generate-header`, `verify-contract`, and `verify-docs` already define the ABI/header verification loop that Phase 2 must keep green.
- `tests/abi/check_header.py` and `tests/abi/handle_layout.c`: static ABI guards already pin the export surface, handle format, and signature rules.
- `docs/ffi-contract.md` + `include/pure_simdjson.h`: the repo already treats the contract as normative, so Phase 2 should implement against those artifacts rather than inventing a new native shape.

### Integration Points
- `build.rs` needs to become the native build entry point for vendored simdjson compilation and header regeneration.
- `src/lib.rs` is the place where the contract-only stubs become real runtime behavior for parser/doc allocation, parse, root resolution, `element_type`, `element_get_int64`, and parser diagnostics.
- A native smoke harness should exercise the built library directly against the committed header, first on `linux/amd64` and then on `windows/amd64 (MSVC)`.

</code_context>

<deferred>
## Deferred Ideas

- Completing the remaining typed accessor and iteration surface beyond the minimal path — later phases, especially Phase 4.
- Public kernel override / reproducibility controls — later diagnostic work, not a Phase 2 feature.
- Full parser-scoped diagnostics across every export — defer until the broader accessor surface and Go wrapper exist.
- Full five-platform build/release coverage, macOS signing concerns, and broader CI matrix productionization — later phases, especially Phase 6.

</deferred>

---

*Phase: 02-rust-shim-minimal-parse-path*
*Context gathered: 2026-04-15*
