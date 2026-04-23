---
phase: 8
reviewers: [gemini, claude]
reviewed_at: 2026-04-23T16:17:34Z
plans_reviewed:
  - 08-01-PLAN.md
  - 08-02-PLAN.md
  - 08-03-PLAN.md
  - 08-04-PLAN.md
  - 08-05-PLAN.md
---

# Cross-AI Plan Review - Phase 8

## Gemini Review

# Phase 8 Plan Review: Low-overhead DOM traversal ABI and specialized Go any materializer

## 1. Summary
Phase 8 presents a well-engineered shift from per-node FFI overhead to a bulk-traversal "frame-stream" architecture. The plans provide a robust path for improving Tier 1 materialization performance while maintaining strict adherence to the project's safety, numeric precision, and public ABI stability requirements. By implementing a repo-owned internal frame layout, the design avoids leaking upstream `simdjson` tape internals while providing Go with the exact metadata needed for efficient, pre-allocated tree construction. The inclusion of a machine-gated improvement script in the final wave ensures that Phase 8 only closes when empirical performance gains are verified.

## 2. Strengths
*   **Correct Bottleneck Targeting:** The shift to a frame-stream ABI directly addresses the "materialization-dominates-parse" finding from Phase 7 by removing the O(N) cost of registry lookups and FFI stack switches.
*   **Defensive ABI Design:** Use of the `psdj_internal_` prefix, `cbindgen` exclusions, and Python-based header audit rules provides multi-layered protection against internal symbol leakage.
*   **Semantic Rigor:** Explicitly preserves the distinction between `int64`/`uint64`/`float64` and handles duplicate-key "last wins" semantics for map materialization, ensuring the fast path is a correct replacement for Tier 1 benchmarks.
*   **TDD Methodology:** Plan 01 establishes failing tests and the performance baseline *before* native or Go implementation code is written, ensuring that correctness is baked in from the start.
*   **Memory Efficiency:** Moving from fixed capacity 8 to exact `ChildCount` preallocation for maps and slices will significantly reduce Go allocation growth noise during benchmarks.
*   **Lifetime Safety:** The use of `runtime.KeepAlive(doc)` after borrowed frame reads and the requirement for public values to own Go memory (via copies) strictly adheres to the project's safety invariants.

## 3. Concerns
*   **Traversal Depth (Severity: LOW):** The C++ recursive walk in Plan 02 and the Go recursive builder in Plan 03 might hit stack limits on extremely deep, nested JSON.
    *   *Context:* `simdjson` already enforces depth limits during the parse phase, so this is likely safe, but an explicit mention of depth-limit parity would be ideal.
*   **Memory Pressure (Severity: LOW):** The `psdj_internal_frame_t` struct is ~72 bytes. For a document with 1 million nodes, the doc-owned frame scratch will consume ~72MB of native memory.
    *   *Context:* This is an acceptable trade-off for multi-GB/s performance, but should be noted in the internal benchmark notes.
*   **FFI Struct Layout (Severity: MEDIUM):** While Plan 03 adds layout tests, any drift in padding or alignment between the C++ compiler and the Go `unsafe` layout on 5 platforms could lead to subtle corruption.
    *   *Context:* Mitigated by the requirement for exact 64-bit layout assertions and CI coverage.

## 4. Suggestions
*   **Go Builder Iteration:** Consider a non-recursive (stack-based) loop for the Go builder in `materializer_fastpath.go` if the benchmarks on large fixtures show high stack pressure, though a recursive helper is likely fine given typical JSON depth limits.
*   **Native Scratch Reuse:** In `simdjson_bridge.cpp`, the `materialize_frames` vector is doc-owned. Ensure that `reserve()` is used judiciously if the approximate node count can be estimated from the tape size before the walk, to minimize reallocations within the native traversal.
*   **Error Detail Granularity:** In Plan 02, ensure that `psimdjson_materialize_build` propagates specific `simdjson` errors (like capacity issues) through `map_error` rather than defaulting to `PURE_SIMDJSON_ERR_INTERNAL`.

## 5. Risk Assessment (LOW)
The overall risk is **LOW**.
*   **Justification:** The design is non-breaking (public accessors are untouched), internal (unexported implementation), and heavily gated by correctness and performance tests. The "Wave 0" strategy ensures that any regression in ABI contract or numeric semantics is caught immediately. The architectural pattern of one FFI crossing per materialization call is a proven optimization in high-performance Go/C++ libraries.

---

## Claude Review

# Phase 8 Plan Review

## Overall Summary

These five plans lay out a coherent, test-first Phase 8 implementation that creates an internal `psdj_internal_*` frame-stream ABI, binds it to Go, replaces the Tier 1 materialization hot path, and closes out with machine-gated benchmark evidence. The dependency graph is clean (01 -> 02 -> 03 -> 04 -> 05), the public ABI stability story is enforced by a hardened header audit, and the scope boundary to Phase 9 is explicit. The design choices (repo-owned frames over raw tape, doc-owned scratch, Go-side escape copies, preorder emission with exact child counts) match the locked CONTEXT decisions and the research memo. The main risks are in a handful of under-specified edge cases (BIGINT, reentrancy, benchmark significance vs. median-only gate) rather than in the overall architecture.

---

## Plan 08-01 (Wave 0 Guardrails)

### Strengths
- Narrow, auditable scope: header rule + six named test shells + artifact directory.
- `FORBIDDEN_INTERNAL_SYMBOL_PREFIXES = ("psdj_internal_", "psimdjson_")` enforces the public/private split declaratively and pairs it with a symmetric positive test.
- Test names are locked before implementation, so Plan 03 has nothing to negotiate.

### Concerns
- **MEDIUM - Skip-guarded tests are silent passes.** The single `requireFastMaterializerLinkedForTest(t)` helper causes all six `TestFastMaterializer*` tests to `t.Skip` in Plan 01. `go test -run 'TestFastMaterializer' -count=1` will report PASS even if the assertion bodies are syntactically valid but semantically wrong (e.g., wrong struct fields, wrong helper signatures). The TDD framing is thin - a typo in a deferred `Object.GetField` call path won't surface until Plan 03 removes the guard, which is the busiest wave of the phase.
- **LOW - `tests/abi/test_check_header.py` is a new Python test harness in a repo that previously ran header audits as a shell command.** The plan assumes `python3 tests/abi/test_check_header.py` is an acceptable entrypoint (unittest/self-driving script), but the runner convention isn't specified.

### Suggestions
- In Plan 01's test bodies, include one or two assertions that don't require the fast materializer - e.g., assert that `mustParseDoc` returns a usable `Element`, that `materializer_fastpath.go` is expected to provide `fastMaterializeElement`, or just run the Skip as `t.Skipf` with a link to the Plan 03 activation checklist. Consider a `go build`-gated placeholder symbol so a compile-time check runs in Wave 0.
- Make the Python test runner mechanism explicit (invoke via `unittest` or add `if __name__ == "__main__": unittest.main()` boilerplate).

### Risk: **LOW**

---

## Plan 08-02 (Native Frame Stream)

### Strengths
- Frame struct is precisely specified at byte level (field order, types, sizes) - downstream Go layout tests can pin it exactly.
- Reuses `with_resolved_view` so root, descendant, forged, reserved-bit, and closed-doc validation all share the established registry discipline instead of inventing a parallel path.
- cbindgen exclusion list update is in the same commit as the Rust export, so header drift is caught immediately.
- Rust integration test uses `uint64 max` to prove split numeric payloads work end-to-end inside the cdylib, independent of Go.

### Concerns
- **HIGH - BIGINT handling is unspecified.** The public ABI maps `simdjson::BIGINT_ERROR` to `PURE_SIMDJSON_ERR_PRECISION_LOSS`, and `pure_simdjson_element_type` already surfaces this for root values. But the plan says `psimdjson_materialize_build` "maps simdjson errors through `map_error`" without defining what the traversal does when a BIGINT node is encountered mid-walk. Options are: (a) abort entire materialize with `ERR_PRECISION_LOSS`, (b) emit an error-frame kind, (c) silently coerce. Current accessor-based materializer fails lazily at the offending node; the fast path needs an explicit decision and test. If Phase 4's "parse-time `ErrInvalidJSON`" claim is wrong for some BIGINT subset, fast-path behavior will diverge from the accessor baseline under Plan 03 parity tests.
- **HIGH - Reentrancy / concurrent-goroutine use of `doc->materialize_frames`.** A single `std::vector` on `psimdjson_doc` means: (a) two concurrent `fastMaterializeElement(doc, ...)` calls race on `push_back` -> UB, and (b) even sequential calls invalidate any still-live frame span from the previous call if any consumer forgot to finish draining. The `Doc` is documented as thread-compatible, so (a) is caller fault, but the plan should state this precondition in the contract doc or in-code comment, and the registry could add a lightweight `materialize_in_progress` bool to turn misuse into `ERR_PARSER_BUSY`-style deterministic error rather than silent corruption.
- **MEDIUM - Saturated child-count handling on large containers.** simdjson's tape stores child counts saturated at 24 bits. The plan says "sets exact `child_count`" - if the implementation walks via simdjson's DOM array/object iterators (which count implicitly), this is fine. But if the implementation uses tape scope words directly (as hinted by `src/native/simdjson_bridge.cpp:212`'s `tape_ref` usage), saturation handling must be explicit. Not called out in acceptance criteria.
- **MEDIUM - Vector growth vs. borrowed pointer stability.** `std::vector<psdj_internal_frame_t>::push_back` invalidates pointers on reallocation. Frames are handed out only after traversal completes, but `out_frames = materialize_frames.data()` is taken only after the last `push_back`, so this is correct - worth a one-line comment in the bridge so future contributors don't `reserve()` suboptimally.
- **LOW - `psimdjson_materialize_build` naming.** Elsewhere in the plan the Rust export is `psdj_internal_*` and the C++ bridge is `psimdjson_*`. This is intentional and matches existing convention, but it means the forbidden-prefix list must include both; Plan 01 already does.

### Suggestions
- Add a task or acceptance criterion that pins BIGINT behavior in Plan 02 Task 1 (`fast path returns ERR_PRECISION_LOSS when traversal hits a BIGINT-classed numeric node`), plus a Rust integration test case containing `{"big": 99999999999999999999999}` that asserts the error code propagates without partial frame emission.
- Add `materialize_in_progress` guard (or document single-call precondition prominently) to prevent double-call scratch aliasing.
- In the C++ action, call `materialize_frames.reserve(child_count_hint)` at the root if tape metadata exposes it, to minimize reallocations during traversal. Also add a comment that frames are handed out via `data()` only after traversal completes.

### Risk: **MEDIUM** (primarily BIGINT gap and reentrancy)

---

## Plan 08-03 (Go Binding + Fast Materializer)

### Strengths
- Explicit `unsafe.Sizeof == 72` plus full offset pinning catches the most common class of FFI layout bug across Go/Rust/C++ boundaries.
- `fastMaterializeElement` reuses `element.usableDoc()` so `ErrClosed`/`ErrInvalidHandle` short-circuit before any FFI crossing - keeps parity with every other public accessor.
- Preallocation via `make([]any, 0, int(frame.ChildCount))` / `make(map[string]any, int(frame.ChildCount))` is locked into acceptance criteria.
- Duplicate-key behavior is asserted in both directions (materializer last-wins + existing `GetField` first-match regression test remains unchanged).

### Concerns
- **HIGH - Borrowed slice lifetime across the recursive helper.** `InternalMaterializeBuild` returns `unsafe.Slice(ptr, count)`. This slice header has no finalizer and no connection to the `Doc` Go object. The recursive builder reads from it repeatedly; if a caller passes a `*Doc` but the runtime decides the `Doc` is unreachable mid-walk (it's not, because the caller holds a live `Element` which holds `doc`, but this chain isn't obvious), the frames could be freed by a concurrent `doc.Close()` racing with the materializer. The `runtime.KeepAlive(doc)` at function end is necessary but not sufficient against concurrent `Close`. Matches project's thread-compatible guarantee but deserves an explicit test.
- **MEDIUM - Acceptance criterion for `runtime.KeepAlive(doc)` placement.** The criterion greps for the call but doesn't verify it's at the end of the function (after the last frame read). A grep-anchor like `defer runtime.KeepAlive(doc)` immediately after the frame fetch would be more robust than a placement-agnostic regex.
- **MEDIUM - Helper recursion on preorder frames.** The `(any, nextIndex, error)` recursion is a sensible pattern but easy to get wrong: if the builder miscounts array/object ChildCount or misses a key frame for an object entry, downstream frames desynchronize silently. Parity tests will catch most of this, but an explicit "frames fully consumed at end of walk" invariant check (`if nextIndex != len(frames) { return ErrInternal }`) would fail loud on frame-builder bugs in Plan 02.
- **MEDIUM - Object member key frames convention.** The spec says "Object member frames carry `key_ptr/key_len`; array member frames carry `key_ptr == NULL` and `key_len == 0`." The Go materializer needs to distinguish object-context from array-context children. If the spec is that every child frame inside an object carries the key inline (rather than alternating key-frame/value-frame pairs), this is simpler - and the plan appears to assume this - but it's worth stating explicitly in Plan 02's frame semantics to avoid ambiguity.
- **LOW - Zero-length string at offset 0.** `string(unsafe.Slice((*byte)(unsafe.Pointer(0)), 0))` is technically UB on some Go versions even though the length is zero. The guarded early-return for `ptr == 0` handles it, but the acceptance criterion should pin that guard explicitly.

### Suggestions
- Add an explicit invariant check after top-level recursion: `if consumed != len(frames) { return nil, wrapStatus(int32(ffi.ErrInvalidJSON)) }` (or `ErrInternal`) to catch frame-builder desync bugs in Plan 02.
- Pin `runtime.KeepAlive(doc)` as `defer runtime.KeepAlive(doc)` immediately after the `InternalMaterializeBuild` call and update the acceptance regex to anchor placement.
- Add a `TestFastMaterializerFramesFullyConsumed` assertion that parses a small doc and verifies the builder returns exactly the expected frame count (via debug hook or by counting allocations in a helper).
- Ensure Plan 02's spec explicitly says "object children carry their key inline" (not "key frame then value frame").

### Risk: **MEDIUM**

---

## Plan 08-04 (Benchmark Wiring)

### Strengths
- Change surface is minimal (two test files) and leaves comparator registry names untouched - benchstat rows stay comparable against Phase 7 raw artifacts.
- Explicit comment requirement in the diagnostic loop defends against "someone caches the Go tree" future regressions.
- Keeps Tier 2 / Tier 3 schema-shaped paths untouched.

### Concerns
- **LOW - Dead-code risk from renamed `benchmarkMaterializePureElementViaAccessors`.** The plan lets the implementer optionally keep the old helper. If retained unused, it becomes silent rot. Either delete it outright or anchor it behind a build tag.
- **LOW - Acceptance criterion grep on the loop shape uses `[[:space:]]*` inside a `rg` regex.** Subtle but `rg` is multiline-aware only with `-U`. The criterion allows file-read fallback, so acceptable, but a simpler verification would be `rg 'benchmarkMaterializePureElement\(root\)' benchmark_diagnostics_test.go`.

### Suggestions
- Drop `benchmarkMaterializePureElementViaAccessors` unless there's a concrete test that exercises it for regression comparison. If it's kept, wire it to a `TestMaterializerParityAccessorsVsFast` test so it has a reason to exist.
- Simplify the loop-shape acceptance check.

### Risk: **LOW**

---

## Plan 08-05 (Evidence Capture + Closeout)

### Strengths
- Three-file evidence layout (raw + benchstat + machine-gated improvement check) makes the closeout auditable after the fact.
- Handoff sentence to Phase 9 is locked verbatim and absence-of-diff enforced via `git diff --exit-code` on README/docs/workflows.
- `check_phase8_tier1_improvement.py` uses only the standard library - no new tool dependencies.

### Concerns
- **HIGH - Median-only gate ignores benchstat significance.** `python3 scripts/bench/check_phase8_tier1_improvement.py` compares medians pairwise and PASSes on any `new_ns < old_ns`, even if the delta is 0.1% and benchstat reports "~" (no statistically significant difference). For a phase whose entire premise is "measured Tier 1 improvement," a weak pass criterion could let an unambitious implementation ship. Benchstat p-values or at least a minimum % delta (e.g. `new_ns < old_ns * 0.9`) would be more defensible.
- **HIGH - Hardware variance of the Phase 7 baseline is ignored.** v0.1.1 was captured on `darwin/arm64` Apple M3 Max. If Plan 05 runs on a different local machine (Linux CI, colder M2, etc.), the `ns/op` comparison is meaningless: a slower host trivially fails and a faster host trivially passes. The script has no hardware assertion. At minimum, `08-BENCHMARK-NOTES.md` should record the capture host and the script should warn when goos/goarch/cpu strings in the two raw files differ.
- **MEDIUM - Wall-clock budget of `-count=5` on large fixtures.** `BenchmarkTier1Diagnostics_canada_json/pure-simdjson-full-16` at ~150ms/iter with Go's default benchtime (~1s each run) and `-count=5` is on the order of minutes, multiplied across three fixtures x six sub-benchmarks. This is an autonomous task run with a 2-minute default tool timeout. The verification command doesn't set `-timeout` or discuss budget.
- **MEDIUM - Python parser brittleness.** Go benchmark output is space-delimited and varies across Go versions (extra metric columns, Unicode separators in some envs). The script is untested and will silently miss rows if it uses naive split-on-whitespace against `BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-full-16 \t 61 \t 27962032 ns/op ...`. Ingesting benchstat's machine-readable CSV output (`benchstat -format csv`) would be more robust than re-parsing raw `go test -bench` lines.
- **MEDIUM - `improvement.txt` schema is defined only by example.** The acceptance criterion looks for `PASS BenchmarkTier1Diagnostics_.../pure-simdjson-(full|materialize-only)`. A failing run doesn't produce a well-defined `FAIL` line in the spec (the script exits 1 and writes to the file via `>`, but the text format on failure isn't pinned). Future readers of a failed Phase 8 won't know what to grep for.
- **LOW - `statistics.median` on 5 samples is coarse.** `benchstat` uses a more robust estimator. For 5 samples the median IS the middle value, which is fine, but unit-testing the script with a synthetic 5-sample fixture would be cheap insurance.

### Suggestions
- Strengthen the machine gate: either (a) require `new_median <= old_median * 0.9` (10% improvement floor) to pass, or (b) parse `benchstat` output for `p<0.05` AND direction-negative deltas.
- Add a hardware check: compare `goos:`, `goarch:`, `pkg:`, `cpu:` headers in both files and fail or loudly warn on mismatch. Record the capture host in `08-BENCHMARK-NOTES.md`.
- Add a `--timeout 1200s` or equivalent to the `go test -bench` command and document the expected wall-clock budget.
- Write a tiny `tests/bench/test_check_phase8_improvement.py` with a synthetic pair of raw files that exercises PASS and FAIL paths.
- Pin the failure-case output format explicitly: e.g., `FAIL <name> old=<...> new=<...> delta=<...> reason=<regressed|missing>`.
- Consider using `benchstat -format csv` as the script's input instead of raw `-bench` output.

### Risk: **MEDIUM-HIGH** (the closeout gate is the phase's contract with Phase 9; a weak gate undermines the whole proof-of-improvement claim)

---

## Cross-Cutting Observations

### Things the plan set gets right
- **ABI hygiene is enforced, not hoped for.** Three layers (cbindgen exclude + header audit + absence grep) all gate the internal/private split.
- **No public API widening.** Absence of `Element.Interface`/`Doc.Interface` is asserted via grep. D-14 is hard to violate accidentally.
- **Doc-owned scratch, Go-owned escape.** The ownership story (borrowed spans during one call, copy at value boundary) is consistent across Plans 02 and 03.
- **Benchmark row stability.** Comparator keys and sub-benchmark names are preserved, which is the single thing you most want right for benchstat to be meaningful.

### Gaps worth addressing before execution
- **BIGINT end-to-end semantics** (Plans 02 + 03 tests).
- **Reentrancy/concurrent-call guard** on doc-owned frame scratch (Plan 02).
- **Benchmark gate quality** (Plan 05: statistical significance + hardware-identity check + wall-clock budget).
- **Test stub assertiveness in Wave 0** (Plan 01: skipped tests can't fail early).
- **Frame-fully-consumed invariant** as a builder/consumer desync detector (Plan 03).

### Scope creep / over-engineering
- None spotted. Plans deliberately stay out of Phase 9 territory (public README, release, Element.Interface). The `.gitkeep` + BENCHMARK-NOTES + absence-diff pattern is exactly enough.

---

## Overall Risk Assessment: **MEDIUM**

The architecture is sound and decisions are well-grounded in the locked CONTEXT. The plans execute cleanly in sequence and the contract gates (header audit, layout assertions, parity tests) are credible. The residual risk comes from three specific places:

1. **BIGINT behavior is unspecified** and could cause Plan 03 parity tests to flake or diverge from accessor baseline.
2. **The Phase 8 success criterion (median `ns/op` improvement on one host)** is weaker than the phase's narrative demands. A clean pass against a noisy baseline on mismatched hardware is possible.
3. **Reentrancy on doc-owned frame scratch** is a correctness cliff if the thread-compatibility contract is ever violated by a consumer - silent corruption rather than a clean error.

Addressing those three before execution (or in Wave 0) would drop the overall risk to LOW. The rest is paper-cut-level polish.

---

## Consensus Summary

### Agreed Strengths
- Both reviewers agree the frame-stream approach targets the measured Phase 7 bottleneck directly by reducing per-node FFI crossings.
- Both reviewers called out strong public ABI hygiene through internal prefixes, cbindgen exclusions, and header audits.
- Both reviewers found the plan set non-breaking: existing public DOM accessors, iterators, and error semantics remain stable while the fast path stays internal.
- Both reviewers agreed exact or near-exact container sizing and Go-owned escape copies are important strengths of the design.

### Agreed Concerns
- Both reviewers identified FFI/layout safety as a meaningful risk area. Gemini focused on cross-platform struct padding/alignment; Claude added concrete placement and frame-consumption checks for the Go binding.
- Both reviewers raised recursion/depth or frame-walk robustness concerns. Gemini suggested stack-based fallback if depth becomes visible; Claude suggested an explicit full-frame-consumption invariant.
- Both reviewers suggested improving native frame allocation discipline with a `reserve()`/capacity hint and clearer comments around when borrowed frame pointers become stable.

### Divergent Views
- Gemini rated the overall plan LOW risk; Claude rated it MEDIUM because it found additional edge cases around BIGINT semantics, doc-owned scratch reentrancy, and the benchmark gate's statistical/hardware validity.
- Gemini treated the Wave 0 test-first structure as a strong guardrail; Claude noted that skip-guarded tests can silently pass until Plan 03 and recommended earlier compile-time or non-fast-path assertions.
- Gemini accepted the closeout improvement script as strong evidence; Claude recommended strengthening it with a minimum delta or benchstat significance plus hardware identity checks.

### Highest-Priority Follow-Ups
- Pin BIGINT/precision-loss behavior in the native traversal and Go parity tests before implementation reaches Plan 03.
- Add a deterministic guard or explicit contract for concurrent/reentrant use of doc-owned materialization scratch.
- Strengthen the Phase 8 benchmark gate so a tiny, statistically insignificant, or cross-hardware-only improvement cannot pass.
- Add a builder invariant that the preorder frame stream is fully consumed and fails loudly on desynchronization.
- Make Wave 0 skipped tests more assertive, or add a compile-time placeholder check, so early test scaffolding catches more mistakes.
