---
phase: 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi
status: passed
verified_at: 2026-04-23
verified_by: codex
requirements:
  - D-01
  - D-02
  - D-03
  - D-04
  - D-05
  - D-06
  - D-07
  - D-08
  - D-09
  - D-10
  - D-11
  - D-12
  - D-13
  - D-14
  - D-15
  - D-16
  - D-17
---

# Phase 8 Verification

## Verdict

Status: passed

Phase 8 delivers the roadmap goal from `.planning/ROADMAP.md`: the repo now has a lower-overhead internal DOM traversal/materialization substrate, an unexported Go fast `any` materializer over one native frame-stream handoff, Tier 1 benchmark wiring through that path, same-host benchmark evidence committed under `testdata/benchmark-results/phase8/`, and an explicit internal handoff that defers public benchmark positioning to Phase 9.

## Evidence

Fresh verification was run on 2026-04-23 from `gsd/phase-08-low-overhead-dom-traversal-abi-and-specialized-go-any-materializer`.

- `python3 tests/abi/test_check_header.py` passed.
- `python3 tests/bench/test_check_phase8_improvement.py` passed.
- `go test ./...` passed.
- `go test ./... -run 'TestFastMaterializer|TestJSONTestSuiteOracle' -count=1` passed.
- `cargo test -- --test-threads=1` passed.
- `make verify-contract` passed.
- `go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=5 -timeout 1200s` completed and wrote `testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt`.
- `scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` completed and wrote `testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt`.
- `python3 scripts/bench/check_phase8_tier1_improvement.py --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` passed with six PASS lines and no FAIL lines.
- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-REVIEW.md` records a clean post-execution code review.

## Requirement Coverage

| Requirement | Status | Evidence |
| --- | --- | --- |
| D-01 | passed | `materializer_fastpath.go` routes Tier 1 `any` materialization through one internal frame-stream handoff. |
| D-02 | passed | `tests/abi/check_header.py`, `tests/abi/test_check_header.py`, and `make verify-contract` keep internal `psdj_internal_` / `psimdjson_` symbols out of the public header. |
| D-03 | passed | `cbindgen.toml` exclusions plus the header-audit rule preserve the internal/private ABI split. |
| D-04 | passed | `src/runtime/registry.rs` validates root and descendant `ValueView` handles once before native traversal begins. |
| D-05 | passed | Oversized integer literals now normalize to parse-time `ERR_INVALID_JSON`, and the public/internal header surface remains stable. |
| D-06 | passed | `fastMaterializeElement` copies keys and strings into Go-owned values at the escape boundary. |
| D-07 | passed | Duplicate-key full materialization is covered by the active fast-materializer test suite and matches the locked last-wins semantics. |
| D-08 | passed | Root and subtree frame/materializer paths are covered by Rust integration tests and Go parity tests. |
| D-09 | passed | Closed-doc and stale-handle cases are covered in Rust and Go tests and return typed errors. |
| D-10 | passed | Busy/reentrant materialization is guarded deterministically through `materialize_in_progress` and `ErrParserBusy`. |
| D-11 | passed | Exact numeric kind preservation is covered by `InternalFrame` layout tests plus Go parity/numeric tests. |
| D-12 | passed | Full-frame consumption and container child-count handling are enforced by the Go builder and its active tests. |
| D-13 | passed | String lifetime after `Doc.Close()` and GC is pinned by `TestFastMaterializerStringOwnershipAfterCloseAndGC`. |
| D-14 | passed | No new public `Interface()`-style API or public benchmark claim surface was added in Phase 8. |
| D-15 | passed | Tier 1 full/materialize-only benchmark helpers and diagnostic row names are wired through the fast path with stable labels. |
| D-16 | passed | `scripts/bench/check_phase8_tier1_improvement.py` proves all six required rows improved by at least `10%` on matching host metadata. |
| D-17 | passed | `08-BENCHMARK-NOTES.md` records the evidence internally and explicitly defers README/result updates and release decisions to Phase 9. |

## Residual Notes

Phase 8 intentionally leaves public benchmark messaging untouched even though the internal diagnostic evidence is now strong. That remaining work is Phase 9 scope, not verification debt. The committed Phase 8 evidence files and notes are sufficient input for that next phase.
