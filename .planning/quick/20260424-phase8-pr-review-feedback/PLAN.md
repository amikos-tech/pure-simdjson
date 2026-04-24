---
quick_id: 20260424
slug: phase8-pr-review-feedback
status: in_progress
created_at: 2026-04-24T04:19:56Z
---

# Phase 8 PR Review Feedback

Review and apply Phase 8 PR feedback for the low-overhead DOM traversal ABI and specialized Go `any` materializer.

## Scope

- Address critical correctness and observability findings before merge.
- Add focused unsafe-boundary coverage for frame protocol violations and guard/span contracts.
- Preserve the Phase 8 benchmark improvement; run the existing Tier 1 improvement gate after changes.

## Tasks

1. Add tests for optional-symbol observability, unavailable telemetry status, fast-path fallback logging, frame protocol failures, guard release, and span invalidation/lifetime contracts.
2. Implement explicit native frame-builder depth limiting and document doc-owned frame scratch invalidation.
3. Improve Go-side error breadcrumbs for malformed frame streams without changing public sentinel matching.
4. Distinguish native allocator telemetry unavailability from internal errors.
5. Apply low-risk comment/tooling fixes from the review list.
6. Run focused tests plus the Phase 8 benchmark regression gate.

## Acceptance

- `go test ./... -run 'TestFastMaterializer|TestNativeAllocStats|TestRegisterOptionalFunc|TestWrapStatus|TestAccessorMaterializerParity' -count=1` passes.
- `cargo test --test rust_shim_fast_materializer -- --test-threads=1` passes.
- `cargo test materialize -- --test-threads=1` passes.
- `python3 tests/abi/test_check_header.py` passes.
- `python3 tests/bench/test_check_phase8_improvement.py` passes.
- `python3 scripts/bench/check_phase8_tier1_improvement.py --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` passes.
