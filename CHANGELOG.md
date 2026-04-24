# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Benchmark harness coverage for Tier 1 full materialization, Tier 2 typed extraction, Tier 3 selective placeholder reads, cold/warm parser lifecycle, and the JSONTestSuite correctness oracle.
- Committed benchmark evidence under `testdata/benchmark-results/v0.1.1/`, including `phase7.bench.txt`, `coldwarm.bench.txt`, and `tier1-diagnostics.bench.txt`.
- Committed linux/amd64 benchmark baseline evidence under `testdata/benchmark-results/v0.1.1-linux-amd64/` for future CI-based claim gates.
- Committed `testdata/benchmark-results/v0.1.2/` evidence and the claim-gated `docs/benchmarks/results-v0.1.2.md` benchmark snapshot.
- Repo-root `LICENSE` and `NOTICE` files covering the project MIT license and vendored simdjson attribution.

### Changed — ABI
- `PURE_SIMDJSON_ABI_VERSION` bumped from `0x00010000` to `0x00010001`. Consumers linking against `include/pure_simdjson.h` must rebuild.
- `pure_simdjson_native_alloc_stats_t` gained two fields: `epoch` (first, so callers can detect a `pure_simdjson_native_alloc_stats_reset()` race between snapshots) and `untracked_free_count` (last, incremented when the native allocator hook observes a free for a pointer it did not record — surfaces double-frees and stray frees that previously went silent). Field order is now `[epoch, live_bytes, total_alloc_bytes, alloc_count, free_count, untracked_free_count]`, all `uint64_t`.

### Changed
- Releases now follow the org-standard tag-driven CI publish flow shared with `pure-onnx` and `pure-tokenizers`.
- Benchmark positioning now treats Tier 1 as the full-`any` worst-case workload on the current DOM ABI, while deferring ABI-level Tier 1 improvement and any new public patch-release decision to Phase 8 and Phase 9.
- README benchmark-positioning now uses the linux/amd64 `v0.1.2` evidence and keeps third-party comparator detail in `docs/benchmarks/results-v0.1.2.md`.

### Documentation
- Added a consumer-facing `README.md` with installation, quick start, supported platforms, and a benchmark snapshot linked to `docs/benchmarks/results-v0.1.1.md`.
- Added `docs/benchmarks.md` to document tier definitions, omission rules, cold-start semantics, native allocator metrics, and rerun commands.

## [0.1.0] - 2026-04-21

### Added
- Initial bootstrap distribution pipeline, five-platform release matrix, staged smoke gates, and release-preparation scaffolding for `v0.1.0`.
