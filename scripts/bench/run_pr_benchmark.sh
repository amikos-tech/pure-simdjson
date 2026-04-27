#!/usr/bin/env bash
set -euo pipefail

PR_BENCH_REGEX='Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_(twitter|canada)_json/(pure-simdjson|encoding-json-any|encoding-json-struct)$'
PR_BENCH_COUNT=5
PR_BENCH_TIMEOUT=600s

usage() {
	echo "Usage: $0 [--baseline <path> | --no-baseline] [--out-dir <path>]" >&2
}

baseline_path=""
no_baseline="false"
out_dir="pr-bench-summary"

while [[ $# -gt 0 ]]; do
	case "$1" in
		--baseline)
			shift
			if [[ $# -eq 0 ]]; then
				usage
				echo "missing value for --baseline" >&2
				exit 1
			fi
			baseline_path="$1"
			;;
		--no-baseline)
			no_baseline="true"
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

if [[ -n "$baseline_path" && "$no_baseline" == "true" ]]; then
	usage
	echo "--baseline and --no-baseline are mutually exclusive" >&2
	exit 1
fi

if [[ -z "$baseline_path" && "$no_baseline" != "true" ]]; then
	no_baseline="${NO_BASELINE:-false}"
	if [[ "$no_baseline" != "true" && "$no_baseline" != "1" ]]; then
		usage
		echo "provide --baseline <path> or --no-baseline" >&2
		exit 1
	fi
	no_baseline="true"
fi

if [[ -n "$baseline_path" && ! -f "$baseline_path" ]]; then
	echo "baseline benchmark file not found: $baseline_path" >&2
	exit 1
fi

for tool in go python3; do
	if ! command -v "$tool" >/dev/null 2>&1; then
		echo "$tool not found in PATH" >&2
		exit 1
	fi
done

if [[ "$no_baseline" != "true" ]]; then
	if ! command -v benchstat >/dev/null 2>&1; then
		echo "benchstat not found; install it with: go install golang.org/x/perf/cmd/benchstat@latest" >&2
		exit 1
	fi
fi

out_parent="$(dirname "$out_dir")"
out_base="$(basename "$out_dir")"
mkdir -p "$out_parent"
stage_dir="$(mktemp -d "${out_parent}/.${out_base}.tmp.XXXXXX")"
current_step="initialization"

cleanup() {
	if [[ -n "${stage_dir:-}" && -d "$stage_dir" ]]; then
		rm -rf "$stage_dir"
	fi
}
trap cleanup EXIT

on_error() {
	status=$?
	echo "PR benchmark failed during: ${current_step}" >&2
	if [[ -n "${stage_dir:-}" && -d "$stage_dir" ]]; then
		echo "partial staged files:" >&2
		find "$stage_dir" -maxdepth 1 -type f -print | sort >&2 || true
	fi
	exit "$status"
}
trap on_error ERR

head_bench="$stage_dir/head.bench.txt"
summary_json="$stage_dir/summary.json"
markdown_md="$stage_dir/markdown.md"

current_step="PR benchmark capture"
go test ./... -run '^$' -bench "$PR_BENCH_REGEX" -benchmem -count="$PR_BENCH_COUNT" -timeout "$PR_BENCH_TIMEOUT" >"$head_bench"

if [[ "$no_baseline" == "true" ]]; then
	current_step="no-baseline regression summary"
	python3 scripts/bench/check_pr_regression.py \
		--no-baseline \
		--summary-out "$summary_json" \
		--markdown-out "$markdown_md"
else
	current_step="baseline copy"
	cp "$baseline_path" "$stage_dir/baseline.bench.txt"

	current_step="head vs baseline benchstat"
	scripts/bench/run_benchstat.sh \
		--old "$stage_dir/baseline.bench.txt" \
		--new "$head_bench" >"$stage_dir/regression.benchstat.txt"

	current_step="regression summary"
	python3 scripts/bench/check_pr_regression.py \
		--benchstat-output "$stage_dir/regression.benchstat.txt" \
		--summary-out "$summary_json" \
		--markdown-out "$markdown_md"
fi

rm -rf "$out_dir"
mv "$stage_dir" "$out_dir"
stage_dir=""
