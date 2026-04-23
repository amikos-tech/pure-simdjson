---
phase: 8
slug: low-overhead-dom-traversal-abi-and-specialized-go-any-materializer
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-23
---

# Phase 8 - Validation Strategy

Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| Framework | Go test/bench, Cargo test, Python ABI header audit |
| Config file | `go.mod`, `Cargo.toml`, `Makefile`, `cbindgen.toml` |
| Quick run command | `go test ./...` |
| Native run command | `cargo test -- --test-threads=1` |
| Contract run command | `make verify-contract` |
| Full suite command | `go test ./... && cargo test -- --test-threads=1 && make verify-contract` |
| Benchmark command | `go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=5 > testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` |
| Benchmark delta command | `scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` |
| Estimated runtime | ~60-180 seconds for full non-benchmark suite locally; benchmarks vary by host |

---

## Sampling Rate

- After every Go-facing task commit: run `go test ./...`.
- After every Rust/C++ FFI task commit: run `cargo test -- --test-threads=1`.
- After every public or internal ABI task commit: run `make verify-contract`.
- After every plan wave: run `go test ./... && cargo test -- --test-threads=1 && make verify-contract`.
- Before phase verification: full suite must be green and Tier 1 diagnostics must be captured with `-benchmem -count=5`.
- Max feedback latency: 3 task commits without a full suite is not allowed.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 08-W0-01 | 01 | 1 | D-01, D-06, D-07, D-10, D-11, D-13 | T-08-01, T-08-03 | Fast materializer parity preserves value shape, numeric kinds, duplicate-key last-wins materialization, and typed errors. | Go integration | `go test ./... -run 'TestFastMaterializer'` | no - Wave 0 creates | pending |
| 08-W0-02 | 01 | 1 | D-08, D-09 | T-08-01 | Borrowed native bytes do not escape; materialized Go strings survive `Doc.Close` and GC pressure. | Go integration | `go test ./... -run 'TestFastMaterializer.*(Lifetime|Close|GC)'` | no - Wave 0 creates | pending |
| 08-W0-03 | 01 | 1 | D-02, D-03, D-05, D-14 | T-08-02, T-08-05 | Internal fast-path symbols stay out of the public generated header and existing public ABI symbols remain stable. | contract | `make verify-contract` | partial - existing guard, update likely needed | pending |
| 08-W0-04 | 01 | 1 | D-04 | T-08-01 | Root and subtree materialization share the same validation rules and return equivalent Go-owned trees. | Go integration | `go test ./... -run 'TestFastMaterializerSubtree'` | no - Wave 0 creates if subtree lands | pending |
| 08-W1-01 | 02 | 1 | D-01, D-02, D-03, D-05 | T-08-02, T-08-05 | Internal frame ABI validates document/view generation once and returns status/out-param errors without callbacks or struct-by-value returns. | Rust/C++ contract | `cargo test -- --test-threads=1 && make verify-contract` | pending implementation | pending |
| 08-W2-01 | 03 | 2 | D-06, D-07, D-08, D-09, D-10, D-11, D-12, D-13 | T-08-01, T-08-03, T-08-04 | Go builder copies keys/strings only at the final value boundary, preallocates containers from metadata, and maps all failures through existing typed errors. | Go integration | `go test ./... -run 'TestFastMaterializer'` | pending implementation | pending |
| 08-W3-01 | 04 | 3 | D-15, D-16, D-17 | T-08-04 | Benchmarks prove materialize-only and full Tier 1 improvement over Phase 7 without changing public benchmark claims. | benchmark | `scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` | pending implementation | pending |

---

## Wave 0 Requirements

- [ ] `materializer_fastpath_test.go` - parity, duplicate-key, numeric preservation, string ownership, closed-doc behavior, and optional subtree coverage.
- [ ] `internal/ffi` frame layout tests - verify Go frame struct size/offsets against native constants if the internal ABI exposes fixed-layout frames.
- [ ] `tests/abi/check_header.py` extension - assert Phase 8 internal symbols are absent from `include/pure_simdjson.h`.
- [ ] `testdata/benchmark-results/phase8/` - raw Phase 8 benchmark output destination for diagnostics and benchstat comparison.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Public README benchmark story is not repositioned in Phase 8 | D-17 | Phase 9 owns public benchmark-story updates, but accidental README/docs drift requires human review. | Review `git diff -- README.md docs/benchmarks.md docs/benchmarks/results-v0.1.1.md` and confirm any Phase 8 edits are internal notes or raw evidence only. |
| Benchmark improvement is interpreted without inventing a release claim | D-15, D-16, D-17 | Benchstat can show deltas, but release/readme positioning remains a Phase 9 decision. | Confirm closeout states materialize-only/full Tier 1 deltas and explicitly defers public benchmark positioning to Phase 9. |

---

## Threat Model

| Ref | Threat | Mitigation | Verification |
|-----|--------|------------|--------------|
| T-08-01 | Borrowed native bytes are read after `Doc.Close` or after the parser/doc finalizer releases native storage. | Borrowed spans stay internal to one materializer call; Go strings are copied before return; `runtime.KeepAlive(doc)` runs after the final borrowed read. | `go test ./... -run 'TestFastMaterializer.*(Lifetime|Close|GC)'` |
| T-08-02 | Internal fast-path symbols leak into the public generated C header and become accidental public ABI. | Use internal symbol naming, keep symbols excluded from cbindgen public output, and extend header audit. | `make verify-contract` |
| T-08-03 | Numeric precision or range behavior changes while building `any` values. | Preserve frame tags for int64, uint64, and float64; route range/precision failures through existing error mapping. | `go test ./... -run 'TestFastMaterializer.*Numeric'` |
| T-08-04 | Benchmark delta is invalid because cached materialized Go trees are reused. | Rebuild maps, slices, and strings on every materializer call; reuse only safe native/frame scratch that does not skip traversal or final Go allocation. | Review materializer implementation and compare `-benchmem` output against Phase 7 diagnostics. |
| T-08-05 | Internal FFI signatures drift into platform-unsafe calling convention shapes. | Use pointer/integer out params, no callbacks, no struct-by-value returns, and fixed layout tests for any mirrored structs. | `cargo test -- --test-threads=1 && make verify-contract` |

---

## Validation Sign-Off

- [x] All phase decisions D-01 through D-17 have at least one automated or manual validation route.
- [x] Sampling continuity requires full-suite coverage at every wave boundary.
- [x] Wave 0 identifies missing test and benchmark scaffolding before implementation.
- [x] No watch-mode flags are used.
- [x] Feedback latency target is defined.
- [x] `nyquist_compliant: true` set in frontmatter.

Approval: pending implementation verification
