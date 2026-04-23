# Phase 8: Low-overhead DOM traversal ABI and specialized Go any materializer - Context

**Gathered:** 2026-04-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace the current accessor-shaped Tier 1 materialization path with a lower-overhead DOM traversal/materialization substrate that preserves correctness while removing avoidable per-node FFI, string handoff, and Go tree-building cost.

This phase remains DOM-era v0.1 work. It does not add On-Demand APIs, path-query public APIs, public benchmark repositioning, or a release decision. Phase 9 consumes the post-Phase-8 evidence and owns benchmark gate recalibration plus any public benchmark-story update.

</domain>

<decisions>
## Implementation Decisions

### Traversal ABI Shape
- **D-01:** Use a bulk traversal/frame-style fast path as the core direction: native should walk a document or subtree once and return compact traversal data that Go can consume without a per-node FFI call.
- **D-02:** Treat the "expose the tape" idea as a strong research/planning vector, not a hard implementation mandate. Planning should explicitly compare a tape-like internal view over Rust-owned buffers against any separate frame-stream design before locking tasks.
- **D-03:** The new traversal/materialization ABI is internal first. It may be used by the Go wrapper and benchmark path, but it is not promised as a public C ABI surface in Phase 8.
- **D-04:** Support both whole-document and subtree materialization as the intended design envelope, while allowing the planner to choose the first implementation slice if doing both at once is too large.
- **D-05:** Preserve the current accessor ABI untouched and add the low-overhead path in parallel. Existing public DOM accessors, iterators, and error semantics remain stable.

### String And Key Handoff
- **D-06:** Object keys should be represented as slices or offsets inside the internal frame/tape view where possible. Go copies keys only when building the final `map[string]any`.
- **D-07:** String values are copied into Go only when materializing a string. Internal traversal may view Rust-owned bytes while the owning `Doc` is alive.
- **D-08:** Borrowed Rust memory must not escape into public Go values. User-visible strings, maps, and slices own Go memory by the time they escape the internal materializer.
- **D-09:** Lifetime safety for internal borrowed views should rely on explicit `Doc`/materializer ownership, `runtime.KeepAlive`, and existing `Close`/finalizer discipline. Debug live-view tracking is not required unless planning finds a cheap, useful way to add it.

### Go any Builder Semantics
- **D-10:** The specialized builder should match current accessor numeric semantics: preserve `int64`, `uint64`, and `float64` distinctions, and surface existing precision/range errors rather than collapsing everything to `float64`.
- **D-11:** Full `map[string]any` materialization uses ordinary Go map assignment semantics for duplicate keys, so the last duplicate key wins. This intentionally differs from `Object.GetField`, which remains a first-match DOM lookup.
- **D-12:** Arrays and objects should use exact or near-exact preallocation from traversal metadata instead of the current fixed small capacities.
- **D-13:** Materialization fails fast with existing typed errors. Wrong type, range, precision, invalid handle, and closed document cases should map to current public error behavior.

### Exposure And Proof Bar
- **D-14:** Keep the specialized materializer internal/benchmark-facing first. Do not add `Element.Interface()`, `Doc.Interface()`, or another public convenience API until the path is measured and validated.
- **D-15:** Phase 8 closeout must prove correctness plus benchmark delta: parity/oracle tests plus Tier 1 diagnostic evidence showing materialization improvement.
- **D-16:** The benchmark target is improvement over the Phase 7 baseline in both materialize-only and full Tier 1 paths. Phase 8 does not require beating `encoding/json + any` before closeout.
- **D-17:** Phase 8 documentation should be internal docs and benchmark notes only. Public README/result repositioning waits for Phase 9.

### the agent's Discretion
- Exact frame/tape struct names and layouts, as long as they preserve the decisions above and remain internal for Phase 8.
- Whether the first implementation slice targets whole-document materialization, subtree materialization, or both, based on risk and testability.
- Whether to add debug-only borrowed-view tracking, if it proves cheap and helpful.
- Exact benchmark command grouping and artifact naming, as long as Phase 7 baseline comparison remains clear.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and prior decisions
- `.planning/ROADMAP.md` - Phase 8 boundary, dependency on Phase 7, Phase 9 follow-up boundary, and research flag.
- `.planning/PROJECT.md` - project constraints: DOM-first v0.1, no cgo consumer build, purego runtime loading, copy-in parse ownership, and honest benchmark positioning.
- `.planning/STATE.md` - current focus, Phase 7 handoff, and locked benchmark-positioning decisions.
- `.planning/phases/01-ffi-contract-design/01-CONTEXT.md` - locked ABI choices: DOM-only v0.1, cursor/pull iteration, no callbacks, copy-out string access, and error-code/out-param discipline.
- `.planning/phases/04-full-typed-accessor-surface/04-CONTEXT.md` - current DOM accessor semantics, string copy-out behavior, iterator model, and `GetStringField`/`GetField` guidance.
- `.planning/phases/07-benchmarks-v0.1-release/07-CONTEXT.md` - benchmark tier definitions and the handoff that routes Tier 1 materialization work to Phase 8.
- `.planning/phases/07-benchmarks-v0.1-release/07-LEARNINGS.md` - materialization-dominates-parse finding, benchmark patterns, native allocator reporting lessons, and Phase 9 boundary.

### Contracts and benchmark evidence
- `docs/ffi-contract.md` - normative public ABI invariants, lifecycle, view/iterator model, copy-out strings, and native telemetry contract.
- `include/pure_simdjson.h` - generated public header that must not be broken by the internal fast path.
- `docs/benchmarks.md` - current Tier 1/2/3 definitions, diagnostics family, native allocation metrics, and interpretation rules.
- `docs/benchmarks/results-v0.1.1.md` - Phase 7 baseline numbers and diagnostic split used as the comparison point.
- `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt` - raw Phase 7 diagnostic evidence for parse-only, materialize-only, and full Tier 1 rows.
- `testdata/benchmark-results/v0.1.1/phase7.bench.txt` - raw Phase 7 Tier 1/2/3 benchmark evidence.

### Tape/frame research inputs
- `third_party/simdjson/doc/tape.md` - upstream tape representation reference to inspect before choosing a tape-like internal view.
- `third_party/simdjson/doc/performance.md` - upstream performance context for DOM traversal and parsing behavior.
- `.planning/research/SUMMARY.md` - project-level research summary and performance/benchmark framing.
- `.planning/research/PITFALLS.md` - FFI, lifetime, UTF-8, numeric precision, and benchmark fairness pitfalls.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `benchmark_comparators_test.go` - current `benchmarkMaterializePureElement` recursively builds `map[string]any` / `[]any` through public accessors; this is the path Phase 8 should replace or bypass for Tier 1.
- `benchmark_diagnostics_test.go` - already isolates full, parse-only, and materialize-only Tier 1 cuts; reuse this shape to prove the Phase 8 delta.
- `benchmark_native_alloc_test.go` and native telemetry bindings - existing native allocation metrics should remain visible beside Go `-benchmem` data.
- `internal/ffi/bindings.go` and `internal/ffi/types.go` - established purego binding style, Go mirror types, `runtime.KeepAlive` discipline, and error-code wrapping patterns.
- `src/lib.rs`, `src/runtime/registry.rs`, and `src/native/simdjson_bridge.cpp` - current export, registry, and C++ bridge layers where any internal fast path will connect.

### Established Patterns
- Public FFI exports return numeric status codes with pointer out-params. Avoid struct-by-value returns on public ABI surfaces.
- Public DOM view and iterator structs are lightweight document-tied state, not independent handles.
- Existing strings are copy-out through native allocation plus `bytes_free`; Phase 8 should avoid this per-string allocation/free in the internal fast path while preserving public copy-out ownership.
- Current object iteration returns a key view and value view, then Go calls `ElementGetString` for every key. This is a likely hot cost to remove.
- Current recursive materialization preallocates arrays and maps with fixed capacity `8`; Phase 8 should use traversal metadata to size containers better.
- The registry currently tracks descendant indices and iterator leases for validation. A bulk traversal or tape-like path may need a different validation strategy that remains safe but avoids per-node bookkeeping cost.

### Integration Points
- Go benchmark comparator path: `benchmarkMaterializePureSimdjsonWithParser` and `benchmarkMaterializePureElement`.
- Public wrapper path: `Parser.Parse`, `Doc.Root`, `Element`, `Array`, `Object`, `ArrayIter`, and `ObjectIter` must keep existing behavior.
- Native ABI path: new internal exports, if any, must be added through Rust `ffi_wrap`, Go purego binding registration, generated header policy decisions, and validation tests.
- Benchmark evidence path: Phase 8 should compare against `docs/benchmarks/results-v0.1.1.md` and raw `testdata/benchmark-results/v0.1.1` files.

</code_context>

<specifics>
## Specific Ideas

- The user provided exploratory research favoring a tape-like internal view: Rust owns the parse arena; Go receives read-only slices over tape/string buffers; Go walks without per-node FFI; strings/keys copy to Go only when public materialized values are constructed; lifetime is managed by the owning `Doc`/handle.
- Treat that research as an exploratory vector, not gospel. It should guide research and planning toward the lowest-crossing-count design, while still requiring verification against simdjson's actual stable/unstable tape access and this repo's safety constraints.
- Compare against the data-model shape of `minio/simdjson-go` before designing the Go-side walker. The relevant idea is lazy traversal over tape/string/message data, not adopting its API wholesale.
- Arrow's C Data Interface was called out as a useful reference for release callbacks, lifetime ownership, and cross-language zero-copy discipline. Use it as design inspiration if the internal ABI exposes borrowed buffers.
- Phase 8 should bias toward one parse/traversal handoff and zero per-node FFI calls. Occasional helper calls are acceptable only if planning proves they do not reintroduce the measured bottleneck.

</specifics>

<deferred>
## Deferred Ideas

- Public `Element.Interface()` / `Doc.Interface()` convenience APIs are deferred until after the internal materializer is measured.
- Borrowed-memory public/unsafe APIs are out of Phase 8. Public Go values must own Go memory.
- JSONPointer/path lookup helpers are not in Phase 8 unless planning proves they are required for the materializer. They may belong to later selective traversal or v0.2 work.
- Public README benchmark repositioning, headline claim changes, and any release decision are Phase 9 work.

</deferred>

---

*Phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materializer*
*Context gathered: 2026-04-23*
