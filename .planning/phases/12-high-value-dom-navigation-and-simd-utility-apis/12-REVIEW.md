---
phase: 12-high-value-dom-navigation-and-simd-utility-apis
reviewed: 2026-07-31T12:37:21Z
depth: standard
files_reviewed: 39
files_reviewed_list:
  - .github/workflows/phase2-rust-shim-smoke.yml
  - .github/workflows/phase3-go-wrapper-smoke.yml
  - cbindgen.toml
  - docs/bootstrap.md
  - docs/ffi-contract.md
  - element.go
  - element_indexed_test.go
  - element_pointer_test.go
  - errors.go
  - include/pure_simdjson.h
  - internal/bootstrap/abi_assertion.go
  - internal/bootstrap/version.go
  - internal/ffi/bindings.go
  - internal/ffi/bindings_test.go
  - internal/ffi/types.go
  - internal/ffi/types_test.go
  - kernel.go
  - library_loading_test.go
  - minify.go
  - minify_test.go
  - scripts/ci/verify_minify_buffer_safety.sh
  - scripts/release/check_bootstrap_abi_state.py
  - scripts/release/test_check_bootstrap_abi_state.py
  - scripts/release/test_release_workflow_contracts.py
  - src/lib.rs
  - src/native/simdjson_bridge.cpp
  - src/native/simdjson_bridge.h
  - src/runtime/mod.rs
  - src/runtime/registry.rs
  - tests/abi/check_header.py
  - tests/abi/handle_layout.c
  - tests/abi/test_check_header.py
  - tests/native/minify_buffer_safety_probe.cpp
  - tests/rust_shim_minify.rs
  - tests/rust_shim_navigation.rs
  - tests/rust_shim_utility_lock.rs
  - tests/smoke/ffi_export_surface.c
  - utf8.go
  - utf8_test.go
findings:
  critical: 2
  warning: 3
  info: 0
  total: 5
status: issues_found
---

# Phase 12: Code Review Report

**Reviewed:** 2026-07-31T12:37:21Z
**Depth:** standard
**Files Reviewed:** 39
**Status:** issues_found

## Narrative Findings (AI reviewer)

The phase adds the intended ABI 1.3 navigation and utility surface, but two fail-closed contracts are not actually upheld. The minify export loses the required capacity exactly when callers need it, and the Go loader accepts an ABI 1.3 library with missing public symbols despite the normative mandatory-surface rule. Stateful iterator constructors also leak leases on a rejected call, while the contract CI is both under-triggered and dependent on an unpinned generator version.

The broad suites passed (`cargo test --release`, `go test ./... -race`, `make verify-contract`, the release-script unit tests, and the ASan/UBSan minify probe). Focused calls still reproduced CR-01 and WR-01, showing that the passing suites do not cover those error paths.

## Critical Issues

### CR-01: Minify discards the required capacity on `BUFFER_TOO_SMALL`

**Classification:** BLOCKER

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/src/runtime/mod.rs:328`

**Related:** `/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:442`, `/Users/tazarov/experiments/amikos/pure-simdjson/src/native/simdjson_bridge.cpp:1130`, `/Users/tazarov/experiments/amikos/pure-simdjson/tests/rust_shim_minify.rs:191`, `/Users/tazarov/experiments/amikos/pure-simdjson/docs/ffi-contract.md:257`

**Issue:** The native bridge writes `src_len` to its local `out_written` and returns status 6 when `dst_cap < src_len`. `runtime::native_minify` then converts every non-success result to `Err(rc)`, throwing the size away, and the public export writes the caller's `out_written` only in the `Ok` branch. A direct call with an 8-byte source and a 7-byte destination returned `rc=6` while leaving the caller's sentinel `out_written=SIZE_MAX`; the normative contract requires `out_written=8`. The existing undersized-buffer test initializes a sentinel but never asserts it, so it passes while the ABI is broken.

**Fix:** Preserve the native status and size separately, and publish the size for success and status 6 only. For example:

```rust
pub(crate) unsafe fn native_minify(/* ... */) -> (pure_simdjson_error_code_t, usize) {
    let mut written = 0;
    let rc = unsafe { psimdjson_minify(src_ptr, src_len, dst_ptr, dst_cap, &mut written) };
    (rc, written)
}

let (rc, written) = runtime::native_minify(src_ptr, src_len, dst_ptr, dst_cap);
if rc == err_ok() || rc == err_buffer_too_small() {
    ptr::write(out_written, written);
}
rc
```

Add `assert_eq!(written, input.len())` to the undersized Rust test and exercise the same status/size pair through the dynamic C smoke harness.

### CR-02: ABI 1.3 loader silently downgrades two mandatory public symbols

**Classification:** BLOCKER

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/internal/ffi/bindings.go:148`

**Related:** `/Users/tazarov/experiments/amikos/pure-simdjson/library_loading_test.go:273`, `/Users/tazarov/experiments/amikos/pure-simdjson/docs/ffi-contract.md:307`, `/Users/tazarov/experiments/amikos/pure-simdjson/tests/abi/check_header.py:48`

**Issue:** `pure_simdjson_native_alloc_stats_reset` and `pure_simdjson_native_alloc_stats_snapshot` are members of the earlier public ABI surface and of the header checker's required-symbol set. The normative ABI handshake says that ABI 1.3 makes the complete earlier surface mandatory and that a missing wrapper-required symbol must fail closed before cache installation. `bindWithRegistrar`, however, resolves both with `registerOptionalFuncWithRegistrar`, clears them when either is absent, and still returns a successful `Bindings`. The loader's mandatory-symbol fixture also omits both names, so an artifact claiming ABI 1.3 can be cached even though its advertised public surface is incomplete.

**Fix:** Bind both exports in the mandatory `symbols` table:

```go
{name: "pure_simdjson_native_alloc_stats_reset", target: &b.nativeAllocStatsReset},
{name: "pure_simdjson_native_alloc_stats_snapshot", target: &b.nativeAllocStatsSnapshot},
```

Remove the optional downgrade for these public symbols, add both names to `abi13MandatoryFixtureSymbols`, and add missing-symbol cases proving that either omission fails binding and leaves `cachedLibrary` nil. Keep only `psdj_internal_*` facilities optional.

## Warnings

### WR-01: Rejected iterator construction consumes an unreachable lease

**Classification:** WARNING

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:960`

**Related:** `/Users/tazarov/experiments/amikos/pure-simdjson/src/lib.rs:1006`, `/Users/tazarov/experiments/amikos/pure-simdjson/src/runtime/registry.rs:1051`, `/Users/tazarov/experiments/amikos/pure-simdjson/src/runtime/registry.rs:1119`

**Issue:** Both iterator constructors allocate and register a lease before `write_out` discovers that `out_iter` is null. The call correctly returns `INVALID_ARGUMENT`, but the lease remains in the owning document's `iter_leases` map and can never be used or released independently. A focused call to `array_iter_new(view, NULL)` followed by one valid construction produced lease id 2 instead of 1. Repeating the rejected call grows document-owned bookkeeping until `Doc` is freed and can eventually exhaust lease space or memory.

**Fix:** Validate `out_iter` before invoking either registry constructor:

```rust
if out_iter.is_null() {
    return err_invalid_argument();
}
```

Add regression coverage for both array and object constructors that observes the test-only lease count before and after a null-output call.

### WR-02: Contract workflow does not trigger for files that define its contract checks

**Classification:** WARNING

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/phase2-rust-shim-smoke.yml:5`

**Issue:** The workflow runs `make verify-contract`, which generates the header with `cbindgen.toml` and executes `tests/abi/**`, but its `push.paths` list includes neither `cbindgen.toml` nor `tests/abi/**`. A config-only ABI generation change or a broken header-checker change therefore skips the very gate intended to validate it. This makes contract coverage depend on an unrelated source file changing in the same push.

**Fix:** Add at least these inputs to the trigger and mirror the gate on pull requests:

```yaml
on:
  push:
    paths:
      - cbindgen.toml
      - tests/abi/**
      # existing paths...
  pull_request:
    paths:
      - cbindgen.toml
      - include/pure_simdjson.h
      - src/**
      - tests/abi/**
```

### WR-03: Contract generation installs an unpinned cbindgen release

**Classification:** WARNING

**File:** `/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/phase2-rust-shim-smoke.yml:41`

**Issue:** `cargo install cbindgen --locked` selects the newest cbindgen release at job time. `--locked` only fixes that selected release's dependency graph; it does not pin the cbindgen version. A new cbindgen release can change generated formatting or raise its minimum Rust version, causing the committed-header check to start failing without any repository change.

**Fix:** Pin the tested generator release, for example `cargo install cbindgen --version <tested-version> --locked`, and update that pin deliberately together with a regenerated header.

---

_Reviewed: 2026-07-31T12:37:21Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
