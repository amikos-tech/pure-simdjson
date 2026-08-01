---
phase: 12
reviewers: [codex, opencode]
reviewed_at: 2026-07-30T18:23:45Z
plans_reviewed:
  - 12-01-PLAN.md
  - 12-02-PLAN.md
  - 12-03-PLAN.md
  - 12-04-PLAN.md
  - 12-05-PLAN.md
  - 12-06-PLAN.md
  - 12-07-PLAN.md
  - 12-08-PLAN.md
---

# Cross-AI Plan Review — Phase 12

Reviewed by 2 independent AI systems (codex `gpt-5.6-sol` @ xhigh reasoning, opencode `gpt-5.5`).
Codex had repository read access and cited concrete file/line evidence; OpenCode reviewed the
plan documents as delivered. The two review modes are complementary — see Consensus Summary.

---

## Codex Review

# Phase 12 Plan Review

## Overall assessment

The architecture is strong: the plans follow existing FFI patterns, preserve handle validation, use additive ABI growth, and give unusually good attention to native ownership and error taxonomy. However, several concrete blockers make the current plan set unsafe to execute unchanged. In particular, ABI 1.3 omits existing compile-time pins and bootstrap policy, the proposed x86 aliasing check is host-count-dependent and will fail unpredictably, wildcard behavior conflicts with D-02, and `MinifyInto` promises arbitrary overlap that upstream cannot safely support.

**Overall risk: HIGH — revise before execution.**

### Cross-plan blockers

| Blocker | Impact |
|---|---|
| ABI pins omitted from plans 12-01/12-05 | `make verify-contract` and `go build ./...` will fail. Existing pins remain at 1.2 in [src/lib.rs](/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:1026), [handle_layout.c](/Users/tazarov/experiments/amikos/pure-simdjson/tests/abi/handle_layout.c:46), [types_test.go](/Users/tazarov/experiments/amikos/pure-simdjson/internal/ffi/types_test.go:9), and the bootstrap canary [abi_assertion.go](/Users/tazarov/experiments/amikos/pure-simdjson/internal/bootstrap/abi_assertion.go:5). |
| D-14 CI command is not portable | [verify.sh](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/spikes/004-minify-buffer-safety/verify.sh:56) hardcodes exactly 24 cases, while [probe.cpp](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/spikes/004-minify-buffer-safety/probe.cpp:171) runs 12 fixtures for every supported kernel. An x86 host can produce 12, 24, 36, or 48 cases. |
| `AtPathAll` contract contradicts upstream | Without `*`, upstream delegates to `at_path` and propagates missing/out-of-range errors; scalar receivers return an empty vector. That does not match D-02’s claimed error surface. |
| Navigation test expects the wrong error | A non-empty pointer applied to a scalar returns `NO_SUCH_FIELD`, not `INCORRECT_TYPE`, in [element-inl.h](/Users/tazarov/experiments/amikos/pure-simdjson/third_party/simdjson/include/simdjson/dom/element-inl.h:434). |
| Arbitrary overlapping minify buffers are unsafe | The plan documents “any overlap,” but the fallback implementation reads and writes one byte at a time in [fallback.cpp](/Users/tazarov/experiments/amikos/pure-simdjson/third_party/simdjson/src/fallback.cpp:345). A destination beginning inside and after the source can overwrite unread input. Only exact `dst == src` was validated. |
| Bootstrap/release state is unresolved | Changing Go’s required ABI to 1.3 while `bootstrap.Version` still points to published ABI 1.2 artifacts makes the default bootstrap path incompatible until Phase 16. |

---

## 12-01 — ABI foundation and native navigation

### Summary

The resolve-then-register design is appropriate and closely follows the proven `Object.GetField` path. The plan correctly delegates pointer/path parsing to vendored simdjson and introduces additive status codes. It is nevertheless incomplete as an ABI bump plan because existing ABI tests and C assertions are omitted.

### Strengths

- Uses upstream `at_pointer` and `at_path` rather than duplicating grammar.
- Reuses `with_resolved_view` and `encode_descendant_view_locked`.
- Separates invalid syntax from index overflow with stable additive codes.
- Makes both new exports required symbols.
- Preserves Rust panic and C++ exception containment.

### Concerns

- **HIGH:** `make verify-contract` will fail because the Rust ABI getter test still expects `0x00010002`, and `tests/abi/handle_layout.c` statically asserts the old version.
- **MEDIUM:** `public_enum_values_are_append_only` is not updated to pin codes 11 and 12.
- **MEDIUM:** The plan adds names to `REQUIRED_SYMBOLS`, but does not add exact signature checks for the new ABI functions. Name-only validation will not detect parameter drift.
- **LOW:** ABI 1.3 is claimed before its full mandatory symbol set exists. Squash merging limits exposure, but intermediate commits have an incomplete ABI claim.

### Suggestions

- Add `tests/abi/handle_layout.c` and the ABI tests in `src/lib.rs` to Task 1.
- Pin both new error-code values in Rust and C ABI layout tests.
- Add a Phase 12 signature rule to `check_header.py`, covering parameter order and pointer indirection.
- Keep the final ABI bump and full symbol surface in the same squash-merged change.

### Risk assessment

**HIGH.** The implementation design is good, but the advertised verification command cannot pass as planned.

---

## 12-02 — Array access and container sizes

### Summary

This is a focused, low-complexity extension that correctly distinguishes element-returning access from scalar helpers. The O(n) and saturation caveats are handled honestly. Test coverage should be expanded around the scalar helpers and ABI signatures.

### Strengths

- Correct resolve-then-register treatment for `Array.At`.
- Correctly permits zero as a valid size.
- Reuses `ErrIndexOutOfRange`.
- Preserves native kind checks even though public Go wrappers also validate hints.
- Explicitly documents O(n) indexed access and 24-bit size saturation.

### Concerns

- **MEDIUM:** Native tests do not cover empty array/object sizes or wrong-type calls to `array_len` and `object_size`.
- **MEDIUM:** Exact C signatures remain unpinned by the contract checker.
- **LOW:** The C++ `uint64_t` to `size_t` cast is safe on the five supported 64-bit targets, but that platform assumption should be stated near the bridge.

### Suggestions

- Add native tests for zero-sized containers and wrong-type size calls.
- Pin all three signatures in the ABI checker.
- Include a static assertion or comment that supported targets require 64-bit `size_t`.

### Risk assessment

**MEDIUM.** The implementation is straightforward, but contract and negative-path coverage are thinner than the rest of the phase.

---

## 12-03 — Wildcard bulk transport

### Summary

The ownership model is thoughtfully designed. Copying transient C++ indices into a tracked Rust-owned array avoids the lifetime problems of Phase 8’s borrowed span. The major unresolved issue is public behavior: direct delegation does not satisfy D-02 for every accepted path.

### Strengths

- Correctly rejects the borrowed-frame design for long-lived `Element` results.
- Uses exact pointer/count allocation tracking and validates frees.
- Separates byte and view-array allocation tables.
- Preserves result ordering.
- Tests double-free and mismatched-length rejection.

### Concerns

- **HIGH:** `AtPathAll` accepts paths without `*`. In that case upstream delegates to ordinary `at_path` and can return `NO_SUCH_FIELD` or `INDEX_OUT_OF_BOUNDS`, contradicting D-02.
- **HIGH:** On a scalar receiver, upstream returns an empty vector rather than `INCORRECT_TYPE`; therefore the stated `ErrWrongType` behavior is not generally available through thin delegation.
- **MEDIUM:** Tests do not cover non-wildcard paths, missing prefixes, out-of-range suffixes, scalar receivers, or wildcard branch type mismatches.
- **MEDIUM:** No test proves that copied result views remain usable after the native array is freed and become invalid after `Doc.Close`.
- **LOW:** Null/free boundary cases such as `(nil, 0)`, `(nil, N)`, and `(ptr, 0)` are omitted.
- **LOW:** Large wildcard matches temporarily exist in several forms—upstream elements, C++ indices, Rust views, Go views, and descendant-set entries—creating substantial bounded memory amplification.

### Suggestions

- Resolve D-02 before implementation. Choose one explicit contract:

  - require at least one wildcard and reject non-wildcard paths as `ErrInvalidPath`, or
  - allow non-wildcard paths and amend D-02 to include missing/out-of-range errors, or
  - normalize missing/out-of-range results to an empty slice.

- Add lifetime tests covering native-array free and `Doc.Close`.
- Add malformed, scalar, missing, and no-wildcard test cases.
- Add allocation overflow/fallible-reserve handling where practical.

### Risk assessment

**HIGH.** Memory ownership is well designed, but the behavior currently does not implement the locked public contract.

---

## 12-04 — Minify, UTF-8 validation, CI, and ABI closeout

### Summary

The bridge design correctly recognizes that upstream cannot detect a short destination and adds the necessary native boundary check. CPU policy is also handled consistently. The D-14 CI task is currently broken and should be redesigned as a durable contract test.

### Strengths

- Checks `dst_cap >= src_len` before native writes.
- Correctly treats minify as non-validating.
- Applies CPU rejection at both the real C++ boundary and Rust test-forcing layer.
- Includes in-place, malformed, empty, and short-buffer tests.
- Finalizes ownership documentation and required symbols.

### Concerns

- **HIGH:** The x86 CI step cannot reliably pass because `verify.sh` requires exactly 24 cases. The result count varies with supported kernels.
- **HIGH:** Production CI references a `.planning/spikes/...` artifact. A release or PR branch that filters planning artifacts will break the workflow.
- **MEDIUM:** The implementation-selection mutex must be released before running the actual minify/validation scan. Holding it across the scan would serialize all utility calls process-wide.
- **MEDIUM:** Tests prove the Rust fallback hook, but do not directly prove that a successful utility call locks native C++ selection.
- **MEDIUM:** The contract checker still validates new symbols mostly by name, not exact signature and ownership semantics.
- **LOW:** The ABI document retains several current-tense ABI 1.2 statements that need classification as historical or updating to 1.3.

### Suggestions

- Promote the probe to a durable `tests/native/` or `scripts/ci/` location.
- Make its expected total dynamic and assert named kernel coverage separately.
- Use a tightly scoped `{ ... }` lock block around implementation selection only.
- Add a Rust test that calls the utility and then asserts native `set_implementation` returns `KERNEL_LOCKED`.
- Add exact ABI checks for minify, validation, wildcard allocation, and free.

### Risk assessment

**HIGH.** Core native logic is good, but the required x86 gate is not executable as written.

---

## 12-05 — Go FFI bindings

### Summary

The binding surface is mechanically sensible and correctly treats all Phase 12 exports as mandatory. This plan has the largest integration gap: changing `ffi.ABIVersion` breaks the existing bootstrap compile-time canary and leaves loader tests describing the old required surface.

### Strengths

- Registers public ABI symbols through the mandatory path.
- Uses `uintptr` for `size_t`.
- Copies wildcard results before freeing native memory.
- Preserves `runtime.KeepAlive` conventions.
- Keeps standalone utilities out of the document registry.

### Concerns

- **HIGH:** Changing `internal/ffi.ABIVersion` to 1.3 makes the compile-time bootstrap canary fail immediately.
- **HIGH:** The plan does not decide how source ABI 1.3 relates to `bootstrap.Version == 0.1.7`, whose downloadable artifacts expose ABI 1.2.
- **HIGH:** `internal/ffi/types_test.go` remains pinned to ABI 1.2 and lacks codes 11/12.
- **MEDIUM:** `TestBindRequiresEveryPhase11Symbol` and loader fixtures are not updated to require every Phase 12 symbol.
- **MEDIUM:** The “later additive ABI” loader test currently uses `0x00010003`; after this plan that value is no longer later. It should become 1.4.
- **MEDIUM:** No focused tests exercise wildcard copying/freeing, nil/count inconsistencies, or minify pointer/length marshaling.
- **LOW:** Reusing the `bytes_free` warning function for view-array free failures produces misleading diagnostics.

### Suggestions

- Add `internal/ffi/types_test.go`, `internal/bootstrap/abi_assertion.go`, loader tests, and release ABI policy files to this plan.
- Decide on an honest unreleased v0.2 bootstrap state now rather than waiting for Phase 16.
- Rename Phase 11 binding fixtures and require all nine new symbols individually.
- Add binding-level tests for every new wrapper, especially wildcard ownership.
- Introduce a generic native-allocation free warning or a view-specific warning.

### Risk assessment

**HIGH.** The planned `go build ./...` command will fail, and default library bootstrap becomes incoherent without an explicit version policy.

---

## 12-06 — Public navigation API

### Summary

The Go API shape and documentation intent are good, but two planned behavioral assertions conflict with the vendored implementation. The plan also needs stronger RFC edge and lifetime coverage.

### Strengths

- Thin public wrappers with no duplicate path parser.
- Correct sentinel mapping through the existing error system.
- Explicit RFC 9535 disclaimer.
- Preserves ordered, non-nil empty wildcard results.
- Documents upstream’s quote-handling asymmetry.

### Concerns

- **HIGH:** `AtPointer("/a/")` on `{"a":1}` returns `ErrElementNotFound`, not `ErrWrongType`. Scalar `at_pointer` maps any well-formed non-empty pointer to `NO_SUCH_FIELD`.
- **HIGH:** `AtPathAll` still inherits the unresolved D-02 mismatch from plan 12-03.
- **MEDIUM:** RFC 6901 tests omit `~0`, `~1`, invalid `~2`, leading-zero array indices, and `"-"`.
- **MEDIUM:** No public test closes the document and verifies all navigation results return `ErrClosed`.
- **LOW:** The commentary calls `'foo'` an “11-character” key; it is five characters including quotes.

### Suggestions

- Change the trailing-scalar expectation to `ErrElementNotFound`; use an array path with a nonnumeric segment to exercise `ErrWrongType`.
- Resolve `AtPathAll` semantics before writing its Go tests.
- Add RFC escape/index boundary tests.
- Add document-lifetime tests for single and wildcard results.

### Risk assessment

**HIGH.** At least one required test will fail against the pinned upstream source, and wildcard semantics remain unsettled.

---

## 12-07 — Public utility API

### Summary

The API shapes match the locked decisions, and the kernel-lock integration is coherent. The serious defect is documenting arbitrary overlapping slices as safe when only identical source/destination starts are supported and tested.

### Strengths

- Clear allocating and buffer-reuse variants.
- Correct first-call library resolution.
- Explicitly distinguishes invalid UTF-8 from operational errors.
- Includes CPU rejection and kernel-lock tests.
- Tests exact in-place operation and short destinations.

### Concerns

- **HIGH:** “`dst == src`, or any overlap” is unsafe. For example, `src := buf[:n]` and `dst := buf[1:n+1]` can overwrite unread input in the fallback kernel.
- **MEDIUM:** An undersized public destination unwraps to `ErrInternal`, even though it is now caller-triggerable. The existing comment saying `ErrBufferTooSmall` always indicates a wrapper bug becomes false.
- **MEDIUM:** `ValidateUTF8` cannot truthfully say errors occur “ONLY” during library resolution; `ErrCPUUnsupported`, ABI mismatch, panic, and native exceptions are also possible.
- **MEDIUM:** [SetKernel’s documentation](/Users/tazarov/experiments/amikos/pure-simdjson/kernel.go:36) remains stale because it says only parser or pool creation locks selection.
- **MEDIUM:** Focused Go verification assumes a prebuilt `target/release` library; [TestMain](/Users/tazarov/experiments/amikos/pure-simdjson/testmain_test.go:34) does not use debug builds.
- **LOW:** No regression test confirms ordinary parsing still rejects invalid UTF-8 independently of the standalone validator.

### Suggestions

- Support only non-overlapping buffers or exact same-start in-place operation.
- Detect and reject partial overlap at a safe boundary; test both overlap directions.
- Expose a meaningful buffer-too-small sentinel or perform an explicit Go validation with a documented error.
- Update `SetKernel` documentation.
- Prepend `cargo build --release` to fresh-checkout Go verification commands.
- Add a parse-invalid-UTF-8 regression test.

### Risk assessment

**HIGH.** The documented overlap contract can cause silent input corruption.

---

## 12-08 — Indexed access and size helpers

### Summary

The public API design follows existing package conventions well. The implementation is likely correct, but the tests do not fully cover the behaviors promised in the task’s own TDD block.

### Strengths

- Correctly avoids a zero-value `Element` fallback for `Array.At`.
- Rejects negative Go indices before FFI.
- Uses the established safe/Err-suffixed scalar convention.
- Documents O(n²) repeated-index traversal risk.
- Documents the exact saturation threshold.

### Concerns

- **MEDIUM:** The behavior section promises closed/invalid-array tests, but Task 2 only covers populated and empty containers.
- **MEDIUM:** The wrong-type test calls `AsArray` and therefore does not exercise `Array.At`’s own kind check.
- **LOW:** Equivalent wrong-type and closed-document tests are missing for `LenErr` and `SizeErr`.
- **LOW:** Focused verification has the same prebuilt-release-library assumption as plans 12-06/12-07.

### Suggestions

- Construct a same-package wrong-kind `Array` in tests and call `At` directly.
- Close the document and verify `Len() == 0`, while `LenErr`/`SizeErr` preserve `ErrClosed`.
- Test zero-value `Array` and `Object` wrappers.
- End the plan with `cargo build --release && make verify-contract && go test ./... -race`.

### Risk assessment

**MEDIUM.** The implementation is sound, but the promised error-preservation contract is not fully tested.

---

## Recommended disposition

Revise the plans before execution, prioritizing:

1. Fix all ABI 1.3 pins, loader fixtures, and bootstrap-version policy.
2. Resolve `AtPathAll`’s no-wildcard and error semantics.
3. Replace the hardcoded/planning-local D-14 CI probe.
4. Restrict `MinifyInto` to exact in-place or non-overlapping buffers.
5. Correct the scalar trailing-pointer error expectation.
6. Add fresh-checkout release-library builds and lifetime/error-path tests.

With those changes, the underlying architecture should be capable of delivering all six Phase 12 requirements without scope expansion.

---

## OpenCode Review

## Summary
The Phase 12 plan set is unusually thorough and mostly implementation-ready. It correctly decomposes native ABI growth before Go bindings and public APIs, preserves the existing handle/lifetime model, and explicitly validates the highest-risk areas: wildcard result ownership, minify buffer bounds, CPU fallback policy, and error taxonomy. The main risks are not scope or missing functionality, but a few execution hazards in concurrency/kernel-lock semantics, test brittleness, and one potentially misleading Go-level `MinifyInto` error contract.

## Strengths
- Clear dependency ordering: native ABI/status foundation lands first, then native capabilities, then Go FFI bindings, then public APIs.
- Good use of existing patterns: `ffi_wrap`, `with_resolved_view`, `encode_descendant_view_locked`, required symbol binding, and tracked allocation free discipline are reused instead of reinvented.
- Strong error taxonomy: `ErrInvalidPath` and `ErrIndexOutOfRange` are distinct and mapped 1:1 to native statuses.
- Wildcard result ownership is well designed: transient C++ scratch, synchronous Rust conversion, Rust-owned returned array, explicit tracked free.
- Minify buffer safety is correctly treated as a native boundary requirement, not just a Go precondition.
- Plans explicitly document upstream quirks instead of hiding them, especially `AtPath` syntax, quoted bracket behavior, `Array.At` O(n), and size saturation.
- Validation strategy is strong across layers: Rust FFI tests, Go public API tests, ABI header checks, docs checks, and CI promotion of the x86-64 minify alias probe.

## Concerns

- **HIGH: Go `MinifyInto` exposes undersized `dst` as `ErrInternal`, which is a poor public contract.**  
  Plan 12-07 says undersized `dst` returns an error “unwrapping to `ErrInternal` via the existing buffer-too-small status mapping.” That directly conflicts with the idea that undersized `dst` is now a legitimate caller error, not a wrapper bug. This will be hard for users to handle correctly and makes the API less honest than the native contract.

- **HIGH: Native and Go kernel-lock semantics may diverge under failure cases.**  
  Plan 12-04 says native `psimdjson_minify` / `validate_utf8` set `selection.locked = true` before rejecting fallback, while Plan 12-07 sets Go `kernelSelectionLocked = true` only after successful calls. If a standalone utility returns `ErrCPUUnsupported`, native selection may be locked while Go still thinks `SetKernel` is allowed. That could create confusing behavior or failed native `SetKernel` calls not predicted by Go state.

- **MEDIUM: `psimdjson_minify` checks `dst_cap < src_len` before null pointer validation and CPU gate.**  
  Bounds-first is defensible, but for `src_len != 0` and `dst_ptr == nil`, `dst_cap >= src_len` proceeds to invalid-argument. That is fine, but tests should explicitly cover nil pointer + nonzero length on the native Rust side if raw FFI callers matter.

- **MEDIUM: Wildcard scratch vector concurrency assumptions rely on existing Go serialization, but native API is independently exposed.**  
  `wildcard_build_guard` prevents reentrancy on the same doc, but the doc scratch vector still creates a native-level mutable field. If native exports can be called by non-Go consumers or concurrent Rust tests, race behavior depends on the broader doc synchronization model. The plan should explicitly verify same-doc concurrent wildcard calls return busy or are serialized safely.

- **MEDIUM: `Array.At` wrong-type test plan does not actually call `Array.At` on a forged/non-array `Array`.**  
  Plan 12-08 says to parse a scalar/object, call `AsArray()`, and confirm it returns `ErrWrongType` before `At` is reachable. That tests `AsArray`, not `Array.At`’s own kind pre-check. Since `Array` has unexported fields this is mostly fine for public callers, but in-package tests can construct `Array{element: root}` and should verify `At` directly.

- **MEDIUM: ABI/doc closeout is delayed until 12-04 while earlier waves already bump ABI to 1.3.**  
  This is acceptable during branch-local execution, but if waves are committed independently, docs may temporarily claim ABI 1.3 without the full mandatory symbol list. The executor must avoid shipping intermediate commits as release candidates.

- **LOW: Plan 12-01 acceptance uses `grep -c '0x00010002' ...` and may fail if docs intentionally mention prior ABI history.**  
  It is okay for a strict migration commit, but brittle if `docs/ffi-contract.md` later keeps historical compatibility notes.

- **LOW: `AtPathAll` malformed wildcard syntax tests may depend on upstream edge behavior.**  
  The plan asks for “malformed wildcard syntax returns `ErrInvalidPath`” but does not specify an exact malformed string known to upstream to reject. This could cause executor churn.

- **LOW: `MinifyInto(dst, src)` doc says “any overlap” may be supported.**  
  Decisions and spike evidence specifically confirm `dst == src`. Partial overlap is a stronger claim and may not have been validated. The public contract should say `dst == src` is supported unless partial overlap has explicit upstream evidence and tests.

- **LOW: `Object.Size` naming is reasonable but asymmetric with `Array.Len`.**  
  This is likely fine, but examples/docs should make the pair discoverable.

## Suggestions
- Add a public `ErrBufferTooSmall` sentinel or map `ffi.ErrBufferTooSmall` to a clearer existing/public error before shipping `MinifyInto`. If that is too large a scope change, at least document this as a known wart and add `errors.Is(err, ErrInternal)` tests.
- Align kernel-lock behavior exactly between Go and native for failure paths. Prefer locking only after successful CPU gate, or make Go reflect native lock-on-attempt behavior and test `SetKernel` after an `ErrCPUUnsupported` utility call.
- Change `MinifyInto` documentation from “any overlap” to “`dst == src` is supported for in-place minification” unless partial overlaps are tested.
- Add native/Rust tests for `psimdjson_minify` invalid pointer combinations and `validate_utf8` invalid pointer combinations.
- Add a direct in-package test for `Array{element: nonArrayElement}.At(0)` returning `ErrWrongType`.
- Specify exact malformed `AtPathAll` strings in the plan based on upstream behavior to avoid ambiguity during execution.
- Consider adding a small public example for `AtPointer`, `AtPathAll`, `MinifyInto`, and `ValidateUTF8` after implementation, even if not required by this phase.

## Risk Assessment
**Overall risk: MEDIUM.**

The architecture and sequencing are strong, and the plans should achieve the Phase 12 goals. The remaining risk is concentrated in public API semantics and process-global kernel-selection behavior, not in the broad design. If the `MinifyInto` error contract and utility kernel-lock semantics are tightened before execution, the risk drops close to LOW.

---

## Consensus Summary

Both reviewers independently judged the **architecture sound and the sequencing correct** — native
ABI foundation, then native capabilities, then Go bindings, then public API — and both concluded the
plans can deliver all six requirements (DOM-01..04, UTIL-01/02) without scope expansion. Neither
reviewer flagged scope creep or over-engineering.

They diverge sharply on **readiness**. OpenCode rated the set MEDIUM risk and executable after
tightening two API contracts. Codex rated it **HIGH risk — revise before execution**, because it read
the actual repository and found that the ABI 1.3 bump breaks compile-time pins in files no plan lists.
That divergence is explained entirely by review mode, not by disagreement: OpenCode could not have
found the pin sites from the plan documents alone.

### Agreed Strengths

- Correct dependency ordering: native ABI/status foundation → native capabilities → Go FFI bindings → public API.
- Reuses proven existing patterns (`ffi_wrap`, `with_resolved_view`, `encode_descendant_view_locked`, required-symbol binding, tracked-allocation discipline) instead of reinventing them.
- Strong error taxonomy — `ErrInvalidPath` and `ErrIndexOutOfRange` are distinct and mapped 1:1 to additive native status codes.
- Wildcard ownership model is well designed: transient C++ scratch → synchronous Rust copy → Rust-owned tracked array → explicit free. Correctly rejects Phase 8's borrowed-span approach for long-lived results.
- Minify short-destination checking is correctly treated as a native boundary requirement, not merely a Go-side precondition.
- Upstream quirks are documented rather than hidden (`AtPath` syntax, quoted-bracket asymmetry, `Array.At` O(n), 24-bit size saturation).
- Layered validation strategy: Rust FFI tests, Go public tests, ABI header checks, docs checks, x86-64 CI probe promotion.

### Agreed Concerns

Raised by **both** reviewers — highest priority:

1. **`MinifyInto` overlap contract overpromises (HIGH / LOW).** Both flagged that documenting "any overlap" as safe exceeds what was validated (`dst == src` only). Codex escalated this to HIGH with proof: `third_party/simdjson/src/fallback.cpp:345` minifies byte-at-a-time in one loop (`dst[pos] = c` while reading `buf[i]`), so a destination starting *after* the source overwrites unread input. **Restrict the contract to exact `dst == src` or fully non-overlapping.**
2. **`MinifyInto` undersized `dst` unwraps to `ErrInternal` (MEDIUM, both).** A now caller-triggerable condition surfaces as a sentinel documented to mean "wrapper bug." Both recommend a meaningful buffer-too-small sentinel or an explicit Go-side check.
3. **Kernel-lock semantics may diverge between native and Go (MEDIUM, both).** 12-04 locks selection on *attempt*, 12-07 locks Go state on *success* — an `ErrCPUUnsupported` utility call leaves the two out of sync. Codex adds that `kernel.go:36` `SetKernel` docs are now stale, and that the selection mutex must be released before the scan to avoid process-wide serialization.
4. **`Array.At` wrong-type test does not test `Array.At` (MEDIUM, both).** 12-08 parses a scalar, calls `AsArray()`, and asserts `ErrWrongType` before `At` is ever reached — that exercises `AsArray`. Both recommend constructing an in-package wrong-kind `Array` and calling `At` directly.
5. **`AtPathAll` malformed-input tests are underspecified (LOW, both).** No exact malformed string known to be rejected upstream is named; likely executor churn.
6. **Native negative-path coverage is thin (MEDIUM, both).** Nil-pointer/nonzero-length combinations for `minify`/`validate_utf8`, zero-sized containers, and wrong-type size calls are untested.

### Codex-Only Blockers — Verified Against the Codebase

Codex was the only reviewer with repo access. I independently confirmed each of these claims; **all are accurate**:

| Claim | Verification | Status |
|---|---|---|
| ABI pins remain at `0x00010002` in files no plan lists | `tests/abi/handle_layout.c:46` (`static_assert`), `internal/ffi/types_test.go:9`, `internal/bootstrap/abi_assertion.go:5,8`. No Phase 12 plan lists any of them. | **CONFIRMED** |
| Bootstrap canary breaks `go build ./...` | `internal/bootstrap/abi_assertion.go:10-11` uses two-way negative-array-size assertions on `ffi.ABIVersion`. Bumping to 1.3 makes one array length negative → **compile error**, not a test failure. | **CONFIRMED — hard blocker** |
| D-14 CI probe hardcodes case count | `.planning/spikes/004-minify-buffer-safety/verify.sh:56` greps `^SUMMARY total=24 any_failure=0$`; `probe.cpp` runs 12 fixtures per supported kernel, so an x86 host yields 12/24/36/48. | **CONFIRMED** |
| Production CI would reference a `.planning/spikes/` artifact | Probe lives under `.planning/`, which `/gsd:pr-branch` filters out of PR branches. | **CONFIRMED** |
| Scalar trailing-pointer returns `NO_SUCH_FIELD`, not `INCORRECT_TYPE` | `third_party/simdjson/include/simdjson/dom/element-inl.h:434-438` — well-formed non-empty pointer on a scalar returns `NO_SUCH_FIELD`. Plan 12-06 expects `ErrWrongType` for `AtPointer("/a/")` on `{"a":1}`; it will get `ErrElementNotFound`. | **CONFIRMED — test will fail** |
| Loader fixture uses `0x00010003` as the "later additive ABI" case | `library_loading_test.go:447` sets `reportedABI: 0x00010003`. After the bump this no longer tests a *later* ABI; should become 1.4. | **CONFIRMED** |
| Release ABI policy has no 1.3 entry | `scripts/release/check_bootstrap_abi_state.py:12-15` maps `"0x00010002" → "0.1.5"` with no `0x00010003` key. Also referenced as a required sync point by `abi_assertion.go:7`. | **CONFIRMED — additional site neither reviewer listed** |

Codex's remaining HIGH — that **`AtPathAll` contradicts D-02** — is a design question rather than a
checkable fact. Its reasoning: without a `*`, upstream delegates to plain `at_path` and propagates
`NO_SUCH_FIELD` / `INDEX_OUT_OF_BOUNDS`, and a scalar receiver yields an empty vector rather than
`INCORRECT_TYPE`. If correct, thin delegation cannot produce D-02's stated error surface. Codex offers
three dispositions: require ≥1 wildcard and reject the rest as `ErrInvalidPath`; accept non-wildcard
paths and amend D-02; or normalize missing/out-of-range to an empty slice. **This needs a decision
before 12-03 and 12-06 are executed.**

### Divergent Views

- **Overall risk: HIGH (codex) vs MEDIUM (opencode).** Not a genuine disagreement — Codex saw the compile-breaking ABI pins, OpenCode structurally could not. Treat **HIGH** as the operative verdict.
- **Overlap severity: HIGH (codex) vs LOW (opencode).** OpenCode read it as a documentation overreach; Codex traced the fallback kernel and showed it is a silent input-corruption path. Treat as **HIGH**.
- **ABI-1.3-before-full-symbol-set:** both raised it, but OpenCode called it acceptable under branch-local execution while Codex wants the bump and the full symbol surface in one squash-merged change. Codex's position is stricter and cheap to adopt.
- **Bootstrap version policy:** Codex alone argues the `bootstrap.Version == 0.1.7` (ABI 1.2 artifacts) vs source ABI 1.3 incoherence must be resolved now rather than deferred to Phase 16. OpenCode did not consider release state at all.

### Recommended Disposition

Revise before execution, in this order:

1. Add every ABI 1.3 pin site to the plans — `tests/abi/handle_layout.c`, `internal/ffi/types_test.go`, `internal/bootstrap/abi_assertion.go`, `scripts/release/check_bootstrap_abi_state.py`, and the `library_loading_test.go:447` fixture (→ 1.4). Decide the unreleased-v0.2 bootstrap version policy.
2. Resolve `AtPathAll` no-wildcard and scalar-receiver semantics against D-02.
3. Relocate the D-14 probe out of `.planning/spikes/` into `tests/native/` or `scripts/ci/` and make its expected total dynamic with per-kernel coverage assertions.
4. Restrict `MinifyInto` to exact in-place or non-overlapping buffers; reject partial overlap at a safe boundary.
5. Correct 12-06's scalar trailing-pointer expectation to `ErrElementNotFound`.
6. Align kernel-lock timing between 12-04 and 12-07; refresh `kernel.go:36` docs.
7. Give `MinifyInto`'s undersized-`dst` a meaningful public error.
8. Add the missing negative-path, lifetime, and RFC 6901 escape tests; prepend `cargo build --release` to fresh-checkout Go verification commands.
