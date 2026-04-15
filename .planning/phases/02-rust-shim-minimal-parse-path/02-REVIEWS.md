---
phase: 2
reviewers:
  - gemini
  - claude
reviewed_at: 2026-04-15T10:26:23Z
plans_reviewed:
  - 02-01-PLAN.md
  - 02-02-PLAN.md
  - 02-03-PLAN.md
prompt_variant: compact_synthesized_from_phase_artifacts
---

# Cross-AI Plan Review — Phase 2

## Review Notes

The initial full-prompt reviewer runs stalled or timed out. The successful Gemini and Claude reviews below were collected from a compact prompt synthesized from the same local artifacts:

- `.planning/PROJECT.md`
- Phase 2 section from `.planning/ROADMAP.md`
- `SHIM-01` through `SHIM-07` from `.planning/REQUIREMENTS.md`
- `.planning/phases/02-rust-shim-minimal-parse-path/02-CONTEXT.md`
- `.planning/phases/02-rust-shim-minimal-parse-path/02-RESEARCH.md`
- `02-01-PLAN.md`, `02-02-PLAN.md`, `02-03-PLAN.md`

## Gemini Review

### Summary

Gemini judged the phase as well-scoped and appropriately incremental. It liked the decision to keep the phase narrowly focused on the `42` smoke path and to isolate the hardest native risks early, but it considered the plan incomplete around low-level FFI safety details.

### Strengths

- Strong scope discipline: the plans resist dragging the full accessor surface into Phase 2.
- Good safety architecture: generation-checked handles and parser-busy semantics are front-loaded instead of deferred.
- Good fallback policy: `ERR_CPU_UNSUPPORTED` plus a hidden test-only bypass is pragmatic.
- Good platform-risk handling: Windows/MSVC is explicitly targeted early instead of being deferred.

### Concerns

- [HIGH] The bridge plan does not make C++ exception containment explicit. If bridge code or simdjson throws across the Rust boundary, the result is undefined behavior.
- [HIGH] Rust panic containment is not explicit in the runtime plan. Panics must not cross `extern "C"` boundaries.
- [MEDIUM] Windows/MSVC CRT linkage details are not explicit. Mismatched CRT settings can break linking or runtime behavior.
- [MEDIUM] The copied+padded input contract is not explicit enough about zero-initialized padding.

### Suggestions

- Make exception safety explicit in the bridge plan: disable simdjson exceptions if appropriate, use `try/catch(...)`, and keep bridge exports `noexcept`.
- Make panic containment explicit in the Rust ABI layer with a mandatory `catch_unwind` wrapper or equivalent mechanical guard.
- Add explicit MSVC CRT alignment handling to the build plan.
- Spell out that the padded allocation tail is zero-initialized.
- Extend the smoke harness or CI verification to prove `doc_free` and `parser_free` behavior and catch memory leaks.

### Risk Assessment

Gemini rated the plan **MEDIUM** risk. Its view was that the architecture is sound, but the execution plan still under-specifies several FFI-boundary safety details that should be made explicit before implementation begins.

## Claude Review

### Summary

Claude judged the phase as well-layered and deliberately narrow, with a clean progression from native build plumbing to runtime safety core to external ABI proof. It strongly endorsed implementing the real generation-checked registry now instead of using a temporary shim, and saw the remaining gaps as fixable clarifications rather than blockers.

### Strengths

- The three-plan decomposition is clean: build plumbing, runtime core, then ABI proof.
- Scope control is strong: only the `42` path is real; later accessor surface stays stubbed.
- The generation-checked registry is the right long-term architectural choice and avoids future refactors.
- Parser-busy semantics align with simdjson's actual parser/doc safety model.
- The native C smoke harness is the right proof mechanism for the committed public header.
- Linux plus Windows as explicit smoke targets is the right early risk burn-down shape.

### Concerns

- [HIGH] Plan 01 does not make MSVC C++ standard handling explicit. Raw `-std=c++17` flags would be wrong on MSVC unless the `cc` crate API is used correctly.
- [HIGH] Plan 02 does not define the parser/doc state machine precisely enough for edge cases such as double-free, freeing a parser while a doc is live, or freeing a doc from the wrong parser context.
- [HIGH] Plan 03 under-specifies the Windows smoke harness compiler/linking strategy.
- [MEDIUM] The plans should reference the exact simdjson amalgamation paths instead of leaving source-file choice implicit.
- [MEDIUM] Panic containment across Rust FFI boundaries should be mandatory in the plan, not assumed.
- [MEDIUM] The padded-buffer requirement should use simdjson's padding constant rather than an implicit hardcoded value.
- [MEDIUM] `element_type` ABI mapping and `get_abi_version` coverage should be called out more explicitly.
- [LOW] Symbol-export verification and thread-safety/registry model details could be sharper.

### Suggestions

- Explicitly rely on `cc::Build::std("c++17")` or equivalent MSVC-safe handling instead of raw compiler flags.
- Pin the exact vendored simdjson source/header paths used by `build.rs`.
- Define the parser lifecycle state machine and add tests for double-free and `parser_free` while a doc is live.
- Make panic containment mechanical at every Rust FFI export.
- Use the simdjson padding constant directly rather than an implicit literal.
- Specify the Windows smoke harness toolchain and library discovery/linking model.
- Add symbol-export verification (`nm`, `dumpbin`, or equivalent) to the build proof.

### Risk Assessment

Claude rated the plan **LOW-MEDIUM** risk. It viewed the architecture and sequencing as correct, with most remaining issues concentrated in native-build specificity and a few under-specified lifecycle edges.

## Consensus Summary

### Agreed Strengths

- The phase is intentionally narrow and resists scope creep. Both reviewers approved keeping the real implementation surface limited to the `42` smoke path plus thin diagnostics.
- The sequencing is sound. Both reviewers liked the build -> runtime -> ABI-proof decomposition because it localizes failures cleanly.
- The real generation-checked registry and parser-busy lifecycle are the right foundations to implement now instead of deferring.
- Targeting Windows/MSVC in Phase 2 is the right early risk-burn-down choice.

### Agreed Concerns

- [HIGH] FFI-boundary safety is not yet explicit enough. Both reviewers want the plans to mechanically specify Rust panic containment and C++ exception containment rather than leaving that to implementation discretion.
- [HIGH] Windows/MSVC build and smoke details are under-specified. Both reviewers want exact compiler/flag/linking behavior called out, especially around C++17 handling and the Windows smoke harness.
- [HIGH] Parser/doc lifecycle edge cases need a clearer state machine. Both reviewers want explicit semantics and tests for stale handles, double-free, parser-busy transitions, and related invalid-handle paths.
- [MEDIUM] The copied+padded input contract needs sharper wording. Both reviewers want the plan to tie padding behavior to simdjson's real requirement and make tail initialization explicit.

### Divergent Views

- Gemini was more worried about MSVC CRT linkage and explicit leak verification in the smoke path.
- Claude was more worried about exact simdjson amalgamation paths, `element_type` ABI mapping, symbol-export verification, and the threading model of the registry.
- Gemini rated the current plan **MEDIUM** risk; Claude rated it **LOW-MEDIUM**. The practical consensus is that the architecture is correct, but a few low-level native and FFI details should be made explicit before execution.

### Recommended Planner Follow-Ups

- Add an explicit FFI-safety requirement to the plans covering Rust `catch_unwind` policy and C++ exception containment/noexcept behavior.
- Make the Windows/MSVC toolchain path concrete in both `build.rs` planning and smoke-harness planning.
- Define the parser/doc lifecycle state machine and add explicit tests for invalid transitions.
- Pin the exact simdjson padding/source-file assumptions in the plan text instead of leaving them implicit.
- Consider adding symbol-export verification and explicit resource-release checks to the smoke plan.
