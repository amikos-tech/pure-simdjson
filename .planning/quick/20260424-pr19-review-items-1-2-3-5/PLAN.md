---
quick_id: 20260424-pr19-review-items-1-2-3-5
slug: pr19-review-items-1-2-3-5
status: in_progress
created_at: 2026-04-24T13:00:00Z
---

# PR #19 Review Items 1, 2, 3, 5

Address the polish items from the automated Claude review on PR #19
(comment 4311664896). Items are documentation and compile-time-assert
additions — no runtime behavior change.

## Scope

- **Item 1:** Doc comment on `Bindings.InternalMaterializeBuild` at
  `internal/ffi/bindings.go:430` naming the invalidation rule (next
  `materialize_build` call on same doc frees the span).
- **Item 2:** Expand the comment at `materializer_fastpath.go:30` to call
  out the intentional LIFO defer ordering (`KeepAlive` must run before
  `Unlock` so the borrowed C++ frame span stays live through
  `buildAnyFromFrames`).
- **Item 3:** Add native-side size asserts for `psdj_internal_frame_t`
  — Rust (`const _: () = assert!(...)` in `src/runtime/mod.rs`) and
  C++ (`static_assert` in `src/native/simdjson_bridge.h`). Assertion
  expressed in terms of field widths so 32-bit builds would still pass
  if ever added. Go side already has `TestInternalFrameLayout` at
  `internal/ffi/types_test.go:8`.
- **Item 5:** Comment at `src/native/simdjson_bridge.cpp:932`
  documenting that `psimdjson_test_hold_materialize_guard` always
  returns `PARSER_BUSY` by design (test scaffolding for the reentry
  path).

Items 4 (`#[no_mangle]` rationale comment) and 6a/6b/6c are intentionally
skipped per PR-feedback analysis.

## Acceptance

- `cargo build --release` passes (Rust asserts compile).
- `go test ./...` passes (no behavior change).
- `make verify-contract` passes (C++ static_assert compiles).
- Pre-change and post-change runs of `make bench-phase7-diagnostics`
  show no material regression on the `Tier1Diagnostics` hot path.
