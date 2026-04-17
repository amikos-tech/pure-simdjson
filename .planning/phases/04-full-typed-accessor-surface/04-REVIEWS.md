---
phase: 4
reviewers: [gemini, claude]
reviewed_at: 2026-04-17T07:11:05Z
plans_reviewed:
  - 04-01-PLAN.md
  - 04-02-PLAN.md
  - 04-03-PLAN.md
  - 04-04-PLAN.md
  - 04-05-PLAN.md
---

# Cross-AI Plan Review — Phase 4

## Gemini Review

### Summary

The Phase 4 plans provide a highly structured and sound approach to completing the DOM accessor API and array/object traversal. The breakdown into native scalar, Go scalar, native iterator, Go iterator, and a final verification/documentation plan is logical and perfectly aligns with the locked ABI constraints. The explicit focus on generalizing the Rust runtime's view validation to handle descendant views before building iterators is an excellent architectural insight that preempts significant memory-safety vulnerabilities.

### Strengths

- **Phase Decomposition**: Splitting the native substrate and Go wrapper work into separate interleaved plans allows for isolated, easily reviewable milestones while hiding raw FFI complexity from the public Go API.
- **View Validation Architecture**: Identifying the gap in root-only view validation and forcing descendant-safe validation in Plan 01 prevents a massive class of dangling-pointer bugs.
- **Error Handling Continuity**: Strong adherence to the established sentinel error model (`wrapStatus`), correctly mapping numeric overflow and precision loss directly from the native bridge.
- **ABI Contract Discipline**: Recognizing that `GetStringField` should be Go composition avoids reopening the locked C ABI, maintaining the Phase 1 contract integrity.

### Concerns

- **HIGH: Iterator Native Allocation Risk (Memory Leak)**
  The committed C ABI in `include/pure_simdjson.h` does not provide `pure_simdjson_array_iter_free` or `pure_simdjson_object_iter_free`. Plan 03 instructs the C++ bridge to store iterator progress in `state0/state1/index/tag`, but it must explicitly forbid native heap allocations (e.g., `new simdjson::dom::array::iterator`) within `psimdjson_array_iter_new`. If the C++ bridge allocates memory on the heap for iterators, it will unavoidably leak because the Go side has no FFI hook to free it.
- **MEDIUM: String Key Extraction in `ObjectIter.Next()`**
  Plan 04 specifies that `ObjectIter.Key()` returns a Go string cached during `Next()`. However, `object_iter_next` returns a `pure_simdjson_value_view_t` for the key, not a string buffer. The plan omits the critical detail that `Next()` must internally invoke the hidden `ElementGetString` binding (and handle the subsequent `BytesFree`) on the key view to perform this conversion.
- **MEDIUM: Safe Ownership Transfer in `bytes_free`**
  Plan 01 correctly introduces `bytes_free`, but does not explicitly mandate that the Go FFI layer (`ElementGetString`) use a `defer` block for `BytesFree`. Without a `defer`, any Go panic during the byte-slice-to-string conversion will result in a leaked native allocation.
- **LOW: Missing Encoder Helper Specification**
  Plan 03 references "descendant-safe view-encoding helpers from Plan 01", but Plan 01 Task 1 only explicitly instructs creating *validation* helpers, not *encoding* helpers.

### Suggestions

1. **Enforce Inline Iterator State (Plan 03)**: Update Task 1 to explicitly state that the C++ bridge must pack `simdjson` iterator state inline into the `uint64_t` fields of the transport structs, strictly forbidding any `new`/`malloc` allocation since there is no ABI destructor.
2. **Clarify Key Extraction (Plan 04)**: In Task 1, explicitly mention that `ObjectIter.Next()` must call the hidden `ElementGetString` binding on the returned key view, copy it to a Go string, free the native bytes, and cache the result for `Key()`.
3. **Require Defers for Native Memory (Plan 01)**: In Task 2, specify that `ElementGetString` must use `defer bindings.BytesFree(...)` immediately upon receiving a successful string buffer from the native call.
4. **Add Encoding Helper (Plan 01)**: Explicitly add the creation of a descendant view-encoding helper (e.g., packing a raw C++ element pointer into a `value_view_t` with a descendant tag) to Task 1 to fulfill Plan 03's prerequisite.

### Risk Assessment

**MEDIUM**. The architectural design is exceptionally strong, but the lack of an iterator `free` function in the ABI combined with the potential for C++ heap allocation in iterators poses a high risk of silent memory leaks. Furthermore, manual memory management across the FFI boundary for strings requires rigid `defer` safety. Addressing the suggested refinements in the plans will easily mitigate these issues and downgrade the execution risk to LOW.

## Claude Review

### Summary

This is a well-structured, pattern-aligned decomposition of Phase 4 into five plans with a clean dependency graph (01 → {02, 03} → 04 → 05) that correctly identifies descendant-safe view validation as the upstream blocker before any iterator or field-lookup work can land. The plans honor the Phase 3 pattern library (small value wrappers, sentinel+detail errors, `runtime.KeepAlive` discipline), defer `GetStringField` to Go composition to avoid reopening the ABI, and preserve Godoc/DOC-03 as a first-class exit gate. The main weaknesses are (a) the descendant-view encoding scheme is under-specified — the single highest-risk design decision in the phase is left to the executor without a locked representation, (b) several task-level verification commands depend on test files created in a *later* task inside the same plan, creating internal sequencing gaps, and (c) a roadmap success criterion (`GetInt64()` on `1e20` → `ErrNumberOutOfRange`) quoted verbatim into Plan 05 conflicts with simdjson's native `INCORRECT_TYPE` behavior for float-kind roots and will bite at verification time.

### Strengths

- **Correct substrate ordering.** Plan 01 lifts `with_validated_view` from root-only to descendant-safe *before* iterators land. This matches Research Risk 1 precisely.
- **`GetStringField` as composition, not ABI.** Plan 04 Task 1 explicitly forbids speculative fast-path work and requires `GetField(name)` + `GetString()` composition. Directly addresses Research Risk 4 and avoids reopening a locked C header.
- **String-ownership boundary is called out.** Plan 01 artifacts include `BytesFree`, Plan 01 Task 2 requires `ElementGetString` to always release the native allocation, and the threat model logs this as `T-04-01-02`. Good mitigation of the first successful-path cleanup contract.
- **Wave parallelization is correct.** Plans 02 and 03 both depend only on Plan 01, so the `wave: 2` tag is accurate; Plan 04 correctly requires both (needs scalar accessors + iterator transport).
- **Pattern reuse discipline.** Every plan extends `element.go`/`bindings.go`/`registry.rs` rather than inventing parallel layers. 04-PATTERNS.md's analog map shows through in task actions (`Element.GetInt64()` → `GetUint64()/GetString()/etc.` shape).
- **Closed-state semantics are test-locked.** Plan 02 Task 2 names `TestIsNull` and closed-doc coverage; Plan 04 Task 2 names `TestIteratorAfterDocClose`. Not left as "add some tests."
- **Source-doc regex gates DOC-03.** Plan 05 acceptance criteria use `rg '^// ElementType|...'` across `element.go` + `iterator.go`, preventing the "code works, docs missing" exit scenario called out in Research Risk 5.

### Concerns

#### HIGH

- **HIGH — Descendant-view storage scheme is not locked.** The single most impactful Phase 4 design decision (how a child `simdjson::dom::element` is encoded into `pure_simdjson_value_view_t.state0/state1`) is delegated to the executor. Options include: (a) encode tape offset + kind inline, (b) heap-allocate `psimdjson_element` per descendant and track in the doc's arena, (c) store the 16-byte `simdjson::dom::element` value directly across `state0+state1`. Each has different lifetime, allocation, and bridge-signature consequences (option (c) requires the bridge to take an element *by value*, conflicting with the current `const psimdjson_element *` pattern). The acceptance criteria only check for symbol presence — an executor can pick (b) with a per-element heap alloc and ship a working but allocation-heavy design, or pick (a) without accounting for the mixed-kind tag namespace. Lock this in PLAN 01's `must_haves.truths` before execution.
- **HIGH — `GetInt64()` on `1e20` conflict with simdjson native behavior.** The roadmap criterion (quoted into Plan 05 Task 2 verbatim) says `GetInt64()` on JSON `1e20` must return `ErrNumberOutOfRange`. But `1e20` parses as a FLOAT64 element; simdjson's `get_int64()` on a float element returns `INCORRECT_TYPE`, which the existing `map_error` in `simdjson_bridge.cpp:56` maps to `ERR_WRONG_TYPE`. Either (a) the bridge needs a custom path that detects "looks numeric but out of int64 range" and overrides the status code, or (b) the roadmap's criterion is wrong and should be pinned to a value like `9223372036854775808` (which parses as UINT64 and *does* yield `NUMBER_OUT_OF_RANGE` via `get_int64`). Plan 05 Task 2 will fail verification as written unless this is resolved pre-execution.
- **HIGH — Plan 02 Task 1 and Plan 04 Task 1 have verify commands that depend on test files from the next task.** Plan 02 Task 1's `<verify>` is `go test -run 'Test(ElementTypeClassification|GetUint64|...)'` but `element_scalar_test.go` is created by Task 2. Same for Plan 04 Task 1 (`TestArrayIterOrder`) vs Task 2 (`iterator_test.go`). Under the gsd-executor's task-by-task acceptance_criteria gate, Task 1 *cannot* pass verification atomically — if you're using TDD-style commits, this also blocks clean rollback. Fix: Task 1 verifies with `go build ./... && go vet ./...`; Task 2 owns the semantic test run. (Plans 01 and 03 are clean because Task 1 verify is `cargo test --lib`.)

#### MEDIUM

- **MEDIUM — Object iteration per-entry FFI cost is not modeled.** Object iteration via `object_iter_next` returns keys as `pure_simdjson_value_view_t`. To expose `Key() string`, Plan 04 Task 1 requires `Next()` to populate the Go string, which (absent a native fast path) means each `Next()` costs: `object_iter_next` + `element_get_string` + `bytes_free` = 3 FFI round-trips + 1 native alloc per entry. For a 100-field object that's 400 cross-boundary operations. This contradicts PROJECT.md's perf story and the "Object iteration is probably the first bottleneck" architecture note. At minimum, Plan 03 should consider whether `object_iter_next` should return the key as an inline `(ptr, len)` pair on the fast path (the ABI already has flexibility — `state0/state1` of the key view could carry a direct bytes pointer if the design treats key-strings as arena-interned).
- **MEDIUM — Iterator `tag` namespace is not pre-assigned.** Plan 03 says "reject unknown tag values" but doesn't declare what values are "known". If Plan 03 picks `0xA2 = ARRAY_ITER_TAG` and Plan 04's tests later check iterator state externally, they may collide. Lock tag constants (`ARRAY_ITER_TAG`, `OBJECT_ITER_TAG`, `DESCENDANT_VIEW_TAG` vs existing `ROOT_VIEW_TAG`) in Plan 01 alongside `ROOT_VIEW_TAG`.
- **MEDIUM — `-race` only runs in Plan 05.** A data race in Plan 03's purego wrappers won't surface until Plan 05's close-out. Given that the phase introduces iterator state held across Go goroutines (Doc can still be Close'd from another goroutine), race coverage should be in Plan 03 Task 2 and Plan 04 Task 2 verify commands too (`go test ./... -race -run 'Test(ArrayIter|ObjectIter|...)'`).
- **MEDIUM — String allocator cross-DLL concern not addressed on Windows.** The C ABI contract says `element_get_string` allocates, `bytes_free` releases. On Windows, crossing the `.dll` boundary with `new uint8_t[]/delete[]` is only safe if both sides use the shim's CRT. Plan 01 doesn't specify allocator (C++ `new`/`delete[]`, C `malloc`/`free`, or Rust `Vec::into_raw_parts` round-tripped via `bytes_free → Vec::from_raw_parts`). Lock the allocator choice or at least require the test harness to exercise round-trip free on all three platforms before Plan 06.
- **MEDIUM — "Stale element" semantics of `Type()` are unclear.** Context D-02 says `Type()` returns `TypeInvalid` on "stale or closed" elements. Closed is handled. Stale — where the owning Doc is alive but the element's state was tampered with or came from a discarded iterator — is not defined anywhere in Plan 02 or 04. Since `Element` is a value copy, there's no natural staleness channel. If `Type()` calls `element_type` which calls `with_validated_view`, tampered `state1`/`reserved` → `ErrInvalidHandle` → must translate to `TypeInvalid` in the Go wrapper. Plan 02 Task 1 doesn't explicitly say `wrapStatus` errors become `TypeInvalid` (error is swallowed). Lock this translation or accept the closed-only interpretation.
- **MEDIUM — "Fuzz corpus" commitment is downgraded.** Roadmap Phase 4 must-haves say "Malformed-UTF-8 fuzz corpus in tests." Plan 05 adds table-driven malformed-UTF-8 cases, not a `go test -fuzz` corpus. Go 1.24 has native fuzzing — either genuinely ship a `FuzzParseThenGetString` corpus or amend the roadmap item to "table-driven malformed-UTF-8 coverage."

#### LOW

- **LOW — Godoc regex is loose.** `rg '^// ElementType'` catches a comment that merely *mentions* `ElementType`, not one that actually precedes the declaration. Tighten with context (`rg -B0 -A1 '^// ElementType' | rg 'type ElementType'`) or rely on `go doc` invocation in verify.
- **LOW — Plan 01 Task 1 is large.** One task touches 5 files across the bridge, runtime, registry, and lib.rs. This is one atomic commit. Splitting into (1a) generalize validation, (1b) scalars/bool/null, (1c) string + bytes_free would produce reviewable-sized commits and cleaner rollback surfaces — but is a nice-to-have if the executor is autonomous.
- **LOW — `Object.GetStringField` semantics on `null` not explicitly tested.** D-09 says present `null` returns a valid Element, and `GetString()` on null returns `ErrWrongType`. Therefore `GetStringField("key_present_as_null")` must propagate `ErrWrongType` (not `ErrElementNotFound`). Plan 04 Task 2's `TestGetStringField` doesn't name this case.
- **LOW — No test for iterating an empty array/object.** Edge case: `[]` and `{}`. `iter.Next()` should return `false` on the first call with `Err() == nil`. Not listed in Plan 04 Task 2.
- **LOW — No negative test for double-call `Next()` after done.** After iteration completes, repeated `Next()` should keep returning `false` with `Err() == nil`. Typical scanner-pattern bug.

### Suggestions

1. **Add a "descendant view encoding" subsection to Plan 01 `must_haves.truths`** that locks: the tag value namespace (e.g. `ROOT_VIEW_TAG`, `DESCENDANT_VIEW_TAG`, `ARRAY_ITER_TAG`, `OBJECT_ITER_TAG` as distinct 8-byte constants), what lives in `state0`/`state1` for descendants (recommend: heap-allocated `psimdjson_element` pointer tracked in a per-doc arena that is freed on `doc_free`, since simdjson `dom::element` is a 16-byte value type but C++ bridge signatures already use pointers — least-invasive option), and that `bytes_free` ownership is the Rust shim's own allocator.
2. **Rewrite Plan 05 Task 2's numeric boundary case.** Replace `1e20 → ErrNumberOutOfRange` with `9223372036854775808 → ErrNumberOutOfRange` (exactly max_int64+1, parses as UINT64). Add a separate case `1e20 → ErrWrongType` to document float-kind-on-int-getter behavior. Coordinate with roadmap amendment.
3. **Reshuffle per-task verify for Plans 02 and 04.** Task 1 verify: `go build ./... && go vet ./...`. Task 2 verify: the `go test -run 'Test(...)'` command. This lets each task pass acceptance atomically.
4. **Add `-race` to Plan 03 Task 2 and Plan 04 Task 2 verify commands** so races in iterator state surface at the earliest plan that can introduce them.
5. **Extend Plan 04 Task 2 test list** with `TestObjectIterEmpty`, `TestArrayIterEmpty`, `TestIteratorNextAfterDone`, and `TestGetStringFieldNullValue` (expecting `ErrWrongType`).
6. **Pre-decide Windows allocator strategy in Plan 01** (recommend: Rust `Box::into_raw`/`Box::from_raw` or `Vec` round-trip inside the Rust shim, so all allocations originate and terminate in the same `.dll`'s allocator) and add a `bytes_free` round-trip smoke test to `tests/rust_shim_accessors.rs`.
7. **Consider a native key-copy fast path for `object_iter_next`** that returns key bytes via a caller-provided buffer or inline `(state0 as *const u8, state1 as usize)` pair — the existing ABI already has the struct fields available. This avoids the per-key `element_get_string` + `bytes_free` cost and matches the "object iteration is the predicted first bottleneck" architecture note. Not strictly required for Phase 4 correctness, but cheap to add now vs. re-doing after benchmarks in Phase 7.
8. **Reduce Plan 01 Task 1 to (1a/1b/1c)** if autonomous commits matter. Each piece stays verifiable with `cargo test --lib`.

### Risk Assessment

**Overall: MEDIUM**

Justification: The architecture is sound, the decomposition maps cleanly onto the research risks, and Plans 02/04 are low-risk Go extensions of an established pattern. The phase's risk concentrates in three places: (1) the unspecified descendant-view encoding in Plan 01 — a single bad choice here costs a Plan 03 rework; (2) a roadmap/simdjson mismatch in Plan 05 that will fail verification as written; (3) per-task verification gaps that complicate rollback. None of these are architecturally fatal — they are planning-artifact fixes that can be pushed into the PLAN files before execution begins, without re-sequencing the phase. If the HIGH items are addressed in an amendment pass, the residual risk drops to LOW.

## Consensus Summary

### Agreed Strengths

- The five-plan decomposition is sound and the dependency graph is correct: native descendant-safe validation first, then scalar and iterator slices, then public Go wrappers, then docs/verification.
- Keeping `GetStringField(name)` as Go composition instead of reopening the ABI is the right constraint-preserving move.
- The plans are aligned with Phase 3 patterns and preserve the sentinel-plus-detail error model, `runtime.KeepAlive(...)` discipline, and DOC-03 as an explicit exit gate.

### Agreed Concerns

- Plan 01 still needs a sharper specification for descendant-view handling. Both reviewers call out missing detail around how child views are encoded or supported beyond the root-only Phase 3 path.
- String ownership across the FFI boundary needs tighter guardrails. Both reviews highlight the importance of the `bytes_free` contract and making the Go string path unambiguously safe.

### Divergent Views

- Gemini’s main high-severity concern is iterator memory leakage if native iterator state is heap-allocated without a matching ABI free path.
- Claude’s strongest concerns are the under-specified descendant-view representation, the `1e20` verification mismatch with simdjson behavior, and task-level verify commands that currently depend on test files created later in the same plan.
- Claude also raises broader performance and portability concerns around object-key iteration cost, Windows allocator ownership, and earlier `-race` coverage; Gemini does not.

### High-Signal Follow-Ups

- Amend Plan 01 before execution to lock descendant-view encoding, iterator/tag state rules, and string allocator ownership.
- Update Plan 01/04 task text so `ObjectIter.Next()` explicitly performs key-string extraction safely and `ElementGetString(...)` frees native allocations via a `defer`-safe path.
- Fix task verification sequencing in Plans 02 and 04 so Task 1 does not depend on tests created in Task 2.
- Reconcile the Plan 05 numeric boundary case with actual simdjson behavior: either change the example away from `1e20` or intentionally change the bridge semantics and document that choice.
