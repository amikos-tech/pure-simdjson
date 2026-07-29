---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
reviewed: 2026-07-29T17:58:19Z
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
  critical: 0
  warning: 2
  info: 0
  total: 2
status: issues_found
---

# Phase 11: Code Review Report

**Reviewed:** 2026-07-29T17:58:19Z
**Depth:** standard
**Files Reviewed:** 37
**Status:** issues_found

## Summary

The prior parser-side `std::bad_alloc` blocker is closed. The parser-aware
handler no longer allocates before its catch-all guard
(`src/native/simdjson_bridge.cpp:242-276`), selector `3` now executes that
production handler while checking an output sentinel
(`src/native/simdjson_bridge.cpp:311-323,1793-1800`), and the subprocess test
requires exact status `97`, one successful child test, and a post-assertion
marker (`tests/rust_shim_minimal.rs:373-410`). The focused four-test exception
suite and all 47 reviewed Rust contract tests passed independently.

The BigInt delimiter, exact-text, ownership, diagnostics, and configured-limit
contracts remain green. No new blocker was found. Two previously reported
fail-closed boundary defects remain: malformed native byte spans can panic or
leak in the Go wrapper, and the release readiness gate does not implement the
SemVer grammar it claims to validate.

## Narrative Findings (AI reviewer)

## Warnings

### WR-01 [WARNING]: Go copy-out trusts inconsistent native spans and can panic or leak

**File:** `internal/ffi/bindings.go:385-408`,
`internal/ffi/bindings_test.go:213-240,348-372`

**Issue:** After a successful native getter call, `copyElementBytes` passes
every pointer/length pair except `nil, 0` directly to `unsafe.Slice`. A
compatible but faulty native artifact can therefore crash the Go process by
returning `nil` with a non-zero length or a length that cannot be represented
by a Go slice. A non-null pointer with zero length is reported as a successful
empty value, but the deferred `BytesFree(ptr, 0)` is rejected by the ABI and
leaks the issued allocation. Current injected-binding tests cover only valid
non-empty and `nil, 0` spans, so these failure paths remain unguarded.

**Fix:** Validate the complete span before constructing the slice:

```go
if ptr == nil {
	if length == 0 {
		return "", int32(OK)
	}
	return "", int32(ErrInternal)
}

maxInt := uintptr(^uint(0) >> 1)
if length == 0 || length > maxInt {
	emitInvalidNativeSpanWarning(ptr, length)
	return "", int32(ErrInternal)
}

defer func() {
	if freeRC := b.BytesFree(ptr, length); freeRC != int32(OK) {
		emitBytesFreeFailureWarning(freeRC, length)
	}
}()
return string(unsafe.Slice(ptr, int(length))), int32(OK)
```

For malformed non-null/zero allocations, add a pointer-validated recovery path
to the allocation registry (or an opaque allocation handle) so the wrapper can
release the real allocation without trusting the bad length. Add injected
tests for nil/non-zero, non-null/zero, and oversized spans; each must return an
internal error without panicking, and recoverable allocations must be freed.

### WR-02 [WARNING]: The release gate accepts malformed versions as SemVer

**File:** `scripts/release/check_bootstrap_abi_state.py:23-27`,
`scripts/release/test_check_bootstrap_abi_state.py:147-156`

**Issue:** `SEMVER_RE` accepts invalid versions such as `0.1.5..`,
`0.1.5.foo`, and `01.01.005`, then reduces them to the same numeric tuple as a
valid release. It also rejects valid build metadata such as
`0.1.5+build.1`. Consequently, the release readiness gate can approve malformed
bootstrap versions and reject legitimate SemVer versions despite reporting
semantic-version validation. The test suite exercises only one valid
prerelease and does not protect either edge.

**Fix:** Replace the permissive expression with a strict SemVer 2.0 parser (or
a fully anchored grammar that rejects leading zeroes and empty identifiers and
accepts valid `+` build metadata). Add table-driven tests containing stable,
prerelease, and build-metadata positives plus the malformed examples above,
then use the same parser wherever release tags and bootstrap versions are
validated.

---

_Reviewed: 2026-07-29T17:58:19Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
