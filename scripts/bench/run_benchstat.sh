#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 --old <path> --new <path>" >&2
}

old_path=""
new_path=""

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
		--new)
			shift
			if [[ $# -eq 0 ]]; then
				usage
				echo "missing value for --new" >&2
				exit 1
			fi
			new_path="$1"
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

if [[ -z "$old_path" ]]; then
	usage
	echo "--old is required" >&2
	exit 1
fi

if [[ -z "$new_path" ]]; then
	usage
	echo "--new is required" >&2
	exit 1
fi

if [[ ! -f "$old_path" ]]; then
	echo "old benchmark file not found: $old_path" >&2
	exit 1
fi

if [[ ! -f "$new_path" ]]; then
	echo "new benchmark file not found: $new_path" >&2
	exit 1
fi

if ! command -v benchstat >/dev/null 2>&1; then
	echo "benchstat not found; install it with: go install golang.org/x/perf/cmd/benchstat@latest" >&2
	exit 1
fi

benchstat "$old_path" "$new_path"
