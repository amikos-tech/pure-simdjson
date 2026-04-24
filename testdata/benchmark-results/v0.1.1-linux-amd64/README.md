# Linux/amd64 Baseline

This directory contains the same-target baseline used by the Phase 9 benchmark claim gate.

The raw files were captured on GitHub Actions `linux/amd64` from the pre-Phase-8 baseline commit recorded in `metadata.json`. They intentionally replace the older `testdata/benchmark-results/v0.1.1/` darwin/arm64 files for old/new CI comparisons, because future benchmark gates run on GitHub Actions linux/amd64 and absolute `ns/op` values are not comparable across different CPU and OS targets.

`phase7.bench.txt` preserves the historical baseline filename expected by `scripts/bench/check_benchmark_claims.py`; it was captured with the current Tier 1/2/3 benchmark command so row names match the `v0.1.2` snapshot.
