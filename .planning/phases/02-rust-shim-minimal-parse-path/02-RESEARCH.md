# Phase 2: Rust Shim + Minimal Parse Path - Research

**Date:** 2026-04-15
**Status:** Ready for planning

## Objective

Turn the Phase 1 ABI contract into a real native shim without collapsing later phases into Phase 2. The plan needs to prove the smallest honest end-to-end parse path, preserve the locked handle/lifetime contract, and burn down the highest native build risk early.

## Constraints That Are Already Locked

- The shim stays a Rust `cdylib` + `staticlib`; Go bindings are a later phase.
- Every parse copies input into Rust-owned padded memory before simdjson sees it.
- Handles stay generation-stamped packed `u64` values; stale handles must fail cleanly.
- One parser may own at most one live document at a time; `doc_free` clears the busy state.
- SIMD fallback is not a silent behavior; unsupported CPUs fail loudly.
- Phase 2 should remain narrow: real minimal parse path now, full accessor surface later.

## Recommended Technical Direction

### 1. Native build integration

- Add `build.rs` to compile vendored simdjson via the `cc` crate rather than `cmake`.
- Vendor simdjson as a pinned git submodule at `third_party/simdjson`.
- Prefer the upstream amalgamation path for Phase 2 so the build graph stays small and deterministic.
- Keep runtime kernel dispatch entirely inside simdjson; do not add `-march=native` or parallel dispatch logic in Rust/Go.
- Preserve the existing cbindgen-driven header flow and keep `make generate-header` / `make verify-contract` green.

### 2. Runtime safety model

- Implement a small custom registry for parsers and docs that maps directly to the locked `{slot:u32, generation:u32}` handle ABI.
- Do not use a temporary raw-pointer shim. The project already locked parser-busy and stale-handle semantics in Phase 1, and simdjson's parser reuse model is the main thing that can go wrong here.
- The minimal real runtime should include:
  - parser allocation / free
  - parser-owned copied+padded input buffer
  - one live doc per parser
  - stale-handle checks
  - parser-busy enforcement
  - doc release that clears parser busy state

### 3. Real exports for Phase 2

Make these exports real in Phase 2:

- `pure_simdjson_get_abi_version`
- `pure_simdjson_get_implementation_name_len`
- `pure_simdjson_copy_implementation_name`
- `pure_simdjson_parser_new`
- `pure_simdjson_parser_free`
- `pure_simdjson_parser_parse`
- `pure_simdjson_parser_get_last_error_len`
- `pure_simdjson_parser_copy_last_error`
- `pure_simdjson_parser_get_last_error_offset`
- `pure_simdjson_doc_free`
- `pure_simdjson_doc_root`
- `pure_simdjson_element_type`
- `pure_simdjson_element_get_int64`

Leave the remaining accessor/iterator exports as explicit stubs behind the same FFI wrapper pattern so the ABI stays intact without pulling Phase 4 work into this phase.

### 4. Diagnostics and CPU gate

- Make the implementation-name helpers report the selected simdjson kernel/implementation for real.
- `parser_new` should reject the `fallback` kernel with `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED`.
- Any bypass for fallback coverage should be hidden to tests/CI only, not exposed as a public Phase 2 feature.
- Parser diagnostics should stay thin and advisory:
  - parse failures populate last-error text
  - offset is best-effort
  - use a pinned unknown-offset sentinel when exact location is unavailable

### 5. Cross-platform proof strategy

- Spend the non-Linux smoke-test budget on `windows/amd64 (MSVC)`, not Darwin.
- The repo and research already call out Windows MSVC + `cc` + simdjson as the risky native path; Phase 2 should burn that down early.
- A pragmatic Phase 2 proof shape is:
  - `linux/amd64`: build + C smoke test
  - `windows/amd64`: build + C smoke test
  - Darwin: optional build-only or deferred until later phases

## Planning Risks To Address Explicitly

### Risk 1: build.rs and vendoring shape

- Need one crisp task to establish `third_party/simdjson`, the pinned commit/tag, and the exact `cc` invocation.
- Plan should explicitly call out the expected source files, include paths, and language standard so the executor is not guessing.

### Risk 2: Rust/C++ exception and panic containment

- Phase 2 must replace the current ad hoc wrapper usage with a macro/helper that makes the unwind policy mechanical.
- The Rust side must not allow panics across FFI, and the C++ seam must use the non-throwing simdjson access patterns documented in Phase 1.

### Risk 3: lifetime correctness in the minimal path

- The plan must verify parser busy, stale handle rejection, and doc-free clearing behavior in addition to the happy path.
- A pure `42` smoke test is necessary but insufficient on its own.

### Risk 4: Windows/MSVC build flags and smoke harness

- The plan should separate "native build plumbing" from "runtime smoke proof" so Windows failures are easier to localize.
- The Windows path needs explicit treatment of produced artifact naming and how the smoke harness loads the built library.

## Validation Architecture

Phase 2 should be planned around fast, repeated native verification rather than waiting for a later release matrix:

- **Quick loop:** `cargo test`
- **Contract loop:** `make verify-contract`
- **Build loop:** `cargo build --release`
- **Native smoke loop:** compile and run a small C harness against the committed header and produced library on `linux/amd64` and `windows/amd64`

The plan should create any missing smoke-test infrastructure early enough that every later task can reuse it.

## Suggested Plan Shape

The phase naturally breaks into a small number of plans:

1. **Native build plumbing**
   - vendored simdjson
   - `build.rs`
   - artifact generation
   - header verification still green

2. **Runtime core**
   - parser/doc registry
   - copied+padded input handling
   - real parse/root/int64 path
   - thin parser diagnostics
   - CPU fallback gate

3. **Smoke and CI proof**
   - C smoke harness
   - linux/amd64 proof
   - windows/amd64 proof
   - header/build checks wired into CI or a Phase 2 verification path

## Deliverables The Planner Should Force

- Concrete `build.rs` behavior with actual simdjson paths and flags
- A real packed-handle registry design, not "implement handle safety"
- Explicit parser/doc state transitions and verification steps
- A named hidden test-only fallback bypass mechanism
- A concrete smoke harness path and invocation strategy for both Linux and Windows

## Research Conclusion

Phase 2 should not be treated as "just get something to build." The real value is proving the locked Phase 1 contract under the minimal happy path, with enough diagnostics to debug failures and enough platform proof to de-risk MSVC early. The safest narrow plan is: real safety core now, minimal real read surface now, thin diagnostics now, Windows smoke now, and everything broader deferred to later phases.
