---
status: complete
completed_at: 2026-04-24T13:45:00Z
slug: pr19-review-items-1-2-3-5
---

# PR #19 Review Items 1, 2, 3, 5 Summary

Addressed four polish items from the automated Claude review on PR #19
(comment 4311664896). All changes are documentation or compile-time
assertions — zero runtime behavior change.

## Changes

- **Item 1 (`internal/ffi/bindings.go:430`):** Added a doc comment on
  `Bindings.InternalMaterializeBuild` naming both the "C++-heap-backed
  slice" and "invalidated by next materialize_build on same doc"
  invariants, plus the caller requirement to keep the doc alive for the
  full read duration.
- **Item 2 (`materializer_fastpath.go:30`):** Expanded the existing
  one-line comment to explicitly call out the LIFO defer dependency
  between `runtime.KeepAlive(doc)` and `doc.mu.Unlock()`, and warned
  against reordering.
- **Item 3 (`src/runtime/mod.rs` + `src/native/simdjson_bridge.h`):**
  Added size assertions for `psdj_internal_frame_t` on both native
  sides. Expressed in terms of field widths
  (`16 + 4*sizeof(ptr) + 24`) rather than a hard-coded `72`, so the
  check remains truth-preserving on 32-bit targets without masking a
  real field addition. Complements the existing Go-side
  offset-by-offset `TestInternalFrameLayout`.
- **Item 5 (`src/native/simdjson_bridge.cpp:932`):** Added a comment on
  `psimdjson_test_hold_materialize_guard` documenting that it always
  returns `PURE_SIMDJSON_ERR_PARSER_BUSY` by design (test scaffolding
  that exercises the reentry guard via nested `materialize_build`
  call).

Skipped from the review: item 4 (cfg(test) rationale — low value,
inflates file), item 6a (saturating uint32→int cast — defending against
impossible 32-bit overflow adds complexity), items 6b/6c (informational).

## Verification

- `cargo build --release` — clean (Rust `const _: () = assert!(…)` compiled)
- `make verify-contract` — passed (C++ `static_assert` compiled via cargo test; ABI header, cbindgen round-trip, handle-layout C compile all clean)
- `go test ./...` — all packages pass, including `TestInternalFrameLayout`
- `make bench-phase7-diagnostics` baseline and post-change (count=5, Apple M3 Max)
  - Geomean sec/op: **-2.28%** (not a regression; within run-to-run noise at n=5)
  - B/op and allocs/op: **identical across every benchmark**
  - No statistically-confident regression on any row (p-values are either inconclusive or favor the post-change run)
- Artifacts saved:
  - `baseline.bench.txt` — pre-change raw `go test -bench` output
  - `post-change.bench.txt` — post-change raw output
  - `benchstat.txt` — benchstat diff of the two

Last action: completed four PR #19 polish items with before/after
benchmark verification; no runtime regression observed.
