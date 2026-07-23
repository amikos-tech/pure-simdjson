#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
output_file=$(mktemp "${TMPDIR:-/tmp}/pure-simdjson-spike-002-output.XXXXXX")
trap 'rm -f "$output_file"' EXIT

"$script_dir/run.sh" | tee "$output_file"
python3 "$script_dir/verify.py" < "$output_file"
