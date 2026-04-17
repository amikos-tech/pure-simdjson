# Phase 4: Full Typed Accessor Surface - Context

**Gathered:** 2026-04-17
**Status:** Ready for planning

<domain>
## Phase Boundary

Complete the public DOM accessor surface so Go callers can extract every JSON value type and traverse arrays and objects through Go-driven cursor/pull iteration.

This phase fills in the v0.1 read API that Phase 3 intentionally left skeletal. It does not add bootstrap/download behavior, On-Demand semantics, zero-copy string views, or broader convenience-helper families beyond the specific helper chosen below.

</domain>

<decisions>
## Implementation Decisions

### Type inspection surface
- **D-01:** `Element.Type()` should expose the concrete JSON value kind directly, including distinct `Int64`, `Uint64`, and `Float64` kinds rather than collapsing them to a generic `Number`.
- **D-02:** `Element.Type()` should remain total (no error return). When called on a stale or closed element, it returns an explicit invalid sentinel such as `TypeInvalid`.
- **D-03:** `Element.NumberKind()` is not needed in `v0.1` because the exact numeric class is already visible through `Type()`.

### Iterator feel for arrays and objects
- **D-04:** Array and object traversal should use scanner-style iterators: `Next() bool`, then `Value()` / `Key()` for the current item, plus `Err() error` for terminal failure.
- **D-05:** `ObjectIter.Key()` should return a copied Go `string`, not a generic `Element`, because JSON object keys are always strings and `v0.1` string access is copy-out.

### Optional helpers
- **D-06:** Add `Object.GetStringField(name)` in Phase 4.
- **D-07:** Do not add additional convenience helpers beyond `GetStringField(name)` in `v0.1`; broader helper families can be evaluated later.

### Missing vs null vs wrong-type behavior
- **D-08:** `Object.GetField(key)` should distinguish a missing key from a present `null`: missing key returns `ErrElementNotFound`.
- **D-09:** A present `null` field returns a valid `Element`, and callers use `IsNull()` to detect that state.
- **D-10:** Typed getters invoked on `null` or any other wrong concrete type should return `ErrWrongType`.

### the agent's Discretion
- Exact exported names for the public type enum and invalid sentinel, as long as the enum preserves the concrete numeric split above.
- Exact file split for the new Go iterator and accessor code, as long as the public semantics above remain unchanged.
- Whether `Object.GetStringField(name)` is implemented as a thin Go composition over `GetField` + `GetString` or as a dedicated native fast path, as long as the public method exists and preserves the locked error behavior. Planning/research should evaluate the single-call native path first because that is the main reason to include this helper.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` — Phase 4 goal, must-haves, nice-to-haves, and success criteria for the full typed accessor surface.
- `.planning/PROJECT.md` — project-level constraints: DOM-first `v0.1`, copy-in parse ownership, cursor/pull iteration, and no-cgo consumer promise.
- `.planning/REQUIREMENTS.md` — `API-04` through `API-08` and `DOC-03`, which define the required accessor, iteration, and Godoc coverage this phase must complete.

### Locked prior decisions
- `.planning/phases/01-ffi-contract-design/01-CONTEXT.md` — locked ABI choices: split numeric accessors, copy-out strings, cursor iterators, direct field lookup, and explicit lifecycle/error semantics.
- `.planning/phases/02-rust-shim-minimal-parse-path/02-CONTEXT.md` — Phase 2 decision to keep the Phase 4 exports present-but-stubbed until this phase, plus the generation-checked safety model that the new accessors must preserve.
- `.planning/phases/03-go-public-api-purego-happy-path/03-CONTEXT.md` — locked Go surface choices: `Element` as a small value type, local loader behavior, sentinel + wrapped-detail errors, and Phase 3’s deliberate placeholder `Array`/`Object` skeleton.

### Normative contract and ABI shape
- `docs/ffi-contract.md` — normative lifecycle, value-view, iterator, string copy-out, and error-code contract for the full accessor surface.
- `include/pure_simdjson.h` — committed exported function signatures, iterator structs, and value-view layout that the implementation and purego bindings must match.

### Research guidance
- `.planning/research/SUMMARY.md` — project-level conclusions that Phase 4 should stay DOM-only, keep cursor/pull iteration, and preserve exact numeric access.
- `.planning/research/PITFALLS.md` — failure modes around parser/doc lifetime, mixed FFI signatures, callback avoidance, null/type handling, and numeric conversion behavior.
- `.planning/research/ARCHITECTURE.md` — recommended Go/Rust/C++ layering, `Element`/`Array`/`Object` shape, and purego binding pattern for extending the existing public surface.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `element.go` — already defines `Element`, `Array`, `Object`, `GetInt64()`, `AsArray()`, and `AsObject()`. Phase 4 should extend this shape rather than redesign it.
- `doc.go` — already caches the root `ffi.ValueView` and enforces `Doc.Close()` semantics. New element accessors and iterators should continue to rely on the live-doc checks already centralized here.
- `errors.go` — already provides the sentinel-plus-detail error model (`ErrWrongType`, `ErrElementNotFound`, `ErrNumberOutOfRange`, `ErrPrecisionLoss`, `ErrClosed`) that the new accessors should keep using.
- `internal/ffi/types.go` — already mirrors `ValueView` and the concrete native `ValueKind` split, including distinct `ValueKindInt64`, `ValueKindUint64`, and `ValueKindFloat64`.
- `internal/ffi/bindings.go` — already binds the current ABI with `purego.RegisterFunc`; Phase 4 can extend this file with the additional accessor and iterator symbols rather than inventing a second binding layer.
- `src/lib.rs` — already exports the full Phase 4 symbol surface, but most of it is still stubbed. The implementation work belongs here and in the runtime helpers it calls.

### Established Patterns
- The Go wrapper favors small value wrappers over pointer-heavy abstractions: `Element` is copied by value and tied to a live `Doc`.
- Lifecycle misuse is surfaced explicitly (`ErrClosed`, `ErrParserBusy`, `ErrInvalidHandle`) instead of being silently repaired.
- FFI calls return status codes plus out-params; Go translates them into sentinel-matchable errors with optional native detail.
- Iteration has already been locked at the ABI level as Go-driven cursor/pull, not native callbacks.

### Integration Points
- `internal/ffi/bindings.go` and `internal/ffi/types.go` are the purego expansion points for the new symbols and iterator state mirrors.
- `src/lib.rs` and its runtime helpers are where `element_get_uint64`, `element_get_float64`, `element_get_string`, `element_get_bool`, `element_is_null`, `array_iter_*`, `object_iter_*`, and `object_get_field` become real.
- `phase3_followup_test.go`, `parser_test.go`, and new accessor/iterator-focused Go tests should become the main correctness harness for the public API semantics locked here.
- Godoc completion for this phase should live on the exported Go types and methods that Phase 3 introduced as placeholders.

</code_context>

<specifics>
## Specific Ideas

- `Type()` should be the one-stop classification API for callers; exact numeric kind should be visible directly instead of hidden behind a second classifier.
- Iterator ergonomics should feel like a Go scanner: tight `for iter.Next()` loops with a final `Err()` check.
- `Object.GetStringField(name)` is the only helper promoted because object-heavy selective extraction is a likely hot path; if a dedicated native fast path materially reduces FFI round-trips, planning should prefer it.
- The API should preserve a sharp semantic distinction between missing fields and present `null` values.

</specifics>

<deferred>
## Deferred Ideas

- `Element.NumberKind()` — deferred because `Type()` already exposes exact numeric kinds in `v0.1`; add later only if real caller ergonomics justify the extra API.
- Broader typed field helper families (`GetInt64Field`, `GetBoolField`, optional/null-aware typed helpers, etc.) — deferred until there is evidence that more than `GetStringField(name)` is worth carrying in the stable surface.

</deferred>

---

*Phase: 04-full-typed-accessor-surface*
*Context gathered: 2026-04-17*
