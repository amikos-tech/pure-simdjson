---
phase: 2
reviewers_requested:
  - gemini
  - claude
reviewers_completed:
  - claude
reviewers_failed:
  - gemini
reviewed_at: 2026-04-15T11:17:35Z
plans_reviewed:
  - 02-01-PLAN.md
  - 02-02-PLAN.md
  - 02-03-PLAN.md
---

# Cross-AI Plan Review — Phase 2

## Gemini Review

Gemini review content could not be collected in this run.

### Failure Details

- CLI detected on `PATH`, but headless review invocation produced no stdout/stderr for an extended period and had to be cancelled manually.
- A trivial probe also stalled:
  - command: `gemini --approval-mode plan -p 'Respond with exactly: ok'`
  - result: no output before manual termination
- No Gemini review text was generated, so there is no reviewer content to incorporate from Gemini for this run.

### Status

- Outcome: failed
- Impact: consensus below is provisional because only one external reviewer completed successfully

---

## Claude Review

---

# Phase 2 Cross-AI Plan Review

## 1. Summary

Phase 2 ("Rust Shim + Minimal Parse Path") turns the Phase 1 ABI contract into a real, buildable native library. It is decomposed into three sequential plans:

- **Plan 01 (Wave 1):** Native build plumbing — vendor simdjson as a git submodule, write `build.rs` with the `cc` crate, create a narrow C++ bridge layer with full exception containment.
- **Plan 02 (Wave 2):** Runtime core — implement a generation-checked parser/doc registry with explicit state machine (`Idle`/`Busy`), wire the minimal `42` parse path, add diagnostics (implementation name, last-error), enforce the CPU fallback gate, and test lifecycle edge cases in Rust.
- **Plan 03 (Wave 3):** Smoke proof and CI — add a C smoke harness compiled against the committed public header, document exact Linux/MSVC compile commands, and wire a dedicated GitHub Actions workflow for both platforms.

The scope targets requirements SHIM-01 through SHIM-07, with the explicit choice to keep all non-minimal accessors and iterators as stubs.

---

## 2. Strengths

**Disciplined scope control.** The plans are ruthlessly narrow. Only the `42` path is real; everything else stays stubbed. This is the right call — it prevents Phase 4 scope from bleeding backward and keeps the proof honest.

**Exception/panic containment is first-class.** Both the C++ → Rust boundary (`noexcept` + `catch (...)`) and the Rust → C ABI boundary (`ffi_wrap`/`catch_unwind`) are explicitly addressed with concrete verification criteria. The STRIDE threat register entries (T-02-01-02, T-02-02-01) reinforce this.

**Windows MSVC is treated as the second proof target, not an afterthought.** The research correctly identifies MSVC + `cc` + simdjson as the riskiest native build path and burns it down early. The workflow uses `ilammy/msvc-dev-cmd@v1` and exact `cl` commands — this is concrete, not aspirational.

**Lifecycle correctness is tested beyond the happy path.** Plan 02 explicitly tests: stale handle rejection, double-free, parser-busy on second parse, `parser_free` while doc is live. These are exactly the edge cases that would cause memory corruption in later phases if missed.

**Padding is derived, not hardcoded.** Querying `psimdjson_padding_bytes()` from simdjson at runtime and zero-initializing the tail avoids a class of subtle buffer-overread bugs.

**Clean dependency ordering.** Wave 1 → 2 → 3 is correct — you can't test runtime behavior without the build substrate, and you can't run a C smoke harness without the runtime.

---

## 3. Concerns

### HIGH

**H1: Submodule pin is underspecified.** The plans reference "pinned simdjson" and the ROADMAP says "v4.6.1," but none of the three plans specify the exact git tag or commit hash. If the executor picks `master` or a different tag, the amalgamation layout, API surface, or `SIMDJSON_PADDING` value could differ. The `build.rs` task should explicitly name the tag (e.g., `v4.6.1`) and verify the submodule checkout matches.

**H2: `static-libstdc++` / `static-libgcc` on Linux may conflict with the `cdylib` output.** Statically linking libstdc++ into a shared library works but creates symbol-visibility and ODR-violation risks if the consumer (or another loaded `.so`) also links libstdc++. The plans don't discuss this trade-off. For a library loaded via `purego` into a Go process (which itself has no C++ runtime), this is likely fine — but it should be explicitly acknowledged, not silently assumed.

**H3: No test for the `ERR_CPP_EXCEPTION` bridge path.** Plan 02's tests cover lifecycle errors and the happy path, but there is no test that forces the C++ bridge to throw and verifies that `catch (...)` returns `PURE_SIMDJSON_ERR_CPP_EXCEPTION` correctly. This is listed as a ROADMAP "nice-to-have" (allocation-failure injection), but given that the entire safety model depends on exception containment, at least one concrete test — even a contrived one (e.g., `parser_parse` with a null pointer or absurd length that triggers a C++ `std::bad_alloc`) — should exist.

### MEDIUM

**M1: The `PURE_SIMDJSON_ALLOW_FALLBACK_FOR_TESTS` bypass lacks definition.** Plan 02 says to implement this env-var bypass but doesn't specify *where* it's checked (at `parser_new` call time? at library init?), whether it's cached, or what happens if the env var changes between calls. For a test-only mechanism this is acceptable, but ambiguity could lead to either a runtime `std::env::var` call on every `parser_new` (slow) or a `lazy_static` init (inflexible). A one-liner decision would clarify.

**M2: Smoke harness doesn't test error paths.** `tests/smoke/minimal_parse.c` tests the happy path (parse, extract, release, re-parse, release). It doesn't test: invalid JSON, stale handle after free, or the `ERR_CPU_UNSUPPORTED` path. While Rust tests cover these, the smoke harness is specifically meant to prove the *public header ABI* — error-code propagation through the C ABI deserves at least one negative case.

**M3: Darwin is entirely absent from Phase 2 proof.** The ROADMAP success criteria say `cargo build --release` should produce artifacts on "darwin (one arch)." The plans defer Darwin to "optional build-only or later phases." This is pragmatic given MSVC risk prioritization, but it creates a gap between the ROADMAP exit criteria and the plan's actual deliverables. Either the ROADMAP criteria should be softened for Phase 2, or a Darwin compile-only CI job should be added.

**M4: `element_type` mapping scope.** Plan 02 says `element_type` must return `PURE_SIMDJSON_VALUE_KIND_INT64` for the `42` path, but doesn't specify what happens if `element_type` is called on a non-int64 root (e.g., a string or object from a different parse). Since the function is "real" (not stubbed), it should handle all simdjson type discriminants, even if only `INT64` has a corresponding `get_*` accessor. If it returns garbage for other types, Phase 4 inherits a latent bug.

### LOW

**L1: The `build.rs` verification command uses `find ... -maxdepth 1` to locate the dynamic library.** On some Rust targets, `cargo build --release` places the `.so`/`.dylib` in a subdirectory (e.g., `target/release/deps/`). This is unlikely for cdylib outputs but worth a note.

**L2: Plan 03's negative acceptance criterion (`! rg 'release|upload|publish|cosign|bootstrap'`) is brittle.** The word "release" could appear in a comment, a Rust toolchain setup step name, or a `cargo build --release` command. This grep will likely false-positive and need adjustment.

**L3: No explicit `cbindgen` re-generation step in Phase 2.** The plans assume the Phase 1 header remains valid, and `make verify-contract` checks for drift. But if Phase 2 adds new bridge symbols or types that should be in the public header, there's no task to run `cbindgen` and verify. The bridge symbols are internal (`psimdjson_*`), so this is likely fine — but it's an implicit assumption.

---

## 4. Suggestions

1. **Pin the submodule explicitly.** Add a concrete git tag (`v4.6.1`) to Plan 01 Task 1's action text, and add a verification step: `cd third_party/simdjson && git describe --tags --exact-match` should output `v4.6.1`.

2. **Add one exception-path test.** In Plan 02 Task 2, add a test that feeds an input designed to trigger a C++ allocation failure or error that exercises the `catch (...)` → `ERR_CPP_EXCEPTION` path. Even a `parser_parse` with `len = usize::MAX` (which will fail C++ allocation) would suffice.

3. **Add one negative case to the C smoke harness.** Have the smoke harness parse an invalid JSON string (e.g., `{invalid}`) and assert it returns an error code ≠ 0. This takes 5 lines and proves error propagation through the real ABI.

4. **Make `element_type` fully functional** even though only `element_get_int64` is real. Map all simdjson type discriminants to their corresponding `PURE_SIMDJSON_VALUE_KIND_*` constants. The mapping is a simple switch statement and prevents Phase 4 from inheriting a partial implementation.

5. **Reconcile Darwin with ROADMAP.** Either add a `darwin-build-only` job to the Phase 2 workflow (no smoke execution, just `cargo build --release`) or update the ROADMAP success criteria to say "darwin deferred to Phase 5/6."

6. **Fix the brittle `! rg 'release'` check.** Use a more specific pattern like `! rg 'upload-artifact|cosign|bootstrap|r2_bucket' .github/workflows/phase2-rust-shim-smoke.yml` to avoid false positives from `cargo build --release`.

---

## 5. Risk Assessment

| Risk | Likelihood | Impact | Mitigation Status |
|------|-----------|--------|-------------------|
| C++ exception crosses into Rust | Low | Critical (UB, crash) | **Well-mitigated.** `noexcept` + `catch(...)` + `ffi_wrap` + STRIDE entries. Gap: no test exercises the path. |
| Stale/forged handle causes use-after-free | Low | Critical | **Well-mitigated.** Generation-checked registry with explicit lifecycle tests. |
| simdjson submodule version drift | Medium | High (build break or subtle API change) | **Partially mitigated.** Submodule is pinned but exact tag not specified in plan text. |
| Windows MSVC build failure | Medium | Medium (blocks cross-platform proof) | **Well-mitigated.** Dedicated MSVC job, explicit `cl` commands, `ilammy/msvc-dev-cmd`. |
| Buffer overread from insufficient padding | Low | High (memory safety) | **Well-mitigated.** Runtime padding query + zero-init tail bytes. |
| Darwin build regression discovered late | Medium | Low-Medium (delays Phase 5/6) | **Not mitigated in Phase 2.** Deferred by design — acceptable if ROADMAP is updated. |
| `static-libstdc++` symbol collision | Low | Medium (subtle runtime issues) | **Not addressed.** Likely safe for purego loading but should be documented. |

**Overall assessment:** The plans are well-structured, appropriately scoped, and address the critical safety concerns (exception containment, handle lifecycle, padding). The main gaps are in test coverage of error/exception paths and a minor ROADMAP-to-plan alignment issue on Darwin. None of the concerns are blocking — the HIGH items are addressable with small additions to existing tasks.

---

## Consensus Summary

Consensus is provisional for this run because only Claude completed successfully; Gemini did not return review content.

### Agreed Strengths

- Single-reviewer signal: the plans keep Phase 2 tightly scoped to the minimal `42` path and explicitly defer Phase 4 surface area.
- Single-reviewer signal: exception containment, handle lifecycle safety, and padding correctness are treated as first-class requirements rather than incidental implementation details.
- Single-reviewer signal: Windows/MSVC is correctly prioritized as the second native proof target.

### Agreed Concerns

- Single-reviewer signal: specify the exact simdjson version or commit in the plan itself rather than relying on vague "pinned" language.
- Single-reviewer signal: add at least one concrete test that proves `ERR_CPP_EXCEPTION` containment instead of relying only on structural safeguards.
- Single-reviewer signal: extend the public C smoke harness with at least one negative/error-path assertion.
- Single-reviewer signal: reconcile the Phase 2 plan with the ROADMAP's Darwin build expectation.

### Divergent Views

- Not enough completed reviewers to assess divergence in this run.
