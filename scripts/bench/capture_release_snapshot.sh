#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 [--snapshot <label>] [--out-dir <path>] [--baseline-dir <path>]" >&2
}

snapshot="v0.1.2"
out_dir="testdata/benchmark-results/v0.1.2"
baseline_dir="testdata/benchmark-results/v0.1.1-linux-amd64"

while [[ $# -gt 0 ]]; do
	case "$1" in
		--snapshot)
			shift
			if [[ $# -eq 0 ]]; then
				usage
				echo "missing value for --snapshot" >&2
				exit 1
			fi
			snapshot="$1"
			;;
		--out-dir)
			shift
			if [[ $# -eq 0 ]]; then
				usage
				echo "missing value for --out-dir" >&2
				exit 1
			fi
			out_dir="$1"
			;;
		--baseline-dir)
			shift
			if [[ $# -eq 0 ]]; then
				usage
				echo "missing value for --baseline-dir" >&2
				exit 1
			fi
			baseline_dir="$1"
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

if [[ ! "$snapshot" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.][A-Za-z0-9.-]+)?$ ]]; then
	echo "snapshot must match v<major>.<minor>.<patch>[-suffix], got: $snapshot" >&2
	exit 1
fi

if ! command -v benchstat >/dev/null 2>&1; then
	echo "benchstat not found; install it with: go install golang.org/x/perf/cmd/benchstat@latest" >&2
	exit 1
fi

out_parent="$(dirname "$out_dir")"
out_base="$(basename "$out_dir")"
mkdir -p "$out_parent"
stage_dir="$(mktemp -d "${out_parent}/.${out_base}.tmp.XXXXXX")"
complete_snapshot="false"
current_step="initialization"

cleanup() {
	if [[ -n "${stage_dir:-}" && -d "$stage_dir" ]]; then
		rm -rf "$stage_dir"
	fi
}
trap cleanup EXIT

on_error() {
	status=$?
	echo "benchmark capture failed during: ${current_step}" >&2
	if [[ -n "${stage_dir:-}" && -d "$stage_dir" ]]; then
		echo "partial staged files:" >&2
		find "$stage_dir" -maxdepth 1 -type f -print | sort >&2 || true
		for file in "$stage_dir"/*.bench.txt "$stage_dir"/*.benchstat.txt "$stage_dir"/summary.json; do
			if [[ -f "$file" ]]; then
				echo "--- tail: $file ---" >&2
				tail -n 80 "$file" >&2 || true
			fi
		done
	fi
	exit "$status"
}
trap on_error ERR

promote_stage() {
	rm -rf "$out_dir"
	mv "$stage_dir" "$out_dir"
	stage_dir=""
}

phase9_bench="$stage_dir/phase9.bench.txt"
coldwarm_bench="$stage_dir/coldwarm.bench.txt"
diagnostics_bench="$stage_dir/tier1-diagnostics.bench.txt"

# Canonical Phase 9 command:
# go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/phase9.bench.txt
current_step="Tier 1/2/3 benchmark capture"
go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchmem -count=10 -timeout 1200s >"$phase9_bench"

# Canonical Phase 9 command:
# go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/coldwarm.bench.txt
current_step="cold/warm benchmark capture"
go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=10 -timeout 1200s >"$coldwarm_bench"

# Canonical Phase 9 command:
# go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt
current_step="Tier 1 diagnostics benchmark capture"
go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=10 -timeout 1200s >"$diagnostics_bench"

current_step="old/new phase9 benchstat"
scripts/bench/run_benchstat.sh --old "$baseline_dir/phase7.bench.txt" --new "$phase9_bench" >"$stage_dir/phase9.benchstat.txt"
current_step="cold/warm benchstat"
scripts/bench/run_benchstat.sh --old "$baseline_dir/coldwarm.bench.txt" --new "$coldwarm_bench" >"$stage_dir/coldwarm.benchstat.txt"
current_step="Tier 1 diagnostics benchstat"
scripts/bench/run_benchstat.sh --old "$baseline_dir/tier1-diagnostics.bench.txt" --new "$diagnostics_bench" >"$stage_dir/tier1-diagnostics.benchstat.txt"

normalized_dir="$(mktemp -d "${out_parent}/.${out_base}.normalized.tmp.XXXXXX")"
trap 'rm -rf "$normalized_dir"; cleanup' EXIT

current_step="Tier 1 stdlib benchstat normalization"
python3 scripts/bench/prepare_stdlib_benchstat_inputs.py --source "$phase9_bench" --family tier1 --base-comparator encoding-json-any --candidate-comparator pure-simdjson --left-out "$normalized_dir/tier1-base.bench.txt" --right-out "$normalized_dir/tier1-candidate.bench.txt"
current_step="Tier 1 stdlib benchstat"
scripts/bench/run_benchstat.sh --old "$normalized_dir/tier1-base.bench.txt" --new "$normalized_dir/tier1-candidate.bench.txt" >"$stage_dir/tier1-vs-stdlib.benchstat.txt"

current_step="Tier 2 stdlib benchstat normalization"
python3 scripts/bench/prepare_stdlib_benchstat_inputs.py --source "$phase9_bench" --family tier2 --base-comparator encoding-json-struct --candidate-comparator pure-simdjson --left-out "$normalized_dir/tier2-base.bench.txt" --right-out "$normalized_dir/tier2-candidate.bench.txt"
current_step="Tier 2 stdlib benchstat"
scripts/bench/run_benchstat.sh --old "$normalized_dir/tier2-base.bench.txt" --new "$normalized_dir/tier2-candidate.bench.txt" >"$stage_dir/tier2-vs-stdlib.benchstat.txt"

current_step="Tier 3 stdlib benchstat normalization"
python3 scripts/bench/prepare_stdlib_benchstat_inputs.py --source "$phase9_bench" --family tier3 --base-comparator encoding-json-struct --candidate-comparator pure-simdjson --left-out "$normalized_dir/tier3-base.bench.txt" --right-out "$normalized_dir/tier3-candidate.bench.txt"
current_step="Tier 3 stdlib benchstat"
scripts/bench/run_benchstat.sh --old "$normalized_dir/tier3-base.bench.txt" --new "$normalized_dir/tier3-candidate.bench.txt" >"$stage_dir/tier3-vs-stdlib.benchstat.txt"

current_step="metadata generation"
python3 - "$snapshot" "$phase9_bench" "$stage_dir/metadata.json" <<'PY'
import json
import os
import pathlib
import subprocess
import sys
from datetime import datetime, timezone

snapshot = sys.argv[1]
bench_path = pathlib.Path(sys.argv[2])
metadata_path = pathlib.Path(sys.argv[3])
raw = {}
for line in bench_path.read_text(encoding="utf-8").splitlines():
    if ":" not in line:
        continue
    key, value = line.split(":", 1)
    if key in {"goos", "goarch", "pkg", "cpu"}:
        raw[key] = value.strip()

commands = [
    "go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/phase9.bench.txt",
    "go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/coldwarm.bench.txt",
    "go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt",
]

metadata = {
    "snapshot": snapshot,
    "goos": raw.get("goos", ""),
    "goarch": raw.get("goarch", ""),
    "pkg": raw.get("pkg", ""),
    "cpu": raw.get("cpu", ""),
    "go_version": subprocess.check_output(["go", "version"], text=True).strip(),
    "rustc_version": subprocess.check_output(["rustc", "--version"], text=True).strip(),
    "commit": subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip(),
    "runner_os": os.environ.get("RUNNER_OS") or subprocess.check_output(["uname", "-s"], text=True).strip(),
    "runner_arch": os.environ.get("RUNNER_ARCH") or subprocess.check_output(["uname", "-m"], text=True).strip(),
    "captured_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "commands": commands,
}
metadata_path.write_text(json.dumps(metadata, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY

complete_snapshot="true"

current_step="benchmark claim gate"
if ! python3 scripts/bench/check_benchmark_claims.py --baseline-dir "$baseline_dir" --snapshot-dir "$stage_dir" --snapshot "$snapshot" --require-target linux/amd64 >"$stage_dir/summary.json"; then
	failed_dir="${out_dir}.failed.$(date -u +%Y%m%dT%H%M%SZ)"
	echo "benchmark claim gate failed; preserving failed snapshot at $failed_dir" >&2
	if [[ -f "$stage_dir/summary.json" ]]; then
		echo "--- summary.json ---" >&2
		cat "$stage_dir/summary.json" >&2
	fi
	if [[ "$complete_snapshot" == "true" ]]; then
		mv "$stage_dir" "$failed_dir"
		stage_dir=""
	fi
	exit 1
fi

promote_stage
