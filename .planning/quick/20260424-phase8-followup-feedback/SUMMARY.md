---
status: complete
completed_at: 2026-04-24T06:20:45Z
slug: phase8-followup-feedback
---

# Phase 8 Follow-Up Feedback Summary

Applied the convergent follow-up feedback for Phase 8 materializer observability and test coverage.

## Changes

- Added `PURE_SIMDJSON_ERR_DEPTH_LIMIT`, `ffi.ErrDepthLimit`, and `ErrDepthLimitExceeded`.
- Mapped native parse/materializer depth failures to the new sentinel instead of generic invalid/internal errors.
- Added a 1025-deep array regression test and depth-limit status mapping coverage.
- Added the missing adversarial string-frame nil-pointer subtest.
- Reset the fast-path fallback warning `sync.Once` via `t.Cleanup`.
- Expanded the `Doc.isClosed` TryLock comment to name the fast-path self-deadlock hazard.
- Tightened comments for materializer depth, bool flags, not-implemented/depth enum contracts, and saturated scope-count reserve handling.

## Verification

- `cargo build --release`
- `go test ./... -run 'TestWrapStatusMapsDepthLimitSeparately|TestFastMaterializerDepthLimitExceeded|TestFastMaterializerRejectsAdversarialFrameStreams|TestFastMaterializerUnavailableWarningIsOneShotDebugLog' -count=1`
- `go test ./...`
- `cargo test materialize -- --test-threads=1`
- `make verify-contract`
- `python3 scripts/bench/check_phase8_tier1_improvement.py --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt`
- Fresh Tier 1 diagnostics sample in `/tmp/pure-simdjson-phase8-followup-tier1.bench.txt` passed the Phase 8 improvement gate.

Last action: completed implementation, verification, benchmark gate checks, state update, and commit preparation.
