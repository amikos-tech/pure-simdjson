---
phase: 12
slug: high-value-dom-navigation-and-simd-utility-apis
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-07-30
---

# Phase 12 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `12-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib, no testify) for the Go layer; Rust `cargo test` (`tests/rust_shim_*.rs` integration convention) for the Rust/C++ layer |
| **Config file** | none — both frameworks use zero-config convention-based discovery |
| **Quick run command** | `go test ./... -run 'TestElement_AtPointer\|TestElement_AtPath\|TestArray_At\|TestMinify\|TestValidateUTF8'` |
| **Full suite command** | `make verify-contract && go test ./... -race` |
| **Estimated runtime** | ~60–120 seconds (full); ~5 seconds (quick) |

---

## Sampling Rate

- **After every task commit:** Run the quick command (focused `-run` pattern) plus `cargo check`
- **After every plan wave:** Run `make verify-contract` (includes `cargo test -- --test-threads=1`, header diff check, and `check_header.py` all-rules including `required-symbols`) plus `go test ./... -race`
- **Before `/gsd:verify-work`:** Full suite green, plus `make phase2-smoke-linux` and the new D-14 x86-64 ASan/UBSan step green in CI
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
| — | 12-01 | 1 | DOM-01 | — | Native `at_pointer` maps `INVALID_JSON_POINTER` to status 11, not the generic internal-error bucket | Rust integration | `cargo test --test rust_shim_navigation` | ❌ W0 | ⬜ pending |
| — | 12-01 | 1 | DOM-02 | — | Native `at_path` maps `INDEX_OUT_OF_BOUNDS` to status 12, distinguishable from a missing field | Rust integration | `cargo test --test rust_shim_navigation` | ❌ W0 | ⬜ pending |
| — | 12-02 | 2 | DOM-04 | — | Native `array_at` returns the out-of-bounds status rather than a zero-initialized view | Rust integration | `cargo test --test rust_shim_navigation` | ❌ W0 | ⬜ pending |
| — | 12-03 | 3 | DOM-03 | T-12-BULK | Bulk view-array allocation is freed exactly once; double-free and mismatched-length free are rejected via the separate `view_array_allocations` map | Rust integration | `cargo test --test rust_shim_navigation` | ❌ W0 | ⬜ pending |
| — | 12-04 | 4 | UTIL-01 | T-12-BOUNDS | Undersized `dst` is rejected at the C++ bridge boundary before `simdjson::minify` is called — never a silent truncation or overflow | C++/Rust contract | `cargo test --test rust_shim_minify -- undersized` | ❌ W0 | ⬜ pending |
| — | 12-04 | 4 | UTIL-01/02 (D-15) | T-12-CPU | Native minify/validate entry points replicate the CPU-unsupported rejection and lock kernel selection on first success | Rust integration | `cargo test --test rust_shim_minify` | ❌ W0 | ⬜ pending |
| — | 12-04 | 4 | D-14 | T-12-ALIAS | `dst == src` aliasing is memory-safe on x86-64 kernels (haswell/westmere/fallback), not just arm64 | CI-matrix (ASan/UBSan) | `bash .planning/spikes/004-minify-buffer-safety/verify.sh` as a step in `linux-smoke` | ❌ W0 | ⬜ pending |
| — | 12-05 | 5 | all | — | Every new symbol resolves at bind time; a missing export fails loudly rather than nil-dereferencing at call time | unit | `go test ./internal/ffi/...` | ❌ W0 | ⬜ pending |
| — | 12-06 | 6 | DOM-01 | — | `AtPointer` rejects malformed pointers with `ErrInvalidPath` rather than traversing | unit | `go test -run TestElement_AtPointer ./...` | ❌ W0 | ⬜ pending |
| — | 12-06 | 6 | DOM-02 | — | `AtPath` bracket-key non-quote-awareness is documented and test-pinned, not silently mis-resolved | unit | `go test -run TestElement_AtPath ./...` | ❌ W0 | ⬜ pending |
| — | 12-06 | 6 | DOM-03 | — | `AtPathAll` returns `([]Element{}, nil)` on zero matches; never a partial or unordered set | unit | `go test -run TestElement_AtPathAll ./...` | ❌ W0 | ⬜ pending |
| — | 12-07 | 6 | UTIL-01 | T-12-ALIAS | `MinifyInto` with `dst == src` produces byte-identical output to the non-aliased path | unit | `go test -run TestMinify ./...` | ❌ W0 | ⬜ pending |
| — | 12-07 | 6 | UTIL-02 | — | `ValidateUTF8` reports invalid UTF-8 as `false`, and library-load failure as a real `error` | unit | `go test -run TestValidateUTF8 ./...` | ❌ W0 | ⬜ pending |
| — | 12-07 | 6 | UTIL-01/02 (D-15) | T-12-CPU | Go entry points return `ErrCPUUnsupported` on unsupported CPUs; `SetKernel` afterward returns `ErrKernelLocked`, and the doc comments say so | unit | `go test -run 'TestMinify\|TestValidateUTF8' ./...` | ❌ W0 | ⬜ pending |
| — | 12-08 | 7 | DOM-04 | — | Out-of-range `Array.At` returns `ErrIndexOutOfRange`, never a zero-value `Element`; O(n) scan and 16,777,215 size saturation are documented | unit | `go test -run 'TestArray_At\|TestArray_Len\|TestObject_Size' ./...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*Plan/wave assignments confirmed by gsd-plan-checker against the 8 committed plans. Per-task IDs are assigned at execution time; every task in every plan already carries a concrete `<automated>` command (verified — no watch-mode flags, no 3-consecutive-task gaps).*

---

## Wave 0 Requirements

- [ ] `tests/rust_shim_navigation.rs` — new Rust integration test covering `element_at_pointer`, `element_at_path`, `element_at_path_wildcard`, `array_at`, `array_size`, `object_size` at the registry/FFI layer (follows existing `rust_shim_accessors.rs` / `rust_shim_iterators.rs` structure)
- [ ] `tests/rust_shim_minify.rs` — new Rust integration test covering overlap, empty, malformed, and undersized-`dst` cases at the FFI layer (Go-level tests cannot exercise the `dst_cap` pre-check in isolation from Go's own slice-length invariants)
- [ ] `tests/abi/check_header.py` — `REQUIRED_SYMBOLS` tuple must gain every new exported name, or `make verify-contract` fails closed
- [ ] `.github/workflows/phase2-rust-shim-smoke.yml` — new step in the `linux-smoke` job invoking the D-14 x86-64 ASan/UBSan probe
- [ ] Go test files for the new navigation / indexed-access / utility surfaces (exact file split left to the planner per CONTEXT.md discretion)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `icelake` (AVX-512) kernel aliasing safety | D-14 | GitHub-hosted `ubuntu-latest` runners do not reliably expose AVX-512; `verify.sh` skips kernels the runtime reports as unsupported, so icelake coverage is opportunistic rather than guaranteed | Run `bash .planning/spikes/004-minify-buffer-safety/verify.sh` on any AVX-512-capable x86-64 host and confirm `icelake` appears in the exercised-kernel list with zero ASan/UBSan traps |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-07-30 (gsd-plan-checker, Dimension 8 — VERIFICATION PASSED across all 8 plans)
