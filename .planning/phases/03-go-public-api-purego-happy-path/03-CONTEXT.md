# Phase 3: Go Public API + purego Happy Path - Context

**Gathered:** 2026-04-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Ship the first public Go wrapper for the existing Rust ABI so a consumer can do:

`NewParser() -> Parse(data) -> doc.Root().GetInt64()`

This phase defines the first stable Go-facing package shape, lifecycle behavior, local native-library loading for development and smoke tests, typed error mapping, and `ParserPool`.

It does not add bootstrap/download behavior from Phase 5, nor the full accessor and iteration surface from Phase 4.

</domain>

<decisions>
## Implementation Decisions

### Public API shape
- **D-01:** `Doc` should expose the root as `Root() Element`, not `Root() (Element, error)` and not a more verbose alternative.
- **D-02:** `Element` is a small value type, not a pointer wrapper and not an interface-based abstraction.
- **D-03:** Phase 3 should publicly declare `Parser`, `Doc`, `Element`, `Array`, `Object`, and `ParserPool` now, even though only the happy-path methods are implemented in this phase.
- **D-04:** The happy path to preserve in examples, tests, and docs is `doc.Root().GetInt64()`.
- **D-05:** `Array` and `Object` are part of the public type skeleton in Phase 3, but their real traversal and field-access behavior remains Phase 4 work.

### Local library loading
- **D-06:** Phase 3 loader behavior is local-only: first honor `PURE_SIMDJSON_LIB_PATH`, then try deterministic repo-local build outputs, then fail with an error that includes the attempted paths.
- **D-07:** Repo-local search order is:
  1. current-platform `target/release`
  2. current-platform `target/debug`
  3. `target/<triple>/release`
  4. `target/<triple>/debug`
- **D-08:** Library discovery must stay deterministic. Do not use a broad recursive scan under `target/`.
- **D-09:** The loader should always resolve and load a concrete full path, not a bare library filename.
- **D-10:** The purego binding path in this phase must use `RegisterFunc`, never `SyscallN`.

### Error and diagnostics surface
- **D-11:** Public sentinel errors remain the main matching surface (`errors.Is` / `errors.As`).
- **D-12:** Returned errors should wrap native detail when available instead of collapsing everything to sentinel-only failures.
- **D-13:** The package should expose one public structured error type carrying native error detail and the wrapped sentinel. The required detail is: code, offset, message, and wrapped sentinel error.
- **D-14:** ABI mismatch and load failures should use the same wrapped-error model rather than a special-case error style.
- **D-15:** Native parse/accessor diagnostics exposed by the Rust shim are worth preserving in Phase 3 instead of discarding at the Go boundary.

### Lifecycle and pool ergonomics
- **D-16:** `Parser.Close()` and `Doc.Close()` are idempotent.
- **D-17:** Any method invoked after close must return `ErrClosed`.
- **D-18:** `ParserPool` should enforce lifecycle correctness rather than silently repairing misuse.
- **D-19:** `ParserPool.Put` must reject parsers that still own a live document.
- **D-20:** Finalizers are warning-only safety nets in test builds. They are silent in production and are never the primary cleanup path.
- **D-21:** The documented concurrency model remains goroutine-per-parser; `ParserPool` exists to support that model, not to make one parser concurrently shareable.

### Carried forward from earlier phases
- **D-22:** The Go wrapper must preserve the locked parser/doc invariant from the ABI: one parser may own at most one live doc at a time, and re-parse must fail with `ErrParserBusy` until the doc is released.
- **D-23:** The Go wrapper must preserve explicit lifecycle semantics from the ABI: releasing the doc is what clears parser busy state; no auto-invalidation of prior docs.
- **D-24:** `v0.1` stays DOM-based; On-Demand behavior remains out of scope for this phase.
- **D-25:** All parsing still copies input into Rust-owned padded storage; Phase 3 must not imply any zero-copy input behavior.

### the agent's Discretion
- Exact Go file split across root package files (`purejson.go`, `parser.go`, `doc.go`, `element.go`, `errors.go`, `library*.go`) as long as the public semantics above are preserved.
- Exact naming and representation of any exported error-code helper type that backs the structured error.
- Exact `ParserPool` method signatures, as long as misuse is surfaced deterministically and not silently repaired.
- Exact wording and logging mechanism for test-build finalizer warnings.

</decisions>

<specifics>
## Specific Ideas

- Keep the first public example intentionally minimal: `p, err := purejson.NewParser(); doc, err := p.Parse(data); v, err := doc.Root().GetInt64()`.
- The public error model should feel idiomatic in Go: sentinel matching for control flow, structured detail for debugging and tests.
- Phase 3 loader behavior is for local development and smoke tests only. Download/bootstrap orchestration belongs to Phase 5 and should not be pulled forward.
- The public package shape should stabilize early so Phase 4 fills in behavior rather than introducing core API churn.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` — Phase 3 goal, must-haves, success criteria, and boundaries against Phases 4 and 5.
- `.planning/PROJECT.md` — core value, package-level constraints, pure-* family expectations, and current-state notes after Phase 2.
- `.planning/REQUIREMENTS.md` — `API-01`, `API-02`, `API-03`, `API-09`, `API-10`, `API-11`, `API-12`, `DOC-03` (partial), and `DOC-04`.

### Locked prior decisions
- `.planning/phases/01-ffi-contract-design/01-CONTEXT.md` — locked ABI and lifecycle decisions the Go wrapper must preserve.
- `.planning/phases/02-rust-shim-minimal-parse-path/02-CONTEXT.md` — narrowed scope and native implementation choices already made for the Rust shim.
- `docs/ffi-contract.md` — normative lifecycle, diagnostics, handle, and ownership rules the Go API must wrap without changing semantics.
- `include/pure_simdjson.h` — concrete exported symbol surface and struct layout the purego bindings must mirror.

### Research and architecture guidance
- `.planning/research/SUMMARY.md` — project-level decisions promoting `ParserPool`, DOM-first v0.1, and explicit lifecycle handling.
- `.planning/research/ARCHITECTURE.md` — recommended Go package structure, loader split, `ParserPool`, finalizer safety-net, and purego integration shape.
- `.planning/research/PITFALLS.md` — lifecycle, finalizer, DLL-loading, purego ABI, and error-surface failure modes Phase 3 must avoid.
- `.planning/research/STACK.md` — purego v0.10.0 expectations and loader/platform constraints relevant to the Go binding.

### Existing implementation anchors
- `src/lib.rs` — current real Rust exports: metadata helpers, parser/doc lifecycle, `doc_root`, `element_type`, and `element_get_int64`.
- `src/runtime/registry.rs` — native parser/doc registry semantics, busy-state handling, and generation-checked lifecycle behavior.
- `tests/rust_shim_minimal.rs` — current executable expectations for parser busy state, diagnostics, invalid-handle behavior, and the happy path.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `src/lib.rs`: already exposes the stable C ABI shape and the only real native path Phase 3 should bind first.
- `src/runtime/registry.rs`: already enforces parser busy state, generation-checked handles, explicit `doc_free` release semantics, and last-error storage.
- `include/pure_simdjson.h`: already defines the concrete symbol names and C shapes that purego must bind.
- `tests/rust_shim_minimal.rs`: already codifies the lifecycle and diagnostics behaviors the Go wrapper should mirror in its own tests.

### Established Patterns
- The native boundary is explicit and conservative: numeric error codes, out-params, no implicit parser reuse, and one live doc per parser.
- The repo has no Go wrapper yet, so Phase 3 is defining the first public Go package rather than conforming to an existing Go layout.
- The current native implementation is intentionally narrow: only metadata helpers, parser/doc lifecycle, `doc_root`, `element_type`, and `element_get_int64` are real. The remaining DOM surface is still stubbed and belongs to later phases.

### Integration Points
- The new Go package must bind directly to the symbols already exported in `include/pure_simdjson.h`.
- The loader needs to bridge local Rust build outputs into the Go happy path before Phase 5 introduces binary bootstrap/download.
- The Phase 3 tests should align with native expectations already proven in `tests/rust_shim_minimal.rs`, especially parser busy state, close semantics, and diagnostics preservation.

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within the Phase 3 boundary.

</deferred>

---

*Phase: 03-go-public-api-purego-happy-path*
*Context gathered: 2026-04-15*
