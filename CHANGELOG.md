# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Advisory pull-request benchmark regression check (Tier 1/2/3 on twitter and canada fixtures, advisory-only). The future blocking-flip is gated by the `REQUIRE_NO_REGRESSION` env var in `.github/workflows/pr-benchmark.yml`.

### Fixed
- Release workflow contracts now cover pull requests to `main`, sanitizer summaries only report measured probe results, and bootstrap ABI checks order prereleases below their final release.
- Wildcard navigation now separates invalid expressions from valid empty matches and clears ABI output sentinels on failure; native carrier and byte releases enforce exact pointer/count pairs.

## [0.1.7] - 2026-07-23

### Fixed
- The pinned Alpine release smoke now marks only the mounted repository and
  audited simdjson submodule paths as safe before `build.rs` verifies the
  upstream commit.

### Changed
- Bootstrap default-install recovery now pins `bootstrap.Version` to `0.1.7`
  after the immutable `v0.1.6` tag failed before artifact publication.
- This patch carries the same ABI 1.2 compatibility surface and is not the
  final v0.2 release.

## [0.1.6] - 2026-07-23

### Fixed
- The pinned Alpine release smoke now installs `git`, which the audited
  simdjson patch gate uses to verify the exact upstream submodule commit.

### Changed
- Bootstrap default-install recovery now pins `bootstrap.Version` to `0.1.6`
  after the immutable `v0.1.5` tag failed before artifact publication.
- This patch carries the ABI 1.2 BigInt, parser-limit, kernel-control, and
  diagnostic-offset surface prepared under `v0.1.5`; it is not the final v0.2
  release.

## [0.1.5] - 2026-07-23

This source state prepares `0.1.5` as an intermediate Phase 11 compatibility
artifact. It is not the final v0.2 release, and it does not claim publication:
default-bootstrap availability remains unproven until Plan 11-14 completes the
tag-driven CI and fresh-runner validation gates.

### Added
- Exact oversized-integer access through `TypeBigInt` and copied decimal text from `GetBigInt`.
- Immutable parser capacity/depth options, process-wide kernel selection diagnostics, and explicit known/unknown parse offsets.

### Changed — ABI
- The audited simdjson base is now v4.6.4 and `PURE_SIMDJSON_ABI_VERSION` is `0x00010002`, with append-only BigInt, parser-control, kernel, and offset-diagnostic surface.
- ABI 1.1 native libraries are rejected by the Phase 11 wrapper. Consumers of the native ABI must rebuild before using this compatibility artifact.

### Changed
- `NewParserPool` now accepts parser options and returns `(*ParserPool, error)`; this is a deliberate Go source break so invalid configuration fails at construction.

## [0.1.4] - 2026-04-27

### Fixed
- Linux release builds no longer depend on custom `std::basic_string` allocator behavior that is rejected by the manylinux libstdc++ toolchain.

### Changed
- Bootstrap default-install recovery now pins `bootstrap.Version` to `0.1.4` after the `v0.1.3` tag failed before artifact publication.

## [0.1.3] - 2026-04-27

### Fixed
- Linux release artifacts now hide non-public dynamic symbols so the exported surface stays limited to the public `pure_simdjson_*` ABI.
- macOS release smoke no longer reports parser diagnostic string cleanup as an untracked native allocator free.

### Changed
- Bootstrap default-install recovery now pins `bootstrap.Version` to `0.1.3` after the `v0.1.2` tag failed before artifact publication.

## [0.1.2] - 2026-04-25

### Added
- Benchmark harness coverage for Tier 1 full materialization, Tier 2 typed extraction, Tier 3 selective placeholder reads, cold/warm parser lifecycle, and the JSONTestSuite correctness oracle.
- Committed benchmark evidence under `testdata/benchmark-results/v0.1.1/`, including `phase7.bench.txt`, `coldwarm.bench.txt`, and `tier1-diagnostics.bench.txt`.
- Committed linux/amd64 benchmark baseline evidence under `testdata/benchmark-results/v0.1.1-linux-amd64/` for future CI-based claim gates.
- Committed `testdata/benchmark-results/v0.1.2/` evidence and the claim-gated `docs/benchmarks/results-v0.1.2.md` benchmark snapshot.
- Bootstrap/ABI release-state checks that reject stale bootstrap pins, Go/Rust ABI mismatches, and unknown ABI policy values before tag publication.
- Repo-root `LICENSE` and `NOTICE` files covering the project MIT license and vendored simdjson attribution.

### Changed — ABI
- `PURE_SIMDJSON_ABI_VERSION` bumped from `0x00010000` to `0x00010001`. Consumers linking against `include/pure_simdjson.h` must rebuild.
- `pure_simdjson_native_alloc_stats_t` gained two fields: `epoch` (first, so callers can detect a `pure_simdjson_native_alloc_stats_reset()` race between snapshots) and `untracked_free_count` (last, incremented when the native allocator hook observes a free for a pointer it did not record — surfaces double-frees and stray frees that previously went silent). Field order is now `[epoch, live_bytes, total_alloc_bytes, alloc_count, free_count, untracked_free_count]`, all `uint64_t`.

### Changed
- Releases now follow the org-standard tag-driven CI publish flow shared with `pure-onnx` and `pure-tokenizers`.
- Bootstrap default-install alignment now pins `bootstrap.Version` to `0.1.2` for artifacts carrying ABI `0x00010001`, while source checksums remain empty and runtime verification continues to use published `SHA256SUMS`.
- Benchmark positioning now treats Tier 1 as the full-`any` worst-case workload on the current DOM ABI, while deferring ABI-level Tier 1 improvement and any new public patch-release decision to Phase 8 and Phase 9.
- README benchmark-positioning now uses the linux/amd64 `v0.1.2` evidence and keeps third-party comparator detail in `docs/benchmarks/results-v0.1.2.md`.

### Documentation
- Added a consumer-facing `README.md` with installation, quick start, supported platforms, and a benchmark snapshot linked to `docs/benchmarks/results-v0.1.1.md`.
- Added `docs/benchmarks.md` to document tier definitions, omission rules, cold-start semantics, native allocator metrics, and rerun commands.

## [0.1.0] - 2026-04-21

### Added
- Initial bootstrap distribution pipeline, five-platform release matrix, staged smoke gates, and release-preparation scaffolding for `v0.1.0`.
