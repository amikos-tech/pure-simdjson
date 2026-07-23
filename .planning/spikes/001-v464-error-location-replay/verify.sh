#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
observations="$(mktemp "${TMPDIR:-/tmp}/pure-simdjson-spike-001-observations.XXXXXX")"
trap 'rm -f "$observations"' EXIT

bash "$script_dir/run.sh" | tee "$observations"
python3 "$script_dir/verify.py" "$observations"
