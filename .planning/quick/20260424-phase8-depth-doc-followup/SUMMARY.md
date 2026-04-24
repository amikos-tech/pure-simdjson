---
status: complete
completed_at: 2026-04-24T06:56:29Z
slug: phase8-depth-doc-followup
---

# Phase 8 Depth Documentation Follow-Up Summary

Resolved the remaining minor depth-limit review feedback.

## Changes

- Clarified that the C++ materializer depth guard is defense-in-depth because simdjson parser depth catches user input first today.
- Strengthened `PURE_SIMDJSON_ERR_NOT_IMPLEMENTED` and `PURE_SIMDJSON_ERR_DEPTH_LIMIT` docs to explain they split user-actionable failures from `ERR_INTERNAL`.
- Added a numeric-contract comment for `ffi.ErrDepthLimit`.
- Added a depth-1023 success boundary test. A depth-1024 fixture fails in the parser for this JSON shape, so the accepted edge is pinned to the actual current parser behavior while the depth-1025 test continues to assert `ErrDepthLimitExceeded`.

## Verification

- `cargo build --release`
- `go test ./... -run 'TestFastMaterializerDepthLimit|TestWrapStatusMapsDepthLimitSeparately' -count=1`
- `go test ./...`
- `make verify-contract`
- `python3 scripts/bench/check_phase8_tier1_improvement.py --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt`

Last action: completed implementation, verification, benchmark gate check, state update, and commit preparation.
