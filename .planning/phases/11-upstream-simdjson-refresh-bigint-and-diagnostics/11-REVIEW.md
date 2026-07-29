---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
reviewed: 2026-07-29T15:53:33Z
depth: standard
files_reviewed: 37
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
  - testdata/jsontestsuite/cases/n_array_bigint_suffix_plus.json
  - testdata/jsontestsuite/cases/n_number_bigint_negative_suffix_underscore.json
  - testdata/jsontestsuite/cases/n_number_bigint_positive_suffix_x.json
  - testdata/jsontestsuite/cases/n_object_bigint_suffix_slash.json
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
  critical: 1
  warning: 2
  info: 0
  total: 3
status: issues_found
---

# Phase 11: Code Review Report

**Reviewed:** 2026-07-29T15:53:33Z
**Depth:** standard
**Files Reviewed:** 37
**Status:** issues_found

## Summary

The BigInt delimiter fix is present in all nine generated architecture
implementations, and the root, nested, malformed-suffix, and exact-text
regressions pass. The `std::bad_alloc` mapper now returns status `97` through
the generic test seam. However, the production parser-aware exception path can
still terminate the process while trying to record that same allocation
failure, so the normative exception contract is not closed. Two existing
fail-closed boundary defects also remain in the Go copy-out and release-version
validation paths.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01 [BLOCKER]: Parser-side `bad_alloc` handling can call `std::terminate` instead of returning status 97

**File:** `src/native/simdjson_bridge.cpp:274-300`,
`tests/rust_shim_minimal.rs:346-358`

**Issue:** `capture_parser_exception` is declared `noexcept`, but its
`std::string("std::bad_alloc: ") + error.what()` argument is constructed before
control enters `try_set_last_error_message` and its catch-all guard. That
construction can allocate and throw another `std::bad_alloc`; because it escapes
the `noexcept` function, C++ invokes `std::terminate`. This is particularly
likely while already handling allocation exhaustion from the real parser path.
The parser catch macro calls this diagnostic helper before `map_cpp_exception`,
so status `PURE_SIMDJSON_ERR_CPP_EXCEPTION` (`97`) is never returned in that
case.

The new forced-exception tests do not cover this path. They call
`psimdjson_test_force_cpp_exception`, which uses
`PSIMDJSON_CATCH_CPP_EXCEPTIONS`; production parsing uses the distinct
`PSIMDJSON_CATCH_PARSER_CPP_EXCEPTIONS` macro containing the unsafe diagnostic
capture.

**Fix:**

```cpp
void capture_parser_exception(
    psimdjson_parser *parser,
    const std::bad_alloc &
) noexcept {
  // Any allocation performed by LastErrorBuffer::assign is now inside the
  // catch-all guard; do not allocate while forming this argument.
  try_set_last_error_message(parser, "std::bad_alloc");
}
```

Alternatively, wrap the entire string construction and assignment in a local
`try`/`catch (...)`. Add a fault-injection test that throws `std::bad_alloc`
inside an entry point using `PSIMDJSON_CATCH_PARSER_CPP_EXCEPTIONS`, and assert
that the process stays alive, the output sentinel is preserved, and status
`97` reaches the Rust ABI.

## Warnings

### WR-01 [WARNING]: Go copy-out trusts inconsistent native spans and can panic or leak

**File:** `internal/ffi/bindings.go:385-408`

**Issue:** `copyElementBytes` accepts every successful pointer/length pair
except the valid empty sentinel as input to `unsafe.Slice`. If a compatible but
faulty native artifact returns `ptr == nil && length > 0`, this panics inside
the Go process. If it returns a non-null pointer with zero length, the method
reports success but its deferred `BytesFree(ptr, 0)` is rejected, leaking the
allocation. An unrepresentable length can likewise panic during slice
construction. The FFI boundary should reject inconsistent spans rather than
turning a native contract violation into a Go crash or silent leak.

**Fix:**

```go
if ptr == nil {
	if length == 0 {
		return "", int32(OK)
	}
	return "", int32(ErrInternal)
}
defer func() {
	if freeRC := b.BytesFree(ptr, length); freeRC != int32(OK) {
		emitBytesFreeFailureWarning(freeRC, length)
	}
}()
maxInt := int(^uint(0) >> 1)
if length == 0 || length > uintptr(maxInt) {
	return "", int32(ErrInternal)
}
return string(unsafe.Slice(ptr, int(length))), int32(OK)
```

Add injected-binding tests for nil/nonzero, nonnil/zero, and oversized spans,
asserting an internal error without a panic.

### WR-02 [WARNING]: The release gate accepts malformed versions as SemVer

**File:** `scripts/release/check_bootstrap_abi_state.py:23-27`

**Issue:** `SEMVER_RE` accepts invalid semantic versions such as `0.1.5..`,
`0.1.5.foo`, and `01.01.005`, while rejecting valid build metadata such as
`0.1.5+build.1`. A malformed bootstrap version can therefore pass the ABI
readiness check even though the command and diagnostics promise semantic
version validation, while a valid SemVer identifier can be rejected.

**Fix:** Use a strict SemVer 2.0 parser or a fully anchored grammar that rejects
leading zeroes, validates prerelease identifiers, and accepts build metadata
when the release policy allows it. Add positive and negative table-driven tests
and use the same parser for tag/readiness validation.

---

_Reviewed: 2026-07-29T15:53:33Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
