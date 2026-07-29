---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
reviewed: 2026-07-29T13:59:08Z
depth: standard
files_reviewed: 33
files_reviewed_list:
  - docs/bootstrap.md
  - docs/concurrency.md
  - docs/ffi-contract.md
  - include/pure_simdjson.h
  - internal/bootstrap/abi_assertion.go
  - internal/bootstrap/checksums.go
  - internal/bootstrap/version.go
  - internal/ffi/bindings.go
  - internal/ffi/bindings_test.go
  - internal/ffi/types.go
  - internal/ffi/types_test.go
  - patches/simdjson-v4.6.4-positive-bigint.patch
  - scripts/release/check_bootstrap_abi_state.py
  - scripts/release/test_check_bootstrap_abi_state.py
  - scripts/release/test_public_bootstrap_validation_contracts.py
  - scripts/release/test_release_workflow_contracts.py
  - src/lib.rs
  - src/native/simdjson_bridge.cpp
  - src/native/simdjson_bridge.h
  - src/runtime/mod.rs
  - src/runtime/registry.rs
  - testdata/jsontestsuite/expectations.tsv
  - tests/abi/check_header.py
  - tests/abi/handle_layout.c
  - tests/abi/test_check_header.py
  - tests/rust_shim_bigint.rs
  - tests/rust_shim_diagnostics.rs
  - tests/rust_shim_fast_materializer.rs
  - tests/rust_shim_kernel.rs
  - tests/rust_shim_limits.rs
  - tests/rust_shim_minimal.rs
  - tests/smoke/ffi_export_surface.c
  - tests/smoke/go_bootstrap_smoke.go
findings:
  critical: 2
  warning: 2
  info: 0
  total: 4
status: issues_found
---

# Phase 11: Code Review Report

**Reviewed:** 2026-07-29T13:59:08Z
**Depth:** standard
**Files Reviewed:** 33
**Status:** issues_found

## Summary

The configured-capacity/depth checks, 1024-depth boundary, diagnostic offset
sentinel, subprocess crash regression, generated header, and cross-language ABI
layouts all passed their existing debug and optimized test paths. A targeted
probe against the optimized shared library nevertheless found a release-blocking
BigInt validation bypass: malformed JSON is accepted and truncated. The review
also found a normative exception-status mismatch and two fail-closed robustness
gaps.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01 [BLOCKER]: BigInt handling accepts malformed numeric tokens and discards their suffix

**File:** `patches/simdjson-v4.6.4-positive-bigint.patch:18-75`,
`src/native/simdjson_bridge.cpp:740`

**Issue:** The patch changes positive `uint64` overflow from
`INVALID_NUMBER(src)` to `BIGINT_NUMBER(src)`, and the bridge enables
`number_as_string(true)`. In simdjson v4.6.4, the BigInt storage path scans and
copies only the optional sign plus digits, then returns success without applying
the normal structural-or-whitespace check to the next byte. Consequently,
malformed tokens such as `18446744073709551616x`,
`-9223372036854775809x`, and `[123456789012345678901x]` parse successfully.

A probe against `target/release/libpure_simdjson.dylib` returned status `0`,
kind `9`, and text `18446744073709551616` for input
`18446744073709551616x`; the invalid `x` was silently discarded. This lets
invalid JSON pass a validation boundary and changes the caller's data.

**Fix:** Extend the audited upstream patch so every generated BigInt storage
path rejects a non-delimiter after the digit scan before appending the BigInt:

```cpp
const uint8_t *p = value;
if (*p == '-') {
  ++p;
}
while (numberparsing::is_digit(*p)) {
  ++p;
}
if (jsoncharutils::is_not_structural_or_whitespace(*p)) {
  return simdjson::NUMBER_ERROR;
}
```

Apply the fix to every generated architecture copy, update the patch-integrity
assertion as needed, and add root and nested regression cases for positive,
negative, very long, and invalid-UTF-8 BigInt suffixes. Each must return
`PURE_SIMDJSON_ERR_INVALID_JSON`.

### CR-02 [BLOCKER]: `std::bad_alloc` violates the normative C++ exception status contract

**File:** `src/native/simdjson_bridge.cpp:253-259`,
`docs/ffi-contract.md:290-292`

**Issue:** The normative ABI contract says trapped C++ exceptions are converted
to `PURE_SIMDJSON_ERR_CPP_EXCEPTION` (`97`), but the dedicated
`std::bad_alloc` overload returns `PURE_SIMDJSON_ERR_INTERNAL` (`127`).
Callers therefore receive the wrong stable ABI status for one entire class of
C++ exceptions, and the existing forced-exception test does not exercise this
overload.

**Fix:** Preserve the documented exception classification:

```cpp
pure_simdjson_error_code_t map_cpp_exception(
    const char *function_name,
    const std::bad_alloc &error
) noexcept {
  log_cpp_exception(function_name, error.what());
  return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
}
```

Add a test-only `std::bad_alloc` throw path and assert that the public Rust ABI
returns `PURE_SIMDJSON_ERR_CPP_EXCEPTION`. If allocation failures are
intentionally meant to be `INTERNAL`, change the normative contract and all
consumer expectations in the same ABI decision rather than leaving the two
definitions inconsistent.

## Warnings

### WR-01 [WARNING]: Go copy-out trusts an inconsistent native span and can panic or leak

**File:** `internal/ffi/bindings.go:385-408`

**Issue:** `copyElementBytes` checks only the valid empty sentinel
`ptr == nil && length == 0`. If an otherwise ABI-compatible artifact returns
success with `ptr == nil && length > 0`, `unsafe.Slice` panics. If it returns a
non-null pointer with zero length, the method reports success and its deferred
`BytesFree(ptr, 0)` fails, leaking the allocation. This boundary already treats
the native length and pointer as untrusted in other span-returning helpers and
should fail closed here as well.

**Fix:** Validate the pair before installing the defer or constructing a slice:

```go
if ptr == nil {
	if length == 0 {
		return "", int32(OK)
	}
	return "", int32(ErrInternal)
}
if length == 0 || length > uintptr(maxInt) {
	return "", int32(ErrInternal)
}
```

Add injected-binding tests for both inconsistent pairs and an oversized length,
asserting an internal error with no panic.

### WR-02 [WARNING]: The release gate labels a loose version grammar as SemVer

**File:** `scripts/release/check_bootstrap_abi_state.py:23-27`

**Issue:** `SEMVER_RE` accepts invalid semantic versions such as `0.1.5..`,
`0.1.5.foo`, and `01.01.005`, while rejecting valid build metadata such as
`0.1.5+build.1`. This can admit malformed release identifiers or reject a valid
SemVer tag even though the command and diagnostics promise semantic-version
validation.

**Fix:** Use a strict SemVer 2.0 parser or a fully anchored grammar that rejects
leading zeroes, validates prerelease identifiers, and optionally accepts
`+build` metadata. Add positive and negative table-driven tests, and keep the
tag/readiness gate grammars synchronized with this canonical check.

---

_Reviewed: 2026-07-29T13:59:08Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
