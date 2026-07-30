# Spike Manifest

## Idea

Reduce execution risk with isolated, reproducible experiments around places where the current implementation does not yet prove the planned behavior. Phase 11: upstream error locations, ABI-first symbol binding, and capacity rejection before the Rust-owned input copy. Phase 12: whether the locked Minify buffer-aliasing contract is actually memory-safe against the real vendored upstream implementation.

## Requirements

- Spike code stays under `.planning/spikes/` and must not modify production sources, the simdjson gitlink, bootstrap versions, tags, or release state.
- Upstream experiments use simdjson v4.6.4 commit `1bcf71bd85059ab6574ea1159de9298dcc1212c5`.
- Error locations are known only when simdjson itself returns a pointer inside `[input, input+len)`; no secondary parser, scanner, message parsing, or estimated byte index is allowed.
- ABI probing happens before Phase 11 mandatory-symbol binding so ABI 1.1 is classified as a version mismatch and incomplete ABI 1.2 as a corrupt surface.
- Capacity rejection happens before padding arithmetic, buffer growth, or input copying and clears stale parser diagnostics.
- Phase 12's `MinifyInto(dst, src []byte)` requires `dst` to be sized only `>= len(src)` (no `SIMDJSON_PADDING` slack) and permits `dst == src` in-place aliasing; this must hold against the real vendored implementation, not just upstream's doc comments.

## Spikes

| # | Name | Type | Validates | Verdict | Tags |
|---|------|------|-----------|---------|------|
| 001 | v464-error-location-replay | standard | Given malformed inputs and pinned simdjson v4.6.4, when an upstream-only On-Demand replay runs after DOM failure, then only stable in-range pointers become known offsets | VALIDATED | [simdjson, diagnostics, offsets, phase-11] |
| 002 | abi-first-staged-binding | standard | Given ABI 1.1, complete ABI 1.2, and incomplete ABI 1.2 libraries, when the loader probes ABI before mandatory symbols, then mismatch and corruption remain distinguishable | VALIDATED | [purego, abi, loader, phase-11] |
| 003 | pre-copy-capacity-proof | standard | Given an oversized input and stale parser diagnostics, when parsing starts, then capacity rejection occurs before Rust buffer growth/copy and clears stale details | VALIDATED | [rust, capacity, memory, diagnostics, phase-11] |
| 004 | minify-buffer-safety | standard | Given dst allocated exactly len(src) bytes, when simdjson::minify() runs directly and with dst aliasing src, then no out-of-bounds write occurs and in-place output matches the non-aliased reference | VALIDATED | [simdjson, minify, memory-safety, ffi, phase-12] |
