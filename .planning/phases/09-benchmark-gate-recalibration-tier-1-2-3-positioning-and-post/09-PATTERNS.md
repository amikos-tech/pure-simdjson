# Phase 9: Benchmark Gate Recalibration, Tier 1/2/3 Positioning, and Post-ABI Evidence Refresh - Pattern Map

**Mapped:** 2026-04-24
**Files analyzed:** 15
**Analogs found:** 14 / 15

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `.github/workflows/benchmark-capture.yml` | config | batch | `.github/workflows/public-bootstrap-validation.yml` + `.github/workflows/release.yml` | role-match |
| `scripts/bench/check_benchmark_claims.py` | utility | transform | `scripts/bench/check_phase8_tier1_improvement.py` | exact |
| `tests/bench/test_check_benchmark_claims.py` | test | transform | `tests/bench/test_check_phase8_improvement.py` | exact |
| `scripts/bench/capture_release_snapshot.sh` | utility | batch | `scripts/bench/run_benchstat.sh` | role-match |
| `testdata/benchmark-results/v0.1.2/phase9.bench.txt` | testdata | batch | `testdata/benchmark-results/v0.1.1/phase7.bench.txt` | exact |
| `testdata/benchmark-results/v0.1.2/coldwarm.bench.txt` | testdata | batch | `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt` | exact |
| `testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt` | testdata | batch | `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt` | exact |
| `testdata/benchmark-results/v0.1.2/phase9.benchstat.txt` | testdata | transform | `testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt` | role-match |
| `testdata/benchmark-results/v0.1.2/coldwarm.benchstat.txt` | testdata | transform | `testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt` | role-match |
| `testdata/benchmark-results/v0.1.2/tier1-diagnostics.benchstat.txt` | testdata | transform | `testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt` | exact |
| `testdata/benchmark-results/v0.1.2/summary.json` | testdata | transform | `testdata/benchmark-results/phase8/tier1-diagnostics.improvement.txt` | partial |
| `docs/benchmarks.md` | documentation | request-response | `docs/benchmarks.md` | exact |
| `docs/benchmarks/results-v0.1.2.md` | documentation | request-response | `docs/benchmarks/results-v0.1.1.md` | exact |
| `README.md` | documentation | request-response | `README.md` | exact |
| `CHANGELOG.md` | documentation | event-driven | `CHANGELOG.md` | exact |

## Pattern Assignments

### `.github/workflows/benchmark-capture.yml` (config, batch)

**Analog:** `.github/workflows/public-bootstrap-validation.yml`; supplement artifact upload from `.github/workflows/release.yml`.

**Dispatch and least-privilege permissions pattern** (`.github/workflows/public-bootstrap-validation.yml` lines 1-21):

```yaml
name: public bootstrap validation

on:
  workflow_dispatch:
    inputs:
      version:
        description: published version without the leading v, for example 0.1.0
        required: true
        type: string
  schedule:
    - cron: "23 6 * * *"

concurrency:
  # Queue runs instead of cancelling them so each latest.json snapshot keeps its own validation signal.
  group: public-bootstrap-validation-${{ github.event_name == 'workflow_dispatch' && inputs.version || 'scheduled' }}
  cancel-in-progress: false

permissions:
  contents: read
  actions: read
```

**Linux amd64 runner and bash defaults pattern** (`.github/workflows/public-bootstrap-validation.yml` lines 53-84):

```yaml
  validate-r2:
    name: validate r2 (${{ matrix.platform_id }})
    runs-on: ${{ matrix.runner }}
    needs: resolve-version
    timeout-minutes: 20
    strategy:
      fail-fast: false
      matrix:
        include:
          - platform_id: linux-amd64
            runner: ubuntu-latest
            goos: linux
            goarch: amd64
    defaults:
      run:
        shell: bash
```

**Checkout, toolchain, cargo build, Go command pattern** (`.github/workflows/phase3-go-wrapper-smoke.yml` lines 20-37):

```yaml
      - name: Check out repository
        uses: actions/checkout@v4
        with:
          submodules: recursive

      - name: Install Rust toolchain
        uses: dtolnay/rust-toolchain@stable

      - name: Install Go toolchain
        uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: Build release library
        run: cargo build --release

      - name: Run Go wrapper race tests
        run: go test ./... -race
```

**Artifact upload pattern** (`.github/workflows/release.yml` lines 165-170):

```yaml
      - name: Upload staged linux artifact bundle
        uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02
        with:
          name: release-${{ matrix.platform_id }}
          if-no-files-found: error
          path: ${{ github.workspace }}/dist/${{ matrix.platform_id }}
```

**Apply to Phase 9:** Use `workflow_dispatch` input for snapshot label, `runs-on: ubuntu-latest`, `contents: read`, bash shell, checkout with recursive submodules, Rust + Go setup, `cargo build --release`, three `go test -bench` captures, benchstat generation, claim gate generation, and artifact upload as transport only. Do not add Pages/write permissions unless explicitly planned.

---

### `scripts/bench/check_benchmark_claims.py` (utility, transform)

**Analog:** `scripts/bench/check_phase8_tier1_improvement.py`

**Imports and constants pattern** (lines 1-24):

```python
#!/usr/bin/env python3

from __future__ import annotations

import argparse
import pathlib
import re
import statistics
import sys


MIN_DELTA_FRACTION = 0.10
METADATA_KEYS = ("goos", "goarch", "pkg", "cpu")
REQUIRED_BENCHMARKS = (
    "BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-full",
    "BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-materialize-only",
)
METADATA_RE = re.compile(r"^(goos|goarch|pkg|cpu):\s*(.+?)\s*$")
```

**Argparse pattern** (lines 30-36):

```python
def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Verify Phase 8 Tier 1 benchmark improvement against Phase 7."
    )
    parser.add_argument("--old", required=True, type=pathlib.Path)
    parser.add_argument("--new", required=True, type=pathlib.Path)
    return parser.parse_args()
```

**Benchmark parser pattern** (lines 39-65):

```python
def parse_benchmark_file(
    path: pathlib.Path,
) -> tuple[dict[str, str], dict[str, list[float]]]:
    metadata: dict[str, str] = {}
    samples: dict[str, list[float]] = {}

    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except (OSError, UnicodeDecodeError) as error:
        raise SystemExit(f"read {path}: {error}") from error

    for line in lines:
        metadata_match = METADATA_RE.match(line)
        if metadata_match is not None:
            metadata[metadata_match.group(1)] = metadata_match.group(2)
            continue

        benchmark_match = BENCHMARK_RE.match(line)
        if benchmark_match is None:
            continue

        benchmark_name = benchmark_match.group(1)
        trailing_fields = benchmark_match.group(2).split()
        ns_value = extract_ns_per_op(path, line, trailing_fields)
        samples.setdefault(benchmark_name, []).append(ns_value)

    return metadata, samples
```

**Error handling pattern** (lines 68-81):

```python
def extract_ns_per_op(path: pathlib.Path, line: str, trailing_fields: list[str]) -> float:
    for index, token in enumerate(trailing_fields):
        if token != "ns/op":
            continue
        if index == 0:
            break
        numeric_token = trailing_fields[index - 1].replace(",", "")
        try:
            return float(numeric_token)
        except ValueError as error:
            raise SystemExit(
                f"parse {path}: benchmark row has invalid ns/op value: {line}"
            ) from error
    raise SystemExit(f"parse {path}: benchmark row missing ns/op value: {line}")
```

**Fail-closed comparison pattern** (lines 121-168):

```python
for benchmark_name in REQUIRED_BENCHMARKS:
    old_values = old_samples.get(benchmark_name)
    new_values = new_samples.get(benchmark_name)

    if not old_values:
        print(
            f"FAIL {benchmark_name} old=missing new="
            f"{format_ns(statistics.median(new_values)) if new_values else 'missing'} "
            "delta=n/a reason=missing-old-row"
        )
        success = False
        continue

    if not new_values:
        print(
            f"FAIL {benchmark_name} old={format_ns(statistics.median(old_values))} "
            "new=missing delta=n/a reason=missing-new-row"
        )
        success = False
        continue
```

**Main/exit pattern** (lines 173-189):

```python
def main() -> int:
    args = parse_args()

    old_metadata, old_samples = parse_benchmark_file(args.old)
    new_metadata, new_samples = parse_benchmark_file(args.new)

    if not compare_metadata(old_metadata, new_metadata):
        return 1

    if not compare_benchmarks(old_samples, new_samples):
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
```

**Apply to Phase 9:** Keep typed functions and deterministic output. Generalize from old/new positional files to `--baseline-dir`, `--snapshot-dir`, `--snapshot`, and `--require-target`. Emit JSON to stdout for `summary.json`, but preserve fail-closed semantics for missing rows, malformed rows, target mismatch, regressions, non-significant/noisy Tier 1 headline evidence, and unsupported claim allowances.

---

### `tests/bench/test_check_benchmark_claims.py` (test, transform)

**Analog:** `tests/bench/test_check_phase8_improvement.py`

**Imports, repo root, script path pattern** (lines 1-14):

```python
#!/usr/bin/env python3

from __future__ import annotations

import pathlib
import subprocess
import sys
import tempfile
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
SCRIPT_PATH = REPO_ROOT / "scripts" / "bench" / "check_phase8_tier1_improvement.py"
```

**Synthetic benchmark text builder pattern** (lines 34-51):

```python
def build_benchmark_text(
    *,
    metadata: dict[str, str],
    samples_by_row: dict[str, list[float]],
) -> str:
    lines = [f"{key}: {value}" for key, value in metadata.items()]
    for row_name, samples in samples_by_row.items():
        for run_index, ns_value in enumerate(samples, start=1):
            lines.append(
                f"{row_name}-16\t{run_index}\t{ns_value:.2f} ns/op\t"
                f"{1000.0 / ns_value:.2f} MB/s\t0 B/op\t0 allocs/op"
            )
    lines.append("PASS")
    return "\n".join(lines) + "\n"
```

**Tempfile writer and subprocess runner pattern** (lines 54-95):

```python
class CheckPhase8ImprovementTests(unittest.TestCase):
    def write_benchmark_file(
        self,
        directory: pathlib.Path,
        name: str,
        *,
        metadata: dict[str, str] | None = None,
        samples_by_row: dict[str, list[float]] | None = None,
    ) -> pathlib.Path:
        metadata = dict(BASE_METADATA if metadata is None else metadata)
        path = directory / name
        path.write_text(
            build_benchmark_text(metadata=metadata, samples_by_row=samples_by_row),
            encoding="utf-8",
        )
        return path

    def run_script(
        self,
        *,
        old_path: pathlib.Path,
        new_path: pathlib.Path,
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(SCRIPT_PATH), "--old", str(old_path), "--new", str(new_path)],
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
            check=False,
        )
```

**Assertion style pattern** (lines 97-117 and 202-227):

```python
def test_accepts_exact_ten_percent_improvement(self) -> None:
    with tempfile.TemporaryDirectory() as temp_dir:
        directory = pathlib.Path(temp_dir)
        old_path = self.write_benchmark_file(directory, "old.bench.txt")
        new_path = self.write_benchmark_file(directory, "new.bench.txt")

        result = self.run_script(old_path=old_path, new_path=new_path)

    self.assertEqual(result.returncode, 0, result.stderr)
    self.assertIn("threshold=10.00%", result.stdout)

def test_rejects_metadata_mismatch(self) -> None:
    result = self.run_script(old_path=old_path, new_path=new_path)
    self.assertNotEqual(result.returncode, 0)
    self.assertIn("reason=metadata-mismatch", result.stdout)
```

**Apply to Phase 9:** Keep `unittest`, temp directories, direct subprocess execution, `check=False`, and stdout assertions. Add synthetic directories for baseline/snapshot, `summary.json` parsing assertions, target metadata mismatch, missing rows, malformed `ns/op`, Tier 1 non-significance/noisy fail-closed behavior, Tier 2/3 regression, and claim allowance booleans/modes.

---

### `scripts/bench/capture_release_snapshot.sh` (utility, batch, optional)

**Analog:** `scripts/bench/run_benchstat.sh`

**Shell safety and usage pattern** (lines 1-7):

```bash
#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 --old <path> --new <path>" >&2
}
```

**Manual argument parsing pattern** (lines 11-40):

```bash
while [[ $# -gt 0 ]]; do
	case "$1" in
		--old)
			shift
			if [[ $# -eq 0 ]]; then
				usage
				echo "missing value for --old" >&2
				exit 1
			fi
			old_path="$1"
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			usage
			echo "unexpected argument: $1" >&2
			exit 1
			;;
	esac
	shift
done
```

**Precondition and command execution pattern** (lines 56-71):

```bash
if [[ ! -f "$old_path" ]]; then
	echo "old benchmark file not found: $old_path" >&2
	exit 1
fi

if ! command -v benchstat >/dev/null 2>&1; then
	echo "benchstat not found; install it with: go install golang.org/x/perf/cmd/benchstat@latest" >&2
	exit 1
fi

benchstat "$old_path" "$new_path"
```

**Apply to Phase 9:** If the optional capture script is used, keep it as a thin deterministic wrapper around existing `go test -bench`, `run_benchstat.sh`, and `check_benchmark_claims.py`; do not embed public wording decisions in shell.

---

### `testdata/benchmark-results/v0.1.2/phase9.bench.txt` (testdata, batch)

**Analog:** `testdata/benchmark-results/v0.1.1/phase7.bench.txt`

**Metadata and Tier 1 row format** (lines 1-15):

```text
goos: darwin
goarch: arm64
pkg: github.com/amikos-tech/pure-simdjson
cpu: Apple M3 Max
BenchmarkTier1FullParse_twitter_json/pure-simdjson-16      	      64	  18028216 ns/op	  35.03 MB/s	         3.000 native-allocs/op	   6105064 native-bytes/op	         0 native-live-bytes	 9588967 B/op	  286521 allocs/op
BenchmarkTier1FullParse_twitter_json/encoding-json-any-16  	     292	   3945813 ns/op	 160.05 MB/s	         0 native-allocs/op	         0 native-bytes/op	         0 native-live-bytes	 2076684 B/op	   32125 allocs/op
BenchmarkTier1FullParse_twitter_json/encoding-json-struct-16         	     399	   2974678 ns/op	 212.30 MB/s	         0 native-allocs/op	         0 native-bytes/op	         0 native-live-bytes	  118056 B/op	    1205 allocs/op
```

**Tier row source pattern** (`benchmark_fullparse_test.go` lines 7-17, 24-31):

```go
func BenchmarkTier1FullParse_twitter_json(b *testing.B) {
	runTier1FullParseBenchmark(b, benchmarkFixtureTwitter)
}

func runTier1FullParseBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	for _, comparator := range availableBenchmarkComparators(b) {
		comparator := comparator
		b.Run(comparator.key, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
```

**Tier 2 comparator pattern** (`benchmark_typed_test.go` lines 22-34, 76-93):

```go
func BenchmarkTier2Typed_twitter_json(b *testing.B) {
	runTier2TypedBenchmark(b, benchmarkFixtureTwitter)
}

func availableTier2TypedComparators(tb testing.TB) []benchmarkComparator {
	tb.Helper()

	var comparators []benchmarkComparator
	for _, comparator := range availableBenchmarkComparators(tb) {
		switch comparator.key {
		case benchmarkComparatorPureSimdjson,
			benchmarkComparatorEncodingStruct,
			benchmarkComparatorBytedanceSonic,
			benchmarkComparatorGoccyGoJSON:
			comparators = append(comparators, comparator)
		}
	}
```

**Tier 3 scope pattern** (`benchmark_selective_test.go` lines 15-26):

```go
func BenchmarkTier3SelectivePlaceholder_twitter_json(b *testing.B) {
	runTier3SelectivePlaceholderBenchmark(b, benchmarkFixtureTwitter)
}

// Tier 3 remains a DOM-era placeholder benchmark. It measures selective reads
// on the current DOM API only and does not imply a new On-Demand or path-query
// surface in v0.1.
func runTier3SelectivePlaceholderBenchmark(b *testing.B, fixtureName string) {
```

**Apply to Phase 9:** Capture on real `linux/amd64`; metadata lines must say `goos: linux` and `goarch: amd64`. Preserve benchmark row names and comparator keys; do not synthesize unavailable comparator rows.

---

### `testdata/benchmark-results/v0.1.2/coldwarm.bench.txt` (testdata, batch)

**Analog:** `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt`

**Raw cold/warm evidence format** (lines 1-24):

```text
goos: darwin
goarch: arm64
pkg: github.com/amikos-tech/pure-simdjson
cpu: Apple M3 Max
BenchmarkColdStart_twitter_json-16         	    5613	    297493 ns/op	2122.79 MB/s	         8.000 native-allocs/op	   8640732 native-bytes/op	         0 native-live-bytes	     488 B/op	      23 allocs/op
BenchmarkWarm_twitter_json-16              	    6733	    245967 ns/op	2567.48 MB/s	         3.000 native-allocs/op	   6105064 native-bytes/op	         0 native-live-bytes	     328 B/op	      13 allocs/op
```

**Cold/warm source pattern** (`benchmark_coldstart_test.go` lines 31-41, 67-88):

```go
// cold-start here means first Parse after NewParser inside an already loaded process.
// It intentionally excludes bootstrap or download time.
// Both cold and warm families report native-bytes/op, native-allocs/op, and
// native-live-bytes through benchmarkRunWithNativeAllocMetrics.
func runColdStartBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	benchmarkRunWithNativeAllocMetrics(b, true, func() {
```

```go
// Warm benchmarks do one warm-up parse before ResetTimer and then reuse the parser.
func runWarmBenchmark(b *testing.B, fixtureName string) {
	data := loadBenchmarkFixture(b, fixtureName)

	parser, err := NewParser()
	if err != nil {
		b.Fatalf("NewParser(%s): %v", fixtureName, err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	benchmarkRunWithNativeAllocMetrics(b, true, func() {
```

**Apply to Phase 9:** Keep cold/warm rows separate from Tier 1/2/3 rows and keep native allocation metrics in the committed raw evidence.

---

### `testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt` (testdata, batch)

**Analog:** `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt`

**Raw diagnostic evidence format** (lines 1-12):

```text
goos: darwin
goarch: arm64
pkg: github.com/amikos-tech/pure-simdjson
cpu: Apple M3 Max
BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-full-16         	      61	  27962032 ns/op	  22.58 MB/s	         3.000 native-allocs/op	   6105064 native-bytes/op	         0 native-live-bytes	 9589193 B/op	  286522 allocs/op
BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-parse-only-16   	    4429	    289270 ns/op	2183.13 MB/s	         3.000 native-allocs/op	   6105064 native-bytes/op	         0 native-live-bytes	     328 B/op	      13 allocs/op
BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-materialize-only-16         	      39	  31103942 ns/op	  20.30 MB/s	         0 native-allocs/op	         0 native-bytes/op	         0 native-live-bytes	 9588046 B/op	  286508 allocs/op
```

**Diagnostic row source pattern** (`benchmark_diagnostics_test.go` lines 8-15, 31-56):

```go
const (
	benchmarkTier1DiagnosticsPureFull            = "pure-simdjson-full"
	benchmarkTier1DiagnosticsPureParseOnly       = "pure-simdjson-parse-only"
	benchmarkTier1DiagnosticsPureMaterializeOnly = "pure-simdjson-materialize-only"
	benchmarkTier1DiagnosticsPureStageReuse      = "pure-simdjson-stage-input-reuse-model"
	benchmarkTier1DiagnosticsPureStageAlloc      = "pure-simdjson-stage-input-alloc-model"
	benchmarkTier1DiagnosticsEncodingAnyFull     = "encoding-json-any-full"
)

// These rows intentionally isolate pieces of the steady-state Tier 1 path.
// They are diagnostic cuts, not additive accounting: materialize-only keeps one
// parsed document open across the loop so the DOM walk and string extraction
// path can be measured without parse/setup noise.
func runTier1DiagnosticsBenchmark(b *testing.B, fixtureName string) {
```

**Apply to Phase 9:** Preserve the diagnostic sub-row names exactly; these are internal explanation rows and should support docs without becoming a headline gate by themselves.

---

### `testdata/benchmark-results/v0.1.2/*.benchstat.txt` (testdata, transform)

**Analog:** `testdata/benchmark-results/phase8/tier1-diagnostics.benchstat.txt`

**Benchstat output format** (lines 1-12):

```text
goos: darwin
goarch: arm64
pkg: github.com/amikos-tech/pure-simdjson
cpu: Apple M3 Max
                                                                            | testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt | testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt |
                                                                            |                            sec/op                             |                sec/op                 vs base                 |
Tier1Diagnostics_twitter_json/pure-simdjson-full-16                                                                          27962.0us +/- inf                           861.7us +/- inf        ~ (p=0.333 n=1+5)
Tier1Diagnostics_twitter_json/pure-simdjson-parse-only-16                                                                      289.3us +/- inf                           177.5us +/- inf        ~ (p=0.333 n=1+5)
```

**Wrapper pattern** (`scripts/bench/run_benchstat.sh` lines 66-71):

```bash
if ! command -v benchstat >/dev/null 2>&1; then
	echo "benchstat not found; install it with: go install golang.org/x/perf/cmd/benchstat@latest" >&2
	exit 1
fi

benchstat "$old_path" "$new_path"
```

**Apply to Phase 9:** Generate separate benchstat files for main Tier 1/2/3, cold/warm, and diagnostics. Use committed `v0.1.1` as baseline and new `v0.1.2` evidence as snapshot. The claim gate must not rely only on median ratios when Tier 1 headline significance is required.

---

### `testdata/benchmark-results/v0.1.2/summary.json` (testdata, transform)

**Analog:** `testdata/benchmark-results/phase8/tier1-diagnostics.improvement.txt` for fail/pass statuses; `scripts/bench/check_phase8_tier1_improvement.py` for generation.

**Prior machine gate output shape** (lines 1-6):

```text
PASS BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-full old=27962032.00 new=861742.00 delta=96.92% threshold=10.00%
PASS BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-materialize-only old=31103942.00 new=678443.00 delta=97.82% threshold=10.00%
PASS BenchmarkTier1Diagnostics_citm_catalog_json/pure-simdjson-full old=75919203.00 new=2349985.00 delta=96.90% threshold=10.00%
```

**Metadata compare pattern** (`scripts/bench/check_phase8_tier1_improvement.py` lines 92-113):

```python
def print_metadata_mismatch(
    *,
    key: str,
    old_value: str | None,
    new_value: str | None,
) -> None:
    normalized_old = "missing" if old_value is None else old_value
    normalized_new = "missing" if new_value is None else new_value
    print(
        f"FAIL metadata old_{key}={normalized_old} "
        f"new_{key}={normalized_new} reason=metadata-mismatch"
    )

def compare_metadata(old_metadata: dict[str, str], new_metadata: dict[str, str]) -> bool:
    for key in METADATA_KEYS:
        old_value = old_metadata.get(key)
        new_value = new_metadata.get(key)
        if old_value != new_value:
            print_metadata_mismatch(key=key, old_value=old_value, new_value=new_value)
            return False
    return True
```

**Apply to Phase 9:** JSON should include snapshot label, target/toolchain metadata, thresholds, per-fixture statuses, ratios, and allowed public claim modes. Unlike the Phase 8 text file, this file should be deterministic JSON emitted by `check_benchmark_claims.py`.

---

### `docs/benchmarks.md` (documentation, request-response)

**Analog:** `docs/benchmarks.md`

**Current snapshot pointer pattern** (lines 1-3):

```markdown
# Benchmark Methodology

This project publishes benchmark results from committed `go test -bench` output under [testdata/benchmark-results/v0.1.1](../testdata/benchmark-results/v0.1.1). The current public snapshot is [results-v0.1.1.md](benchmarks/results-v0.1.1.md).
```

**Tier and comparator rules pattern** (lines 5-15):

```markdown
## Tier Definitions

- Tier 1: `BenchmarkTier1FullParse_*` measures full parse plus full Go `any` materialization. For `pure-simdjson`, this means parse the document, walk the DOM, and recursively build `map[string]any`, `[]any`, and scalar Go values. This is the strict full-materialization parity benchmark and the current worst-case workload for the DOM API.
- Tier 2: `BenchmarkTier2Typed_*` measures schema-shaped typed extraction using the current public API. It reflects the intended `[]byte -> Doc -> typed accessors` path much better than Tier 1.
- Tier 3: `BenchmarkTier3SelectivePlaceholder_*` measures selective reads on the current DOM API only. It is a DOM-era placeholder benchmark, not a shipped On-Demand or path-query API.

## Comparator Rules

- Comparator tables only include libraries that actually run on that exact target/toolchain combination.
- Unsupported comparators are omitted from that target table instead of being rendered as `N/A`.
```

**Rerun command pattern** (lines 29-43):

````markdown
## Rerun Commands

Capture the main benchmark snapshot:

```sh
go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchmem -count=5 > testdata/benchmark-results/v0.1.1/phase7.bench.txt
go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=5 > testdata/benchmark-results/v0.1.1/coldwarm.bench.txt
go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=1 > testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt
```

Compare two benchmark captures with `benchstat`:

```sh
./scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/phase7.bench.txt --new /path/to/new-phase7.bench.txt
```
````

**Apply to Phase 9:** Update snapshot path to `v0.1.2`, command `-count=10` where planned, and include claim gate command. Keep methodology page as the pointer owner from README to current result snapshot.

---

### `docs/benchmarks/results-v0.1.2.md` (documentation, request-response)

**Analog:** `docs/benchmarks/results-v0.1.1.md`

**Status block pattern** (lines 1-11):

```markdown
# Benchmark Results v0.1.1

This snapshot records the current Phase 7 benchmark/docs/legal baseline after the steady-state harness fixes, parser input-buffer reuse, and Tier 1 diagnostic split were added. It is intentionally a truthful evidence snapshot, not a forced release gate.

## Status

BENCH-07 truthful-positioning: PASS
Tier 1 headline on current DOM ABI: NOT SUPPORTED
Tier 2/Tier 3 headline on current DOM ABI: SUPPORTED
x86_64 minio parity on this snapshot: UNAVAILABLE
```

**Target/evidence pattern** (lines 12-23):

```markdown
## Target and Raw Evidence

- `darwin/arm64` on `Apple M3 Max`
  - Go toolchain: `go1.26.2`
  - Rust toolchain: `rustc 1.89.0 (29483883e 2025-08-04)`
  - OS: `macOS 26.4.1 (25E253)`
  - Tier 1/2/3 evidence: `testdata/benchmark-results/v0.1.1/phase7.bench.txt`
  - Cold/warm evidence: `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt`
  - Tier 1 diagnostics: `testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt`
```

**Tier table and interpretation pattern** (lines 25-35):

```markdown
## Tier 1: Full Parse + Full `any` Materialization

Median `ns/op` from `testdata/benchmark-results/v0.1.1/phase7.bench.txt`:

| Fixture | pure-simdjson | encoding/json + any | Relative to stdlib |
| --- | ---: | ---: | ---: |
| `twitter.json` | `18,028,216` | `3,838,594` | `0.21x` |

On the current DOM ABI, Tier 1 is still dominated by building a full generic Go tree rather than by raw parse throughput.
```

**Tier 2/3 and cold/warm table pattern** (lines 49-82):

```markdown
## Tier 2: Typed Extraction

Median `ns/op` from `testdata/benchmark-results/v0.1.1/phase7.bench.txt`:

| Fixture | pure-simdjson | encoding/json + struct | bytedance/sonic | Speedup vs stdlib |
| --- | ---: | ---: | ---: | ---: |

## Cold vs Warm Parser Lifecycle

Median `ns/op` and native allocation profile from `testdata/benchmark-results/v0.1.1/coldwarm.bench.txt`:

| Fixture | cold-start median | warm median | cold native allocs/op | warm native allocs/op |
| --- | ---: | ---: | ---: | ---: |
```

**Apply to Phase 9:** Create release-scoped result doc, not phase-scoped. Populate status from `summary.json`, include `linux/amd64` target metadata, link raw/benchstat/summary files, keep full comparator tables here, and avoid claims not unlocked by the gate.

---

### `README.md` (documentation, request-response)

**Analog:** `README.md`

**Benchmark section pattern** (lines 61-67):

```markdown
## Benchmark Snapshot

The current benchmark evidence is published in [results-v0.1.1.md](docs/benchmarks/results-v0.1.1.md) with the methodology in [benchmarks.md](docs/benchmarks.md). Comparator tables omit unsupported libraries on a given target instead of showing synthetic `N/A` rows.

Tier 1 is a strict full `any` materialization benchmark, and on the current `darwin/arm64` DOM ABI it is still slower than `encoding/json` for the three published corpus files: `0.21x` on `twitter.json`, `0.20x` on `citm_catalog.json`, and `0.17x` on `canada.json`. The current strength story is Tier 2 typed extraction and Tier 3 selective traversal on the DOM API, where the same snapshot shows `10.08x` to `14.52x` wins over `encoding/json` struct decoding in Tier 2 and `15.19x` to `20.05x` wins in the Tier 3 placeholder rows.

Use Tier 1 as the worst-case "parse and build a full generic Go tree" reference point. Use Tier 2 and Tier 3 as the representative performance story for the current API. Bootstrap and install details remain in [docs/bootstrap.md](docs/bootstrap.md).
```

**Apply to Phase 9:** README should link to `docs/benchmarks.md` and mention only stdlib-relative ratios. Do not include full comparator tables in README. If Phase 09.1 has not validated default installs, frame strong results as upcoming-release evidence rather than default-install release completion.

---

### `CHANGELOG.md` (documentation, event-driven)

**Analog:** `CHANGELOG.md`

**Keep-a-Changelog header and Unreleased pattern** (lines 1-12):

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Benchmark harness coverage for Tier 1 full materialization, Tier 2 typed extraction, Tier 3 selective placeholder reads, cold/warm parser lifecycle, and the JSONTestSuite correctness oracle.
- Committed benchmark evidence under `testdata/benchmark-results/v0.1.1/`, including `phase7.bench.txt`, `coldwarm.bench.txt`, and `tier1-diagnostics.bench.txt`.
```

**Changed/Documentation pattern** (lines 19-25):

```markdown
### Changed
- Releases now follow the org-standard tag-driven CI publish flow shared with `pure-onnx` and `pure-tokenizers`.
- Benchmark positioning now treats Tier 1 as the full-`any` worst-case workload on the current DOM ABI, while deferring ABI-level Tier 1 improvement and any new public patch-release decision to Phase 8 and Phase 9.

### Documentation
- Added a consumer-facing `README.md` with installation, quick start, supported platforms, and a benchmark snapshot linked to `docs/benchmarks/results-v0.1.1.md`.
- Added `docs/benchmarks.md` to document tier definitions, omission rules, cold-start semantics, native allocator metrics, and rerun commands.
```

**Apply to Phase 9:** Add concise Unreleased entries for `v0.1.2` benchmark capture, claim gate, and benchmark positioning. Do not claim a release was published or bootstrap artifacts aligned; Phase 09.1 owns that.

## Shared Patterns

### Stable Benchmark Row Names

**Source:** `benchmark_comparators_test.go` lines 14-37; `benchmark_fullparse_test.go` lines 7-17; `benchmark_typed_test.go` lines 22-31; `benchmark_selective_test.go` lines 15-20; `benchmark_coldstart_test.go` lines 7-28.

```go
const (
	benchmarkFixtureTwitter = "twitter.json"
	benchmarkFixtureCITM    = "citm_catalog.json"
	benchmarkFixtureCanada  = "canada.json"
)

const (
	benchmarkComparatorEncodingAny    = "encoding-json-any"
	benchmarkComparatorEncodingStruct = "encoding-json-struct"
	benchmarkComparatorMinioSimdjson  = "minio-simdjson-go"
	benchmarkComparatorBytedanceSonic = "bytedance-sonic"
	benchmarkComparatorGoccyGoJSON    = "goccy-go-json"
)
```

Apply to all raw evidence, gate parsing, docs tables, and README ratios. Do not rename rows during Phase 9.

### Comparator Omission

**Source:** `benchmark_comparators_test.go` lines 56-58, 107-123, 144-159.

```go
func (c benchmarkComparator) available() bool {
	return c.materialize != nil && c.omissionReason == ""
}

func registerOmittedBenchmarkComparator(key, reason string) {
	if key == "" {
		panic("benchmark comparator key must not be empty")
	}
	if reason == "" {
		panic(fmt.Sprintf("benchmark comparator %q omission reason must not be empty", key))
	}
```

Apply to benchmark docs and result tables: omit unsupported comparators instead of rendering fake `N/A`.

### Native Allocation Metrics

**Source:** `benchmark_native_alloc_test.go` lines 5-11, 22-44.

```go
const (
	benchmarkMetricNativeBytesPerOp  = "native-bytes/op"
	benchmarkMetricNativeAllocsPerOp = "native-allocs/op"
	benchmarkMetricNativeLiveBytes   = "native-live-bytes"
)

func benchmarkRunWithNativeAllocMetrics(b *testing.B, requireNativeAllocs bool, run func()) {
	b.Helper()
```

```go
b.ResetTimer()
run()
b.StopTimer()

stats, rc := library.bindings.NativeAllocStatsSnapshot()
if err := wrapStatus(rc); err != nil {
	b.Fatalf("NativeAllocStatsSnapshot(): %v", err)
}
```

Apply to result docs and evidence validation; native metrics are part of the published benchmark shape.

### Fail-Closed Gate Output

**Source:** `scripts/bench/check_phase8_tier1_improvement.py` lines 125-168.

```python
if not new_values:
    print(
        f"FAIL {benchmark_name} old={format_ns(statistics.median(old_values))} "
        "new=missing delta=n/a reason=missing-new-row"
    )
    success = False
    continue

if new_median > old_median:
    print(
        f"FAIL {benchmark_name} old={format_ns(old_median)} "
        f"new={format_ns(new_median)} delta={format_delta(delta_fraction)} "
        "reason=regressed"
    )
    success = False
    continue
```

Apply to Tier 1/2/3 claim allowances. Noisy or incomplete evidence should produce conservative README/docs modes.

### Release Boundary

**Source:** `.agents/skills/pure-simdjson-release/SKILL.md` lines 11-25.

```markdown
1. Read `docs/releases.md` before suggesting any release action.
2. Run `bash scripts/release/check_readiness.sh --strict --version <semver-without-v>` before recommending a tag push.
3. Treat `main -> annotated tag -> CI publish` as the supported sequencing.

## Constraints

- CI is the only publish path.
- Do not hand-upload artifacts.
- Do not bypass CI publication.
```

Apply to docs/changelog wording. Phase 9 may recommend benchmark positioning, but must not imply tag publication, CI release completion, or bootstrap artifact alignment.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `testdata/benchmark-results/v0.1.2/summary.json` | testdata | transform | No existing committed JSON benchmark claim summary exists. Use `check_phase8_tier1_improvement.py` for parsing/fail-closed behavior and `tier1-diagnostics.improvement.txt` for status vocabulary, but define the JSON schema from Phase 9 decisions D-12 and D-22. |

## Metadata

**Analog search scope:** `.github/workflows/`, `scripts/bench/`, `tests/bench/`, `docs/`, `testdata/benchmark-results/`, `benchmark_*_test.go`, `.agents/skills/`
**Files scanned:** 24 targeted files, 1155 repository files available by `rg --files`
**Pattern extraction date:** 2026-04-24
